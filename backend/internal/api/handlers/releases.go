package handlers

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/veilence/veilence-mx/backend/internal/apperror"
	"github.com/veilence/veilence-mx/backend/internal/models"
	"github.com/veilence/veilence-mx/backend/internal/queue"
	"github.com/veilence/veilence-mx/backend/internal/rbac"
)

type releaseDetail struct {
	models.Release
	Diff       *models.Diff     `json:"diff,omitempty"`
	Analysis   *models.Analysis `json:"analysis,omitempty"`
	Package    *models.Package  `json:"package,omitempty"`
	IsBaseline bool             `json:"isBaseline,omitempty"`
}

// ListPackageReleases returns releases for a specific package scoped to the current org.
func (h *PackageHandlers) ListPackageReleases(w http.ResponseWriter, r *http.Request) {
	orgID := rbac.OrgIDFromContext(r.Context())

	packageID, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		respondAppError(w, apperror.BadRequest("invalid package ID"))
		return
	}

	// Verify package belongs to the requesting org
	var pkg models.Package
	if err := h.DB.Where("id = ? AND org_id = ?", packageID, orgID).First(&pkg).Error; err != nil {
		respondAppError(w, apperror.NotFound("package"))
		return
	}

	page, limit := parsePagination(r)

	var total int64
	if err := h.DB.Model(&models.Release{}).Where("package_id = ?", packageID).Count(&total).Error; err != nil {
		respondAppError(w, apperror.Internal("failed to count releases"))
		return
	}

	var releases []models.Release
	if err := h.DB.Where("package_id = ?", packageID).
		Order("created_at DESC").
		Offset((page - 1) * limit).
		Limit(limit).
		Find(&releases).Error; err != nil {
		respondAppError(w, apperror.Internal("failed to list releases"))
		return
	}

	respondJSON(w, http.StatusOK, releases, &Meta{Page: page, Limit: limit, Total: total})
}

// GetRelease returns a single release with its diff and analysis, scoped to the current org.
func (h *PackageHandlers) GetRelease(w http.ResponseWriter, r *http.Request) {
	orgID := rbac.OrgIDFromContext(r.Context())

	id, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		respondAppError(w, apperror.BadRequest("invalid release ID"))
		return
	}

	var release models.Release
	if err := h.DB.Preload("Package").First(&release, id).Error; err != nil {
		respondAppError(w, apperror.NotFound("release"))
		return
	}

	// Verify the release's package belongs to the requesting org
	if release.Package.OrgID != orgID {
		respondAppError(w, apperror.NotFound("release"))
		return
	}

	detail := releaseDetail{
		Release: release,
		Package: &release.Package,
	}

	var diff models.Diff
	if tx := h.DB.Where("release_id = ?", release.ID).Limit(1).Find(&diff); tx.RowsAffected > 0 {
		detail.Diff = &diff
		var analysis models.Analysis
		if tx := h.DB.Where("diff_id = ?", diff.ID).Limit(1).Find(&analysis); tx.RowsAffected > 0 {
			detail.Analysis = &analysis
		}
	} else if release.Status == models.ReleaseStatusCompleted {
		// Completed release with no diff is a baseline (first tracked version)
		detail.IsBaseline = true
	}

	respondJSON(w, http.StatusOK, detail, nil)
}

// ReanalyzeRelease re-queues a single release for analysis.
// POST /api/releases/{id}/reanalyze
func (h *PackageHandlers) ReanalyzeRelease(w http.ResponseWriter, r *http.Request) {
	orgID := rbac.OrgIDFromContext(r.Context())

	id, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		respondAppError(w, apperror.BadRequest("invalid release ID"))
		return
	}

	if h.Queue == nil {
		respondAppError(w, apperror.Internal("analysis queue not configured"))
		return
	}

	// Load release with package to verify org ownership
	var release models.Release
	if err := h.DB.Preload("Package").First(&release, id).Error; err != nil {
		respondAppError(w, apperror.NotFound("release"))
		return
	}

	if release.Package.OrgID != orgID {
		respondAppError(w, apperror.NotFound("release"))
		return
	}

	// Check if a diff exists for this release
	var diff models.Diff
	result := h.DB.Where("release_id = ?", release.ID).Limit(1).Find(&diff)
	if result.RowsAffected == 0 {
		// No diff yet — re-enqueue as a diff job from scratch
		h.DB.Model(&release).Update("status", models.ReleaseStatusPending)
		jobID, err := h.Queue.Enqueue(r.Context(), queue.JobTypeDiff, release.ID)
		if err != nil {
			slog.Error("failed to enqueue diff job for reanalysis", "release_id", release.ID, "error", err)
			respondAppError(w, apperror.Internal("failed to enqueue reanalysis"))
			return
		}

		respondJSON(w, http.StatusOK, map[string]any{
			"message": "release re-queued for diffing and analysis",
			"jobId":   jobID,
		}, nil)
		return
	}

	// Diff exists — re-enqueue analysis job
	h.DB.Model(&release).Update("status", models.ReleaseStatusAnalyzing)
	jobID, err := h.Queue.Enqueue(r.Context(), queue.JobTypeAnalyze, diff.ID)
	if err != nil {
		slog.Error("failed to enqueue analysis job for reanalysis", "release_id", release.ID, "diff_id", diff.ID, "error", err)
		respondAppError(w, apperror.Internal("failed to enqueue reanalysis"))
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"message": "release re-queued for analysis",
		"jobId":   jobID,
	}, nil)
}

// analysisHistoryEntry holds a single entry in the analysis history for a package.
type analysisHistoryEntry struct {
	ReleaseID      uint   `json:"releaseId"`
	Version        string `json:"version"`
	Classification string `json:"classification"`
	Confidence     float64 `json:"confidence"`
	Reasoning      string `json:"reasoning"`
	ModelUsed      string `json:"modelUsed"`
	AnalyzerType   string `json:"analyzerType"`
	AnalyzedAt     string `json:"analyzedAt"`
	PublishedAt    string `json:"publishedAt"`
}

// GetAnalysisHistory returns the analysis history for a package across all its releases.
// GET /api/packages/{id}/analysis-history
func (h *PackageHandlers) GetAnalysisHistory(w http.ResponseWriter, r *http.Request) {
	orgID := rbac.OrgIDFromContext(r.Context())

	packageID, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		respondAppError(w, apperror.BadRequest("invalid package ID"))
		return
	}

	// Verify package belongs to org
	var pkg models.Package
	if err := h.DB.Where("id = ? AND org_id = ?", packageID, orgID).First(&pkg).Error; err != nil {
		respondAppError(w, apperror.NotFound("package"))
		return
	}

	// Load all releases with their diffs and analyses
	var releases []models.Release
	if err := h.DB.Where("package_id = ?", packageID).
		Order("published_at DESC, created_at DESC").
		Find(&releases).Error; err != nil {
		respondAppError(w, apperror.Internal("failed to load releases"))
		return
	}

	if len(releases) == 0 {
		respondJSON(w, http.StatusOK, []analysisHistoryEntry{}, nil)
		return
	}

	// Batch-load diffs
	releaseIDs := make([]uint, len(releases))
	for i, rel := range releases {
		releaseIDs[i] = rel.ID
	}

	var diffs []models.Diff
	h.DB.Where("release_id IN ?", releaseIDs).Find(&diffs)
	diffByRelease := make(map[uint]models.Diff)
	for _, d := range diffs {
		diffByRelease[d.ReleaseID] = d
	}

	// Batch-load analyses
	diffIDs := make([]uint, 0, len(diffs))
	for _, d := range diffs {
		diffIDs = append(diffIDs, d.ID)
	}
	analysisByDiff := make(map[uint]models.Analysis)
	if len(diffIDs) > 0 {
		var analyses []models.Analysis
		h.DB.Where("diff_id IN ?", diffIDs).Find(&analyses)
		for _, a := range analyses {
			analysisByDiff[a.DiffID] = a
		}
	}

	// Build history entries
	result := make([]analysisHistoryEntry, 0)
	for _, rel := range releases {
		d, hasDiff := diffByRelease[rel.ID]
		if !hasDiff {
			// Baseline or pending — include with baseline classification
			if rel.Status == models.ReleaseStatusCompleted {
				result = append(result, analysisHistoryEntry{
					ReleaseID:      rel.ID,
					Version:        rel.Version,
					Classification: "baseline",
					PublishedAt:    rel.PublishedAt.Format("2006-01-02T15:04:05Z"),
				})
			}
			continue
		}

		a, hasAnalysis := analysisByDiff[d.ID]
		if !hasAnalysis {
			continue // diff exists but no analysis yet
		}

		result = append(result, analysisHistoryEntry{
			ReleaseID:      rel.ID,
			Version:        rel.Version,
			Classification: string(a.Classification),
			Confidence:     a.Confidence,
			Reasoning:      a.Reasoning,
			ModelUsed:      a.ModelUsed,
			AnalyzerType:   string(a.AnalyzerType),
			AnalyzedAt:     a.CreatedAt.Format("2006-01-02T15:04:05Z"),
			PublishedAt:    rel.PublishedAt.Format("2006-01-02T15:04:05Z"),
		})
	}

	respondJSON(w, http.StatusOK, result, nil)
}
