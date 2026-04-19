package persistent

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/veilence/veilence-mx/backend/internal/entity"
)

// RefreshTokenRepo implements entity.RefreshTokenRepository using GORM.
type RefreshTokenRepo struct {
	db *gorm.DB
}

// NewRefreshTokenRepo creates a new RefreshTokenRepo.
func NewRefreshTokenRepo(db *gorm.DB) *RefreshTokenRepo {
	return &RefreshTokenRepo{db: db}
}

func (r *RefreshTokenRepo) FindByTokenHash(ctx context.Context, tokenHash string) (*entity.RefreshToken, error) {
	var m RefreshToken
	if err := r.db.WithContext(ctx).Where("token_hash = ?", tokenHash).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("refresh token %w", entity.ErrNotFound)
		}
		return nil, fmt.Errorf("finding refresh token: %w", err)
	}
	return refreshTokenToDomain(&m), nil
}

func (r *RefreshTokenRepo) Create(ctx context.Context, token *entity.RefreshToken) error {
	m := refreshTokenToModel(token)
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return fmt.Errorf("creating refresh token: %w", err)
	}
	token.ID = m.ID
	token.CreatedAt = m.CreatedAt
	return nil
}

func (r *RefreshTokenRepo) Delete(ctx context.Context, id uint) error {
	if err := r.db.WithContext(ctx).Delete(&RefreshToken{}, id).Error; err != nil {
		return fmt.Errorf("deleting refresh token: %w", err)
	}
	return nil
}

func (r *RefreshTokenRepo) DeleteByTokenHash(ctx context.Context, tokenHash string) error {
	if err := r.db.WithContext(ctx).Where("token_hash = ?", tokenHash).Delete(&RefreshToken{}).Error; err != nil {
		return fmt.Errorf("deleting refresh token by hash: %w", err)
	}
	return nil
}

func (r *RefreshTokenRepo) DeleteByUserID(ctx context.Context, userID uint) error {
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&RefreshToken{}).Error; err != nil {
		return fmt.Errorf("deleting refresh tokens by user: %w", err)
	}
	return nil
}

// --- Converters ---

func refreshTokenToDomain(m *RefreshToken) *entity.RefreshToken {
	return &entity.RefreshToken{
		ID:        m.ID,
		UserID:    m.UserID,
		TokenHash: m.TokenHash,
		ExpiresAt: m.ExpiresAt,
		CreatedAt: m.CreatedAt,
	}
}

func refreshTokenToModel(d *entity.RefreshToken) *RefreshToken {
	return &RefreshToken{
		ID:        d.ID,
		UserID:    d.UserID,
		TokenHash: d.TokenHash,
		ExpiresAt: d.ExpiresAt,
		CreatedAt: d.CreatedAt,
	}
}
