package models

import (
	"time"
)

// Setting represents a configurable system setting stored as key-value.
// OrgID=0 represents a global default setting; org-specific settings override globals.
type Setting struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	OrgID     uint      `gorm:"index;not null;default:0;uniqueIndex:idx_settings_org_key" json:"orgId"`
	Key       string    `gorm:"not null;type:varchar(100);uniqueIndex:idx_settings_org_key" json:"key"`
	Value     string    `gorm:"type:text" json:"value"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// TableName returns the table name for Setting.
func (Setting) TableName() string {
	return "settings"
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
