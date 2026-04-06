package models

import (
	"time"
)

// Classification represents the LLM's assessment of a diff.
type Classification string

const (
	// ClassificationBenign indicates the diff is safe.
	ClassificationBenign Classification = "benign"
	// ClassificationSuspicious indicates the diff may be malicious.
	ClassificationSuspicious Classification = "suspicious"
	// ClassificationMalicious indicates the diff is likely malicious.
	ClassificationMalicious Classification = "malicious"
)

// AnalyzerType represents which analysis backend was used.
type AnalyzerType string

const (
	// AnalyzerTypeAPI uses the Anthropic Claude API.
	AnalyzerTypeAPI AnalyzerType = "api"
	// AnalyzerTypeCLI uses the Claude Code CLI subprocess.
	AnalyzerTypeCLI AnalyzerType = "cli"
	// AnalyzerTypeCopilot uses the copilot-api proxy.
	AnalyzerTypeCopilot AnalyzerType = "copilot"
)

// Analysis represents the LLM analysis of a diff.
type Analysis struct {
	ID             uint           `gorm:"primarykey" json:"id"`
	DiffID         uint           `gorm:"not null;index" json:"diffId"`
	Diff           Diff           `gorm:"foreignKey:DiffID" json:"-"`
	Classification Classification `gorm:"not null;type:varchar(20)" json:"classification"`
	Confidence     float64        `gorm:"not null" json:"confidence"`
	Reasoning      string         `gorm:"type:text" json:"reasoning"`
	ModelUsed      string         `gorm:"type:varchar(100)" json:"modelUsed"`
	AnalyzerType   AnalyzerType   `gorm:"not null;type:varchar(10)" json:"analyzerType"`
	RawResponse    string         `gorm:"type:text" json:"-"`
	CreatedAt      time.Time      `json:"createdAt"`
}

// TableName returns the table name for Analysis.
func (Analysis) TableName() string {
	return "analyses"
}
