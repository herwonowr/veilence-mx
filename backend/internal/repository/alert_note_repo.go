package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/veilence/veilence-mx/backend/internal/domain"
	"github.com/veilence/veilence-mx/backend/internal/models"
)

// AlertNoteRepo implements domain.AlertNoteRepository using GORM.
type AlertNoteRepo struct {
	db *gorm.DB
}

// NewAlertNoteRepo creates a new AlertNoteRepo.
func NewAlertNoteRepo(db *gorm.DB) *AlertNoteRepo {
	return &AlertNoteRepo{db: db}
}

func (r *AlertNoteRepo) FindByAlertID(ctx context.Context, alertID uint) ([]domain.AlertNote, error) {
	var ms []models.AlertNote
	if err := r.db.WithContext(ctx).
		Where("alert_id = ?", alertID).
		Order("created_at DESC").
		Find(&ms).Error; err != nil {
		return nil, fmt.Errorf("listing alert notes: %w", err)
	}

	result := make([]domain.AlertNote, len(ms))
	for i := range ms {
		result[i] = *alertNoteToDomain(&ms[i])
	}
	return result, nil
}

func (r *AlertNoteRepo) Create(ctx context.Context, note *domain.AlertNote) error {
	m := alertNoteToModel(note)
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return fmt.Errorf("creating alert note: %w", err)
	}
	note.ID = m.ID
	note.CreatedAt = m.CreatedAt
	note.UpdatedAt = m.UpdatedAt
	return nil
}

func (r *AlertNoteRepo) FindByID(ctx context.Context, id uint) (*domain.AlertNote, error) {
	var m models.AlertNote
	if err := r.db.WithContext(ctx).First(&m, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("alert note %w", domain.ErrNotFound)
		}
		return nil, fmt.Errorf("finding alert note: %w", err)
	}
	return alertNoteToDomain(&m), nil
}

func (r *AlertNoteRepo) Update(ctx context.Context, note *domain.AlertNote) error {
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

func (r *AlertNoteRepo) Delete(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Delete(&models.AlertNote{}, id)
	if result.Error != nil {
		return fmt.Errorf("deleting alert note: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("alert note %w", domain.ErrNotFound)
	}
	return nil
}

// --- Converters ---

func alertNoteToDomain(m *models.AlertNote) *domain.AlertNote {
	return &domain.AlertNote{
		ID:        m.ID,
		AlertID:   m.AlertID,
		OrgID:     m.OrgID,
		UserID:    m.UserID,
		UserEmail: m.UserEmail,
		Content:   m.Content,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}

func alertNoteToModel(d *domain.AlertNote) *models.AlertNote {
	return &models.AlertNote{
		ID:        d.ID,
		AlertID:   d.AlertID,
		OrgID:     d.OrgID,
		UserID:    d.UserID,
		UserEmail: d.UserEmail,
		Content:   d.Content,
		CreatedAt: d.CreatedAt,
		UpdatedAt: d.UpdatedAt,
	}
}
