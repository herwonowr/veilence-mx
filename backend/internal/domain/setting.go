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
	SettingPyPIPollInterval    = "pypi_poll_interval"
	SettingNPMPollInterval     = "npm_poll_interval"
	SettingPyPITopN            = "pypi_top_n"
	SettingNPMTopN             = "npm_top_n"
	SettingAnalyzerMode        = "analyzer_mode"
	SettingTopNRefreshInterval = "top_n_refresh_interval"
	SettingDiffSizeLimit       = "diff_size_limit"
	SettingVersionDepthMode    = "version_depth_mode"
	SettingVersionDepthCount   = "version_depth_count"
)
