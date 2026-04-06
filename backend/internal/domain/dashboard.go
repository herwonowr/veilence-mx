package domain

import (
	"context"
	"time"
)

// DashboardStats holds the overview statistics for the dashboard, org-scoped.
type DashboardStats struct {
	TotalPackages   int64 `json:"totalPackages"`
	TotalReleases   int64 `json:"totalReleases"`
	PendingAnalyses int64 `json:"pendingAnalyses"`
	ActiveAlerts    int64 `json:"activeAlerts"`
	RecentMalicious int64 `json:"recentMalicious"`
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

// RegistryCount holds a registry type and its package count.
type RegistryCount struct {
	Registry string
	Count    int64
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

// DashboardRepository defines read-only aggregation queries for the dashboard.
// All queries are scoped to an organization via orgID.
type DashboardRepository interface {
	// GetStats returns overview statistics for an organization.
	GetStats(ctx context.Context, orgID uint) (*DashboardStats, error)
	// GetReleaseActivity returns release counts per day within a date range for an org.
	GetReleaseActivity(ctx context.Context, orgID uint, from, to time.Time) ([]ReleaseActivityPoint, error)
	// GetClassificationDistribution returns analysis classification counts within a date range.
	GetClassificationDistribution(ctx context.Context, orgID uint, from, to time.Time) ([]ClassificationCount, error)
	// GetBaselineCount returns the number of completed releases without diffs within a date range.
	GetBaselineCount(ctx context.Context, orgID uint, from, to time.Time) (int64, error)
	// GetRegistryDistribution returns package counts per registry for an org.
	GetRegistryDistribution(ctx context.Context, orgID uint) ([]RegistryCount, error)
	// GetAlertsBySeverity returns alert counts per severity within a date range for an org.
	GetAlertsBySeverity(ctx context.Context, orgID uint, from, to time.Time) ([]AlertSeverityCount, error)
	// GetReleaseStatusDistribution returns release status counts within a date range for an org.
	GetReleaseStatusDistribution(ctx context.Context, orgID uint, from, to time.Time) ([]ReleaseStatusCount, error)
	// GetUnanalyzedDiffIDs returns diff IDs without analyses, scoped to an org's packages.
	GetUnanalyzedDiffIDs(ctx context.Context, orgID uint) ([]uint, error)
}
