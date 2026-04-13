// Package alertnote implements the business logic for alert notes (comments on alerts).
package alertnote

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/veilence/veilence-mx/backend/internal/domain"
)

// Service implements domain.AlertNoteService.
type Service struct {
	notes  domain.AlertNoteRepository
	alerts domain.AlertRepository
	users  domain.UserRepository
}

// NewService creates a new alert note service.
func NewService(notes domain.AlertNoteRepository, alerts domain.AlertRepository, users domain.UserRepository) *Service {
	return &Service{
		notes:  notes,
		alerts: alerts,
		users:  users,
	}
}

// ListByAlert returns all notes for an alert, verifying that the alert
// belongs to the given org for tenant isolation.
func (s *Service) ListByAlert(ctx context.Context, orgID, alertID uint) ([]domain.AlertNote, error) {
	// Verify alert exists and belongs to the org
	alert, err := s.alerts.FindByID(ctx, alertID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, fmt.Errorf("alert %w", domain.ErrNotFound)
		}
		return nil, fmt.Errorf("verifying alert ownership: %w", err)
	}
	if alert.OrgID != orgID {
		return nil, fmt.Errorf("alert %w", domain.ErrNotFound)
	}

	notes, err := s.notes.FindByAlertID(ctx, alertID)
	if err != nil {
		return nil, fmt.Errorf("listing notes for alert %d: %w", alertID, err)
	}

	return notes, nil
}

// Create adds a new note to an alert. It verifies org ownership, validates
// the content, resolves the user's email, and persists the note.
func (s *Service) Create(ctx context.Context, orgID, alertID, userID uint, content string) (*domain.AlertNote, error) {
	// Validate content
	if content == "" {
		return nil, fmt.Errorf("content is required")
	}
	if len(content) > domain.MaxNoteLength {
		return nil, fmt.Errorf("content must be at most %d characters", domain.MaxNoteLength)
	}

	// Verify alert exists and belongs to the org
	alert, err := s.alerts.FindByID(ctx, alertID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, fmt.Errorf("alert %w", domain.ErrNotFound)
		}
		return nil, fmt.Errorf("verifying alert ownership: %w", err)
	}
	if alert.OrgID != orgID {
		return nil, fmt.Errorf("alert %w", domain.ErrNotFound)
	}

	// Resolve user email
	userEmail := s.resolveUserEmail(ctx, userID)

	note := &domain.AlertNote{
		AlertID:   alertID,
		OrgID:     orgID,
		UserID:    userID,
		UserEmail: userEmail,
		Content:   content,
	}

	if err := s.notes.Create(ctx, note); err != nil {
		return nil, fmt.Errorf("creating alert note: %w", err)
	}

	return note, nil
}

// resolveUserEmail looks up a user's email by ID. Returns an empty string
// if the user is not found (non-fatal — the note is still created).
func (s *Service) resolveUserEmail(ctx context.Context, userID uint) string {
	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		slog.Warn("failed to resolve user email for alert note",
			"user_id", userID,
			"error", err,
		)
		return ""
	}
	return user.Email
}
