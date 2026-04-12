package repository

import (
	"context"
	"errors"
	"fmt"

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
	query := r.db.WithContext(ctx).Model(&models.Package{}).Where("org_id = ?", orgID)

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

func (r *PackageRepo) SoftDelete(ctx context.Context, orgID, id uint) error {
	result := r.db.WithContext(ctx).Where("id = ? AND org_id = ?", id, orgID).Delete(&models.Package{})
	if result.Error != nil {
		return fmt.Errorf("deleting package: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("package %w", domain.ErrNotFound)
	}
	return nil
}

func (r *PackageRepo) CountByOrg(ctx context.Context, orgID uint, ecosystem *domain.Ecosystem) (int64, error) {
	query := r.db.WithContext(ctx).Model(&models.Package{}).Where("org_id = ?", orgID)
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
		IsCustom:      m.IsCustom,
		Rank:          m.Rank,
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
		IsCustom:      d.IsCustom,
		Rank:          d.Rank,
		CreatedAt:     d.CreatedAt,
		UpdatedAt:     d.UpdatedAt,
	}
}
