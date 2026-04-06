package repository

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/veilence/veilence-mx/backend/internal/domain"
	"github.com/veilence/veilence-mx/backend/internal/models"
)

// AnalysisRepo implements domain.AnalysisRepository using GORM.
type AnalysisRepo struct {
	db *gorm.DB
}

// NewAnalysisRepo creates a new AnalysisRepo.
func NewAnalysisRepo(db *gorm.DB) *AnalysisRepo {
	return &AnalysisRepo{db: db}
}

func (r *AnalysisRepo) FindByID(ctx context.Context, id uint) (*domain.Analysis, error) {
	var m models.Analysis
	if err := r.db.WithContext(ctx).First(&m, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("analysis %w", domain.ErrNotFound)
		}
		return nil, fmt.Errorf("finding analysis: %w", err)
	}
	return analysisToDomain(&m), nil
}

func (r *AnalysisRepo) FindByDiffID(ctx context.Context, diffID uint) ([]domain.Analysis, error) {
	var ms []models.Analysis
	if err := r.db.WithContext(ctx).Where("diff_id = ?", diffID).Find(&ms).Error; err != nil {
		return nil, fmt.Errorf("finding analyses by diff: %w", err)
	}
	result := make([]domain.Analysis, len(ms))
	for i := range ms {
		result[i] = *analysisToDomain(&ms[i])
	}
	return result, nil
}

func (r *AnalysisRepo) Create(ctx context.Context, analysis *domain.Analysis) error {
	m := analysisToModel(analysis)
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return fmt.Errorf("creating analysis: %w", err)
	}
	analysis.ID = m.ID
	analysis.CreatedAt = m.CreatedAt
	return nil
}

func (r *AnalysisRepo) CountByDiffID(ctx context.Context, diffID uint) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&models.Analysis{}).Where("diff_id = ?", diffID).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("counting analyses: %w", err)
	}
	return count, nil
}

// --- Converters ---

func analysisToDomain(m *models.Analysis) *domain.Analysis {
	return &domain.Analysis{
		ID:             m.ID,
		DiffID:         m.DiffID,
		Classification: domain.Classification(m.Classification),
		Confidence:     m.Confidence,
		Reasoning:      m.Reasoning,
		ModelUsed:      m.ModelUsed,
		AnalyzerType:   domain.AnalyzerType(m.AnalyzerType),
		RawResponse:    m.RawResponse,
		CreatedAt:      m.CreatedAt,
	}
}

func analysisToModel(d *domain.Analysis) *models.Analysis {
	return &models.Analysis{
		ID:             d.ID,
		DiffID:         d.DiffID,
		Classification: models.Classification(d.Classification),
		Confidence:     d.Confidence,
		Reasoning:      d.Reasoning,
		ModelUsed:      d.ModelUsed,
		AnalyzerType:   models.AnalyzerType(d.AnalyzerType),
		RawResponse:    d.RawResponse,
		CreatedAt:      d.CreatedAt,
	}
}
