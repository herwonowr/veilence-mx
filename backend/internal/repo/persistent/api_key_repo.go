package persistent

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/veilence/veilence-mx/backend/internal/entity"
)

// APIKeyRepo implements entity.APIKeyRepository using GORM.
type APIKeyRepo struct {
	db *gorm.DB
}

// NewAPIKeyRepo creates a new APIKeyRepo.
func NewAPIKeyRepo(db *gorm.DB) *APIKeyRepo {
	return &APIKeyRepo{db: db}
}

func (r *APIKeyRepo) FindActiveByPrefix(ctx context.Context, prefix string) ([]entity.APIKey, error) {
	var ms []APIKey
	if err := r.db.WithContext(ctx).Where("is_active = ? AND key_prefix = ?", true, prefix).Find(&ms).Error; err != nil {
		return nil, fmt.Errorf("finding api keys by prefix: %w", err)
	}
	result := make([]entity.APIKey, len(ms))
	for i := range ms {
		result[i] = *apiKeyToDomain(&ms[i])
	}
	return result, nil
}

func (r *APIKeyRepo) FindByUserIDAndWorkspaceID(ctx context.Context, userID, workspaceID string) ([]entity.APIKey, error) {
	var ms []APIKey
	if err := r.db.WithContext(ctx).Where("user_id = ? AND workspace_id = ?", userID, workspaceID).Find(&ms).Error; err != nil {
		return nil, fmt.Errorf("APIKeyRepo.FindByUserIDAndWorkspaceID: %w", err)
	}
	result := make([]entity.APIKey, len(ms))
	for i := range ms {
		result[i] = *apiKeyToDomain(&ms[i])
	}
	return result, nil
}

func (r *APIKeyRepo) Create(ctx context.Context, key *entity.APIKey) error {
	m := apiKeyToModel(key)
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return fmt.Errorf("creating api key: %w", err)
	}
	key.ID = m.ID
	key.CreatedAt = m.CreatedAt
	return nil
}

func (r *APIKeyRepo) SoftDeleteScoped(ctx context.Context, userID, workspaceID, keyID string) error {
	result := r.db.WithContext(ctx).Where("id = ? AND user_id = ? AND workspace_id = ?", keyID, userID, workspaceID).Delete(&APIKey{})
	if result.Error != nil {
		return fmt.Errorf("APIKeyRepo.SoftDeleteScoped: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("api key %w", entity.ErrNotFound)
	}
	return nil
}

// --- Converters ---

func apiKeyToDomain(m *APIKey) *entity.APIKey {
	return &entity.APIKey{
		ID:          m.ID,
		UserID:      m.UserID,
		WorkspaceID: m.WorkspaceID,
		Name:        m.Name,
		KeyHash:     m.KeyHash,
		KeyPrefix:   m.KeyPrefix,
		Role:        entity.APIKeyRole(m.Role),
		LastUsedAt:  m.LastUsedAt,
		ExpiresAt:   m.ExpiresAt,
		IsActive:    m.IsActive,
		CreatedAt:   m.CreatedAt,
	}
}

func apiKeyToModel(d *entity.APIKey) *APIKey {
	return &APIKey{
		ID:          d.ID,
		UserID:      d.UserID,
		WorkspaceID: d.WorkspaceID,
		Name:        d.Name,
		KeyHash:     d.KeyHash,
		KeyPrefix:   d.KeyPrefix,
		Role:        string(d.Role),
		LastUsedAt:  d.LastUsedAt,
		ExpiresAt:   d.ExpiresAt,
		IsActive:    d.IsActive,
		CreatedAt:   d.CreatedAt,
	}
}
