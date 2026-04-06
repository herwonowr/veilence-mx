package repository

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/veilence/veilence-mx/backend/internal/domain"
	"github.com/veilence/veilence-mx/backend/internal/models"
)

// DiffRepo implements domain.DiffRepository using GORM.
type DiffRepo struct {
	db *gorm.DB
}

// NewDiffRepo creates a new DiffRepo.
func NewDiffRepo(db *gorm.DB) *DiffRepo {
	return &DiffRepo{db: db}
}

func (r *DiffRepo) FindByID(ctx context.Context, id uint) (*domain.Diff, error) {
	var m models.Diff
	if err := r.db.WithContext(ctx).First(&m, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("diff %w", domain.ErrNotFound)
		}
		return nil, fmt.Errorf("finding diff: %w", err)
	}
	return diffToDomain(&m), nil
}

func (r *DiffRepo) FindByReleaseID(ctx context.Context, releaseID uint) ([]domain.Diff, error) {
	var ms []models.Diff
	if err := r.db.WithContext(ctx).Where("release_id = ?", releaseID).Find(&ms).Error; err != nil {
		return nil, fmt.Errorf("finding diffs by release: %w", err)
	}
	result := make([]domain.Diff, len(ms))
	for i := range ms {
		result[i] = *diffToDomain(&ms[i])
	}
	return result, nil
}

func (r *DiffRepo) Create(ctx context.Context, diff *domain.Diff) error {
	m := diffToModel(diff)
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return fmt.Errorf("creating diff: %w", err)
	}
	diff.ID = m.ID
	diff.CreatedAt = m.CreatedAt
	return nil
}

// --- Converters ---

func diffToDomain(m *models.Diff) *domain.Diff {
	return &domain.Diff{
		ID:               m.ID,
		ReleaseID:        m.ReleaseID,
		PrevReleaseID:    m.PrevReleaseID,
		DiffContent:      m.DiffContent,
		FileChangesCount: m.FileChangesCount,
		LinesAdded:       m.LinesAdded,
		LinesRemoved:     m.LinesRemoved,
		CreatedAt:        m.CreatedAt,
	}
}

func diffToModel(d *domain.Diff) *models.Diff {
	return &models.Diff{
		ID:               d.ID,
		ReleaseID:        d.ReleaseID,
		PrevReleaseID:    d.PrevReleaseID,
		DiffContent:      d.DiffContent,
		FileChangesCount: d.FileChangesCount,
		LinesAdded:       d.LinesAdded,
		LinesRemoved:     d.LinesRemoved,
		CreatedAt:        d.CreatedAt,
	}
}
