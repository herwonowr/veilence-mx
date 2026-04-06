package repository

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/veilence/veilence-mx/backend/internal/domain"
	"github.com/veilence/veilence-mx/backend/internal/models"
)

// NotificationRepo implements domain.NotificationRepository using GORM.
type NotificationRepo struct {
	db *gorm.DB
}

// NewNotificationRepo creates a new NotificationRepo.
func NewNotificationRepo(db *gorm.DB) *NotificationRepo {
	return &NotificationRepo{db: db}
}

func (r *NotificationRepo) Create(ctx context.Context, notification *domain.Notification) error {
	m := notifToModel(notification)
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return fmt.Errorf("creating notification: %w", err)
	}
	notification.ID = m.ID
	notification.CreatedAt = m.CreatedAt
	return nil
}

func (r *NotificationRepo) FindByUserAndOrg(ctx context.Context, orgID, userID uint, onlyUnread bool) ([]domain.Notification, error) {
	query := r.db.WithContext(ctx).Model(&models.Notification{})

	if orgID != 0 {
		query = query.Where("org_id = ?", orgID)
	}

	// Show org-wide (user_id=0) and user-specific notifications
	query = query.Where("user_id = ? OR user_id = 0", userID)

	if onlyUnread {
		query = query.Where("is_read = ?", false)
	}

	var ms []models.Notification
	err := query.
		Order("created_at DESC").
		Find(&ms).Error
	if err != nil {
		return nil, fmt.Errorf("listing notifications: %w", err)
	}

	result := make([]domain.Notification, len(ms))
	for i := range ms {
		result[i] = *notifToDomain(&ms[i])
	}
	return result, nil
}

func (r *NotificationRepo) MarkRead(ctx context.Context, id, userID uint) (int64, error) {
	result := r.db.WithContext(ctx).Model(&models.Notification{}).
		Where("id = ? AND (user_id = ? OR user_id = 0)", id, userID).
		Update("is_read", true)
	if result.Error != nil {
		return 0, fmt.Errorf("marking notification as read: %w", result.Error)
	}
	return result.RowsAffected, nil
}

func (r *NotificationRepo) CountUnread(ctx context.Context, orgID, userID uint) (int64, error) {
	query := r.db.WithContext(ctx).Model(&models.Notification{}).
		Where("is_read = ?", false).
		Where("user_id = ? OR user_id = 0", userID)

	if orgID != 0 {
		query = query.Where("org_id = ?", orgID)
	}

	var count int64
	if err := query.Count(&count).Error; err != nil {
		return 0, fmt.Errorf("counting unread notifications: %w", err)
	}
	return count, nil
}

// --- Converters ---

func notifToDomain(m *models.Notification) *domain.Notification {
	return &domain.Notification{
		ID:        m.ID,
		OrgID:     m.OrgID,
		UserID:    m.UserID,
		ChannelID: m.ChannelID,
		Title:     m.Title,
		Message:   m.Message,
		IsRead:    m.IsRead,
		SentAt:    m.SentAt,
		CreatedAt: m.CreatedAt,
	}
}

func notifToModel(d *domain.Notification) *models.Notification {
	return &models.Notification{
		ID:        d.ID,
		OrgID:     d.OrgID,
		UserID:    d.UserID,
		ChannelID: d.ChannelID,
		Title:     d.Title,
		Message:   d.Message,
		IsRead:    d.IsRead,
		SentAt:    d.SentAt,
		CreatedAt: d.CreatedAt,
	}
}
