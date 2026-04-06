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
		query = query.Where("packages.name ILIKE ?", "%"+search+"%")
	}

	var total int64
	query.Count(&total)

	var releases []models.Release
	query.Preload("Package").
		Order(sortOrder).
		Offset((page - 1) * limit).
		Limit(limit).
		Find(&releases)

	result := make([]recentRelease, 0, len(releases))
	for _, rel := range releases {
		rr := recentRelease{
			Release:         rel,
			PackageName:     rel.Package.Name,
			PackageRegistry: string(rel.Package.Registry),
		}

		// Fetch classification if available
		var diff models.Diff
		if tx := h.DB.Where("release_id = ?", rel.ID).Limit(1).Find(&diff); tx.RowsAffected > 0 {
			var analysis models.Analysis
			if tx := h.DB.Where("diff_id = ?", diff.ID).Limit(1).Find(&analysis); tx.RowsAffected > 0 {
				rr.Classification = string(analysis.Classification)
			}
		} else if rel.Status == models.ReleaseStatusCompleted {
			// Completed release with no diff is a baseline (first tracked version)
			rr.Classification = "baseline"
		}

		// Apply classification filter after computing classification
		classificationFilter := r.URL.Query().Get("classification")
		if classificationFilter != "" && rr.Classification != classificationFilter {
			continue
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
