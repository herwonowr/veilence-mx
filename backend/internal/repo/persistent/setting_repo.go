package persistent

import (
	"context"
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

func (r *SettingRepo) FindByWorkspaceID(ctx context.Context, workspaceID string) ([]entity.Setting, error) {
	var ms []Setting
	if err := r.db.WithContext(ctx).Where("workspace_id = ?", workspaceID).Find(&ms).Error; err != nil {
		return nil, fmt.Errorf("listing settings: %w", err)
	}
	result := make([]entity.Setting, len(ms))
	for i := range ms {
		result[i] = *settingToDomain(&ms[i])
	}
	return result, nil
}

func (r *SettingRepo) UpsertByWorkspaceAndKey(ctx context.Context, workspaceID string, key, value string) error {
	m := &Setting{WorkspaceID: &workspaceID, Key: key, Value: value}
	result := r.db.WithContext(ctx).
		Where("workspace_id = ? AND key = ?", workspaceID, key).
		Assign(Setting{Value: value}).
		FirstOrCreate(m)
	if result.Error != nil {
		return fmt.Errorf("upserting setting by workspace and key: %w", result.Error)
	}
	return nil
}

func (r *SettingRepo) FindPlatformSettings(ctx context.Context, keys []string) ([]entity.Setting, error) {
	var ms []Setting
	if err := r.db.WithContext(ctx).Where("workspace_id IS NULL AND key IN ?", keys).Find(&ms).Error; err != nil {
		return nil, fmt.Errorf("SettingRepo.FindPlatformSettings: %w", err)
	}
	result := make([]entity.Setting, len(ms))
	for i := range ms {
		result[i] = *settingToDomain(&ms[i])
	}
	return result, nil
}

func (r *SettingRepo) UpsertPlatformSetting(ctx context.Context, key, value string) error {
	m := &Setting{Key: key, Value: value}
	result := r.db.WithContext(ctx).
		Where("workspace_id IS NULL AND key = ?", key).
		Assign(Setting{Value: value}).
		FirstOrCreate(m)
	if result.Error != nil {
		return fmt.Errorf("SettingRepo.UpsertPlatformSetting: %w", result.Error)
	}
	return nil
}

// --- Converters ---

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func settingToDomain(m *Setting) *entity.Setting {
	return &entity.Setting{
		ID:          m.ID,
		WorkspaceID: derefString(m.WorkspaceID),
		Key:         m.Key,
		Value:       m.Value,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}
}
