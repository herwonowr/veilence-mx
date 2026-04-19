package persistent

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/veilence/veilence-mx/backend/internal/entity"
)

// PipelineRepo implements analyzer.PipelineRepository using GORM.
type PipelineRepo struct {
	db *gorm.DB
}

// NewPipelineRepo creates a new PipelineRepo.
func NewPipelineRepo(db *gorm.DB) *PipelineRepo {
	return &PipelineRepo{db: db}
}

// FindDiffWithRelease loads a diff by ID along with its release and package.
func (r *PipelineRepo) FindDiffWithRelease(ctx context.Context, diffID uint) (*entity.Diff, *entity.Release, *entity.Package, error) {
	var model Diff
	if err := r.db.WithContext(ctx).Preload("Release.Package").First(&model, diffID).Error; err != nil {
		return nil, nil, nil, fmt.Errorf("PipelineRepo.FindDiffWithRelease: %w", err)
	}

	diff := &entity.Diff{
		ID:               model.ID,
		ReleaseID:        model.ReleaseID,
		PrevReleaseID:    model.PrevReleaseID,
		DiffContent:      model.DiffContent,
		FileChangesCount: model.FileChangesCount,
		LinesAdded:       model.LinesAdded,
		LinesRemoved:     model.LinesRemoved,
		CreatedAt:        model.CreatedAt,
	}

	release := &entity.Release{
		ID:           model.Release.ID,
		PackageID:    model.Release.PackageID,
		Version:      model.Release.Version,
		PublishedAt:  model.Release.PublishedAt,
		TarballURL:   model.Release.TarballURL,
		SHA256:       model.Release.SHA256,
		Status:       entity.ReleaseStatus(model.Release.Status),
		ErrorMessage: model.Release.ErrorMessage,
		CreatedAt:    model.Release.CreatedAt,
	}

	pkg := &entity.Package{
		ID:          model.Release.Package.ID,
		WorkspaceID: model.Release.Package.WorkspaceID,
		Name:        model.Release.Package.Name,
		Ecosystem:   entity.Ecosystem(model.Release.Package.Ecosystem),
	}

	return diff, release, pkg, nil
}

// FindReleaseByID returns a release by its ID.
func (r *PipelineRepo) FindReleaseByID(ctx context.Context, id uint) (*entity.Release, error) {
	var model Release
	if err := r.db.WithContext(ctx).First(&model, id).Error; err != nil {
		return nil, fmt.Errorf("PipelineRepo.FindReleaseByID: %w", err)
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

// CreateAnalysis persists a new analysis record.
func (r *PipelineRepo) CreateAnalysis(ctx context.Context, analysis *entity.Analysis) error {
	model := Analysis{
		DiffID:         analysis.DiffID,
		Classification: Classification(analysis.Classification),
		Confidence:     analysis.Confidence,
		Reasoning:      analysis.Reasoning,
		ModelUsed:      analysis.ModelUsed,
		AnalyzerType:   AnalyzerType(analysis.AnalyzerType),
		RawResponse:    analysis.RawResponse,
	}
	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		return fmt.Errorf("PipelineRepo.CreateAnalysis: %w", err)
	}
	analysis.ID = model.ID
	analysis.CreatedAt = model.CreatedAt
	return nil
}

// CreateAlert persists a new alert record.
func (r *PipelineRepo) CreateAlert(ctx context.Context, alert *entity.Alert) error {
	model := Alert{
		WorkspaceID: alert.WorkspaceID,
		AnalysisID:  alert.AnalysisID,
		PackageID:   alert.PackageID,
		Severity:    AlertSeverity(alert.Severity),
		Status:      AlertStatus(alert.Status),
		Message:     alert.Message,
	}
	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		return fmt.Errorf("PipelineRepo.CreateAlert: %w", err)
	}
	alert.ID = model.ID
	alert.CreatedAt = model.CreatedAt
	return nil
}

// UpdateReleaseStatus updates the status of a release.
func (r *PipelineRepo) UpdateReleaseStatus(ctx context.Context, id uint, status entity.ReleaseStatus) error {
	if err := r.db.WithContext(ctx).Model(&Release{}).Where("id = ?", id).Update("status", string(status)).Error; err != nil {
		return fmt.Errorf("PipelineRepo.UpdateReleaseStatus: %w", err)
	}
	return nil
}
