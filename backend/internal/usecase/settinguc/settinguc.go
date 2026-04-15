// Package settinguc implements the business logic for settings management.
package settinguc

import (
	"context"
	"fmt"
	"net/mail"
	"strconv"
	"strings"
	"time"

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

		// Validate numeric settings have sane ranges.
		switch key {
		case entity.SettingDiscoveryScanDepth:
			n, err := strconv.Atoi(value)
			if err != nil || n < 1 || n > 1000 {
				return nil, fmt.Errorf("discovery_scan_depth must be an integer between 1 and 1000")
			}
		case entity.SettingDiffSizeLimit:
			n, err := strconv.Atoi(value)
			if err != nil || n < 1024 || n > 10485760 {
				return nil, fmt.Errorf("diff_size_limit must be an integer between 1024 and 10485760")
			}
		case entity.SettingMonitoringInterval:
			if err := validateDurationRange(value, "monitoring_interval"); err != nil {
				return nil, err
			}
		case entity.SettingDiscoveryInterval:
			if err := validateDurationRange(value, "discovery_interval"); err != nil {
				return nil, err
			}
		case entity.SettingEmailDigestEnabled:
			if value != "true" && value != "false" {
				return nil, fmt.Errorf("email_digest_enabled must be 'true' or 'false'")
			}
		case entity.SettingEmailDigestFrequency:
			if value != "daily" && value != "weekly" {
				return nil, fmt.Errorf("email_digest_frequency must be 'daily' or 'weekly'")
			}
		case entity.SettingEmailDigestRecipients:
			if err := validateEmailRecipients(value); err != nil {
				return nil, err
			}
		case entity.SettingDiscoveryAutoApprove:
			if value != "true" && value != "false" {
				return nil, fmt.Errorf("discovery_auto_approve must be 'true' or 'false'")
			}
		case entity.SettingStaleAutoRemoveMonths:
			n, err := strconv.Atoi(value)
			if err != nil || n < 0 || n > 36 {
				return nil, fmt.Errorf("stale_auto_remove_months must be an integer between 0 and 36")
			}
		case entity.SettingPackageCountWarningThreshold:
			n, err := strconv.Atoi(value)
			if err != nil || n < 1 || n > 10000 {
				return nil, fmt.Errorf("package_count_warning_threshold must be an integer between 1 and 10000")
			}
		}

		if err := uc.settings.UpsertByOrgAndKey(ctx, orgID, key, value); err != nil {
			return nil, fmt.Errorf("SettingUseCase.UpdateSettings: upserting %s: %w", key, err)
		}
	}

	// Return updated settings
	return uc.GetSettings(ctx, orgID)
}

// validateDurationRange parses a Go duration string and checks it is between 1m and 168h.
func validateDurationRange(value, fieldName string) error {
	d, err := time.ParseDuration(value)
	if err != nil {
		return fmt.Errorf("%s must be a valid duration (e.g. '5m', '1h')", fieldName)
	}
	if d < time.Minute || d > 168*time.Hour {
		return fmt.Errorf("%s must be between 1m and 168h", fieldName)
	}
	return nil
}

// validateEmailRecipients validates a comma-separated list of email addresses.
// At least one valid address is required. Each address is checked using net/mail.ParseAddress.
func validateEmailRecipients(value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("email_digest_recipients must contain at least one email address")
	}
	parts := strings.Split(value, ",")
	for _, part := range parts {
		addr := strings.TrimSpace(part)
		if addr == "" {
			continue
		}
		if _, err := mail.ParseAddress(addr); err != nil {
			return fmt.Errorf("email_digest_recipients contains invalid email: %s", addr)
		}
	}
	return nil
}
