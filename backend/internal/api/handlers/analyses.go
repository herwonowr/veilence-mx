package handlers

import (
	"net/http"

	"github.com/veilence/veilence-mx/backend/internal/models"
	"github.com/veilence/veilence-mx/backend/internal/queue"
	"github.com/veilence/veilence-mx/backend/internal/rbac"
)

// dashboardStats holds the statistics for the dashboard.
type dashboardStats struct {
	TotalPackages   int64 `json:"totalPackages"`
	TotalReleases   int64 `json:"totalReleases"`
	PendingAnalyses int64 `json:"pendingAnalyses"`
	ActiveAlerts    int64 `json:"activeAlerts"`
	RecentMalicious int64 `json:"recentMalicious"`
}

// GetDashboardStats returns overview statistics for the dashboard, scoped to the current org.
func (h *Handlers) GetDashboardStats(w http.ResponseWriter, r *http.Request) {
	orgID := rbac.OrgIDFromContext(r.Context())
	var stats dashboardStats

	h.DB.Model(&models.Package{}).Where("org_id = ?", orgID).Count(&stats.TotalPackages)
	h.DB.Model(&models.Release{}).
		Joins("JOIN packages ON packages.id = releases.package_id").
		Where("packages.org_id = ?", orgID).
		Count(&stats.TotalReleases)
	h.DB.Model(&models.Release{}).
		Joins("JOIN packages ON packages.id = releases.package_id").
		Where("packages.org_id = ? AND releases.status IN ?", orgID, []string{"pending", "diffing", "analyzing"}).
		Count(&stats.PendingAnalyses)
	h.DB.Model(&models.Alert{}).Where("org_id = ? AND status = ?", orgID, "new").Count(&stats.ActiveAlerts)
	h.DB.Model(&models.Analysis{}).
		Joins("JOIN diffs ON diffs.id = analyses.diff_id").
		Joins("JOIN releases ON releases.id = diffs.release_id").
		Joins("JOIN packages ON packages.id = releases.package_id").
		Where("packages.org_id = ? AND analyses.classification = ?", orgID, "malicious").
		Count(&stats.RecentMalicious)

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
func (h *Handlers) GetRecentReleases(w http.ResponseWriter, r *http.Request) {
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
func (h *Handlers) ReanalyzeAll(w http.ResponseWriter, r *http.Request) {
	orgID := rbac.OrgIDFromContext(r.Context())

	if h.Queue == nil {
		respondError(w, http.StatusInternalServerError, "analysis queue not configured")
		return
	}

	// Find diffs without analysis results, scoped to current org's packages
	var diffIDs []uint
	h.DB.Model(&models.Diff{}).
		Select("diffs.id").
		Joins("JOIN releases ON releases.id = diffs.release_id").
		Joins("JOIN packages ON packages.id = releases.package_id").
		Joins("LEFT JOIN analyses ON analyses.diff_id = diffs.id").
		Where("packages.org_id = ? AND analyses.id IS NULL", orgID).
		Pluck("diffs.id", &diffIDs)

	queued := 0
	for _, id := range diffIDs {
		if _, err := h.Queue.Enqueue(r.Context(), queue.JobTypeAnalyze, id); err != nil {
			respondError(w, http.StatusInternalServerError, "failed to enqueue: "+err.Error())
			return
		}
		queued++
	}

	// Also retry dead-letter jobs
	deadRetried, _ := h.Queue.RequeueAllDead(r.Context(), queue.JobTypeAnalyze)

	respondJSON(w, http.StatusOK, map[string]any{
		"message":      "re-analysis triggered",
		"queued":       queued,
		"dead_retried": deadRetried,
	}, nil)
}
