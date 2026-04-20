package entity

import "time"

// Setting represents a configurable system setting stored as key-value.
// WorkspaceID=0 represents a global default setting; workspace-specific settings override globals.
type Setting struct {
	ID             string
	WorkspaceID    string
	Key            string
	Value          string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// Default setting keys.
const (
	SettingDiscoveryScanDepth = "discovery_scan_depth"
	SettingDiscoveryInterval  = "discovery_interval"
	SettingMonitoringInterval = "monitoring_interval"
	// Email digest settings (per-org).
	SettingEmailDigestEnabled    = "email_digest_enabled"
	SettingEmailDigestFrequency  = "email_digest_frequency"
	SettingEmailDigestRecipients = "email_digest_recipients"

	// Discovery auto-approval: when "true", newly discovered packages are
	// automatically approved for monitoring instead of being left as suggestions.
	SettingDiscoveryAutoApprove = "discovery_auto_approve"

	// Stale auto-removal: number of months after which active packages with
	// no new releases are automatically removed. "0" disables auto-removal.
	SettingStaleAutoRemoveMonths = "stale_auto_remove_months"

	// Package count warning threshold: the number of monitored packages at which
	// a warning is surfaced in the dashboard/settings UI.
	SettingPackageCountWarningThreshold = "package_count_warning_threshold"

)

// ValidSettingKeys is the canonical set of all accepted setting keys.
// Used by the handler layer for input validation.
var ValidSettingKeys = map[string]bool{
	SettingDiscoveryScanDepth:            true,
	SettingDiscoveryInterval:             true,
	SettingMonitoringInterval:            true,
	SettingEmailDigestEnabled:            true,
	SettingEmailDigestFrequency:          true,
	SettingEmailDigestRecipients:         true,
	SettingDiscoveryAutoApprove:          true,
	SettingStaleAutoRemoveMonths:         true,
	SettingPackageCountWarningThreshold:  true,
}
