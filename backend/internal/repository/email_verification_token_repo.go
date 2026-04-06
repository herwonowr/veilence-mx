package repository

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/veilence/veilence-mx/backend/internal/domain"
	"github.com/veilence/veilence-mx/backend/internal/models"
)

// EmailVerificationTokenRepo implements domain.EmailVerificationTokenRepository using GORM.
type EmailVerificationTokenRepo struct {
	db *gorm.DB
}

// NewEmailVerificationTokenRepo creates a new EmailVerificationTokenRepo.
func NewEmailVerificationTokenRepo(db *gorm.DB) *EmailVerificationTokenRepo {
	return &EmailVerificationTokenRepo{db: db}
}

func (r *EmailVerificationTokenRepo) FindByTokenHash(ctx context.Context, tokenHash string) (*domain.EmailVerificationToken, error) {
	var m models.EmailVerificationToken
	if err := r.db.WithContext(ctx).Where("token_hash = ?", tokenHash).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("email verification token %w", domain.ErrNotFound)
		}
		return nil, fmt.Errorf("finding email verification token: %w", err)
	}
	return toDomainEmailVerificationToken(&m), nil
}

func (r *EmailVerificationTokenRepo) Create(ctx context.Context, token *domain.EmailVerificationToken) error {
	m := toModelEmailVerificationToken(token)
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return fmt.Errorf("creating email verification token: %w", err)
	}
	token.ID = m.ID
	token.CreatedAt = m.CreatedAt
	return nil
}

func (r *EmailVerificationTokenRepo) Delete(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Delete(&models.EmailVerificationToken{}, id)
	if result.Error != nil {
		return fmt.Errorf("deleting email verification token: %w", result.Error)
	}
	return nil
}

func (r *EmailVerificationTokenRepo) DeleteByUserID(ctx context.Context, userID uint) error {
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&models.EmailVerificationToken{}).Error; err != nil {
		return fmt.Errorf("deleting email verification tokens for user: %w", err)
	}
	return nil
}

func toDomainEmailVerificationToken(m *models.EmailVerificationToken) *domain.EmailVerificationToken {
	return &domain.EmailVerificationToken{
		ID:        m.ID,
		UserID:    m.UserID,
		TokenHash: m.TokenHash,
		ExpiresAt: m.ExpiresAt,
		CreatedAt: m.CreatedAt,
	}
}

func toModelEmailVerificationToken(d *domain.EmailVerificationToken) *models.EmailVerificationToken {
	return &models.EmailVerificationToken{
		ID:        d.ID,
		UserID:    d.UserID,
		TokenHash: d.TokenHash,
		ExpiresAt: d.ExpiresAt,
	}
}
