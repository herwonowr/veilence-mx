package persistent

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/veilence/veilence-mx/backend/internal/entity"
)

// PollerRepo implements poller.PollerRepository using GORM.
type PollerRepo struct {
	db *gorm.DB
}

// NewPollerRepo creates a new PollerRepo.
func NewPollerRepo(db *gorm.DB) *PollerRepo {
	return &PollerRepo{db: db}
}

// DistinctActiveWorkspaceIDs returns workspace IDs that have at least one active package.
func (r *PollerRepo) DistinctActiveWorkspaceIDs(ctx context.Context) ([]string, error) {
	var wsIDs []string
	if err := r.db.WithContext(ctx).Model(&Package{}).
		Where("status = ?", PackageStatusActive).
		Distinct("workspace_id").
		Pluck("workspace_id", &wsIDs).Error; err != nil {
		return nil, fmt.Errorf("PollerRepo.DistinctActiveWorkspaceIDs: %w", err)
	}
	return wsIDs, nil
}

// DistinctWorkspaceIDs returns all workspace IDs from settings and packages (merged, unique).
func (r *PollerRepo) DistinctWorkspaceIDs(ctx context.Context) ([]string, error) {
	var settingWsIDs []string
	if err := r.db.WithContext(ctx).Model(&Setting{}).
		Where("workspace_id IS NOT NULL").
		Distinct("workspace_id").
		Pluck("workspace_id", &settingWsIDs).Error; err != nil {
		return nil, fmt.Errorf("PollerRepo.DistinctWorkspaceIDs (settings): %w", err)
	}

	var pkgWsIDs []string
	if err := r.db.WithContext(ctx).Model(&Package{}).
		Distinct("workspace_id").
		Pluck("workspace_id", &pkgWsIDs).Error; err != nil {
		return nil, fmt.Errorf("PollerRepo.DistinctWorkspaceIDs (packages): %w", err)
	}

	seen := make(map[string]bool, len(settingWsIDs)+len(pkgWsIDs))
	for _, id := range settingWsIDs {
		seen[id] = true
	}
	for _, id := range pkgWsIDs {
		seen[id] = true
	}

	var result []string
	for id := range seen {
		result = append(result, id)
	}
	return result, nil
}

// FindActivePackagesByWorkspace loads all active packages for a workspace.
func (r *PollerRepo) FindActivePackagesByWorkspace(ctx context.Context, workspaceID string) ([]entity.Package, error) {
	var models []Package
	if err := r.db.WithContext(ctx).
		Where("workspace_id = ? AND status = ?", workspaceID, PackageStatusActive).
		Find(&models).Error; err != nil {
		return nil, fmt.Errorf("PollerRepo.FindActivePackagesByWorkspace: %w", err)
	}

	packages := make([]entity.Package, len(models))
	for i, m := range models {
		packages[i] = *packageToDomain(&m)
	}
	return packages, nil
}

// UpdatePackageMetadata updates a package's latest version and description.
func (r *PollerRepo) UpdatePackageMetadata(ctx context.Context, packageID string, latestVersion, description string) error {
	return r.db.WithContext(ctx).Model(&Package{}).Where("id = ?", packageID).Updates(map[string]interface{}{
		"latest_version": latestVersion,
		"description":    description,
	}).Error
}

// FindLatestRelease returns the most recent release for a package by publish time.
func (r *PollerRepo) FindLatestRelease(ctx context.Context, packageID string) (*entity.Release, error) {
	var model Release
	result := r.db.WithContext(ctx).Where("package_id = ?", packageID).Order("published_at DESC").Limit(1).Find(&model)
	if result.Error != nil {
		return nil, fmt.Errorf("PollerRepo.FindLatestRelease: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, nil
	}
	return &entity.Release{
		ID:          model.ID,
		PackageID:   model.PackageID,
		Version:     model.Version,
		PublishedAt: model.PublishedAt,
		TarballURL:  model.TarballURL,
		SHA256:      model.SHA256,
		Status:      entity.ReleaseStatus(model.Status),
		CreatedAt:   model.CreatedAt,
	}, nil
}

// FindReleaseByPackageAndVersion checks if a release already exists.
func (r *PollerRepo) FindReleaseByPackageAndVersion(ctx context.Context, packageID string, version string) (*entity.Release, error) {
	var model Release
	result := r.db.WithContext(ctx).Where("package_id = ? AND version = ?", packageID, version).Limit(1).Find(&model)
	if result.Error != nil {
		return nil, fmt.Errorf("PollerRepo.FindReleaseByPackageAndVersion: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, nil
	}
	return &entity.Release{
		ID:        model.ID,
		PackageID: model.PackageID,
		Version:   model.Version,
		Status:    entity.ReleaseStatus(model.Status),
	}, nil
}

// CreateRelease persists a new release.
func (r *PollerRepo) CreateRelease(ctx context.Context, release *entity.Release) error {
	model := Release{
		PackageID:   release.PackageID,
		Version:     release.Version,
		PublishedAt: release.PublishedAt,
		TarballURL:  release.TarballURL,
		SHA256:      release.SHA256,
		Status:      ReleaseStatus(release.Status),
	}
	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		return fmt.Errorf("PollerRepo.CreateRelease: %w", err)
	}
	release.ID = model.ID
	release.CreatedAt = model.CreatedAt
	return nil
}

// FindPackageByWorkspaceAndName looks up an existing package.
func (r *PollerRepo) FindPackageByWorkspaceAndName(ctx context.Context, workspaceID string, name string, ecosystem entity.Ecosystem) (*entity.Package, error) {
	var model Package
	result := r.db.WithContext(ctx).Where("workspace_id = ? AND name = ? AND ecosystem = ?", workspaceID, name, string(ecosystem)).Limit(1).Find(&model)
	if result.Error != nil {
		return nil, fmt.Errorf("PollerRepo.FindPackageByWorkspaceAndName: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, nil
	}
	pkg := *packageToDomain(&model)
	return &pkg, nil
}

// CreatePackage persists a new package.
func (r *PollerRepo) CreatePackage(ctx context.Context, pkg *entity.Package) error {
	model := Package{
		WorkspaceID:            pkg.WorkspaceID,
		Name:                   pkg.Name,
		Ecosystem:              Ecosystem(pkg.Ecosystem),
		Source:                 PackageSource(pkg.Source),
		Status:                 PackageStatus(pkg.Status),
		Rank:                   pkg.Rank,
		DownloadCount:          pkg.DownloadCount,
		DownloadCountUpdatedAt: pkg.DownloadCountUpdatedAt,
	}
	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		return fmt.Errorf("PollerRepo.CreatePackage: %w", err)
	}
	pkg.ID = model.ID
	pkg.CreatedAt = model.CreatedAt
	return nil
}

// UpdatePackageRank updates a package's rank.
func (r *PollerRepo) UpdatePackageRank(ctx context.Context, packageID string, rank int) error {
	return r.db.WithContext(ctx).Model(&Package{}).Where("id = ?", packageID).Update("rank", &rank).Error
}

// UpdatePackageDiscoveryMetrics updates a suggested/removed package's metrics.
func (r *PollerRepo) UpdatePackageDiscoveryMetrics(ctx context.Context, packageID string, updates map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&Package{}).Where("id = ?", packageID).Updates(updates).Error
}

// UpdateDownloadCounts batch-updates download metrics for active packages in a workspace.
func (r *PollerRepo) UpdateDownloadCounts(ctx context.Context, workspaceID string, updates []entity.PackageDownloadUpdate) error {
	now := time.Now()
	for _, u := range updates {
		if err := r.db.WithContext(ctx).Model(&Package{}).
			Where("id = ? AND workspace_id = ?", u.PackageID, workspaceID).
			Updates(map[string]interface{}{
				"download_count":            u.DownloadCount,
				"download_count_updated_at": now,
			}).Error; err != nil {
			return fmt.Errorf("PollerRepo.UpdateDownloadCounts: package %s: %w", u.PackageID, err)
		}
	}
	return nil
}

// RemoveStalePackages marks active packages with no updates in the given period as removed.
func (r *PollerRepo) RemoveStalePackages(ctx context.Context, workspaceID string, staleBefore time.Time) (int64, error) {
	result := r.db.WithContext(ctx).Model(&Package{}).
		Where("workspace_id = ? AND status = ? AND updated_at < ?", workspaceID, PackageStatusActive, staleBefore).
		Update("status", PackageStatusRemoved)
	if result.Error != nil {
		return 0, fmt.Errorf("PollerRepo.RemoveStalePackages: %w", result.Error)
	}
	return result.RowsAffected, nil
}

// GetSetting retrieves a setting value by workspace and key.
func (r *PollerRepo) GetSetting(ctx context.Context, workspaceID string, key string) (string, error) {
	var setting Setting
	result := r.db.WithContext(ctx).Where("key = ? AND workspace_id = ?", key, workspaceID).Limit(1).Find(&setting)
	if result.Error != nil {
		return "", fmt.Errorf("PollerRepo.GetSetting: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return "", nil
	}
	return setting.Value, nil
}

