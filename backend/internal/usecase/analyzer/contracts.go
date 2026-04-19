package analyzer

import (
	"context"

	"github.com/veilence/veilence-mx/backend/internal/entity"
)

// PipelineRepository defines the persistence operations needed by the analysis pipeline.
type PipelineRepository interface {
	// FindDiffWithRelease loads a diff by ID along with its release and package.
	FindDiffWithRelease(ctx context.Context, diffID uint) (*entity.Diff, *entity.Release, *entity.Package, error)
	// FindReleaseByID returns a release by its ID.
	FindReleaseByID(ctx context.Context, id uint) (*entity.Release, error)
	// CreateAnalysis persists a new analysis record.
	CreateAnalysis(ctx context.Context, analysis *entity.Analysis) error
	// CreateAlert persists a new alert record.
	CreateAlert(ctx context.Context, alert *entity.Alert) error
	// UpdateReleaseStatus updates the status of a release.
	UpdateReleaseStatus(ctx context.Context, id uint, status entity.ReleaseStatus) error
}
