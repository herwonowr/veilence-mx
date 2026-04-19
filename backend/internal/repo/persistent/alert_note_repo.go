package persistent

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/veilence/veilence-mx/backend/internal/entity"
)

// AlertNoteRepo implements entity.AlertNoteRepository using GORM.
type AlertNoteRepo struct {
	db *gorm.DB
}

// NewAlertNoteRepo creates a new AlertNoteRepo.
func NewAlertNoteRepo(db *gorm.DB) *AlertNoteRepo {
	return &AlertNoteRepo{db: db}
}

func (r *AlertNoteRepo) FindByAlertID(ctx context.Context, alertID, workspaceID uint) ([]entity.AlertNote, error) {
	var ms []AlertNote
	if err := r.db.WithContext(ctx).
		Where("alert_id = ? AND workspace_id = ?", alertID, workspaceID).
		Order("created_at DESC").
		Find(&ms).Error; err != nil {
		return nil, fmt.Errorf("listing alert notes: %w", err)
	}

	result := make([]entity.AlertNote, len(ms))
	for i := range ms {
		result[i] = *alertNoteToDomain(&ms[i])
	}
	return result, nil
}

func (r *AlertNoteRepo) Create(ctx context.Context, note *entity.AlertNote) error {
	m := alertNoteToModel(note)
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return fmt.Errorf("creating alert note: %w", err)
	}
	note.ID = m.ID
	note.CreatedAt = m.CreatedAt
	note.UpdatedAt = m.UpdatedAt
	return nil
}

func (r *AlertNoteRepo) FindByID(ctx context.Context, id, workspaceID uint) (*entity.AlertNote, error) {
	var m AlertNote
	if err := r.db.WithContext(ctx).Where("id = ? AND workspace_id = ?", id, workspaceID).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("alert note %w", entity.ErrNotFound)
		}
		return nil, fmt.Errorf("finding alert note: %w", err)
	}
	return alertNoteToDomain(&m), nil
}

func (r *AlertNoteRepo) Update(ctx context.Context, note *entity.AlertNote) error {
	m := alertNoteToModel(note)
	now := time.Now()
	if err := r.db.WithContext(ctx).Model(m).Updates(map[string]any{
		"content":    m.Content,
		"updated_at": now,
	}).Error; err != nil {
		return fmt.Errorf("updating alert note: %w", err)
	}
	note.UpdatedAt = now
	return nil
}

func (r *AlertNoteRepo) Delete(ctx context.Context, id, workspaceID uint) error {
	result := r.db.WithContext(ctx).Where("id = ? AND workspace_id = ?", id, workspaceID).Delete(&AlertNote{})
	if result.Error != nil {
		return fmt.Errorf("deleting alert note: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("alert note %w", entity.ErrNotFound)
	}
	return nil
}

// --- Converters ---

func alertNoteToDomain(m *AlertNote) *entity.AlertNote {
	return &entity.AlertNote{
		ID:        m.ID,
		AlertID:   m.AlertID,
		WorkspaceID:     m.WorkspaceID,
		UserID:    m.UserID,
		UserEmail: m.UserEmail,
		Content:   m.Content,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}

func alertNoteToModel(d *entity.AlertNote) *AlertNote {
	return &AlertNote{
		ID:        d.ID,
		AlertID:   d.AlertID,
		WorkspaceID:     d.WorkspaceID,
		UserID:    d.UserID,
		UserEmail: d.UserEmail,
		Content:   d.Content,
		CreatedAt: d.CreatedAt,
		UpdatedAt: d.UpdatedAt,
	}
}
