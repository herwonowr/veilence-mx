package repository

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/veilence/veilence-mx/backend/internal/domain"
	"github.com/veilence/veilence-mx/backend/internal/models"
)

// NotificationRuleRepo implements domain.NotificationRuleRepository using GORM.
type NotificationRuleRepo struct {
	db *gorm.DB
}

// NewNotificationRuleRepo creates a new NotificationRuleRepo.
func NewNotificationRuleRepo(db *gorm.DB) *NotificationRuleRepo {
	return &NotificationRuleRepo{db: db}
}

func (r *NotificationRuleRepo) FindByOrgID(ctx context.Context, orgID uint) ([]domain.NotificationRule, error) {
	var ms []models.NotificationRule
	if err := r.db.WithContext(ctx).Where("org_id = ?", orgID).Find(&ms).Error; err != nil {
		return nil, fmt.Errorf("listing notification rules: %w", err)
	}
	result := make([]domain.NotificationRule, len(ms))
	for i := range ms {
		result[i] = *notifRuleToDomain(&ms[i])
	}
	return result, nil
}

func (r *NotificationRuleRepo) FindActiveByOrgID(ctx context.Context, orgID uint) ([]domain.NotificationRule, error) {
	var ms []models.NotificationRule
	if err := r.db.WithContext(ctx).Where("org_id = ? AND is_active = ?", orgID, true).Find(&ms).Error; err != nil {
		return nil, fmt.Errorf("listing active notification rules: %w", err)
	}
	result := make([]domain.NotificationRule, len(ms))
	for i := range ms {
		result[i] = *notifRuleToDomain(&ms[i])
	}
	return result, nil
}

func (r *NotificationRuleRepo) Create(ctx context.Context, rule *domain.NotificationRule) error {
	m := notifRuleToModel(rule)
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return fmt.Errorf("creating notification rule: %w", err)
	}
	rule.ID = m.ID
	rule.CreatedAt = m.CreatedAt
	rule.UpdatedAt = m.UpdatedAt
	return nil
}

func (r *NotificationRuleRepo) DeleteByIDAndOrg(ctx context.Context, id, orgID uint) (int64, error) {
	result := r.db.WithContext(ctx).Where("id = ? AND org_id = ?", id, orgID).Delete(&models.NotificationRule{})
	if result.Error != nil {
		return 0, fmt.Errorf("deleting notification rule: %w", result.Error)
	}
	return result.RowsAffected, nil
}

// --- Converters ---

func notifRuleToDomain(m *models.NotificationRule) *domain.NotificationRule {
	return &domain.NotificationRule{
		ID:        m.ID,
		OrgID:     m.OrgID,
		ChannelID: m.ChannelID,
		Severity:  m.Severity,
		IsActive:  m.IsActive,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}

func notifRuleToModel(d *domain.NotificationRule) *models.NotificationRule {
	return &models.NotificationRule{
		ID:        d.ID,
		OrgID:     d.OrgID,
		ChannelID: d.ChannelID,
		Severity:  d.Severity,
		IsActive:  d.IsActive,
		CreatedAt: d.CreatedAt,
		UpdatedAt: d.UpdatedAt,
	}
}
