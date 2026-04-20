package persistent

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/veilence/veilence-mx/backend/internal/entity"
)

// DifferRepo implements differ.DifferRepository using GORM.
type DifferRepo struct {
	db *gorm.DB
}

// NewDifferRepo creates a new DifferRepo.
func NewDifferRepo(db *gorm.DB) *DifferRepo {
	return &DifferRepo{db: db}
}

// FindReleaseByIDWithPackage loads a release by ID along with its package.
func (r *DifferRepo) FindReleaseByIDWithPackage(ctx context.Context, id string) (*entity.Release, *entity.Package, error) {
	var model Release
	if err := r.db.WithContext(ctx).Preload("Package").First(&model, id).Error; err != nil {
		return nil, nil, fmt.Errorf("DifferRepo.FindReleaseByIDWithPackage: %w", err)
	}

	release := &entity.Release{
		ID:           model.ID,
		PackageID:    model.PackageID,
		Version:      model.Version,
		PublishedAt:  model.PublishedAt,
		TarballURL:   model.TarballURL,
		SHA256:       model.SHA256,
		Status:       entity.ReleaseStatus(model.Status),
		ErrorMessage: model.ErrorMessage,
		CreatedAt:    model.CreatedAt,
	}

	pkg := &entity.Package{
		ID:          model.Package.ID,
		WorkspaceID: model.Package.WorkspaceID,
		Name:        model.Package.Name,
		Ecosystem:   entity.Ecosystem(model.Package.Ecosystem),
	}

	return release, pkg, nil
}

// FindPreviousCompletedRelease finds the latest completed release before the given publish time.
func (r *DifferRepo) FindPreviousCompletedRelease(ctx context.Context, packageID string, beforePublishedAt time.Time) (*entity.Release, error) {
	var model Release
	err := r.db.WithContext(ctx).
		Where("package_id = ? AND status = ? AND published_at < ?", packageID, string(ReleaseStatusCompleted), beforePublishedAt).
		Order("published_at DESC").
		First(&model).Error
	if err != nil {
		return nil, fmt.Errorf("DifferRepo.FindPreviousCompletedRelease: %w", err)
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

// UpdateReleaseStatus updates the status of a release.
func (r *DifferRepo) UpdateReleaseStatus(ctx context.Context, id string, status entity.ReleaseStatus) error {
	if err := r.db.WithContext(ctx).Model(&Release{}).Where("id = ?", id).Update("status", string(status)).Error; err != nil {
		return fmt.Errorf("DifferRepo.UpdateReleaseStatus: %w", err)
	}
	return nil
}

// UpdateReleaseError updates the status and error message on a release.
func (r *DifferRepo) UpdateReleaseError(ctx context.Context, id string, status entity.ReleaseStatus, errorMessage string) error {
	if err := r.db.WithContext(ctx).Model(&Release{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":        string(status),
		"error_message": errorMessage,
	}).Error; err != nil {
		return fmt.Errorf("DifferRepo.UpdateReleaseError: %w", err)
	}
	return nil
}

// CreateDiff persists a new diff record.
func (r *DifferRepo) CreateDiff(ctx context.Context, diff *entity.Diff) error {
	model := Diff{
		ReleaseID:        diff.ReleaseID,
		PrevReleaseID:    diff.PrevReleaseID,
		DiffContent:      diff.DiffContent,
		FileChangesCount: diff.FileChangesCount,
		LinesAdded:       diff.LinesAdded,
		LinesRemoved:     diff.LinesRemoved,
	}
	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		return fmt.Errorf("DifferRepo.CreateDiff: %w", err)
	}
	diff.ID = model.ID
	diff.CreatedAt = model.CreatedAt
	return nil
}
