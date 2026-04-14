// Package settinguc implements the business logic for settings management.
package settinguc

import (
	"context"
	"fmt"

	"github.com/veilence/veilence-mx/backend/internal/entity"
	"github.com/veilence/veilence-mx/backend/internal/usecase"
)

// UseCase implements usecase.SettingService.
type UseCase struct {
	settings usecase.SettingRepository
}

// New creates a new setting UseCase.
func New(settings usecase.SettingRepository) *UseCase {
	return &UseCase{settings: settings}
}

// GetSettings returns all settings as a key-value map for the given org.
func (uc *UseCase) GetSettings(ctx context.Context, orgID uint) (map[string]string, error) {
	settings, err := uc.settings.FindByOrgID(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("SettingUseCase.GetSettings: %w", err)
	}

	result := make(map[string]string, len(settings))
	for _, s := range settings {
		result[s.Key] = s.Value
	}
	return result, nil
}

// UpdateSettings updates settings from a key-value map for the given org.
// Returns the updated settings map.
func (uc *UseCase) UpdateSettings(ctx context.Context, orgID uint, settings map[string]string) (map[string]string, error) {
	for key, value := range settings {
		if !entity.ValidSettingKeys[key] {
			return nil, fmt.Errorf("invalid setting key: %s", key)
		}
		if err := uc.settings.UpsertByOrgAndKey(ctx, orgID, key, value); err != nil {
			return nil, fmt.Errorf("SettingUseCase.UpdateSettings: upserting %s: %w", key, err)
		}
	}

	// Return updated settings
	return uc.GetSettings(ctx, orgID)
}
