package poller

import (
	"context"
	"time"

	"github.com/veilence/veilence-mx/backend/internal/entity"
)

// PollerRepository defines the persistence operations needed by the poller.
type PollerRepository interface {
	// DistinctActiveWorkspaceIDs returns workspace IDs that have at least one active package.
	DistinctActiveWorkspaceIDs(ctx context.Context) ([]uint, error)

	// DistinctWorkspaceIDs returns all workspace IDs from settings and packages.
	DistinctWorkspaceIDs(ctx context.Context) ([]uint, error)

	// FindActivePackagesByWorkspace loads all active packages for a workspace.
	FindActivePackagesByWorkspace(ctx context.Context, workspaceID uint) ([]entity.Package, error)

	// UpdatePackageMetadata updates a package's latest version and description.
	UpdatePackageMetadata(ctx context.Context, packageID uint, latestVersion, description string) error

	// FindLatestRelease returns the most recent release for a package by publish time.
	FindLatestRelease(ctx context.Context, packageID uint) (*entity.Release, error)

	// FindReleaseByPackageAndVersion checks if a release already exists.
	FindReleaseByPackageAndVersion(ctx context.Context, packageID uint, version string) (*entity.Release, error)

	// CreateRelease persists a new release.
	CreateRelease(ctx context.Context, release *entity.Release) error

	// FindPackageByWorkspaceAndName looks up an existing package.
	FindPackageByWorkspaceAndName(ctx context.Context, workspaceID uint, name string, ecosystem entity.Ecosystem) (*entity.Package, error)

	// CreatePackage persists a new package.
	CreatePackage(ctx context.Context, pkg *entity.Package) error

	// UpdatePackageRank updates a package's rank.
	UpdatePackageRank(ctx context.Context, packageID uint, rank uint) error

	// UpdatePackageDiscoveryMetrics updates a suggested/removed package's metrics.
	UpdatePackageDiscoveryMetrics(ctx context.Context, packageID uint, updates map[string]interface{}) error

	// UpdateDownloadCounts batch-updates download metrics for active packages in a workspace.
	UpdateDownloadCounts(ctx context.Context, workspaceID uint, updates []entity.PackageDownloadUpdate) error

	// RemoveStalePackages marks active packages with no updates in the given period as removed.
	RemoveStalePackages(ctx context.Context, workspaceID uint, staleBefore time.Time) (int64, error)

	// GetSetting retrieves a setting value by workspace and key.
	GetSetting(ctx context.Context, workspaceID uint, key string) (string, error)
}
