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

// GetSettings returns all settings as a key-value map for the given workspace.
func (uc *UseCase) GetSettings(ctx context.Context, workspaceID string) (map[string]string, error) {
	settings, err := uc.settings.FindByWorkspaceID(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("SettingUseCase.GetSettings: %w", err)
	}

	result := make(map[string]string, len(settings))
	for _, s := range settings {
		result[s.Key] = s.Value
	}

	return result, nil
}

// validationError returns a validation error wrapping entity.ErrValidation so
// callers can distinguish it from infrastructure errors using errors.Is.
func validationError(msg string) error {
	return fmt.Errorf("%s: %w", msg, entity.ErrValidation)
}

// UpdateSettings updates settings from a key-value map for the given workspace.
// Returns the updated settings map.
// Validation failures are wrapped with entity.ErrValidation.
func (uc *UseCase) UpdateSettings(ctx context.Context, workspaceID string, settings map[string]string) (map[string]string, error) {
	for key, value := range settings {
		if !entity.ValidSettingKeys[key] {
			return nil, validationError(fmt.Sprintf("invalid setting key: %s", key))
		}

		// Validate settings have sane values.
		switch key {
		case entity.SettingDiscoveryScanDepth:
			n, err := strconv.Atoi(value)
			if err != nil || n < 1 || n > 1000 {
				return nil, validationError("discovery_scan_depth must be an integer between 1 and 1000")
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
				return nil, validationError("email_digest_enabled must be 'true' or 'false'")
			}
		case entity.SettingEmailDigestFrequency:
			if value != "daily" && value != "weekly" {
				return nil, validationError("email_digest_frequency must be 'daily' or 'weekly'")
			}
		case entity.SettingEmailDigestRecipients:
			if err := validateEmailRecipients(value); err != nil {
				return nil, err
			}
		case entity.SettingDiscoveryAutoApprove:
			if value != "true" && value != "false" {
				return nil, validationError("discovery_auto_approve must be 'true' or 'false'")
			}
		case entity.SettingStaleAutoRemoveMonths:
			n, err := strconv.Atoi(value)
			if err != nil || n < 0 || n > 36 {
				return nil, validationError("stale_auto_remove_months must be an integer between 0 and 36")
			}
		case entity.SettingPackageCountWarningThreshold:
			n, err := strconv.Atoi(value)
			if err != nil || n < 0 || n > 100000 {
				return nil, validationError("package_count_warning_threshold must be an integer between 0 and 100000")
			}
		}

		if err := uc.settings.UpsertByWorkspaceAndKey(ctx, workspaceID, key, value); err != nil {
			return nil, fmt.Errorf("SettingUseCase.UpdateSettings: upserting %s: %w", key, err)
		}
	}

	// Return updated settings
	return uc.GetSettings(ctx, workspaceID)
}

// validateDurationRange parses a Go duration string and checks it is between 1m and 168h.
func validateDurationRange(value, fieldName string) error {
	d, err := time.ParseDuration(value)
	if err != nil {
		return validationError(fmt.Sprintf("%s must be a valid duration (e.g. '5m', '1h')", fieldName))
	}
	if d < time.Minute || d > 168*time.Hour {
		return validationError(fmt.Sprintf("%s must be between 1m and 168h", fieldName))
	}
	return nil
}

// validateEmailRecipients validates a comma-separated list of email addresses.
// At least one valid address is required. Each address is checked using net/mail.ParseAddress.
func validateEmailRecipients(value string) error {
	if strings.TrimSpace(value) == "" {
		return validationError("email_digest_recipients must contain at least one email address")
	}
	parts := strings.Split(value, ",")
	for _, part := range parts {
		addr := strings.TrimSpace(part)
		if addr == "" {
			continue
		}
		if _, err := mail.ParseAddress(addr); err != nil {
			return validationError(fmt.Sprintf("email_digest_recipients contains invalid email: %s", addr))
		}
	}
	return nil
}
