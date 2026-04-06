package repository

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/veilence/veilence-mx/backend/internal/domain"
	"github.com/veilence/veilence-mx/backend/internal/models"
)

// APIKeyRepo implements domain.APIKeyRepository using GORM.
type APIKeyRepo struct {
	db *gorm.DB
}

// NewAPIKeyRepo creates a new APIKeyRepo.
func NewAPIKeyRepo(db *gorm.DB) *APIKeyRepo {
	return &APIKeyRepo{db: db}
}

func (r *APIKeyRepo) FindByID(ctx context.Context, id uint) (*domain.APIKey, error) {
	var m models.APIKey
	if err := r.db.WithContext(ctx).First(&m, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("api key %w", domain.ErrNotFound)
		}
		return nil, fmt.Errorf("finding api key: %w", err)
	}
	return apiKeyToDomain(&m), nil
}

func (r *APIKeyRepo) FindActiveByPrefix(ctx context.Context, prefix string) ([]domain.APIKey, error) {
	var ms []models.APIKey
	if err := r.db.WithContext(ctx).Where("is_active = ? AND key_prefix = ?", true, prefix).Find(&ms).Error; err != nil {
		return nil, fmt.Errorf("finding api keys by prefix: %w", err)
	}
	result := make([]domain.APIKey, len(ms))
	for i := range ms {
		result[i] = *apiKeyToDomain(&ms[i])
	}
	return result, nil
}

func (r *APIKeyRepo) FindByUserID(ctx context.Context, userID uint) ([]domain.APIKey, error) {
	var ms []models.APIKey
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&ms).Error; err != nil {
		return nil, fmt.Errorf("listing api keys: %w", err)
	}
	result := make([]domain.APIKey, len(ms))
	for i := range ms {
		result[i] = *apiKeyToDomain(&ms[i])
	}
	return result, nil
}

func (r *APIKeyRepo) Create(ctx context.Context, key *domain.APIKey) error {
	m := apiKeyToModel(key)
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return fmt.Errorf("creating api key: %w", err)
	}
	key.ID = m.ID
	key.CreatedAt = m.CreatedAt
	return nil
}

func (r *APIKeyRepo) Update(ctx context.Context, key *domain.APIKey) error {
	m := apiKeyToModel(key)
	if err := r.db.WithContext(ctx).Save(m).Error; err != nil {
		return fmt.Errorf("updating api key: %w", err)
	}
	return nil
}

func (r *APIKeyRepo) SoftDelete(ctx context.Context, userID, keyID uint) error {
	result := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", keyID, userID).Delete(&models.APIKey{})
	if result.Error != nil {
		return fmt.Errorf("revoking api key: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("api key %w", domain.ErrNotFound)
	}
	return nil
}

// --- Converters ---

func apiKeyToDomain(m *models.APIKey) *domain.APIKey {
	return &domain.APIKey{
		ID:         m.ID,
		UserID:     m.UserID,
		Name:       m.Name,
		KeyHash:    m.KeyHash,
		KeyPrefix:  m.KeyPrefix,
		Scope:      domain.APIKeyScope(m.Scope),
		LastUsedAt: m.LastUsedAt,
		ExpiresAt:  m.ExpiresAt,
		IsActive:   m.IsActive,
		CreatedAt:  m.CreatedAt,
	}
}

func apiKeyToModel(d *domain.APIKey) *models.APIKey {
	return &models.APIKey{
		ID:         d.ID,
		UserID:     d.UserID,
		Name:       d.Name,
		KeyHash:    d.KeyHash,
		KeyPrefix:  d.KeyPrefix,
		Scope:      string(d.Scope),
		LastUsedAt: d.LastUsedAt,
		ExpiresAt:  d.ExpiresAt,
		IsActive:   d.IsActive,
		CreatedAt:  d.CreatedAt,
	}
}
