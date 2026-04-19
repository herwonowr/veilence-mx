// Package alertnote implements the business logic for alert notes (comments on alerts).
package alertnote

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/veilence/veilence-mx/backend/internal/entity"
	"github.com/veilence/veilence-mx/backend/internal/usecase"
)

// UseCase implements usecase.AlertNoteService.
type UseCase struct {
	notes  usecase.AlertNoteRepository
	alerts usecase.AlertRepository
	users  usecase.UserRepository
}

// New creates a new alert note UseCase.
func New(notes usecase.AlertNoteRepository, alerts usecase.AlertRepository, users usecase.UserRepository) *UseCase {
	return &UseCase{
		notes:  notes,
		alerts: alerts,
		users:  users,
	}
}

// ListByAlert returns all notes for an alert, verifying that the alert
// belongs to the given org for tenant isolation.
func (uc *UseCase) ListByAlert(ctx context.Context, workspaceID, alertID uint) ([]entity.AlertNote, error) {
	// Verify alert exists and belongs to the org
	_, err := uc.alerts.FindByIDAndWorkspaceID(ctx, alertID, workspaceID)
	if err != nil {
		if errors.Is(err, entity.ErrNotFound) {
			return nil, fmt.Errorf("alert %w", entity.ErrNotFound)
		}
		return nil, fmt.Errorf("verifying alert ownership: %w", err)
	}

	notes, err := uc.notes.FindByAlertID(ctx, alertID, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("listing notes for alert %d: %w", alertID, err)
	}

	return notes, nil
}

// Create adds a new note to an alert. It verifies org ownership, validates
// the content, resolves the user's email, and persists the note.
func (uc *UseCase) Create(ctx context.Context, workspaceID, alertID, userID uint, content string) (*entity.AlertNote, error) {
	// Validate content
	if content == "" {
		return nil, fmt.Errorf("content is required")
	}
	if len(content) > entity.MaxNoteLength {
		return nil, fmt.Errorf("content must be at most %d characters", entity.MaxNoteLength)
	}

	// Verify alert exists and belongs to the org
	_, err := uc.alerts.FindByIDAndWorkspaceID(ctx, alertID, workspaceID)
	if err != nil {
		if errors.Is(err, entity.ErrNotFound) {
			return nil, fmt.Errorf("alert %w", entity.ErrNotFound)
		}
		return nil, fmt.Errorf("verifying alert ownership: %w", err)
	}

	// Resolve user email
	userEmail := uc.resolveUserEmail(ctx, userID)

	note := &entity.AlertNote{
		AlertID:   alertID,
		WorkspaceID:     workspaceID,
		UserID:    userID,
		UserEmail: userEmail,
		Content:   content,
	}

	if err := uc.notes.Create(ctx, note); err != nil {
		return nil, fmt.Errorf("creating alert note: %w", err)
	}

	return note, nil
}

// resolveUserEmail looks up a user's email by ID. Returns an empty string
// if the user is not found (non-fatal — the note is still created).
func (uc *UseCase) resolveUserEmail(ctx context.Context, userID uint) string {
	user, err := uc.users.FindByID(ctx, userID)
	if err != nil {
		slog.Warn("failed to resolve user email for alert note",
			"user_id", userID,
			"error", err,
		)
		return ""
	}
	return user.Email
}

// Update edits an existing note's content. Verifies:
// 1. Content validation
// 2. Alert exists and belongs to org (tenant isolation)
// 3. Note exists
// 4. Note belongs to org (belt-and-suspenders tenant check)
// 5. Note belongs to the given alert (URL consistency)
// 6. Calling user is the note's author (owner verification)
func (uc *UseCase) Update(ctx context.Context, workspaceID, alertID, noteID, userID uint, content string) (*entity.AlertNote, error) {
	// Validate content
	if content == "" {
		return nil, fmt.Errorf("content is required")
	}
	if len(content) > entity.MaxNoteLength {
		return nil, fmt.Errorf("content must be at most %d characters", entity.MaxNoteLength)
	}

	// Verify alert exists and belongs to the org
	_, err := uc.alerts.FindByIDAndWorkspaceID(ctx, alertID, workspaceID)
	if err != nil {
		if errors.Is(err, entity.ErrNotFound) {
			return nil, fmt.Errorf("alert %w", entity.ErrNotFound)
		}
		return nil, fmt.Errorf("verifying alert ownership: %w", err)
	}

	// Find the note
	note, err := uc.notes.FindByID(ctx, noteID, workspaceID)
	if err != nil {
		if errors.Is(err, entity.ErrNotFound) {
			return nil, fmt.Errorf("alert note %w", entity.ErrNotFound)
		}
		return nil, fmt.Errorf("finding alert note: %w", err)
	}

	// Verify note belongs to org (belt-and-suspenders)
	if note.WorkspaceID != workspaceID {
		return nil, fmt.Errorf("alert note %w", entity.ErrNotFound)
	}

	// Verify note belongs to this alert (URL consistency)
	if note.AlertID != alertID {
		return nil, fmt.Errorf("alert note %w", entity.ErrNotFound)
	}

	// Verify user is the note's author
	if note.UserID != userID {
		return nil, fmt.Errorf("only the note author can edit: %w", entity.ErrForbidden)
	}

	// Update the content
	note.Content = content
	if err := uc.notes.Update(ctx, note); err != nil {
		return nil, fmt.Errorf("updating alert note: %w", err)
	}

	return note, nil
}

// Delete removes a note. Verifies:
// 1. Alert exists and belongs to org (tenant isolation)
// 2. Note exists
// 3. Note belongs to org (belt-and-suspenders tenant check)
// 4. Note belongs to the given alert (URL consistency)
// 5. Calling user is the note's author (owner verification)
func (uc *UseCase) Delete(ctx context.Context, workspaceID, alertID, noteID, userID uint) error {
	// Verify alert exists and belongs to the org
	_, err := uc.alerts.FindByIDAndWorkspaceID(ctx, alertID, workspaceID)
	if err != nil {
		if errors.Is(err, entity.ErrNotFound) {
			return fmt.Errorf("alert %w", entity.ErrNotFound)
		}
		return fmt.Errorf("verifying alert ownership: %w", err)
	}

	// Find the note
	note, err := uc.notes.FindByID(ctx, noteID, workspaceID)
	if err != nil {
		if errors.Is(err, entity.ErrNotFound) {
			return fmt.Errorf("alert note %w", entity.ErrNotFound)
		}
		return fmt.Errorf("finding alert note: %w", err)
	}

	// Verify note belongs to org (belt-and-suspenders)
	if note.WorkspaceID != workspaceID {
		return fmt.Errorf("alert note %w", entity.ErrNotFound)
	}

	// Verify note belongs to this alert (URL consistency)
	if note.AlertID != alertID {
		return fmt.Errorf("alert note %w", entity.ErrNotFound)
	}

	// Verify user is the note's author
	if note.UserID != userID {
		return fmt.Errorf("only the note author can delete: %w", entity.ErrForbidden)
	}

	if err := uc.notes.Delete(ctx, noteID, workspaceID); err != nil {
		return fmt.Errorf("deleting alert note: %w", err)
	}

	return nil
}
