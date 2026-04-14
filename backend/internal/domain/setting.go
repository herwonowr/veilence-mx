package domain

import "time"

// Setting represents a configurable system setting stored as key-value.
// OrgID=0 represents a global default setting; org-specific settings override globals.
type Setting struct {
	ID        uint      `json:"id"`
	OrgID     uint      `json:"orgId"`
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
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
)

// ValidSettingKeys is the canonical set of all accepted setting keys.
// Used by the handler layer for input validation.
var ValidSettingKeys = map[string]bool{
	SettingDiscoveryScanDepth:     true,
	SettingDiscoveryInterval:      true,
	SettingMonitoringInterval:     true,
	SettingAnalyzerMode:           true,
	SettingDiffSizeLimit:          true,
	SettingEmailDigestEnabled:     true,
	SettingEmailDigestFrequency:   true,
	SettingEmailDigestRecipients:  true,
}
