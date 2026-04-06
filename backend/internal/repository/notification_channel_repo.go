package repository

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/veilence/veilence-mx/backend/internal/domain"
	"github.com/veilence/veilence-mx/backend/internal/models"
)

// NotificationChannelRepo implements domain.NotificationChannelRepository using GORM.
type NotificationChannelRepo struct {
	db *gorm.DB
}

// NewNotificationChannelRepo creates a new NotificationChannelRepo.
func NewNotificationChannelRepo(db *gorm.DB) *NotificationChannelRepo {
	return &NotificationChannelRepo{db: db}
}

func (r *NotificationChannelRepo) FindByID(ctx context.Context, id uint) (*domain.NotificationChannel, error) {
	var m models.NotificationChannel
	if err := r.db.WithContext(ctx).First(&m, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("notification channel %w", domain.ErrNotFound)
		}
		return nil, fmt.Errorf("finding notification channel: %w", err)
	}
	return notifChannelToDomain(&m), nil
}

func (r *NotificationChannelRepo) FindByIDAndOrg(ctx context.Context, id, orgID uint) (*domain.NotificationChannel, error) {
	var m models.NotificationChannel
	if err := r.db.WithContext(ctx).Where("id = ? AND org_id = ?", id, orgID).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("notification channel %w", domain.ErrNotFound)
		}
		return nil, fmt.Errorf("finding notification channel: %w", err)
	}
	return notifChannelToDomain(&m), nil
}

func (r *NotificationChannelRepo) FindByOrgID(ctx context.Context, orgID uint) ([]domain.NotificationChannel, error) {
	var ms []models.NotificationChannel
	if err := r.db.WithContext(ctx).Where("org_id = ?", orgID).Find(&ms).Error; err != nil {
		return nil, fmt.Errorf("listing notification channels: %w", err)
	}
	result := make([]domain.NotificationChannel, len(ms))
	for i := range ms {
		result[i] = *notifChannelToDomain(&ms[i])
	}
	return result, nil
}

func (r *NotificationChannelRepo) Create(ctx context.Context, channel *domain.NotificationChannel) error {
	m := notifChannelToModel(channel)
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return fmt.Errorf("creating notification channel: %w", err)
	}
	channel.ID = m.ID
	channel.CreatedAt = m.CreatedAt
	channel.UpdatedAt = m.UpdatedAt
	return nil
}

func (r *NotificationChannelRepo) Update(ctx context.Context, channel *domain.NotificationChannel) error {
	m := notifChannelToModel(channel)
	if err := r.db.WithContext(ctx).Save(m).Error; err != nil {
		return fmt.Errorf("updating notification channel: %w", err)
	}
	channel.UpdatedAt = m.UpdatedAt
	return nil
}

func (r *NotificationChannelRepo) DeleteByIDAndOrg(ctx context.Context, id, orgID uint) (int64, error) {
	result := r.db.WithContext(ctx).Where("id = ? AND org_id = ?", id, orgID).Delete(&models.NotificationChannel{})
	if result.Error != nil {
		return 0, fmt.Errorf("deleting notification channel: %w", result.Error)
	}
	return result.RowsAffected, nil
}

// --- Converters ---

func notifChannelToDomain(m *models.NotificationChannel) *domain.NotificationChannel {
	return &domain.NotificationChannel{
		ID:        m.ID,
		OrgID:     m.OrgID,
		Name:      m.Name,
		Type:      domain.NotificationChannelType(m.Type),
		Config:    m.Config,
		IsActive:  m.IsActive,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}

func notifChannelToModel(d *domain.NotificationChannel) *models.NotificationChannel {
	return &models.NotificationChannel{
		ID:        d.ID,
		OrgID:     d.OrgID,
		Name:      d.Name,
		Type:      models.NotificationChannelType(d.Type),
		Config:    d.Config,
		IsActive:  d.IsActive,
		CreatedAt: d.CreatedAt,
		UpdatedAt: d.UpdatedAt,
	}
}
