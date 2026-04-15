package settinguc_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/veilence/veilence-mx/backend/internal/entity"
	"github.com/veilence/veilence-mx/backend/internal/usecase/settinguc"
)

// ---------------------------------------------------------------------------
// Mock SettingRepository
// ---------------------------------------------------------------------------

// mockSettingRepo is a simple in-memory mock of usecase.SettingRepository.
type mockSettingRepo struct {
	// store holds settings keyed by "orgID:key".
	store map[string]entity.Setting

	// errors lets tests inject errors for specific methods.
	findByOrgIDErr    error
	upsertByOrgKeyErr error
}

func newMockRepo() *mockSettingRepo {
	return &mockSettingRepo{store: make(map[string]entity.Setting)}
}

func storeKey(orgID uint, key string) string {
	return fmt.Sprintf("%d:%s", orgID, key)
}

func (m *mockSettingRepo) FindByOrgID(_ context.Context, orgID uint) ([]entity.Setting, error) {
	if m.findByOrgIDErr != nil {
		return nil, m.findByOrgIDErr
	}
	var result []entity.Setting
	prefix := fmt.Sprintf("%d:", orgID)
	for k, v := range m.store {
		if len(k) >= len(prefix) && k[:len(prefix)] == prefix {
			result = append(result, v)
		}
	}
	return result, nil
}

func (m *mockSettingRepo) FindByKey(_ context.Context, orgID uint, key string) (*entity.Setting, error) {
	s, ok := m.store[storeKey(orgID, key)]
	if !ok {
		return nil, entity.ErrNotFound
	}
	return &s, nil
}

func (m *mockSettingRepo) Upsert(_ context.Context, setting *entity.Setting) error {
	m.store[storeKey(setting.OrgID, setting.Key)] = *setting
	return nil
}

func (m *mockSettingRepo) UpsertByOrgAndKey(_ context.Context, orgID uint, key, value string) error {
	if m.upsertByOrgKeyErr != nil {
		return m.upsertByOrgKeyErr
	}
	m.store[storeKey(orgID, key)] = entity.Setting{OrgID: orgID, Key: key, Value: value}
	return nil
}

func (m *mockSettingRepo) FindOrCreateByKey(_ context.Context, key, defaultValue string) (*entity.Setting, error) {
	sk := storeKey(0, key)
	s, ok := m.store[sk]
	if !ok {
		s = entity.Setting{Key: key, Value: defaultValue}
		m.store[sk] = s
	}
	return &s, nil
}

// ---------------------------------------------------------------------------
// GetSettings
// ---------------------------------------------------------------------------

func TestGetSettings_ReturnsKeyValueMap(t *testing.T) {
	repo := newMockRepo()
	repo.store[storeKey(1, "monitoring_interval")] = entity.Setting{OrgID: 1, Key: "monitoring_interval", Value: "5m"}
	repo.store[storeKey(1, "analyzer_mode")] = entity.Setting{OrgID: 1, Key: "analyzer_mode", Value: "auto"}

	uc := settinguc.New(repo)
	result, err := uc.GetSettings(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, "5m", result["monitoring_interval"])
	assert.Equal(t, "auto", result["analyzer_mode"])
	assert.Len(t, result, 2)
}

func TestGetSettings_RepoError(t *testing.T) {
	repo := newMockRepo()
	repo.findByOrgIDErr = fmt.Errorf("db connection lost")

	uc := settinguc.New(repo)
	_, err := uc.GetSettings(context.Background(), 1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "db connection lost")
}

func TestGetSettings_EmptyOrg(t *testing.T) {
	repo := newMockRepo()
	uc := settinguc.New(repo)
	result, err := uc.GetSettings(context.Background(), 99)
	require.NoError(t, err)
	assert.Empty(t, result)
}

// ---------------------------------------------------------------------------
// UpdateSettings — invalid key
// ---------------------------------------------------------------------------

func TestUpdateSettings_InvalidKey(t *testing.T) {
	repo := newMockRepo()
	uc := settinguc.New(repo)

	_, err := uc.UpdateSettings(context.Background(), 1, map[string]string{
		"totally_invalid_key": "value",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid setting key")
}

// ---------------------------------------------------------------------------
// UpdateSettings — discovery_scan_depth validation
// ---------------------------------------------------------------------------

func TestUpdateSettings_DiscoveryScanDepth(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"valid min", "1", false},
		{"valid mid", "500", false},
		{"valid max", "1000", false},
		{"too low", "0", true},
		{"too high", "1001", true},
		{"negative", "-1", true},
		{"not a number", "abc", true},
		{"float", "1.5", true},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockRepo()
			uc := settinguc.New(repo)
			_, err := uc.UpdateSettings(context.Background(), 1, map[string]string{
				entity.SettingDiscoveryScanDepth: tt.value,
			})
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "discovery_scan_depth")
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// UpdateSettings — diff_size_limit validation
// ---------------------------------------------------------------------------

func TestUpdateSettings_DiffSizeLimit(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"valid min", "1024", false},
		{"valid mid", "5242880", false},
		{"valid max", "10485760", false},
		{"too low", "1023", true},
		{"too high", "10485761", true},
		{"zero", "0", true},
		{"not a number", "abc", true},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockRepo()
			uc := settinguc.New(repo)
			_, err := uc.UpdateSettings(context.Background(), 1, map[string]string{
				entity.SettingDiffSizeLimit: tt.value,
			})
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "diff_size_limit")
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// UpdateSettings — monitoring_interval validation
// ---------------------------------------------------------------------------

func TestUpdateSettings_MonitoringInterval(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"valid 1 minute", "1m", false},
		{"valid 5 minutes", "5m", false},
		{"valid 1 hour", "1h", false},
		{"valid 24 hours", "24h", false},
		{"valid 168 hours (max)", "168h", false},
		{"valid compound", "1h30m", false},
		{"too short — 59 seconds", "59s", true},
		{"too short — 0s", "0s", true},
		{"too long — 169 hours", "169h", true},
		{"too long — 200h", "200h", true},
		{"not a duration", "abc", true},
		{"empty", "", true},
		{"negative", "-5m", true},
		{"just a number", "300", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockRepo()
			uc := settinguc.New(repo)
			_, err := uc.UpdateSettings(context.Background(), 1, map[string]string{
				entity.SettingMonitoringInterval: tt.value,
			})
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "monitoring_interval")
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// UpdateSettings — discovery_interval validation
// ---------------------------------------------------------------------------

func TestUpdateSettings_DiscoveryInterval(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"valid 1 minute", "1m", false},
		{"valid 1 hour", "1h", false},
		{"valid 168 hours (max)", "168h", false},
		{"too short", "30s", true},
		{"too long", "169h", true},
		{"not a duration", "nope", true},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockRepo()
			uc := settinguc.New(repo)
			_, err := uc.UpdateSettings(context.Background(), 1, map[string]string{
				entity.SettingDiscoveryInterval: tt.value,
			})
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "discovery_interval")
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// UpdateSettings — email_digest_enabled validation
// ---------------------------------------------------------------------------

func TestUpdateSettings_EmailDigestEnabled(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"true", "true", false},
		{"false", "false", false},
		{"True (wrong case)", "True", true},
		{"yes", "yes", true},
		{"1", "1", true},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockRepo()
			uc := settinguc.New(repo)
			_, err := uc.UpdateSettings(context.Background(), 1, map[string]string{
				entity.SettingEmailDigestEnabled: tt.value,
			})
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "email_digest_enabled")
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// UpdateSettings — email_digest_frequency validation
// ---------------------------------------------------------------------------

func TestUpdateSettings_EmailDigestFrequency(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"daily", "daily", false},
		{"weekly", "weekly", false},
		{"monthly", "monthly", true},
		{"Daily (wrong case)", "Daily", true},
		{"empty", "", true},
		{"hourly", "hourly", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockRepo()
			uc := settinguc.New(repo)
			_, err := uc.UpdateSettings(context.Background(), 1, map[string]string{
				entity.SettingEmailDigestFrequency: tt.value,
			})
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "email_digest_frequency")
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// UpdateSettings — email_digest_recipients validation
// ---------------------------------------------------------------------------

func TestUpdateSettings_EmailDigestRecipients(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"single valid email", "alice@example.com", false},
		{"two valid emails", "alice@example.com,bob@example.com", false},
		{"emails with spaces", " alice@example.com , bob@example.com ", false},
		{"email with display name", "Alice <alice@example.com>", false},
		{"empty string", "", true},
		{"only spaces", "   ", true},
		{"invalid email", "not-an-email", true},
		{"one valid one invalid", "alice@example.com,bad", true},
		{"missing domain", "alice@", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockRepo()
			uc := settinguc.New(repo)
			_, err := uc.UpdateSettings(context.Background(), 1, map[string]string{
				entity.SettingEmailDigestRecipients: tt.value,
			})
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "email_digest_recipients")
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// UpdateSettings — valid settings are persisted
// ---------------------------------------------------------------------------

func TestUpdateSettings_PersistsValidSettings(t *testing.T) {
	repo := newMockRepo()
	uc := settinguc.New(repo)

	result, err := uc.UpdateSettings(context.Background(), 1, map[string]string{
		entity.SettingMonitoringInterval:    "10m",
		entity.SettingEmailDigestEnabled:    "true",
		entity.SettingEmailDigestFrequency:  "daily",
		entity.SettingEmailDigestRecipients: "alice@example.com",
		entity.SettingDiscoveryScanDepth:    "100",
	})
	require.NoError(t, err)
	assert.Equal(t, "10m", result["monitoring_interval"])
	assert.Equal(t, "true", result["email_digest_enabled"])
	assert.Equal(t, "daily", result["email_digest_frequency"])
	assert.Equal(t, "alice@example.com", result["email_digest_recipients"])
	assert.Equal(t, "100", result["discovery_scan_depth"])
}

// ---------------------------------------------------------------------------
// UpdateSettings — repo error propagation
// ---------------------------------------------------------------------------

func TestUpdateSettings_RepoUpsertError(t *testing.T) {
	repo := newMockRepo()
	repo.upsertByOrgKeyErr = fmt.Errorf("database write failed")

	uc := settinguc.New(repo)
	_, err := uc.UpdateSettings(context.Background(), 1, map[string]string{
		entity.SettingMonitoringInterval: "5m",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "database write failed")
}

// ---------------------------------------------------------------------------
// UpdateSettings — analyzer_mode validation
// ---------------------------------------------------------------------------

func TestUpdateSettings_AnalyzerMode(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"auto", "auto", false},
		{"manual", "manual", false},
		{"disabled", "disabled", false},
		{"Auto (wrong case)", "Auto", true},
		{"MANUAL (wrong case)", "MANUAL", true},
		{"copilot (not a mode)", "copilot", true},
		{"api (not a mode)", "api", true},
		{"empty", "", true},
		{"arbitrary string", "foobar", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockRepo()
			uc := settinguc.New(repo)
			_, err := uc.UpdateSettings(context.Background(), 1, map[string]string{
				entity.SettingAnalyzerMode: tt.value,
			})
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "analyzer_mode")
				assert.ErrorIs(t, err, entity.ErrValidation)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// UpdateSettings — discovery_auto_approve validation
// ---------------------------------------------------------------------------

func TestUpdateSettings_DiscoveryAutoApprove(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"true", "true", false},
		{"false", "false", false},
		{"True (wrong case)", "True", true},
		{"yes", "yes", true},
		{"1", "1", true},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockRepo()
			uc := settinguc.New(repo)
			_, err := uc.UpdateSettings(context.Background(), 1, map[string]string{
				entity.SettingDiscoveryAutoApprove: tt.value,
			})
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "discovery_auto_approve")
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// UpdateSettings — stale_auto_remove_months validation
// ---------------------------------------------------------------------------

func TestUpdateSettings_StaleAutoRemoveMonths(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"zero (disabled)", "0", false},
		{"valid 1 month", "1", false},
		{"valid 6 months", "6", false},
		{"valid 12 months", "12", false},
		{"valid max 36", "36", false},
		{"too high", "37", true},
		{"negative", "-1", true},
		{"not a number", "abc", true},
		{"float", "1.5", true},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockRepo()
			uc := settinguc.New(repo)
			_, err := uc.UpdateSettings(context.Background(), 1, map[string]string{
				entity.SettingStaleAutoRemoveMonths: tt.value,
			})
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "stale_auto_remove_months")
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// UpdateSettings — package_count_warning_threshold validation
// ---------------------------------------------------------------------------

func TestUpdateSettings_PackageCountWarningThreshold(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"zero (disabled)", "0", false},
		{"valid 1", "1", false},
		{"valid mid", "500", false},
		{"valid high", "10000", false},
		{"valid max", "100000", false},
		{"too high", "100001", true},
		{"negative", "-1", true},
		{"not a number", "abc", true},
		{"float", "1.5", true},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockRepo()
			uc := settinguc.New(repo)
			_, err := uc.UpdateSettings(context.Background(), 1, map[string]string{
				entity.SettingPackageCountWarningThreshold: tt.value,
			})
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "package_count_warning_threshold")
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// UpdateSettings — validation errors wrap entity.ErrValidation
// ---------------------------------------------------------------------------

func TestUpdateSettings_ValidationErrorsWrapErrValidation(t *testing.T) {
	tests := []struct {
		name string
		key  string
		val  string
	}{
		{"invalid key", "totally_invalid_key", "value"},
		{"bad scan depth", entity.SettingDiscoveryScanDepth, "abc"},
		{"bad diff limit", entity.SettingDiffSizeLimit, "abc"},
		{"bad monitoring interval", entity.SettingMonitoringInterval, "abc"},
		{"bad discovery interval", entity.SettingDiscoveryInterval, "abc"},
		{"bad digest enabled", entity.SettingEmailDigestEnabled, "abc"},
		{"bad digest frequency", entity.SettingEmailDigestFrequency, "abc"},
		{"bad digest recipients", entity.SettingEmailDigestRecipients, ""},
		{"bad auto approve", entity.SettingDiscoveryAutoApprove, "abc"},
		{"bad stale months", entity.SettingStaleAutoRemoveMonths, "abc"},
		{"bad warning threshold", entity.SettingPackageCountWarningThreshold, "abc"},
		{"bad analyzer mode", entity.SettingAnalyzerMode, "abc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockRepo()
			uc := settinguc.New(repo)
			_, err := uc.UpdateSettings(context.Background(), 1, map[string]string{
				tt.key: tt.val,
			})
			require.Error(t, err)
			assert.True(t, errors.Is(err, entity.ErrValidation),
				"expected error to wrap ErrValidation, got: %v", err)
		})
	}
}

// ---------------------------------------------------------------------------
// UpdateSettings — repo errors do NOT wrap entity.ErrValidation
// ---------------------------------------------------------------------------

func TestUpdateSettings_RepoErrorsNotValidation(t *testing.T) {
	repo := newMockRepo()
	repo.upsertByOrgKeyErr = fmt.Errorf("database write failed")

	uc := settinguc.New(repo)
	_, err := uc.UpdateSettings(context.Background(), 1, map[string]string{
		entity.SettingMonitoringInterval: "5m",
	})
	require.Error(t, err)
	assert.False(t, errors.Is(err, entity.ErrValidation),
		"repo errors should not be validation errors")
}
