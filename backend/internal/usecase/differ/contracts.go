package differ

import (
	"context"
	"time"

	"github.com/veilence/veilence-mx/backend/internal/entity"
)

// DifferRepository defines the persistence operations needed by the differ.
type DifferRepository interface {
	// FindReleaseByIDWithPackage loads a release by ID along with its package.
	FindReleaseByIDWithPackage(ctx context.Context, id uint) (*entity.Release, *entity.Package, error)
	// FindPreviousCompletedRelease finds the latest completed release before the given one.
	FindPreviousCompletedRelease(ctx context.Context, packageID uint, beforePublishedAt time.Time) (*entity.Release, error)
	// UpdateReleaseStatus updates the status of a release.
	UpdateReleaseStatus(ctx context.Context, id uint, status entity.ReleaseStatus) error
	// UpdateReleaseError updates the status and error message on a release.
	UpdateReleaseError(ctx context.Context, id uint, status entity.ReleaseStatus, errorMessage string) error
	// CreateDiff persists a new diff record.
	CreateDiff(ctx context.Context, diff *entity.Diff) error
}
