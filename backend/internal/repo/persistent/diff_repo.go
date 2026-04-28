package persistent

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/veilence/veilence-mx/backend/internal/entity"
)

// DiffRepo implements entity.DiffRepository using GORM.
type DiffRepo struct {
	db *gorm.DB
}

// NewDiffRepo creates a new DiffRepo.
func NewDiffRepo(db *gorm.DB) *DiffRepo {
	return &DiffRepo{db: db}
}

func (r *DiffRepo) FindFirstByReleaseID(ctx context.Context, releaseID string) (*entity.Diff, error) {
	var ms []Diff
	err := r.db.WithContext(ctx).
		Where("release_id = ?", releaseID).
		Order("created_at ASC").
		Limit(1).
		Find(&ms).Error
	if err != nil {
		return nil, fmt.Errorf("finding first diff by release: %w", err)
	}
	if len(ms) == 0 {
		return nil, nil
	}
	return diffToDomain(&ms[0]), nil
}

func (r *DiffRepo) FindByReleaseIDs(ctx context.Context, releaseIDs []string) ([]entity.Diff, error) {
	if len(releaseIDs) == 0 {
		return []entity.Diff{}, nil
	}
	var ms []Diff
	if err := r.db.WithContext(ctx).Where("release_id IN ?", releaseIDs).Find(&ms).Error; err != nil {
		return nil, fmt.Errorf("finding diffs by release IDs: %w", err)
	}
	result := make([]entity.Diff, len(ms))
	for i := range ms {
		result[i] = *diffToDomain(&ms[i])
	}
	return result, nil
}

// --- Converters ---

func diffToDomain(m *Diff) *entity.Diff {
	return &entity.Diff{
		ID:               m.ID,
		ReleaseID:        m.ReleaseID,
		PrevReleaseID:    m.PrevReleaseID,
		DiffContent:      m.DiffContent,
		FileChangesCount: m.FileChangesCount,
		LinesAdded:       m.LinesAdded,
		LinesRemoved:     m.LinesRemoved,
		Truncated:        m.Truncated,
		OriginalSize:     m.OriginalSize,
		CreatedAt:        m.CreatedAt,
	}
}
