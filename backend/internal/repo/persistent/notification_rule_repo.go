package persistent

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/veilence/veilence-mx/backend/internal/entity"
)

// NotificationRuleRepo implements entity.NotificationRuleRepository using GORM.
type NotificationRuleRepo struct {
	db *gorm.DB
}

// NewNotificationRuleRepo creates a new NotificationRuleRepo.
func NewNotificationRuleRepo(db *gorm.DB) *NotificationRuleRepo {
	return &NotificationRuleRepo{db: db}
}

func (r *NotificationRuleRepo) FindByWorkspaceID(ctx context.Context, workspaceID uint) ([]entity.NotificationRule, error) {
	var ms []NotificationRule
	if err := r.db.WithContext(ctx).Where("workspace_id = ?", workspaceID).Find(&ms).Error; err != nil {
		return nil, fmt.Errorf("listing notification rules: %w", err)
	}
	result := make([]entity.NotificationRule, len(ms))
	for i := range ms {
		result[i] = *notifRuleToDomain(&ms[i])
	}
	return result, nil
}

func (r *NotificationRuleRepo) FindActiveByWorkspaceID(ctx context.Context, workspaceID uint) ([]entity.NotificationRule, error) {
	var ms []NotificationRule
	if err := r.db.WithContext(ctx).Where("workspace_id = ? AND is_active = ?", workspaceID, true).Find(&ms).Error; err != nil {
		return nil, fmt.Errorf("listing active notification rules: %w", err)
	}
	result := make([]entity.NotificationRule, len(ms))
	for i := range ms {
		result[i] = *notifRuleToDomain(&ms[i])
	}
	return result, nil
}

func (r *NotificationRuleRepo) Create(ctx context.Context, rule *entity.NotificationRule) error {
	m := notifRuleToModel(rule)
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return fmt.Errorf("creating notification rule: %w", err)
	}
	rule.ID = m.ID
	rule.CreatedAt = m.CreatedAt
	rule.UpdatedAt = m.UpdatedAt
	return nil
}

func (r *NotificationRuleRepo) DeleteByIDAndWorkspace(ctx context.Context, id, workspaceID uint) (int64, error) {
	result := r.db.WithContext(ctx).Where("id = ? AND workspace_id = ?", id, workspaceID).Delete(&NotificationRule{})
	if result.Error != nil {
		return 0, fmt.Errorf("deleting notification rule: %w", result.Error)
	}
	return result.RowsAffected, nil
}

// --- Converters ---

func notifRuleToDomain(m *NotificationRule) *entity.NotificationRule {
	return &entity.NotificationRule{
		ID:        m.ID,
		WorkspaceID:     m.WorkspaceID,
		ChannelID: m.ChannelID,
		Severity:  m.Severity,
		IsActive:  m.IsActive,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}

func notifRuleToModel(d *entity.NotificationRule) *NotificationRule {
	return &NotificationRule{
		ID:        d.ID,
		WorkspaceID:     d.WorkspaceID,
		ChannelID: d.ChannelID,
		Severity:  d.Severity,
		IsActive:  d.IsActive,
		CreatedAt: d.CreatedAt,
		UpdatedAt: d.UpdatedAt,
	}
}
