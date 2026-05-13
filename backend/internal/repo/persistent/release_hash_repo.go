package persistent

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/veilence/veilence-mx/backend/internal/entity"
)

// ReleaseHashRepo implements usecase.ReleaseHashRepository using GORM.
type ReleaseHashRepo struct {
	db *gorm.DB
}

// NewReleaseHashRepo creates a new ReleaseHashRepo.
func NewReleaseHashRepo(db *gorm.DB) *ReleaseHashRepo {
	return &ReleaseHashRepo{db: db}
}

// CreateBatch inserts multiple release hashes in a single batch.
func (r *ReleaseHashRepo) CreateBatch(ctx context.Context, hashes []entity.ReleaseHash) error {
	if len(hashes) == 0 {
		return nil
	}

	models := make([]ReleaseHash, len(hashes))
	for i, h := range hashes {
		models[i] = ReleaseHash{
			ReleaseID: h.ReleaseID,
			Filename:  h.Filename,
			Algorithm: h.Algorithm,
			Hash:      h.Hash,
		}
	}

	if err := r.db.WithContext(ctx).Create(&models).Error; err != nil {
		return fmt.Errorf("ReleaseHashRepo.CreateBatch: %w", err)
	}
	return nil
}

// FindByReleaseID returns all hashes for a given release.
func (r *ReleaseHashRepo) FindByReleaseID(ctx context.Context, releaseID string) ([]entity.ReleaseHash, error) {
	var models []ReleaseHash
	if err := r.db.WithContext(ctx).Where("release_id = ?", releaseID).Order("filename, algorithm").Find(&models).Error; err != nil {
		return nil, fmt.Errorf("ReleaseHashRepo.FindByReleaseID: %w", err)
	}

	result := make([]entity.ReleaseHash, len(models))
	for i, m := range models {
		result[i] = entity.ReleaseHash{
			ID:        m.ID,
			ReleaseID: m.ReleaseID,
			Filename:  m.Filename,
			Algorithm: m.Algorithm,
			Hash:      m.Hash,
			CreatedAt: m.CreatedAt,
		}
	}
	return result, nil
}

// CountByReleaseID returns the number of hashes for a given release.
func (r *ReleaseHashRepo) CountByReleaseID(ctx context.Context, releaseID string) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&ReleaseHash{}).Where("release_id = ?", releaseID).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("ReleaseHashRepo.CountByReleaseID: %w", err)
	}
	return count, nil
}

// CountByReleaseIDs returns hash counts for multiple releases in a single query.
func (r *ReleaseHashRepo) CountByReleaseIDs(ctx context.Context, releaseIDs []string) (map[string]int64, error) {
	result := make(map[string]int64, len(releaseIDs))
	if len(releaseIDs) == 0 {
		return result, nil
	}

	type countRow struct {
		ReleaseID string
		Count     int64
	}
	var rows []countRow
	if err := r.db.WithContext(ctx).
		Model(&ReleaseHash{}).
		Select("release_id, count(*) as count").
		Where("release_id IN ?", releaseIDs).
		Group("release_id").
		Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("ReleaseHashRepo.CountByReleaseIDs: %w", err)
	}

	for _, row := range rows {
		result[row.ReleaseID] = row.Count
	}
	return result, nil
}
