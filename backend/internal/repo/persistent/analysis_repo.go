package persistent

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/veilence/veilence-mx/backend/internal/entity"
)

// AnalysisRepo implements entity.AnalysisRepository using GORM.
type AnalysisRepo struct {
	db *gorm.DB
}

// NewAnalysisRepo creates a new AnalysisRepo.
func NewAnalysisRepo(db *gorm.DB) *AnalysisRepo {
	return &AnalysisRepo{db: db}
}

func (r *AnalysisRepo) FindByDiffID(ctx context.Context, diffID string) ([]entity.Analysis, error) {
	var ms []Analysis
	if err := r.db.WithContext(ctx).Where("diff_id = ?", diffID).Find(&ms).Error; err != nil {
		return nil, fmt.Errorf("finding analyses by diff: %w", err)
	}
	result := make([]entity.Analysis, len(ms))
	for i := range ms {
		result[i] = *analysisToDomain(&ms[i])
	}
	return result, nil
}

func (r *AnalysisRepo) Create(ctx context.Context, analysis *entity.Analysis) error {
	m := analysisToModel(analysis)
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return fmt.Errorf("creating analysis: %w", err)
	}
	analysis.ID = m.ID
	analysis.CreatedAt = m.CreatedAt
	return nil
}

func (r *AnalysisRepo) FindByDiffIDs(ctx context.Context, diffIDs []string) ([]entity.Analysis, error) {
	if len(diffIDs) == 0 {
		return []entity.Analysis{}, nil
	}
	var ms []Analysis
	if err := r.db.WithContext(ctx).Where("diff_id IN ?", diffIDs).Find(&ms).Error; err != nil {
		return nil, fmt.Errorf("finding analyses by diff IDs: %w", err)
	}
	result := make([]entity.Analysis, len(ms))
	for i := range ms {
		result[i] = *analysisToDomain(&ms[i])
	}
	return result, nil
}

// --- Converters ---

func analysisToDomain(m *Analysis) *entity.Analysis {
	return &entity.Analysis{
		ID:             m.ID,
		DiffID:         m.DiffID,
		Classification: entity.Classification(m.Classification),
		Confidence:     m.Confidence,
		Reasoning:      m.Reasoning,
		ModelUsed:      m.ModelUsed,
		AnalyzerType:   entity.AnalyzerType(m.AnalyzerType),
		RawResponse:    m.RawResponse,
		CreatedAt:      m.CreatedAt,
	}
}

func analysisToModel(d *entity.Analysis) *Analysis {
	return &Analysis{
		ID:             d.ID,
		DiffID:         d.DiffID,
		Classification: Classification(d.Classification),
		Confidence:     d.Confidence,
		Reasoning:      d.Reasoning,
		ModelUsed:      d.ModelUsed,
		AnalyzerType:   AnalyzerType(d.AnalyzerType),
		RawResponse:    d.RawResponse,
		CreatedAt:      d.CreatedAt,
	}
}
