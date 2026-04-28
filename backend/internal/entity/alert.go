package entity

import (
	"fmt"
	"time"
)

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
	ID          string
	WorkspaceID string
	AnalysisID  string
	ReleaseID   string
	PackageID   string
	Severity    AlertSeverity
	Status      AlertStatus
	Message     string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// AlertNote represents a comment/note on an alert.
type AlertNote struct {
	ID          string
	AlertID     string
	WorkspaceID string
	UserID      string
	UserEmail   string
	Content     string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// MaxNoteLength is the maximum allowed length of a note's content.
const MaxNoteLength = 10000

// AlertFilters holds optional query filters for listing alerts.
type AlertFilters struct {
	Severity *AlertSeverity
	Status   *AlertStatus
	Search   *string
}

// validAlertTransitions defines the allowed status transitions.
var validAlertTransitions = map[AlertStatus][]AlertStatus{
	AlertStatusNew:          {AlertStatusAcknowledged, AlertStatusResolved},
	AlertStatusAcknowledged: {AlertStatusResolved},
	AlertStatusResolved:     {AlertStatusNew},
}

// ValidateStatusTransition checks whether transitioning from the current status
// to the given next status is allowed. Returns ErrValidation if not.
func (a Alert) ValidateStatusTransition(next AlertStatus) error {
	allowed, ok := validAlertTransitions[a.Status]
	if !ok {
		return fmt.Errorf("unknown current alert status %q: %w", a.Status, ErrValidation)
	}
	for _, s := range allowed {
		if s == next {
			return nil
		}
	}
	return fmt.Errorf("invalid alert status transition from %q to %q: %w", a.Status, next, ErrValidation)
}

// AlertWithPackage combines an alert with its package info for list responses.
type AlertWithPackage struct {
	Alert
	PackageName      string
	PackageEcosystem string
}
