package entity

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
	ID             uint `json:"id"`
	OrgID          uint `json:"orgId"`
	AnalysisID     uint `json:"analysisId"`
	ReleaseID      uint `json:"releaseId"`
	PackageID      uint `json:"packageId"`
	Severity       AlertSeverity `json:"severity"`
	Status         AlertStatus `json:"status"`
	Message        string `json:"message"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

// AlertNote represents a comment/note on an alert.
type AlertNote struct {
	ID             uint `json:"id"`
	AlertID        uint `json:"alertId"`
	OrgID          uint `json:"orgId"`
	UserID         uint `json:"userId"`
	UserEmail      string `json:"userEmail"`
	Content        string `json:"content"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

// MaxNoteLength is the maximum allowed length of a note's content.
const MaxNoteLength = 10000

// AlertFilters holds optional query filters for listing alerts.
type AlertFilters struct {
	Severity       *AlertSeverity `json:"severity,omitempty"`
	Status         *AlertStatus `json:"status,omitempty"`
	Search         *string `json:"search,omitempty"`
}

// AlertWithPackage combines an alert with its package info for list responses.
type AlertWithPackage struct {
	Alert
	PackageName    string `json:"packageName"`
	PackageEcosystem string `json:"packageEcosystem"`
}
