package repository

import (
	"context"
	"fmt"

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
		Order("created_at ASC").
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
