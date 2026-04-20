package persistent

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/veilence/veilence-mx/backend/internal/entity"
)

// NotificationRepo implements entity.NotificationRepository using GORM.
type NotificationRepo struct {
	db *gorm.DB
}

// NewNotificationRepo creates a new NotificationRepo.
func NewNotificationRepo(db *gorm.DB) *NotificationRepo {
	return &NotificationRepo{db: db}
}

func (r *NotificationRepo) Create(ctx context.Context, notification *entity.Notification) error {
	m := notifToModel(notification)
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return fmt.Errorf("creating notification: %w", err)
	}
	notification.ID = m.ID
	notification.CreatedAt = m.CreatedAt
	return nil
}

func (r *NotificationRepo) FindByUserAndWorkspace(ctx context.Context, workspaceID, userID uint, onlyUnread bool) ([]entity.Notification, error) {
	query := r.db.WithContext(ctx).Model(&Notification{})

	// Always require workspace scoping - callers must resolve workspaceID before calling.
	if workspaceID != 0 {
		query = query.Where("workspace_id = ?", workspaceID)
	} else {
		// Safety: refuse to query without workspace scope to prevent cross-tenant access.
		return nil, nil
	}

	// Show org-wide (user_id=0) and user-specific notifications
	query = query.Where("user_id = ? OR user_id = 0", userID)

	if onlyUnread {
		query = query.Where("is_read = ?", false)
	}

	var ms []Notification
	err := query.
		Order("created_at DESC").
		Find(&ms).Error
	if err != nil {
		return nil, fmt.Errorf("listing notifications: %w", err)
	}

	result := make([]entity.Notification, len(ms))
	for i := range ms {
		result[i] = *notifToDomain(&ms[i])
	}
	return result, nil
}

func (r *NotificationRepo) FindByUserAndWorkspaceIDs(ctx context.Context, workspaceIDs []uint, userID uint, onlyUnread bool) ([]entity.Notification, error) {
	if len(workspaceIDs) == 0 {
		return nil, nil
	}

	query := r.db.WithContext(ctx).Model(&Notification{}).
		Where("workspace_id IN ?", workspaceIDs).
		Where("user_id = ? OR user_id = 0", userID)

	if onlyUnread {
		query = query.Where("is_read = ?", false)
	}

	var ms []Notification
	err := query.Order("created_at DESC").Find(&ms).Error
	if err != nil {
		return nil, fmt.Errorf("listing notifications by workspace IDs: %w", err)
	}

	result := make([]entity.Notification, len(ms))
	for i := range ms {
		result[i] = *notifToDomain(&ms[i])
	}
	return result, nil
}

func (r *NotificationRepo) MarkRead(ctx context.Context, id, userID uint) (int64, error) {
	result := r.db.WithContext(ctx).Model(&Notification{}).
		Where("id = ? AND (user_id = ? OR user_id = 0)", id, userID).
		Update("is_read", true)
	if result.Error != nil {
		return 0, fmt.Errorf("marking notification as read: %w", result.Error)
	}
	return result.RowsAffected, nil
}

func (r *NotificationRepo) CountUnread(ctx context.Context, workspaceID, userID uint) (int64, error) {
	query := r.db.WithContext(ctx).Model(&Notification{}).
		Where("is_read = ?", false).
		Where("user_id = ? OR user_id = 0", userID)

	// Always require workspace scoping - callers must resolve workspaceID before calling.
	if workspaceID != 0 {
		query = query.Where("workspace_id = ?", workspaceID)
	} else {
		// Safety: refuse to count without workspace scope to prevent cross-tenant access.
		return 0, nil
	}

	var count int64
	if err := query.Count(&count).Error; err != nil {
		return 0, fmt.Errorf("counting unread notifications: %w", err)
	}
	return count, nil
}

func (r *NotificationRepo) CountUnreadByWorkspaceIDs(ctx context.Context, workspaceIDs []uint, userID uint) (int64, error) {
	if len(workspaceIDs) == 0 {
		return 0, nil
	}

	var count int64
	err := r.db.WithContext(ctx).Model(&Notification{}).
		Where("is_read = ?", false).
		Where("workspace_id IN ?", workspaceIDs).
		Where("user_id = ? OR user_id = 0", userID).
		Count(&count).Error
	if err != nil {
		return 0, fmt.Errorf("counting unread notifications by workspace IDs: %w", err)
	}
	return count, nil
}

func (r *NotificationRepo) MarkAllRead(ctx context.Context, workspaceID, userID uint) (int64, error) {
	query := r.db.WithContext(ctx).Model(&Notification{}).
		Where("is_read = ?", false).
		Where("user_id = ? OR user_id = 0", userID)

	// Always require workspace scoping - callers must resolve workspaceID before calling.
	if workspaceID != 0 {
		query = query.Where("workspace_id = ?", workspaceID)
	} else {
		// Safety: refuse to update without workspace scope to prevent cross-tenant access.
		return 0, nil
	}

	result := query.Update("is_read", true)
	if result.Error != nil {
		return 0, fmt.Errorf("marking all notifications as read: %w", result.Error)
	}
	return result.RowsAffected, nil
}

func (r *NotificationRepo) MarkAllReadByWorkspaceIDs(ctx context.Context, workspaceIDs []uint, userID uint) (int64, error) {
	if len(workspaceIDs) == 0 {
		return 0, nil
	}

	result := r.db.WithContext(ctx).Model(&Notification{}).
		Where("is_read = ?", false).
		Where("workspace_id IN ?", workspaceIDs).
		Where("user_id = ? OR user_id = 0", userID).
		Update("is_read", true)
	if result.Error != nil {
		return 0, fmt.Errorf("marking all notifications as read by workspace IDs: %w", result.Error)
	}
	return result.RowsAffected, nil
}

func (r *NotificationRepo) DeleteByID(ctx context.Context, id, workspaceID, userID uint) (int64, error) {
	query := r.db.WithContext(ctx).Where("id = ? AND (user_id = ? OR user_id = 0)", id, userID)

	// Always require workspace scoping - callers must resolve workspaceID before calling.
	if workspaceID != 0 {
		query = query.Where("workspace_id = ?", workspaceID)
	} else {
		// Safety: refuse to delete without workspace scope to prevent cross-tenant access.
		return 0, nil
	}

	result := query.Delete(&Notification{})
	if result.Error != nil {
		return 0, fmt.Errorf("deleting notification: %w", result.Error)
	}
	return result.RowsAffected, nil
}

func (r *NotificationRepo) DeleteAll(ctx context.Context, workspaceID, userID uint) (int64, error) {
	query := r.db.WithContext(ctx).Where("user_id = ? OR user_id = 0", userID)

	// Always require workspace scoping - callers must resolve workspaceID before calling.
	if workspaceID != 0 {
		query = query.Where("workspace_id = ?", workspaceID)
	} else {
		// Safety: refuse to delete without workspace scope to prevent cross-tenant access.
		return 0, nil
	}

	result := query.Delete(&Notification{})
	if result.Error != nil {
		return 0, fmt.Errorf("deleting all notifications: %w", result.Error)
	}
	return result.RowsAffected, nil
}

func (r *NotificationRepo) DeleteAllByWorkspaceIDs(ctx context.Context, workspaceIDs []uint, userID uint) (int64, error) {
	if len(workspaceIDs) == 0 {
		return 0, nil
	}

	result := r.db.WithContext(ctx).
		Where("workspace_id IN ?", workspaceIDs).
		Where("user_id = ? OR user_id = 0", userID).
		Delete(&Notification{})
	if result.Error != nil {
		return 0, fmt.Errorf("deleting all notifications by workspace IDs: %w", result.Error)
	}
	return result.RowsAffected, nil
}

func (r *NotificationRepo) DeleteBatch(ctx context.Context, ids []uint, workspaceID, userID uint) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}

	query := r.db.WithContext(ctx).Where("id IN ? AND (user_id = ? OR user_id = 0)", ids, userID)

	// Always require workspace scoping - callers must resolve workspaceID before calling.
	if workspaceID != 0 {
		query = query.Where("workspace_id = ?", workspaceID)
	} else {
		// Safety: refuse to delete without workspace scope to prevent cross-tenant access.
		return 0, nil
	}

	result := query.Delete(&Notification{})
	if result.Error != nil {
		return 0, fmt.Errorf("batch deleting notifications: %w", result.Error)
	}
	return result.RowsAffected, nil
}

func (r *NotificationRepo) FindByID(ctx context.Context, id uint) (*entity.Notification, error) {
	var m Notification
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&m).Error; err != nil {
		return nil, fmt.Errorf("finding notification by ID: %w", err)
	}
	return notifToDomain(&m), nil
}

func (r *NotificationRepo) DeleteBatchByWorkspaceIDs(ctx context.Context, ids []uint, workspaceIDs []uint, userID uint) (int64, error) {
	if len(ids) == 0 || len(workspaceIDs) == 0 {
		return 0, nil
	}

	result := r.db.WithContext(ctx).
		Where("id IN ? AND workspace_id IN ? AND (user_id = ? OR user_id = 0)", ids, workspaceIDs, userID).
		Delete(&Notification{})
	if result.Error != nil {
		return 0, fmt.Errorf("batch deleting notifications by workspace IDs: %w", result.Error)
	}
	return result.RowsAffected, nil
}

// --- Converters ---

func notifToDomain(m *Notification) *entity.Notification {
	return &entity.Notification{
		ID:            m.ID,
		WorkspaceID:         m.WorkspaceID,
		UserID:        m.UserID,
		ChannelID:     m.ChannelID,
		Severity:      m.Severity,
		EventType:     m.EventType,
		ReferenceID:   m.ReferenceID,
		ReferenceType: m.ReferenceType,
		Title:         m.Title,
		Message:       m.Message,
		IsRead:        m.IsRead,
		SentAt:        m.SentAt,
		CreatedAt:     m.CreatedAt,
	}
}

func notifToModel(d *entity.Notification) *Notification {
	return &Notification{
		ID:            d.ID,
		WorkspaceID:         d.WorkspaceID,
		UserID:        d.UserID,
		ChannelID:     d.ChannelID,
		Severity:      d.Severity,
		EventType:     d.EventType,
		ReferenceID:   d.ReferenceID,
		ReferenceType: d.ReferenceType,
		Title:         d.Title,
		Message:       d.Message,
		IsRead:        d.IsRead,
		SentAt:        d.SentAt,
		CreatedAt:     d.CreatedAt,
	}
}
