package response

import "github.com/veilence/veilence-mx/backend/internal/entity"

// DashboardStatsResponse is the JSON representation of dashboard overview statistics.
type DashboardStatsResponse struct {
	TotalPackages   int64 `json:"totalPackages"`
	TotalReleases   int64 `json:"totalReleases"`
	PendingAnalyses int64 `json:"pendingAnalyses"`
	ActiveAlerts    int64 `json:"activeAlerts"`
	RecentMalicious int64 `json:"recentMalicious"`
}

// DashboardStatsFromEntity maps a domain DashboardStats to a response DTO.
func DashboardStatsFromEntity(s *entity.DashboardStats) DashboardStatsResponse {
	return DashboardStatsResponse{
		TotalPackages:   s.TotalPackages,
		TotalReleases:   s.TotalReleases,
		PendingAnalyses: s.PendingAnalyses,
		ActiveAlerts:    s.ActiveAlerts,
		RecentMalicious: s.RecentMalicious,
	}
}

// ChartReleaseActivityPointResponse is a single point on the release activity chart.
type ChartReleaseActivityPointResponse struct {
	Date     string `json:"date"`
	Releases int64  `json:"releases"`
}

// ChartClassificationCountResponse is a single bar in the classification chart.
type ChartClassificationCountResponse struct {
	Classification string `json:"classification"`
	Count          int64  `json:"count"`
}

// ChartEcosystemCountResponse is a single slice in the ecosystem pie chart.
type ChartEcosystemCountResponse struct {
	Ecosystem string `json:"ecosystem"`
	Count     int64  `json:"count"`
}

// ChartAlertSeverityCountResponse is a single bar in the alerts-by-severity chart.
type ChartAlertSeverityCountResponse struct {
	Severity string `json:"severity"`
	Count    int64  `json:"count"`
}

// ChartReleaseStatusCountResponse is a single bar in the release status chart.
type ChartReleaseStatusCountResponse struct {
	Status string `json:"status"`
	Count  int64  `json:"count"`
}

// ChartDataResponse is the full dashboard chart data payload.
type ChartDataResponse struct {
	ReleaseActivity  []ChartReleaseActivityPointResponse `json:"releaseActivity"`
	Classifications  []ChartClassificationCountResponse  `json:"classifications"`
	Ecosystems       []ChartEcosystemCountResponse       `json:"ecosystems"`
	AlertsBySeverity []ChartAlertSeverityCountResponse   `json:"alertsBySeverity"`
	ReleaseStatuses  []ChartReleaseStatusCountResponse   `json:"releaseStatuses"`
}

// ChartDataFromEntity maps a domain ChartData to a response DTO.
func ChartDataFromEntity(d *entity.ChartData) ChartDataResponse {
	resp := ChartDataResponse{}

	resp.ReleaseActivity = make([]ChartReleaseActivityPointResponse, len(d.ReleaseActivity))
	for i, p := range d.ReleaseActivity {
		resp.ReleaseActivity[i] = ChartReleaseActivityPointResponse{
			Date:     p.Date,
			Releases: p.Releases,
		}
	}

	resp.Classifications = make([]ChartClassificationCountResponse, len(d.Classifications))
	for i, c := range d.Classifications {
		resp.Classifications[i] = ChartClassificationCountResponse{
			Classification: c.Classification,
			Count:          c.Count,
		}
	}

	resp.Ecosystems = make([]ChartEcosystemCountResponse, len(d.Ecosystems))
	for i, e := range d.Ecosystems {
		resp.Ecosystems[i] = ChartEcosystemCountResponse{
			Ecosystem: e.Ecosystem,
			Count:     e.Count,
		}
	}

	resp.AlertsBySeverity = make([]ChartAlertSeverityCountResponse, len(d.AlertsBySeverity))
	for i, a := range d.AlertsBySeverity {
		resp.AlertsBySeverity[i] = ChartAlertSeverityCountResponse{
			Severity: a.Severity,
			Count:    a.Count,
		}
	}

	resp.ReleaseStatuses = make([]ChartReleaseStatusCountResponse, len(d.ReleaseStatuses))
	for i, s := range d.ReleaseStatuses {
		resp.ReleaseStatuses[i] = ChartReleaseStatusCountResponse{
			Status: s.Status,
			Count:  s.Count,
		}
	}

	return resp
}

// ReanalyzeAllResponse is the JSON representation of a bulk reanalysis result.
type ReanalyzeAllResponse struct {
	Message string `json:"message"`
	Queued  int    `json:"queued"`
}
