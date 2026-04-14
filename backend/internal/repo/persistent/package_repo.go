package persistent

import (
	"context"
	"errors"
	"fmt"
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

func (r *PackageRepo) FindByID(ctx context.Context, id uint) (*entity.Package, error) {
	var m Package
	if err := r.db.WithContext(ctx).First(&m, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("package %w", entity.ErrNotFound)
		}
		return nil, fmt.Errorf("finding package: %w", err)
	}
	return packageToDomain(&m), nil
}

func (r *PackageRepo) FindByOrgID(ctx context.Context, orgID uint, page, limit int, sortClause string, filters entity.PackageFilters) ([]entity.Package, int64, error) {
	var total int64
	query := r.db.WithContext(ctx).Model(&Package{}).Where("org_id = ?", orgID)

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
		query = query.Where("LOWER(name) LIKE LOWER(?)", "%"+*filters.Search+"%")
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

func (r *PackageRepo) FindActiveByOrgID(ctx context.Context, orgID uint) ([]entity.Package, error) {
	var ms []Package
	err := r.db.WithContext(ctx).
		Where("org_id = ? AND status = ?", orgID, PackageStatusActive).
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

func (r *PackageRepo) FindByOrgAndName(ctx context.Context, orgID uint, name string, ecosystem entity.Ecosystem) (*entity.Package, error) {
	var m Package
	err := r.db.WithContext(ctx).
		Where("org_id = ? AND name = ? AND ecosystem = ?", orgID, name, string(ecosystem)).
		First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("package %w", entity.ErrNotFound)
		}
		return nil, fmt.Errorf("finding package by name: %w", err)
	}
	return packageToDomain(&m), nil
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

func (r *PackageRepo) BlockPackage(ctx context.Context, orgID, pkgID uint, reason string) error {
	now := time.Now()
	result := r.db.WithContext(ctx).
		Model(&Package{}).
		Where("id = ? AND org_id = ? AND status = ?", pkgID, orgID, PackageStatusActive).
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

func (r *PackageRepo) UnblockPackage(ctx context.Context, orgID, pkgID uint) error {
	result := r.db.WithContext(ctx).
		Model(&Package{}).
		Where("id = ? AND org_id = ? AND status = ?", pkgID, orgID, PackageStatusBlocked).
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

func (r *PackageRepo) RemovePackage(ctx context.Context, orgID, pkgID uint) error {
	result := r.db.WithContext(ctx).
		Model(&Package{}).
		Where("id = ? AND org_id = ? AND status IN ?", pkgID, orgID,
			[]PackageStatus{PackageStatusActive, PackageStatusBlocked}).
		Update("status", PackageStatusRemoved)
	if result.Error != nil {
		return fmt.Errorf("removing package: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("package %w", entity.ErrNotFound)
	}
	return nil
}

func (r *PackageRepo) CountByOrg(ctx context.Context, orgID uint, ecosystem *entity.Ecosystem) (int64, error) {
	query := r.db.WithContext(ctx).Model(&Package{}).Where("org_id = ? AND status = ?", orgID, PackageStatusActive)
	if ecosystem != nil {
		query = query.Where("ecosystem = ?", string(*ecosystem))
	}
	var count int64
	if err := query.Count(&count).Error; err != nil {
		return 0, fmt.Errorf("counting packages: %w", err)
	}
	return count, nil
}

func (r *PackageRepo) ExistsByOrgAndName(ctx context.Context, orgID uint, name string, ecosystem entity.Ecosystem) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&Package{}).
		Where("org_id = ? AND name = ? AND ecosystem = ?", orgID, name, string(ecosystem)).
		Count(&count).Error
	if err != nil {
		return false, fmt.Errorf("checking package existence: %w", err)
	}
	return count > 0, nil
}

// --- Converters ---

func packageToDomain(m *Package) *entity.Package {
	return &entity.Package{
		ID:            m.ID,
		OrgID:         m.OrgID,
		Name:          m.Name,
		Ecosystem:     entity.Ecosystem(m.Ecosystem),
		LatestVersion: m.LatestVersion,
		Description:   m.Description,
		Source:        entity.PackageSource(m.Source),
		Status:        entity.PackageStatus(m.Status),
		Rank:          m.Rank,
		BlockedAt:     m.BlockedAt,
		BlockedReason: m.BlockedReason,
		CreatedAt:     m.CreatedAt,
		UpdatedAt:     m.UpdatedAt,
	}
}

func packageToModel(d *entity.Package) *Package {
	return &Package{
		ID:            d.ID,
		OrgID:         d.OrgID,
		Name:          d.Name,
		Ecosystem:     Ecosystem(d.Ecosystem),
		LatestVersion: d.LatestVersion,
		Description:   d.Description,
		Source:        PackageSource(d.Source),
		Status:        PackageStatus(d.Status),
		Rank:          d.Rank,
		BlockedAt:     d.BlockedAt,
		BlockedReason: d.BlockedReason,
		CreatedAt:     d.CreatedAt,
		UpdatedAt:     d.UpdatedAt,
	}
}
