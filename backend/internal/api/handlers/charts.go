package handlers

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/veilence/veilence-mx/backend/internal/rbac"
)

type releaseActivityPoint struct {
	Date     string `json:"date"`
	Releases int64  `json:"releases"`
}

type classificationCount struct {
	Classification string `json:"classification"`
	Count          int64  `json:"count"`
}

type registryCount struct {
	Registry string `json:"registry"`
	Count    int64  `json:"count"`
}

type alertSeverityCount struct {
	Severity string `json:"severity"`
	Count    int64  `json:"count"`
}

type releaseStatusCount struct {
	Status string `json:"status"`
	Count  int64  `json:"count"`
}

type chartData struct {
	ReleaseActivity  []releaseActivityPoint `json:"releaseActivity"`
	Classifications  []classificationCount  `json:"classifications"`
	Registries       []registryCount        `json:"registries"`
	AlertsBySeverity []alertSeverityCount   `json:"alertsBySeverity"`
	ReleaseStatuses  []releaseStatusCount   `json:"releaseStatuses"`
}

// GetChartData returns aggregated data for dashboard charts, scoped to the current org.
func (h *DashboardHandlers) GetChartData(w http.ResponseWriter, r *http.Request) {
	orgID := rbac.OrgIDFromContext(r.Context())
	ctx := r.Context()
	var data chartData

	// Parse time range from query params
	now := time.Now()
	var from, to time.Time

	if fromStr := r.URL.Query().Get("from"); fromStr != "" {
		if parsed, err := time.Parse("2006-01-02", fromStr); err == nil {
			from = parsed
		}
	}
	if toStr := r.URL.Query().Get("to"); toStr != "" {
		if parsed, err := time.Parse("2006-01-02", toStr); err == nil {
			to = parsed.Add(24*time.Hour - time.Second) // end of day
		}
	}

	// Default to last 30 days if no range specified
	if from.IsZero() {
		from = now.AddDate(0, 0, -30)
	}
	if to.IsZero() {
		to = now
	}

	// 1. Release activity — releases per day within range, scoped to org
	activityRows, err := h.Dashboard.GetReleaseActivity(ctx, orgID, from, to)
	if err != nil {
		slog.Error("failed to get release activity", "org_id", orgID, "error", err)
		respondError(w, http.StatusInternalServerError, "failed to get chart data")
		return
	}

	// Fill in missing days with 0
	dayMap := make(map[string]int64)
	for _, row := range activityRows {
		dayMap[row.Date.Format("2006-01-02")] = row.Count
	}
	for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
		key := d.Format("2006-01-02")
		data.ReleaseActivity = append(data.ReleaseActivity, releaseActivityPoint{
			Date:     key,
			Releases: dayMap[key],
		})
	}

	// 2. Classification distribution (within range), scoped to org
	classRows, err := h.Dashboard.GetClassificationDistribution(ctx, orgID, from, to)
	if err != nil {
		slog.Error("failed to get classification distribution", "org_id", orgID, "error", err)
		respondError(w, http.StatusInternalServerError, "failed to get chart data")
		return
	}

	baselineCount, err := h.Dashboard.GetBaselineCount(ctx, orgID, from, to)
	if err != nil {
		slog.Error("failed to get baseline count", "org_id", orgID, "error", err)
		respondError(w, http.StatusInternalServerError, "failed to get chart data")
		return
	}

	for _, row := range classRows {
		data.Classifications = append(data.Classifications, classificationCount{
			Classification: row.Classification,
			Count:          row.Count,
		})
	}
	if baselineCount > 0 {
		data.Classifications = append(data.Classifications, classificationCount{
			Classification: "baseline",
			Count:          baselineCount,
		})
	}
	if len(data.Classifications) == 0 {
		data.Classifications = []classificationCount{}
	}

	// 3. Registry distribution, scoped to org
	regRows, err := h.Dashboard.GetRegistryDistribution(ctx, orgID)
	if err != nil {
		slog.Error("failed to get registry distribution", "org_id", orgID, "error", err)
		respondError(w, http.StatusInternalServerError, "failed to get chart data")
		return
	}

	for _, row := range regRows {
		data.Registries = append(data.Registries, registryCount{
			Registry: row.Registry,
			Count:    row.Count,
		})
	}
	if len(data.Registries) == 0 {
		data.Registries = []registryCount{}
	}

	// 4. Alerts by severity (within range), scoped to org
	alertRows, err := h.Dashboard.GetAlertsBySeverity(ctx, orgID, from, to)
	if err != nil {
		slog.Error("failed to get alerts by severity", "org_id", orgID, "error", err)
		respondError(w, http.StatusInternalServerError, "failed to get chart data")
		return
	}

	severityMap := make(map[string]int64)
	for _, row := range alertRows {
		severityMap[row.Severity] = row.Count
	}
	for _, sev := range []string{"critical", "high", "medium", "low"} {
		data.AlertsBySeverity = append(data.AlertsBySeverity, alertSeverityCount{
			Severity: sev,
			Count:    severityMap[sev],
		})
	}

	// 5. Release statuses (within range), scoped to org
	statusRows, err := h.Dashboard.GetReleaseStatusDistribution(ctx, orgID, from, to)
	if err != nil {
		slog.Error("failed to get release status distribution", "org_id", orgID, "error", err)
		respondError(w, http.StatusInternalServerError, "failed to get chart data")
		return
	}

	for _, row := range statusRows {
		data.ReleaseStatuses = append(data.ReleaseStatuses, releaseStatusCount{
			Status: row.Status,
			Count:  row.Count,
		})
	}
	if len(data.ReleaseStatuses) == 0 {
		data.ReleaseStatuses = []releaseStatusCount{}
	}

	respondJSON(w, http.StatusOK, data, nil)
}
