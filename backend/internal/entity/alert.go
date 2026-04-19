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
	ID             uint
	WorkspaceID    uint
	AnalysisID     uint
	ReleaseID      uint
	PackageID      uint
	Severity       AlertSeverity
	Status         AlertStatus
	Message        string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// AlertNote represents a comment/note on an alert.
type AlertNote struct {
	ID             uint
	AlertID        uint
	WorkspaceID    uint
	UserID         uint
	UserEmail      string
	Content        string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// MaxNoteLength is the maximum allowed length of a note's content.
const MaxNoteLength = 10000

// AlertFilters holds optional query filters for listing alerts.
type AlertFilters struct {
	Severity       *AlertSeverity
	Status         *AlertStatus
	Search         *string
}

// AlertWithPackage combines an alert with its package info for list responses.
type AlertWithPackage struct {
	Alert
	PackageName    string
	PackageEcosystem string
}
