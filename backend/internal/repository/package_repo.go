package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/veilence/veilence-mx/backend/internal/domain"
	"github.com/veilence/veilence-mx/backend/internal/models"
)

// PackageRepo implements domain.PackageRepository using GORM.
type PackageRepo struct {
	db *gorm.DB
}

// NewPackageRepo creates a new PackageRepo.
func NewPackageRepo(db *gorm.DB) *PackageRepo {
	return &PackageRepo{db: db}
}

func (r *PackageRepo) FindByID(ctx context.Context, id uint) (*domain.Package, error) {
	var m models.Package
	if err := r.db.WithContext(ctx).First(&m, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("package %w", domain.ErrNotFound)
		}
		return nil, fmt.Errorf("finding package: %w", err)
	}
	return packageToDomain(&m), nil
}

func (r *PackageRepo) FindByOrgID(ctx context.Context, orgID uint, page, limit int, sortClause string) ([]domain.Package, int64, error) {
	var total int64
	// Exclude removed packages by default
	query := r.db.WithContext(ctx).Model(&models.Package{}).Where("org_id = ? AND status != ?", orgID, models.PackageStatusRemoved)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("counting packages: %w", err)
	}

	var ms []models.Package
	err := query.
		Order(sortClause).
		Offset((page - 1) * limit).
		Limit(limit).
		Find(&ms).Error
	if err != nil {
		return nil, 0, fmt.Errorf("listing packages: %w", err)
	}

	result := make([]domain.Package, len(ms))
	for i := range ms {
		result[i] = *packageToDomain(&ms[i])
	}
	return result, total, nil
}

func (r *PackageRepo) FindActiveByOrgID(ctx context.Context, orgID uint) ([]domain.Package, error) {
	var ms []models.Package
	err := r.db.WithContext(ctx).
		Where("org_id = ? AND status = ?", orgID, models.PackageStatusActive).
		Find(&ms).Error
	if err != nil {
		return nil, fmt.Errorf("finding active packages: %w", err)
	}

	result := make([]domain.Package, len(ms))
	for i := range ms {
		result[i] = *packageToDomain(&ms[i])
	}
	return result, nil
}

func (r *PackageRepo) FindByOrgAndName(ctx context.Context, orgID uint, name string, ecosystem domain.Ecosystem) (*domain.Package, error) {
	var m models.Package
	err := r.db.WithContext(ctx).
		Where("org_id = ? AND name = ? AND ecosystem = ?", orgID, name, string(ecosystem)).
		First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("package %w", domain.ErrNotFound)
		}
		return nil, fmt.Errorf("finding package by name: %w", err)
	}
	return packageToDomain(&m), nil
}

func (r *PackageRepo) Create(ctx context.Context, pkg *domain.Package) error {
	m := packageToModel(pkg)
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return fmt.Errorf("creating package: %w", err)
	}
	pkg.ID = m.ID
	pkg.CreatedAt = m.CreatedAt
	pkg.UpdatedAt = m.UpdatedAt
	return nil
}

func (r *PackageRepo) Update(ctx context.Context, pkg *domain.Package) error {
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
		Model(&models.Package{}).
		Where("id = ? AND org_id = ? AND status = ?", pkgID, orgID, models.PackageStatusActive).
		Updates(map[string]any{
			"status":         models.PackageStatusBlocked,
			"blocked_at":     now,
			"blocked_reason": reason,
		})
	if result.Error != nil {
		return fmt.Errorf("blocking package: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("package %w", domain.ErrNotFound)
	}
	return nil
}

func (r *PackageRepo) UnblockPackage(ctx context.Context, orgID, pkgID uint) error {
	result := r.db.WithContext(ctx).
		Model(&models.Package{}).
		Where("id = ? AND org_id = ? AND status = ?", pkgID, orgID, models.PackageStatusBlocked).
		Updates(map[string]any{
			"status":         models.PackageStatusActive,
			"blocked_at":     nil,
			"blocked_reason": "",
		})
	if result.Error != nil {
		return fmt.Errorf("unblocking package: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("package %w", domain.ErrNotFound)
	}
	return nil
}

func (r *PackageRepo) RemovePackage(ctx context.Context, orgID, pkgID uint) error {
	result := r.db.WithContext(ctx).
		Model(&models.Package{}).
		Where("id = ? AND org_id = ? AND status IN ?", pkgID, orgID,
			[]models.PackageStatus{models.PackageStatusActive, models.PackageStatusBlocked}).
		Update("status", models.PackageStatusRemoved)
	if result.Error != nil {
		return fmt.Errorf("removing package: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("package %w", domain.ErrNotFound)
	}
	return nil
}

func (r *PackageRepo) CountByOrg(ctx context.Context, orgID uint, ecosystem *domain.Ecosystem) (int64, error) {
	query := r.db.WithContext(ctx).Model(&models.Package{}).Where("org_id = ? AND status = ?", orgID, models.PackageStatusActive)
	if ecosystem != nil {
		query = query.Where("ecosystem = ?", string(*ecosystem))
	}
	var count int64
	if err := query.Count(&count).Error; err != nil {
		return 0, fmt.Errorf("counting packages: %w", err)
	}
	return count, nil
}

// --- Converters ---

func packageToDomain(m *models.Package) *domain.Package {
	return &domain.Package{
		ID:            m.ID,
		OrgID:         m.OrgID,
		Name:          m.Name,
		Ecosystem:     domain.Ecosystem(m.Ecosystem),
		LatestVersion: m.LatestVersion,
		Description:   m.Description,
		Source:        domain.PackageSource(m.Source),
		Status:        domain.PackageStatus(m.Status),
		Rank:          m.Rank,
		BlockedAt:     m.BlockedAt,
		BlockedReason: m.BlockedReason,
		CreatedAt:     m.CreatedAt,
		UpdatedAt:     m.UpdatedAt,
	}
}

func packageToModel(d *domain.Package) *models.Package {
	return &models.Package{
		ID:            d.ID,
		OrgID:         d.OrgID,
		Name:          d.Name,
		Ecosystem:     models.Ecosystem(d.Ecosystem),
		LatestVersion: d.LatestVersion,
		Description:   d.Description,
		Source:        models.PackageSource(d.Source),
		Status:        models.PackageStatus(d.Status),
		Rank:          d.Rank,
		BlockedAt:     d.BlockedAt,
		BlockedReason: d.BlockedReason,
		CreatedAt:     d.CreatedAt,
		UpdatedAt:     d.UpdatedAt,
	}
}
