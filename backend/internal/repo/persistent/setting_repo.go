package persistent

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/veilence/veilence-mx/backend/internal/entity"
)

// SettingRepo implements entity.SettingRepository using GORM.
type SettingRepo struct {
	db *gorm.DB
}

// NewSettingRepo creates a new SettingRepo.
func NewSettingRepo(db *gorm.DB) *SettingRepo {
	return &SettingRepo{db: db}
}

func (r *SettingRepo) FindByOrgID(ctx context.Context, orgID uint) ([]entity.Setting, error) {
	var ms []Setting
	if err := r.db.WithContext(ctx).Where("org_id = ?", orgID).Find(&ms).Error; err != nil {
		return nil, fmt.Errorf("listing settings: %w", err)
	}
	result := make([]entity.Setting, len(ms))
	for i := range ms {
		result[i] = *settingToDomain(&ms[i])
	}
	return result, nil
}

func (r *SettingRepo) FindByKey(ctx context.Context, orgID uint, key string) (*entity.Setting, error) {
	var m Setting
	err := r.db.WithContext(ctx).Where("org_id = ? AND key = ?", orgID, key).First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("setting %w", entity.ErrNotFound)
		}
		return nil, fmt.Errorf("finding setting: %w", err)
	}
	return settingToDomain(&m), nil
}

func (r *SettingRepo) Upsert(ctx context.Context, setting *entity.Setting) error {
	m := settingToModel(setting)
	result := r.db.WithContext(ctx).
		Where("org_id = ? AND key = ?", m.OrgID, m.Key).
		Assign(Setting{Value: m.Value}).
		FirstOrCreate(m)
	if result.Error != nil {
		return fmt.Errorf("upserting setting: %w", result.Error)
	}
	setting.ID = m.ID
	setting.CreatedAt = m.CreatedAt
	setting.UpdatedAt = m.UpdatedAt
	return nil
}

func (r *SettingRepo) FindOrCreateByKey(ctx context.Context, key, defaultValue string) (*entity.Setting, error) {
	m := &Setting{Key: key, Value: defaultValue}
	if err := r.db.WithContext(ctx).Where("key = ?", key).FirstOrCreate(m).Error; err != nil {
		return nil, fmt.Errorf("find or create setting: %w", err)
	}
	return settingToDomain(m), nil
}

func (r *SettingRepo) UpsertByOrgAndKey(ctx context.Context, orgID uint, key, value string) error {
	m := &Setting{OrgID: orgID, Key: key, Value: value}
	result := r.db.WithContext(ctx).
		Where("org_id = ? AND key = ?", orgID, key).
		Assign(Setting{Value: value}).
		FirstOrCreate(m)
	if result.Error != nil {
		return fmt.Errorf("upserting setting by org and key: %w", result.Error)
	}
	return nil
}

// GetSettingValue implements usecase.SettingGetter. It returns the value of a
// single setting by orgID and key. Returns entity.ErrNotFound if the key does
// not exist.
func (r *SettingRepo) GetSettingValue(ctx context.Context, orgID uint, key string) (string, error) {
	setting, err := r.FindByKey(ctx, orgID, key)
	if err != nil {
		return "", err
	}
	return setting.Value, nil
}

// --- Converters ---

func settingToDomain(m *Setting) *entity.Setting {
	return &entity.Setting{
		ID:        m.ID,
		OrgID:     m.OrgID,
		Key:       m.Key,
		Value:     m.Value,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}

func settingToModel(d *entity.Setting) *Setting {
	return &Setting{
		ID:        d.ID,
		OrgID:     d.OrgID,
		Key:       d.Key,
		Value:     d.Value,
		CreatedAt: d.CreatedAt,
		UpdatedAt: d.UpdatedAt,
	}
}
