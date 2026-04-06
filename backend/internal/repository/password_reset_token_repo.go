package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/veilence/veilence-mx/backend/internal/domain"
	"github.com/veilence/veilence-mx/backend/internal/models"
)

// PasswordResetTokenRepo implements domain.PasswordResetTokenRepository using GORM.
type PasswordResetTokenRepo struct {
	db *gorm.DB
}

// NewPasswordResetTokenRepo creates a new PasswordResetTokenRepo.
func NewPasswordResetTokenRepo(db *gorm.DB) *PasswordResetTokenRepo {
	return &PasswordResetTokenRepo{db: db}
}

func (r *PasswordResetTokenRepo) FindByTokenHash(ctx context.Context, tokenHash string) (*domain.PasswordResetToken, error) {
	var m models.PasswordResetToken
	if err := r.db.WithContext(ctx).Where("token_hash = ?", tokenHash).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("password reset token %w", domain.ErrNotFound)
		}
		return nil, fmt.Errorf("finding password reset token: %w", err)
	}
	return toDomainPasswordResetToken(&m), nil
}

func (r *PasswordResetTokenRepo) Create(ctx context.Context, token *domain.PasswordResetToken) error {
	m := toModelPasswordResetToken(token)
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return fmt.Errorf("creating password reset token: %w", err)
	}
	token.ID = m.ID
	token.CreatedAt = m.CreatedAt
	return nil
}

func (r *PasswordResetTokenRepo) MarkUsed(ctx context.Context, id uint) error {
	now := time.Now()
	result := r.db.WithContext(ctx).Model(&models.PasswordResetToken{}).Where("id = ?", id).Update("used_at", &now)
	if result.Error != nil {
		return fmt.Errorf("marking password reset token used: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("password reset token %w", domain.ErrNotFound)
	}
	return nil
}

func (r *PasswordResetTokenRepo) DeleteExpiredByUserID(ctx context.Context, userID uint) error {
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND (expires_at < ? OR used_at IS NOT NULL)", userID, time.Now()).
		Delete(&models.PasswordResetToken{}).Error; err != nil {
		return fmt.Errorf("deleting expired password reset tokens: %w", err)
	}
	return nil
}

func toDomainPasswordResetToken(m *models.PasswordResetToken) *domain.PasswordResetToken {
	return &domain.PasswordResetToken{
		ID:        m.ID,
		UserID:    m.UserID,
		TokenHash: m.TokenHash,
		ExpiresAt: m.ExpiresAt,
		UsedAt:    m.UsedAt,
		CreatedAt: m.CreatedAt,
	}
}

func toModelPasswordResetToken(d *domain.PasswordResetToken) *models.PasswordResetToken {
	return &models.PasswordResetToken{
		ID:        d.ID,
		UserID:    d.UserID,
		TokenHash: d.TokenHash,
		ExpiresAt: d.ExpiresAt,
		UsedAt:    d.UsedAt,
	}
}
