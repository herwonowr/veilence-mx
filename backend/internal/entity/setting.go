package entity

import "time"

// Setting represents a configurable system setting stored as key-value.
// OrgID=0 represents a global default setting; org-specific settings override globals.
type Setting struct {
	ID             uint
	OrgID          uint
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
	SettingAnalyzerMode       = "analyzer_mode"
	SettingDiffSizeLimit      = "diff_size_limit"

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

	// Require email verification: when "true", users with unverified emails
	// are blocked from logging in. Default is "false".
	SettingRequireEmailVerification = "require_email_verification"
)

// ValidAnalyzerModes is the set of allowed values for the analyzer_mode setting.
// "auto" = automatically analyze new releases, "manual" = user-triggered, "disabled" = skip analysis.
var ValidAnalyzerModes = map[string]bool{
	"auto":     true,
	"manual":   true,
	"disabled": true,
}

// ValidSettingKeys is the canonical set of all accepted setting keys.
// Used by the handler layer for input validation.
var ValidSettingKeys = map[string]bool{
	SettingDiscoveryScanDepth:            true,
	SettingDiscoveryInterval:             true,
	SettingMonitoringInterval:            true,
	SettingAnalyzerMode:                  true,
	SettingDiffSizeLimit:                 true,
	SettingEmailDigestEnabled:            true,
	SettingEmailDigestFrequency:          true,
	SettingEmailDigestRecipients:         true,
	SettingDiscoveryAutoApprove:          true,
	SettingStaleAutoRemoveMonths:         true,
	SettingPackageCountWarningThreshold:  true,
	SettingRequireEmailVerification:      true,
}
