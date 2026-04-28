package persistent

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"github.com/veilence/veilence-mx/backend/internal/entity"
)

// ReleaseRepo implements entity.ReleaseRepository using GORM.
type ReleaseRepo struct {
	db *gorm.DB
}

// NewReleaseRepo creates a new ReleaseRepo.
func NewReleaseRepo(db *gorm.DB) *ReleaseRepo {
	return &ReleaseRepo{db: db}
}

func (r *ReleaseRepo) FindByID(ctx context.Context, id string) (*entity.Release, error) {
	var m Release
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("release %w", entity.ErrNotFound)
		}
		return nil, fmt.Errorf("finding release: %w", err)
	}
	return releaseToDomain(&m), nil
}

func (r *ReleaseRepo) FindByPackageID(ctx context.Context, packageID string, page, limit int) ([]entity.Release, int64, error) {
	var total int64
	query := r.db.WithContext(ctx).Model(&Release{}).Where("package_id = ?", packageID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("counting releases: %w", err)
	}

	var ms []Release
	err := query.
		Order("created_at DESC").
		Offset((page - 1) * limit).
		Limit(limit).
		Find(&ms).Error
	if err != nil {
		return nil, 0, fmt.Errorf("listing releases: %w", err)
	}

	result := make([]entity.Release, len(ms))
	for i := range ms {
		result[i] = *releaseToDomain(&ms[i])
	}
	return result, total, nil
}

func (r *ReleaseRepo) FindByPackageIDAndWorkspace(ctx context.Context, packageID, workspaceID string, page, limit int) ([]entity.Release, int64, error) {
	var total int64
	query := r.db.WithContext(ctx).Model(&Release{}).
		Joins("JOIN packages ON packages.id = releases.package_id").
		Where("releases.package_id = ? AND packages.workspace_id = ?", packageID, workspaceID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("counting releases: %w", err)
	}

	var ms []Release
	err := query.
		Select("releases.*").
		Order("releases.created_at DESC").
		Offset((page - 1) * limit).
		Limit(limit).
		Find(&ms).Error
	if err != nil {
		return nil, 0, fmt.Errorf("listing releases: %w", err)
	}

	result := make([]entity.Release, len(ms))
	for i := range ms {
		result[i] = *releaseToDomain(&ms[i])
	}
	return result, total, nil
}

func (r *ReleaseRepo) Create(ctx context.Context, release *entity.Release) error {
	m := releaseToModel(release)
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return fmt.Errorf("creating release: %w", err)
	}
	release.ID = m.ID
	release.CreatedAt = m.CreatedAt
	return nil
}

func (r *ReleaseRepo) FindByIDWithPackage(ctx context.Context, id string) (*entity.Release, *entity.Package, error) {
	var m Release
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, fmt.Errorf("release %w", entity.ErrNotFound)
		}
		return nil, nil, fmt.Errorf("finding release: %w", err)
	}

	var pkg Package
	if err := r.db.WithContext(ctx).Where("id = ?", m.PackageID).First(&pkg).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, fmt.Errorf("package %w", entity.ErrNotFound)
		}
		return nil, nil, fmt.Errorf("finding release package: %w", err)
	}

	return releaseToDomain(&m), packageToDomain(&pkg), nil
}

func (r *ReleaseRepo) FindByIDWithPackageAndWorkspace(ctx context.Context, id, workspaceID string) (*entity.Release, *entity.Package, error) {
	var m Release
	err := r.db.WithContext(ctx).
		Joins("JOIN packages ON packages.id = releases.package_id").
		Where("releases.id = ? AND packages.workspace_id = ?", id, workspaceID).
		Select("releases.*").
		First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, fmt.Errorf("release %w", entity.ErrNotFound)
		}
		return nil, nil, fmt.Errorf("finding release: %w", err)
	}

	var pkg Package
	if err := r.db.WithContext(ctx).Where("id = ? AND workspace_id = ?", m.PackageID, workspaceID).First(&pkg).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, fmt.Errorf("package %w", entity.ErrNotFound)
		}
		return nil, nil, fmt.Errorf("finding release package: %w", err)
	}

	return releaseToDomain(&m), packageToDomain(&pkg), nil
}

func (r *ReleaseRepo) FindByWorkspaceID(ctx context.Context, workspaceID string, page, limit int, sortClause string, filters entity.ReleaseFilters) ([]entity.Release, int64, error) {
	var total int64
	query := r.db.WithContext(ctx).Model(&Release{}).
		Joins("JOIN packages ON packages.id = releases.package_id").
		Where("packages.workspace_id = ? AND packages.status = ?", workspaceID, PackageStatusActive)

	if filters.Ecosystem != nil {
		query = query.Where("packages.ecosystem = ?", string(*filters.Ecosystem))
	}
	if filters.Status != nil {
		if *filters.Status == entity.ReleaseStatusInProgress {
			query = query.Where("releases.status IN ?", entity.InProgressStatuses)
		} else {
			query = query.Where("releases.status = ?", string(*filters.Status))
		}
	}
	if filters.Search != nil && *filters.Search != "" {
		escaped := escapeLikeRepo(*filters.Search)
		query = query.Where("LOWER(packages.name) LIKE LOWER(?)", "%"+escaped+"%")
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("counting releases: %w", err)
	}
	if total == 0 {
		return []entity.Release{}, 0, nil
	}

	// Qualify sort clause if it doesn't contain a table prefix
	if sortClause != "" && !strings.Contains(sortClause, ".") {
		sortClause = "releases." + sortClause
	}

	var ms []Release
	err := query.
		Select("releases.*").
		Order(sortClause).
		Offset((page - 1) * limit).
		Limit(limit).
		Find(&ms).Error
	if err != nil {
		return nil, 0, fmt.Errorf("listing releases by workspace: %w", err)
	}

	result := make([]entity.Release, len(ms))
	for i := range ms {
		result[i] = *releaseToDomain(&ms[i])
	}
	return result, total, nil
}

func (r *ReleaseRepo) FindByWorkspaceIDWithDetails(ctx context.Context, workspaceID string, page, limit int, sortClause string, filters entity.ReleaseFilters) ([]entity.ReleaseWithDetails, int64, error) {
	var total int64
	query := r.db.WithContext(ctx).Model(&Release{}).
		Joins("JOIN packages ON packages.id = releases.package_id").
		Where("packages.workspace_id = ? AND packages.status = ?", workspaceID, PackageStatusActive)

	if filters.Ecosystem != nil {
		query = query.Where("packages.ecosystem = ?", string(*filters.Ecosystem))
	}
	if filters.Status != nil {
		if *filters.Status == entity.ReleaseStatusInProgress {
			query = query.Where("releases.status IN ?", entity.InProgressStatuses)
		} else {
			query = query.Where("releases.status = ?", string(*filters.Status))
		}
	}
	if filters.Search != nil && *filters.Search != "" {
		escaped := escapeLikeRepo(*filters.Search)
		query = query.Where("LOWER(packages.name) LIKE LOWER(?)", "%"+escaped+"%")
	}
	if filters.LatestPerPackage {
		// Sub-select latest release per package
		query = query.Where("releases.id IN (SELECT DISTINCT ON (r2.package_id) r2.id FROM releases r2 ORDER BY r2.package_id, r2.created_at DESC)")
	}
	if filters.Classification != nil && *filters.Classification != "" {
		cls := *filters.Classification
		if cls == "baseline" {
			// Baseline = completed release without a diff
			query = query.Where("releases.status = 'completed' AND NOT EXISTS (SELECT 1 FROM diffs WHERE diffs.release_id = releases.id)")
		} else {
			// Filter by analysis classification through diff
			query = query.
				Joins("LEFT JOIN diffs ON diffs.release_id = releases.id").
				Joins("LEFT JOIN analyses ON analyses.diff_id = diffs.id").
				Where("analyses.classification = ?", cls)
		}
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("counting releases with details: %w", err)
	}
	if total == 0 {
		return []entity.ReleaseWithDetails{}, 0, nil
	}

	// Qualify sort clause if needed
	if sortClause != "" && !strings.Contains(sortClause, ".") {
		sortClause = "releases." + sortClause
	}

	type releaseRow struct {
		Release
		PackageName      string `gorm:"column:package_name"`
		PackageEcosystem string `gorm:"column:package_ecosystem"`
		Classification   string `gorm:"column:classification"`
	}

	var rows []releaseRow
	err := query.
		Select("releases.*, packages.name AS package_name, packages.ecosystem AS package_ecosystem, " +
			"(SELECT a.classification FROM analyses a JOIN diffs d ON d.id = a.diff_id WHERE d.release_id = releases.id ORDER BY a.created_at DESC LIMIT 1) AS classification").
		Order(sortClause).
		Offset((page - 1) * limit).
		Limit(limit).
		Find(&rows).Error
	if err != nil {
		return nil, 0, fmt.Errorf("listing releases with details by workspace: %w", err)
	}

	result := make([]entity.ReleaseWithDetails, len(rows))
	for i, row := range rows {
		rel := releaseToDomain(&row.Release)
		cls := row.Classification
		if cls == "" && rel.Status == entity.ReleaseStatusCompleted {
			cls = "baseline"
		}
		result[i] = entity.ReleaseWithDetails{
			Release:          *rel,
			PackageName:      row.PackageName,
			PackageEcosystem: row.PackageEcosystem,
			Classification:   cls,
		}
	}
	return result, total, nil
}

func (r *ReleaseRepo) FindByPackageIDAll(ctx context.Context, packageID string) ([]entity.Release, error) {
	var ms []Release
	err := r.db.WithContext(ctx).
		Where("package_id = ?", packageID).
		Order("created_at DESC").
		Find(&ms).Error
	if err != nil {
		return nil, fmt.Errorf("listing all releases for package: %w", err)
	}

	result := make([]entity.Release, len(ms))
	for i := range ms {
		result[i] = *releaseToDomain(&ms[i])
	}
	return result, nil
}

func (r *ReleaseRepo) UpdateStatus(ctx context.Context, id string, status entity.ReleaseStatus) error {
	result := r.db.WithContext(ctx).
		Model(&Release{}).
		Where("id = ?", id).
		Update("status", string(status))
	if result.Error != nil {
		return fmt.Errorf("updating release status: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("release %w", entity.ErrNotFound)
	}
	return nil
}

func (r *ReleaseRepo) Update(ctx context.Context, release *entity.Release) error {
	m := releaseToModel(release)
	if err := r.db.WithContext(ctx).Save(m).Error; err != nil {
		return fmt.Errorf("updating release: %w", err)
	}
	return nil
}

// --- Converters ---

func releaseToDomain(m *Release) *entity.Release {
	return &entity.Release{
		ID:           m.ID,
		PackageID:    m.PackageID,
		Version:      m.Version,
		PublishedAt:  m.PublishedAt,
		TarballURL:   m.TarballURL,
		SHA256:       m.SHA256,
		Status:       entity.ReleaseStatus(m.Status),
		ErrorMessage: m.ErrorMessage,
		CreatedAt:    m.CreatedAt,
	}
}

func releaseToModel(d *entity.Release) *Release {
	return &Release{
		ID:           d.ID,
		PackageID:    d.PackageID,
		Version:      d.Version,
		PublishedAt:  d.PublishedAt,
		TarballURL:   d.TarballURL,
		SHA256:       d.SHA256,
		Status:       ReleaseStatus(d.Status),
		ErrorMessage: d.ErrorMessage,
		CreatedAt:    d.CreatedAt,
	}
}
