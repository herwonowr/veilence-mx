package analyzer

import (
	"context"

	"github.com/veilence/veilence-mx/backend/internal/entity"
)

// PipelineRepository defines the persistence operations needed by the analysis pipeline.
type PipelineRepository interface {
	// FindDiffWithRelease loads a diff by ID along with its release and package.
	FindDiffWithRelease(ctx context.Context, diffID string) (*entity.Diff, *entity.Release, *entity.Package, error)
	// FindReleaseByID returns a release by its ID.
	FindReleaseByID(ctx context.Context, id string) (*entity.Release, error)
	// CreateAnalysis persists a new analysis record.
	CreateAnalysis(ctx context.Context, analysis *entity.Analysis) error
	// CreateAlert persists a new alert record.
	CreateAlert(ctx context.Context, alert *entity.Alert) error
	// UpdateReleaseStatus updates the status of a release.
	UpdateReleaseStatus(ctx context.Context, id string, status entity.ReleaseStatus) error
}

// LLMProvider defines the interface for LLM-based diff analysis.
// Implementations live in pkg/ (copilot-api, openai, anthropic, ollama, etc.)
type LLMProvider interface {
	// Analyze classifies a diff using an LLM.
	Analyze(ctx context.Context, diff string, packageName string, ecosystem string, oldVersion string, newVersion string, truncated bool) (*LLMResult, error)
	// Type returns a provider identifier (e.g., "copilot", "openai", "ollama").
	Type() string
}

// LLMResult holds the output of an LLM analysis.
type LLMResult struct {
	Classification string
	Confidence     float64
	Reasoning      string
	RawResponse    string
}
