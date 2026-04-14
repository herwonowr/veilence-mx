package entity

import "time"

// DashboardStats holds the overview statistics for the dashboard, org-scoped.
type DashboardStats struct {
	TotalPackages  int64 `json:"totalPackages"`
	TotalReleases  int64 `json:"totalReleases"`
	PendingAnalyses int64 `json:"pendingAnalyses"`
	ActiveAlerts   int64 `json:"activeAlerts"`
	RecentMalicious int64 `json:"recentMalicious"`
}

// ReleaseActivityPoint represents releases per day.
type ReleaseActivityPoint struct {
	Date           time.Time `json:"date"`
	Count          int64 `json:"count"`
}

// ClassificationCount holds an analysis classification and its count.
type ClassificationCount struct {
	Classification string `json:"classification"`
	Count          int64 `json:"count"`
}

// EcosystemCount holds an ecosystem type and its package count.
type EcosystemCount struct {
	Ecosystem      string `json:"ecosystem"`
	Count          int64 `json:"count"`
}

// AlertSeverityCount holds an alert severity and its count.
type AlertSeverityCount struct {
	Severity       string `json:"severity"`
	Count          int64 `json:"count"`
}

// ReleaseStatusCount holds a release status and its count.
type ReleaseStatusCount struct {
	Status         string `json:"status"`
	Count          int64 `json:"count"`
}

// ChartData holds all aggregated chart data for the dashboard.
type ChartData struct {
	ReleaseActivity []ChartReleaseActivityPoint `json:"releaseActivity,omitempty"`
	Classifications []ChartClassificationCount `json:"classifications,omitempty"`
	Ecosystems     []ChartEcosystemCount `json:"ecosystems,omitempty"`
	AlertsBySeverity []ChartAlertSeverityCount `json:"alertsBySeverity,omitempty"`
	ReleaseStatuses []ChartReleaseStatusCount `json:"releaseStatuses,omitempty"`
}

// ChartReleaseActivityPoint represents a single day's release count for charting.
type ChartReleaseActivityPoint struct {
	Date           string `json:"date"`
	Releases       int64 `json:"releases"`
}

// ChartClassificationCount represents a classification and its count for charting.
type ChartClassificationCount struct {
	Classification string `json:"classification"`
	Count          int64 `json:"count"`
}

// ChartEcosystemCount represents an ecosystem and its count for charting.
type ChartEcosystemCount struct {
	Ecosystem      string `json:"ecosystem"`
	Count          int64 `json:"count"`
}

// ChartAlertSeverityCount represents a severity and its count for charting.
type ChartAlertSeverityCount struct {
	Severity       string `json:"severity"`
	Count          int64 `json:"count"`
}

// ChartReleaseStatusCount represents a status and its count for charting.
type ChartReleaseStatusCount struct {
	Status         string `json:"status"`
	Count          int64 `json:"count"`
}

// HealthResponse is the response for the health check endpoint.
type HealthResponse struct {
	Status         string `json:"status"`
	Database       HealthStatus `json:"database"`
	Redis          HealthStatus `json:"redis"`
}

// HealthStatus represents the status of a dependency.
type HealthStatus struct {
	Status         string `json:"status"`
	Latency        string `json:"latency"`
	Error          string `json:"error"`
}

// ReadinessResponse is the response for the readiness check endpoint.
type ReadinessResponse struct {
	Ready          bool `json:"ready"`
	Database       HealthStatus `json:"database"`
	Redis          HealthStatus `json:"redis"`
}
