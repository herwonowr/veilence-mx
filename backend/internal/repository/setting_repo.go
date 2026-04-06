package repository

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/veilence/veilence-mx/backend/internal/domain"
	"github.com/veilence/veilence-mx/backend/internal/models"
)

// SettingRepo implements domain.SettingRepository using GORM.
type SettingRepo struct {
	db *gorm.DB
}

// NewSettingRepo creates a new SettingRepo.
func NewSettingRepo(db *gorm.DB) *SettingRepo {
	return &SettingRepo{db: db}
}

func (r *SettingRepo) FindByOrgID(ctx context.Context, orgID uint) ([]domain.Setting, error) {
	var ms []models.Setting
	if err := r.db.WithContext(ctx).Where("org_id = ?", orgID).Find(&ms).Error; err != nil {
		return nil, fmt.Errorf("listing settings: %w", err)
	}
	result := make([]domain.Setting, len(ms))
	for i := range ms {
		result[i] = *settingToDomain(&ms[i])
	}
	return result, nil
}

func (r *SettingRepo) FindByKey(ctx context.Context, orgID uint, key string) (*domain.Setting, error) {
	var m models.Setting
	err := r.db.WithContext(ctx).Where("org_id = ? AND key = ?", orgID, key).First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("setting %w", domain.ErrNotFound)
		}
		return nil, fmt.Errorf("finding setting: %w", err)
	}
	return settingToDomain(&m), nil
}

func (r *SettingRepo) Upsert(ctx context.Context, setting *domain.Setting) error {
	m := settingToModel(setting)
	result := r.db.WithContext(ctx).
		Where("org_id = ? AND key = ?", m.OrgID, m.Key).
		Assign(models.Setting{Value: m.Value}).
		FirstOrCreate(m)
	if result.Error != nil {
		return fmt.Errorf("upserting setting: %w", result.Error)
	}
	setting.ID = m.ID
	setting.CreatedAt = m.CreatedAt
	setting.UpdatedAt = m.UpdatedAt
	return nil
}

func (r *SettingRepo) FindOrCreateByKey(ctx context.Context, key, defaultValue string) (*domain.Setting, error) {
	m := &models.Setting{Key: key, Value: defaultValue}
	if err := r.db.WithContext(ctx).Where("key = ?", key).FirstOrCreate(m).Error; err != nil {
		return nil, fmt.Errorf("find or create setting: %w", err)
	}
	return settingToDomain(m), nil
}

// --- Converters ---

func settingToDomain(m *models.Setting) *domain.Setting {
	return &domain.Setting{
		ID:        m.ID,
		OrgID:     m.OrgID,
		Key:       m.Key,
		Value:     m.Value,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}

func settingToModel(d *domain.Setting) *models.Setting {
	return &models.Setting{
		ID:        d.ID,
		OrgID:     d.OrgID,
		Key:       d.Key,
		Value:     d.Value,
		CreatedAt: d.CreatedAt,
		UpdatedAt: d.UpdatedAt,
	}
}
