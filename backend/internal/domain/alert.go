package domain

import "time"

// AlertSeverity represents the severity level of an alert.
type AlertSeverity string

const (
	AlertSeverityLow      AlertSeverity = "low"
	AlertSeverityMedium   AlertSeverity = "medium"
	AlertSeverityHigh     AlertSeverity = "high"
	AlertSeverityCritical AlertSeverity = "critical"
)

// AlertStatus represents the current status of an alert.
type AlertStatus string

const (
	AlertStatusNew          AlertStatus = "new"
	AlertStatusAcknowledged AlertStatus = "acknowledged"
	AlertStatusResolved     AlertStatus = "resolved"
)

// Alert represents a security alert generated from an analysis.
type Alert struct {
	ID         uint          `json:"id"`
	OrgID      uint          `json:"orgId"`
	AnalysisID uint          `json:"analysisId"`
	PackageID  uint          `json:"packageId"`
	Severity   AlertSeverity `json:"severity"`
	Status     AlertStatus   `json:"status"`
	Message    string        `json:"message"`
	CreatedAt  time.Time     `json:"createdAt"`
	UpdatedAt  time.Time     `json:"updatedAt"`
}
