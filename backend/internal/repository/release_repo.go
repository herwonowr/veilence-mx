package repository

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/veilence/veilence-mx/backend/internal/domain"
	"github.com/veilence/veilence-mx/backend/internal/models"
)

// ReleaseRepo implements domain.ReleaseRepository using GORM.
type ReleaseRepo struct {
	db *gorm.DB
}

// NewReleaseRepo creates a new ReleaseRepo.
func NewReleaseRepo(db *gorm.DB) *ReleaseRepo {
	return &ReleaseRepo{db: db}
}

func (r *ReleaseRepo) FindByID(ctx context.Context, id uint) (*domain.Release, error) {
	var m models.Release
	if err := r.db.WithContext(ctx).First(&m, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("release %w", domain.ErrNotFound)
		}
		return nil, fmt.Errorf("finding release: %w", err)
	}
	return releaseToDomain(&m), nil
}

func (r *ReleaseRepo) FindByPackageID(ctx context.Context, packageID uint, page, limit int) ([]domain.Release, int64, error) {
	var total int64
	query := r.db.WithContext(ctx).Model(&models.Release{}).Where("package_id = ?", packageID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("counting releases: %w", err)
	}

	var ms []models.Release
	err := query.
		Order("created_at DESC").
		Offset((page - 1) * limit).
		Limit(limit).
		Find(&ms).Error
	if err != nil {
		return nil, 0, fmt.Errorf("listing releases: %w", err)
	}

	result := make([]domain.Release, len(ms))
	for i := range ms {
		result[i] = *releaseToDomain(&ms[i])
	}
	return result, total, nil
}

func (r *ReleaseRepo) Create(ctx context.Context, release *domain.Release) error {
	m := releaseToModel(release)
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return fmt.Errorf("creating release: %w", err)
	}
	release.ID = m.ID
	release.CreatedAt = m.CreatedAt
	return nil
}

func (r *ReleaseRepo) Update(ctx context.Context, release *domain.Release) error {
	m := releaseToModel(release)
	if err := r.db.WithContext(ctx).Save(m).Error; err != nil {
		return fmt.Errorf("updating release: %w", err)
	}
	return nil
}

// --- Converters ---

func releaseToDomain(m *models.Release) *domain.Release {
	return &domain.Release{
		ID:           m.ID,
		PackageID:    m.PackageID,
		Version:      m.Version,
		PublishedAt:  m.PublishedAt,
		TarballURL:   m.TarballURL,
		SHA256:       m.SHA256,
		Status:       domain.ReleaseStatus(m.Status),
		ErrorMessage: m.ErrorMessage,
		CreatedAt:    m.CreatedAt,
	}
}

func releaseToModel(d *domain.Release) *models.Release {
	return &models.Release{
		ID:           d.ID,
		PackageID:    d.PackageID,
		Version:      d.Version,
		PublishedAt:  d.PublishedAt,
		TarballURL:   d.TarballURL,
		SHA256:       d.SHA256,
		Status:       models.ReleaseStatus(d.Status),
		ErrorMessage: d.ErrorMessage,
		CreatedAt:    d.CreatedAt,
	}
}
