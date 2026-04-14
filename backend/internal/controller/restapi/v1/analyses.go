package v1

import (
	"log/slog"
	"net/http"

	"github.com/veilence/veilence-mx/backend/internal/controller/restapi/v1/response"
	"github.com/veilence/veilence-mx/backend/internal/entity"
	"github.com/veilence/veilence-mx/backend/internal/usecase/rbac"
)

// GetDashboardStats returns overview statistics for the dashboard, scoped to the current org.
func (h *DashboardHandlers) GetDashboardStats(w http.ResponseWriter, r *http.Request) {
	orgID := rbac.OrgIDFromContext(r.Context())

	stats, err := h.DashboardSvc.GetStats(r.Context(), orgID)
	if err != nil {
		slog.Error("failed to get dashboard stats", "org_id", orgID, "error", err)
		respondError(w, http.StatusInternalServerError, "failed to get dashboard stats")
		return
	}

	respondJSON(w, http.StatusOK, response.DashboardStatsFromEntity(stats), nil)
}

// GetRecentReleases returns the most recent releases across all packages, scoped to the current org.
func (h *DashboardHandlers) GetRecentReleases(w http.ResponseWriter, r *http.Request) {
	orgID := rbac.OrgIDFromContext(r.Context())

	page, limit := parsePagination(r)
	sortOrder := parseSort(r, map[string]string{
		"createdAt":        "releases.created_at",
		"version":          "releases.version",
		"publishedAt":      "releases.published_at",
		"status":           "releases.status",
		"packageName":      "packages.name",
		"packageEcosystem": "packages.ecosystem",
	}, "releases.created_at DESC")

	var filters entity.ReleaseFilters
	if eco := r.URL.Query().Get("ecosystem"); eco != "" {
		e := entity.Ecosystem(eco)
		filters.Ecosystem = &e
	}
	if st := r.URL.Query().Get("status"); st != "" {
		s := entity.ReleaseStatus(st)
		filters.Status = &s
	}
	if search := r.URL.Query().Get("search"); search != "" {
		filters.Search = &search
	}
	if r.URL.Query().Get("latest_per_package") == "true" {
		filters.LatestPerPackage = true
	}
	if cls := r.URL.Query().Get("classification"); cls != "" {
		filters.Classification = &cls
	}

	releases, total, err := h.DashboardSvc.GetRecentReleases(r.Context(), orgID, page, limit, sortOrder, filters)
	if err != nil {
		slog.Error("failed to load recent releases", "org_id", orgID, "error", err)
		respondError(w, http.StatusInternalServerError, "failed to load releases")
		return
	}

	respondJSON(w, http.StatusOK, response.RecentReleasesFromEntities(releases), &Meta{Page: page, Limit: limit, Total: total})
}

// ReanalyzeAll re-queues all un-analyzed diffs for analysis, scoped to the current org.
func (h *DashboardHandlers) ReanalyzeAll(w http.ResponseWriter, r *http.Request) {
	orgID := rbac.OrgIDFromContext(r.Context())

	queued, err := h.DashboardSvc.ReanalyzeAll(r.Context(), orgID)
	if err != nil {
		slog.Error("failed to reanalyze all", "org_id", orgID, "error", err)
		respondError(w, http.StatusInternalServerError, "failed to enqueue analysis jobs")
		return
	}

	respondJSON(w, http.StatusOK, response.ReanalyzeAllResponse{
		Message: "re-analysis triggered",
		Queued:  queued,
	}, nil)
}
