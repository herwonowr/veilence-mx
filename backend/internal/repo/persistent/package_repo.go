package persistent

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/veilence/veilence-mx/backend/internal/entity"
)

// PackageRepo implements entity.PackageRepository using GORM.
type PackageRepo struct {
	db *gorm.DB
}

// NewPackageRepo creates a new PackageRepo.
func NewPackageRepo(db *gorm.DB) *PackageRepo {
	return &PackageRepo{db: db}
}

func (r *PackageRepo) FindByIDAndWorkspaceID(ctx context.Context, id, workspaceID string) (*entity.Package, error) {
	var m Package
	if err := r.db.WithContext(ctx).Where("id = ? AND workspace_id = ?", id, workspaceID).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("package %w", entity.ErrNotFound)
		}
		return nil, fmt.Errorf("finding package: %w", err)
	}
	return packageToDomain(&m), nil
}

func (r *PackageRepo) FindByWorkspaceID(ctx context.Context, workspaceID string, page, limit int, sortClause string, filters entity.PackageFilters) ([]entity.Package, int64, error) {
	var total int64
	query := r.db.WithContext(ctx).Model(&Package{}).Where("workspace_id = ?", workspaceID)

	// Apply filters
	if filters.Status != nil {
		query = query.Where("status = ?", string(*filters.Status))
	} else {
		query = query.Where("status = ?", PackageStatusActive)
	}
	if filters.Ecosystem != nil {
		query = query.Where("ecosystem = ?", string(*filters.Ecosystem))
	}
	if filters.Source != nil {
		query = query.Where("source = ?", string(*filters.Source))
	}
	if filters.Search != nil && *filters.Search != "" {
		escaped := strings.ReplaceAll(strings.ReplaceAll(*filters.Search, "%", "\\%"), "_", "\\_")
		query = query.Where("LOWER(name) LIKE LOWER(?)", "%"+escaped+"%")
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("counting packages: %w", err)
	}

	var ms []Package
	err := query.
		Order(sortClause).
		Offset((page - 1) * limit).
		Limit(limit).
		Find(&ms).Error
	if err != nil {
		return nil, 0, fmt.Errorf("listing packages: %w", err)
	}

	result := make([]entity.Package, len(ms))
	for i := range ms {
		result[i] = *packageToDomain(&ms[i])
	}
	return result, total, nil
}

func (r *PackageRepo) FindActiveByWorkspaceID(ctx context.Context, workspaceID string) ([]entity.Package, error) {
	var ms []Package
	err := r.db.WithContext(ctx).
		Where("workspace_id = ? AND status = ?", workspaceID, PackageStatusActive).
		Find(&ms).Error
	if err != nil {
		return nil, fmt.Errorf("finding active packages: %w", err)
	}

	result := make([]entity.Package, len(ms))
	for i := range ms {
		result[i] = *packageToDomain(&ms[i])
	}
	return result, nil
}

func (r *PackageRepo) Create(ctx context.Context, pkg *entity.Package) error {
	m := packageToModel(pkg)
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return fmt.Errorf("creating package: %w", err)
	}
	pkg.ID = m.ID
	pkg.CreatedAt = m.CreatedAt
	pkg.UpdatedAt = m.UpdatedAt
	return nil
}

func (r *PackageRepo) Update(ctx context.Context, pkg *entity.Package) error {
	m := packageToModel(pkg)
	if err := r.db.WithContext(ctx).Save(m).Error; err != nil {
		return fmt.Errorf("updating package: %w", err)
	}
	pkg.UpdatedAt = m.UpdatedAt
	return nil
}

func (r *PackageRepo) BlockPackage(ctx context.Context, workspaceID, pkgID string, reason string) error {
	now := time.Now()
	result := r.db.WithContext(ctx).
		Model(&Package{}).
		Where("id = ? AND workspace_id = ? AND status = ?", pkgID, workspaceID, PackageStatusActive).
		Updates(map[string]any{
			"status":         PackageStatusBlocked,
			"blocked_at":     now,
			"blocked_reason": reason,
		})
	if result.Error != nil {
		return fmt.Errorf("blocking package: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("package %w", entity.ErrNotFound)
	}
	return nil
}

func (r *PackageRepo) UnblockPackage(ctx context.Context, workspaceID, pkgID string) error {
	result := r.db.WithContext(ctx).
		Model(&Package{}).
		Where("id = ? AND workspace_id = ? AND status = ?", pkgID, workspaceID, PackageStatusBlocked).
		Updates(map[string]any{
			"status":         PackageStatusActive,
			"blocked_at":     nil,
			"blocked_reason": "",
		})
	if result.Error != nil {
		return fmt.Errorf("unblocking package: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("package %w", entity.ErrNotFound)
	}
	return nil
}

func (r *PackageRepo) RemovePackage(ctx context.Context, workspaceID, pkgID string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Verify package exists and is removable
		var pkg Package
		if err := tx.Where("id = ? AND workspace_id = ? AND status IN ?", pkgID, workspaceID,
			[]PackageStatus{PackageStatusActive, PackageStatusBlocked}).
			First(&pkg).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("package %w", entity.ErrNotFound)
			}
			return fmt.Errorf("removing package: %w", err)
		}

		// Collect release IDs for this package
		var releaseIDs []string
		tx.Model(&Release{}).Where("package_id = ? AND workspace_id = ?", pkgID, workspaceID).Pluck("id", &releaseIDs)

		// Collect diff IDs for these releases
		var diffIDs []string
		if len(releaseIDs) > 0 {
			tx.Model(&Diff{}).Where("release_id IN ?", releaseIDs).Pluck("id", &diffIDs)
		}

		// Alert notes (via alerts for this package)
		var alertIDs []string
		tx.Model(&Alert{}).Where("package_id = ? AND workspace_id = ?", pkgID, workspaceID).Pluck("id", &alertIDs)
		if len(alertIDs) > 0 {
			if err := tx.Where("alert_id IN ?", alertIDs).Delete(&AlertNote{}).Error; err != nil {
				return fmt.Errorf("removing package: deleting alert notes: %w", err)
			}
		}

		// Alerts
		if err := tx.Where("package_id = ? AND workspace_id = ?", pkgID, workspaceID).Delete(&Alert{}).Error; err != nil {
			return fmt.Errorf("removing package: deleting alerts: %w", err)
		}

		// Analyses (depend on diffs)
		if len(diffIDs) > 0 {
			if err := tx.Where("diff_id IN ?", diffIDs).Delete(&Analysis{}).Error; err != nil {
				return fmt.Errorf("removing package: deleting analyses: %w", err)
			}
		}

		// Diffs (depend on releases)
		if len(releaseIDs) > 0 {
			if err := tx.Where("release_id IN ?", releaseIDs).Delete(&Diff{}).Error; err != nil {
				return fmt.Errorf("removing package: deleting diffs: %w", err)
			}
		}

		// Releases
		if err := tx.Where("package_id = ? AND workspace_id = ?", pkgID, workspaceID).Delete(&Release{}).Error; err != nil {
			return fmt.Errorf("removing package: deleting releases: %w", err)
		}

		// Set package status to removed
		result := tx.Model(&Package{}).Where("id = ?", pkgID).Update("status", PackageStatusRemoved)
		if result.Error != nil {
			return fmt.Errorf("removing package: %w", result.Error)
		}
		return nil
	})
}

func (r *PackageRepo) CountByWorkspace(ctx context.Context, workspaceID string, ecosystem *entity.Ecosystem) (int64, error) {
	query := r.db.WithContext(ctx).Model(&Package{}).Where("workspace_id = ? AND status = ?", workspaceID, PackageStatusActive)
	if ecosystem != nil {
		query = query.Where("ecosystem = ?", string(*ecosystem))
	}
	var count int64
	if err := query.Count(&count).Error; err != nil {
		return 0, fmt.Errorf("counting packages: %w", err)
	}
	return count, nil
}

func (r *PackageRepo) ExistsByWorkspaceAndName(ctx context.Context, workspaceID string, name string, ecosystem entity.Ecosystem) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&Package{}).
		Where("workspace_id = ? AND name = ? AND ecosystem = ?", workspaceID, name, string(ecosystem)).
		Count(&count).Error
	if err != nil {
		return false, fmt.Errorf("checking package existence: %w", err)
	}
	return count > 0, nil
}

func (r *PackageRepo) FindSuggestionsByWorkspaceID(ctx context.Context, workspaceID string, page, limit int, sortClause string, filters entity.PackageFilters) ([]entity.Package, int64, error) {
	var total int64
	query := r.db.WithContext(ctx).Model(&Package{}).
		Where("workspace_id = ? AND status = ?", workspaceID, PackageStatusSuggested)

	// Apply filters (follow FindByWorkspaceID pattern)
	if filters.Ecosystem != nil {
		query = query.Where("ecosystem = ?", string(*filters.Ecosystem))
	}
	if filters.Search != nil && *filters.Search != "" {
		escaped := strings.ReplaceAll(strings.ReplaceAll(*filters.Search, "%", "\\%"), "_", "\\_")
		query = query.Where("LOWER(name) LIKE LOWER(?)", "%"+escaped+"%")
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("counting suggestions: %w", err)
	}

	if sortClause == "" {
		sortClause = "download_count DESC, name ASC"
	}

	var ms []Package
	err := query.
		Order(sortClause).
		Offset((page - 1) * limit).
		Limit(limit).
		Find(&ms).Error
	if err != nil {
		return nil, 0, fmt.Errorf("listing suggestions: %w", err)
	}

	result := make([]entity.Package, len(ms))
	for i := range ms {
		result[i] = *packageToDomain(&ms[i])
	}
	return result, total, nil
}

func (r *PackageRepo) ApprovePackage(ctx context.Context, workspaceID, pkgID string) error {
	result := r.db.WithContext(ctx).
		Model(&Package{}).
		Where("id = ? AND workspace_id = ? AND status = ?", pkgID, workspaceID, PackageStatusSuggested).
		Update("status", PackageStatusActive)
	if result.Error != nil {
		return fmt.Errorf("approving package: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("package %w", entity.ErrNotFound)
	}
	return nil
}

func (r *PackageRepo) RejectPackage(ctx context.Context, workspaceID, pkgID string) error {
	result := r.db.WithContext(ctx).
		Model(&Package{}).
		Where("id = ? AND workspace_id = ? AND status = ?", pkgID, workspaceID, PackageStatusSuggested).
		Update("status", PackageStatusRemoved)
	if result.Error != nil {
		return fmt.Errorf("rejecting package: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("package %w", entity.ErrNotFound)
	}
	return nil
}

func (r *PackageRepo) BulkApprovePackages(ctx context.Context, workspaceID string, pkgIDs []string) (int, error) {
	if len(pkgIDs) == 0 {
		return 0, nil
	}
	result := r.db.WithContext(ctx).
		Model(&Package{}).
		Where("workspace_id = ? AND id IN ? AND status = ?", workspaceID, pkgIDs, PackageStatusSuggested).
		Update("status", PackageStatusActive)
	if result.Error != nil {
		return 0, fmt.Errorf("bulk approving packages: %w", result.Error)
	}
	return int(result.RowsAffected), nil
}

// BulkApproveAllSuggestions approves all suggested packages for a workspace in a single query.
func (r *PackageRepo) BulkApproveAllSuggestions(ctx context.Context, workspaceID string) (int, error) {
	result := r.db.WithContext(ctx).
		Model(&Package{}).
		Where("workspace_id = ? AND status = ?", workspaceID, PackageStatusSuggested).
		Update("status", PackageStatusActive)
	if result.Error != nil {
		return 0, fmt.Errorf("bulk approving all suggestions: %w", result.Error)
	}
	return int(result.RowsAffected), nil
}

func (r *PackageRepo) UpdateDownloadCounts(ctx context.Context, workspaceID string, updates []entity.PackageDownloadUpdate) error {
	if len(updates) == 0 {
		return nil
	}
	now := time.Now()
	for _, u := range updates {
		result := r.db.WithContext(ctx).
			Model(&Package{}).
			Where("id = ? AND workspace_id = ?", u.PackageID, workspaceID).
			Updates(map[string]any{
				"download_count":            u.DownloadCount,
				"download_count_updated_at": now,
			})
		if result.Error != nil {
			return fmt.Errorf("updating download count for package %s: %w", u.PackageID, result.Error)
		}
	}
	return nil
}

func (r *PackageRepo) FindStaleByWorkspaceID(ctx context.Context, workspaceID string, staleBefore time.Time) ([]entity.Package, error) {
	var ms []Package
	err := r.db.WithContext(ctx).
		Where("workspace_id = ? AND status = ? AND (download_count_updated_at < ? OR download_count_updated_at IS NULL)", workspaceID, PackageStatusActive, staleBefore).
		Find(&ms).Error
	if err != nil {
		return nil, fmt.Errorf("finding stale packages: %w", err)
	}

	result := make([]entity.Package, len(ms))
	for i := range ms {
		result[i] = *packageToDomain(&ms[i])
	}
	return result, nil
}

func (r *PackageRepo) RemoveStaleByWorkspaceID(ctx context.Context, workspaceID string, staleBefore time.Time) (int, error) {
	result := r.db.WithContext(ctx).
		Model(&Package{}).
		Where("workspace_id = ? AND status = ? AND updated_at < ?", workspaceID, PackageStatusActive, staleBefore).
		Update("status", PackageStatusRemoved)
	if result.Error != nil {
		return 0, fmt.Errorf("removing stale packages: %w", result.Error)
	}
	return int(result.RowsAffected), nil
}

// --- Converters ---

func packageToDomain(m *Package) *entity.Package {
	return &entity.Package{
		ID:                     m.ID,
		WorkspaceID:            m.WorkspaceID,
		Name:                   m.Name,
		Ecosystem:              entity.Ecosystem(m.Ecosystem),
		LatestVersion:          m.LatestVersion,
		Description:            m.Description,
		Source:                 entity.PackageSource(m.Source),
		Status:                 entity.PackageStatus(m.Status),
		DownloadCount:          m.DownloadCount,
		DownloadCountUpdatedAt: m.DownloadCountUpdatedAt,
		BlockedAt:              m.BlockedAt,
		BlockedReason:          m.BlockedReason,
		CreatedAt:              m.CreatedAt,
		UpdatedAt:              m.UpdatedAt,
	}
}

func packageToModel(d *entity.Package) *Package {
	return &Package{
		ID:                     d.ID,
		WorkspaceID:            d.WorkspaceID,
		Name:                   d.Name,
		Ecosystem:              Ecosystem(d.Ecosystem),
		LatestVersion:          d.LatestVersion,
		Description:            d.Description,
		Source:                 PackageSource(d.Source),
		Status:                 PackageStatus(d.Status),
		DownloadCount:          d.DownloadCount,
		DownloadCountUpdatedAt: d.DownloadCountUpdatedAt,
		BlockedAt:              d.BlockedAt,
		BlockedReason:          d.BlockedReason,
		CreatedAt:              d.CreatedAt,
		UpdatedAt:              d.UpdatedAt,
	}
}
