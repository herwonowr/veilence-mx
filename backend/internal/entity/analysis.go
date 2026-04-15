package entity

import "time"

// Classification represents the LLM's assessment of a diff.
type Classification string

const (
	// ClassificationBenign indicates the diff is safe.
	ClassificationBenign Classification = "benign"
	// ClassificationSuspicious indicates the diff may be malicious.
	ClassificationSuspicious Classification = "suspicious"
	// ClassificationMalicious indicates the diff is likely malicious.
	ClassificationMalicious Classification = "malicious"
	// ClassificationBaseline indicates the first release with no prior version to diff against.
	ClassificationBaseline Classification = "baseline"
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
	ID             uint
	DiffID         uint
	Classification Classification
	Confidence     float64
	Reasoning      string
	ModelUsed      string
	AnalyzerType   AnalyzerType
	RawResponse    string
	CreatedAt      time.Time
}
