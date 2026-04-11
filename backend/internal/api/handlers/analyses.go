package handlers

import (
	"log/slog"
	"net/http"

	"github.com/veilence/veilence-mx/backend/internal/models"
	"github.com/veilence/veilence-mx/backend/internal/queue"
	"github.com/veilence/veilence-mx/backend/internal/rbac"
)

// GetDashboardStats returns overview statistics for the dashboard, scoped to the current org.
func (h *DashboardHandlers) GetDashboardStats(w http.ResponseWriter, r *http.Request) {
	orgID := rbac.OrgIDFromContext(r.Context())

	stats, err := h.Dashboard.GetStats(r.Context(), orgID)
	if err != nil {
		slog.Error("failed to get dashboard stats", "org_id", orgID, "error", err)
		respondError(w, http.StatusInternalServerError, "failed to get dashboard stats")
		return
	}

	respondJSON(w, http.StatusOK, stats, nil)
}

// recentRelease holds a release with its package info for the dashboard.
type recentRelease struct {
	models.Release
	PackageName     string `json:"packageName"`
	PackageRegistry string `json:"packageRegistry"`
	Classification  string `json:"classification,omitempty"`
}

// GetRecentReleases returns the most recent releases across all packages, scoped to the current org.
// NOTE: This handler still uses direct DB access for the complex paginated query with dynamic filters.
// The simpler aggregation queries have been moved to the DashboardRepository.
func (h *DashboardHandlers) GetRecentReleases(w http.ResponseWriter, r *http.Request) {
	orgID := rbac.OrgIDFromContext(r.Context())

	page, limit := parsePagination(r)
	sortOrder := parseSort(r, map[string]string{
		"createdAt":       "releases.created_at",
		"version":         "releases.version",
		"publishedAt":     "releases.published_at",
		"status":          "releases.status",
		"packageName":     "packages.name",
		"packageRegistry": "packages.registry",
	}, "releases.created_at DESC")

	registryFilter := r.URL.Query().Get("registry")
	statusFilter := r.URL.Query().Get("status")
	search := r.URL.Query().Get("search")
	latestPerPackage := r.URL.Query().Get("latest_per_package") == "true"
	classificationFilter := r.URL.Query().Get("classification")

	query := h.DB.Model(&models.Release{}).
		Joins("JOIN packages ON packages.id = releases.package_id").
		Where("packages.org_id = ?", orgID)
	if latestPerPackage {
		query = query.Where("releases.id IN (SELECT DISTINCT ON (package_id) id FROM releases ORDER BY package_id, created_at DESC)")
	}
	if registryFilter != "" {
		query = query.Where("packages.registry = ?", registryFilter)
	}
	if statusFilter != "" {
		query = query.Where("releases.status = ?", statusFilter)
	}
	if search != "" {
		search = escapeLike(search)
		query = query.Where("packages.name ILIKE ?", "%"+search+"%")
	}

	// Apply classification filter in SQL before pagination via LEFT JOINs on diffs/analyses
	if classificationFilter != "" {
		query = query.
			Joins("LEFT JOIN diffs ON diffs.release_id = releases.id").
			Joins("LEFT JOIN analyses ON analyses.diff_id = diffs.id")

		if classificationFilter == "baseline" {
			// Baseline: completed releases with no diff
			query = query.Where("releases.status = ? AND diffs.id IS NULL", models.ReleaseStatusCompleted)
		} else {
			query = query.Where("analyses.classification = ?", classificationFilter)
		}
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		slog.Error("failed to count releases", "org_id", orgID, "error", err)
		respondError(w, http.StatusInternalServerError, "failed to count releases")
		return
	}

	var releases []models.Release
	if err := query.Preload("Package").
		Order(sortOrder).
		Offset((page - 1) * limit).
		Limit(limit).
		Find(&releases).Error; err != nil {
		slog.Error("failed to load releases", "org_id", orgID, "error", err)
		respondError(w, http.StatusInternalServerError, "failed to load releases")
		return
	}

	// Batch-load diffs and analyses for the returned releases to avoid N+1
	releaseIDs := make([]uint, len(releases))
	for i, rel := range releases {
		releaseIDs[i] = rel.ID
	}

	// Load diffs for all releases in one query
	var diffs []models.Diff
	diffMap := make(map[uint]models.Diff)
	if len(releaseIDs) > 0 {
		h.DB.Where("release_id IN ?", releaseIDs).Find(&diffs)
		for _, d := range diffs {
			diffMap[d.ReleaseID] = d
		}
	}

	// Load analyses for all diffs in one query
	analysisMap := make(map[uint]models.Analysis) // keyed by diff_id
	if len(diffs) > 0 {
		diffIDs := make([]uint, len(diffs))
		for i, d := range diffs {
			diffIDs[i] = d.ID
		}
		var analyses []models.Analysis
		h.DB.Where("diff_id IN ?", diffIDs).Find(&analyses)
		for _, a := range analyses {
			analysisMap[a.DiffID] = a
		}
	}

	result := make([]recentRelease, 0, len(releases))
	for _, rel := range releases {
		rr := recentRelease{
			Release:         rel,
			PackageName:     rel.Package.Name,
			PackageRegistry: string(rel.Package.Registry),
		}

		// Look up classification from pre-loaded maps
		if diff, ok := diffMap[rel.ID]; ok {
			if analysis, ok := analysisMap[diff.ID]; ok {
				rr.Classification = string(analysis.Classification)
			}
		} else if rel.Status == models.ReleaseStatusCompleted {
			// Completed release with no diff is a baseline (first tracked version)
			rr.Classification = "baseline"
		}

		result = append(result, rr)
	}

	respondJSON(w, http.StatusOK, result, &Meta{Page: page, Limit: limit, Total: total})
}

// ReanalyzeAll re-queues all un-analyzed diffs for analysis, scoped to the current org.
// Unlike the global RetryDeadJobs queue endpoint, this only enqueues diffs that belong
// to the current organization's packages.
func (h *DashboardHandlers) ReanalyzeAll(w http.ResponseWriter, r *http.Request) {
	orgID := rbac.OrgIDFromContext(r.Context())

	if h.Queue == nil {
		respondError(w, http.StatusInternalServerError, "analysis queue not configured")
		return
	}

	// Find diffs without analysis results, scoped to current org's packages
	diffIDs, err := h.Dashboard.GetUnanalyzedDiffIDs(r.Context(), orgID)
	if err != nil {
		slog.Error("failed to get unanalyzed diff IDs", "org_id", orgID, "error", err)
		respondError(w, http.StatusInternalServerError, "failed to find unanalyzed diffs")
		return
	}

	queued := 0
	for _, id := range diffIDs {
		if _, err := h.Queue.Enqueue(r.Context(), queue.JobTypeAnalyze, id); err != nil {
			slog.Error("failed to enqueue analysis job", "diff_id", id, "org_id", orgID, "error", err)
			respondError(w, http.StatusInternalServerError, "failed to enqueue analysis job")
			return
		}
		queued++
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"message": "re-analysis triggered",
		"queued":  queued,
	}, nil)
}
