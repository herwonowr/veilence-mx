package entity

import "time"

// DashboardStats holds the overview statistics for the dashboard, workspace-scoped.
type DashboardStats struct {
	TotalPackages   int64
	TotalReleases   int64
	PendingAnalyses int64
	ActiveAlerts    int64
	RecentMalicious int64
}

// ReleaseActivityPoint represents releases per day.
type ReleaseActivityPoint struct {
	Date  time.Time
	Count int64
}

// ClassificationCount holds an analysis classification and its count.
type ClassificationCount struct {
	Classification string
	Count          int64
}

// EcosystemCount holds an ecosystem type and its package count.
type EcosystemCount struct {
	Ecosystem string
	Count     int64
}

// AlertSeverityCount holds an alert severity and its count.
type AlertSeverityCount struct {
	Severity string
	Count    int64
}

// ReleaseStatusCount holds a release status and its count.
type ReleaseStatusCount struct {
	Status string
	Count  int64
}

// ChartData holds all aggregated chart data for the dashboard.
type ChartData struct {
	ReleaseActivity  []ChartReleaseActivityPoint
	Classifications  []ChartClassificationCount
	Ecosystems       []ChartEcosystemCount
	AlertsBySeverity []ChartAlertSeverityCount
	ReleaseStatuses  []ChartReleaseStatusCount
}

// ChartReleaseActivityPoint represents a single day's release count for charting.
type ChartReleaseActivityPoint struct {
	Date     string
	Releases int64
}

// ChartClassificationCount represents a classification and its count for charting.
type ChartClassificationCount struct {
	Classification string
	Count          int64
}

// ChartEcosystemCount represents an ecosystem and its count for charting.
type ChartEcosystemCount struct {
	Ecosystem string
	Count     int64
}

// ChartAlertSeverityCount represents a severity and its count for charting.
type ChartAlertSeverityCount struct {
	Severity string
	Count    int64
}

// ChartReleaseStatusCount represents a status and its count for charting.
type ChartReleaseStatusCount struct {
	Status string
	Count  int64
}

// HealthResponse is the response for the health check endpoint.
type HealthResponse struct {
	Status   string
	Database HealthStatus
	Redis    HealthStatus
}

// HealthStatus represents the status of a dependency.
type HealthStatus struct {
	Status  string
	Latency string
	Error   string
}

// ReadinessResponse is the response for the readiness check endpoint.
type ReadinessResponse struct {
	Ready    bool
	Database HealthStatus
	Redis    HealthStatus
}
