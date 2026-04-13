package domain

import (
	"context"
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
	ID         uint          `json:"id"`
	OrgID      uint          `json:"orgId"`
	AnalysisID uint          `json:"analysisId"`
	ReleaseID  uint          `json:"releaseId"`
	PackageID  uint          `json:"packageId"`
	Severity   AlertSeverity `json:"severity"`
	Status     AlertStatus   `json:"status"`
	Message    string        `json:"message"`
	CreatedAt  time.Time     `json:"createdAt"`
	UpdatedAt  time.Time     `json:"updatedAt"`
}

// AlertNote represents a comment/note on an alert.
type AlertNote struct {
	ID        uint      `json:"id"`
	AlertID   uint      `json:"alertId"`
	OrgID     uint      `json:"orgId"`
	UserID    uint      `json:"userId"`
	UserEmail string    `json:"userEmail"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// MaxNoteLength is the maximum allowed length of a note's content.
const MaxNoteLength = 10000

// AlertNoteService defines the business logic operations for alert notes.
type AlertNoteService interface {
	// ListByAlert returns all notes for an alert (newest first), verifying org ownership.
	ListByAlert(ctx context.Context, orgID, alertID uint) ([]AlertNote, error)
	// Create adds a new note to an alert, verifying org ownership and resolving user email.
	Create(ctx context.Context, orgID, alertID, userID uint, content string) (*AlertNote, error)
	// Update edits an existing note's content. Verifies alert+org ownership, note ownership,
	// and that the note belongs to the given alert (URL consistency).
	// Returns ErrForbidden if the caller is not the owner.
	Update(ctx context.Context, orgID, alertID, noteID, userID uint, content string) (*AlertNote, error)
	// Delete removes a note. Verifies alert+org ownership, note ownership,
	// and that the note belongs to the given alert (URL consistency).
	// Returns ErrForbidden if the caller is not the owner.
	Delete(ctx context.Context, orgID, alertID, noteID, userID uint) error
}
