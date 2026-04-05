package handlers

import (
	"net/http"
	"time"

	"github.com/veilence/veilence-mx/backend/internal/models"
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
func (h *Handlers) GetChartData(w http.ResponseWriter, r *http.Request) {
	orgID := rbac.OrgIDFromContext(r.Context())
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
	var activityRows []struct {
		Date  time.Time
		Count int64
	}
	h.DB.Model(&models.Release{}).
		Joins("JOIN packages ON packages.id = releases.package_id").
		Select("DATE(releases.created_at) as date, COUNT(*) as count").
		Where("packages.org_id = ? AND releases.created_at >= ? AND releases.created_at <= ?", orgID, from, to).
		Group("DATE(releases.created_at)").
		Order("date ASC").
		Scan(&activityRows)

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
	var classRows []struct {
		Classification string
		Count          int64
	}
	h.DB.Model(&models.Analysis{}).
		Joins("JOIN diffs ON diffs.id = analyses.diff_id").
		Joins("JOIN releases ON releases.id = diffs.release_id").
		Joins("JOIN packages ON packages.id = releases.package_id").
		Select("analyses.classification, COUNT(*) as count").
		Where("packages.org_id = ? AND analyses.created_at >= ? AND analyses.created_at <= ?", orgID, from, to).
		Group("analyses.classification").
		Scan(&classRows)

	// Also count baselines (completed releases with no diff, within range), scoped to org
	var baselineCount int64
	h.DB.Model(&models.Release{}).
		Joins("JOIN packages ON packages.id = releases.package_id").
		Where("packages.org_id = ? AND releases.status = ? AND releases.id NOT IN (SELECT release_id FROM diffs) AND releases.created_at >= ? AND releases.created_at <= ?", orgID, models.ReleaseStatusCompleted, from, to).
		Count(&baselineCount)

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
	var regRows []struct {
		Registry string
		Count    int64
	}
	h.DB.Model(&models.Package{}).
		Select("registry, COUNT(*) as count").
		Where("org_id = ?", orgID).
		Group("registry").
		Scan(&regRows)
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
	var alertRows []struct {
		Severity string
		Count    int64
	}
	h.DB.Model(&models.Alert{}).
		Select("severity, COUNT(*) as count").
		Where("org_id = ? AND created_at >= ? AND created_at <= ?", orgID, from, to).
		Group("severity").
		Scan(&alertRows)

	// Ensure all severities appear
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
	var statusRows []struct {
		Status string
		Count  int64
	}
	h.DB.Model(&models.Release{}).
		Joins("JOIN packages ON packages.id = releases.package_id").
		Select("releases.status, COUNT(*) as count").
		Where("packages.org_id = ? AND releases.created_at >= ? AND releases.created_at <= ?", orgID, from, to).
		Group("releases.status").
		Scan(&statusRows)
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
