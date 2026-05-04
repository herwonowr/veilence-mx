package persistent

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/veilence/veilence-mx/backend/internal/entity"
)

// UserIdentityRepo implements usecase.UserIdentityRepository using GORM.
type UserIdentityRepo struct {
	db *gorm.DB
}

// NewUserIdentityRepo creates a new UserIdentityRepo.
func NewUserIdentityRepo(db *gorm.DB) *UserIdentityRepo {
	return &UserIdentityRepo{db: db}
}

func (r *UserIdentityRepo) FindByUserID(ctx context.Context, userID string) ([]entity.UserIdentity, error) {
	var ms []UserIdentity
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("created_at DESC").Find(&ms).Error; err != nil {
		return nil, fmt.Errorf("UserIdentityRepo.FindByUserID: %w", err)
	}
	result := make([]entity.UserIdentity, len(ms))
	for i := range ms {
		result[i] = *userIdentityToDomain(&ms[i])
	}
	return result, nil
}

func (r *UserIdentityRepo) FindByProviderAndProviderUserID(ctx context.Context, provider entity.AuthProvider, providerUserID string) (*entity.UserIdentity, error) {
	var m UserIdentity
	if err := r.db.WithContext(ctx).Where("provider = ? AND provider_user_id = ?", string(provider), providerUserID).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("user identity %w", entity.ErrNotFound)
		}
		return nil, fmt.Errorf("UserIdentityRepo.FindByProviderAndProviderUserID: %w", err)
	}
	return userIdentityToDomain(&m), nil
}

func (r *UserIdentityRepo) Create(ctx context.Context, identity *entity.UserIdentity) error {
	m := userIdentityToModel(identity)
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return fmt.Errorf("UserIdentityRepo.Create: %w", err)
	}
	identity.ID = m.ID
	identity.CreatedAt = m.CreatedAt
	identity.UpdatedAt = m.UpdatedAt
	return nil
}

func (r *UserIdentityRepo) Update(ctx context.Context, identity *entity.UserIdentity) error {
	m := userIdentityToModel(identity)
	if err := r.db.WithContext(ctx).Save(m).Error; err != nil {
		return fmt.Errorf("UserIdentityRepo.Update: %w", err)
	}
	identity.UpdatedAt = m.UpdatedAt
	return nil
}

func (r *UserIdentityRepo) Delete(ctx context.Context, id string) error {
	if err := r.db.WithContext(ctx).Where("id = ?", id).Delete(&UserIdentity{}).Error; err != nil {
		return fmt.Errorf("UserIdentityRepo.Delete: %w", err)
	}
	return nil
}

func (r *UserIdentityRepo) DeleteByUserID(ctx context.Context, userID string) error {
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&UserIdentity{}).Error; err != nil {
		return fmt.Errorf("UserIdentityRepo.DeleteByUserID: %w", err)
	}
	return nil
}

// --- Converters ---

func userIdentityToDomain(m *UserIdentity) *entity.UserIdentity {
	return &entity.UserIdentity{
		ID:             m.ID,
		UserID:         m.UserID,
		Provider:       entity.AuthProvider(m.Provider),
		ProviderUserID: m.ProviderUserID,
		Email:          m.Email,
		CreatedAt:      m.CreatedAt,
		UpdatedAt:      m.UpdatedAt,
	}
}

func userIdentityToModel(d *entity.UserIdentity) *UserIdentity {
	return &UserIdentity{
		ID:             d.ID,
		UserID:         d.UserID,
		Provider:       string(d.Provider),
		ProviderUserID: d.ProviderUserID,
		Email:          d.Email,
		CreatedAt:      d.CreatedAt,
		UpdatedAt:      d.UpdatedAt,
	}
}
