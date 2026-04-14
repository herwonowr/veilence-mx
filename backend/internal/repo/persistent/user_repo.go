package persistent

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/veilence/veilence-mx/backend/internal/entity"
)

// UserRepo implements entity.UserRepository using GORM.
type UserRepo struct {
	db *gorm.DB
}

// NewUserRepo creates a new UserRepo.
func NewUserRepo(db *gorm.DB) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) FindByID(ctx context.Context, id uint) (*entity.User, error) {
	var m User
	if err := r.db.WithContext(ctx).First(&m, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("user %w", entity.ErrNotFound)
		}
		return nil, fmt.Errorf("finding user by id: %w", err)
	}
	return userToDomain(&m), nil
}

func (r *UserRepo) FindByEmail(ctx context.Context, email string) (*entity.User, error) {
	var m User
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("user %w", entity.ErrNotFound)
		}
		return nil, fmt.Errorf("finding user by email: %w", err)
	}
	return userToDomain(&m), nil
}

func (r *UserRepo) Create(ctx context.Context, user *entity.User) error {
	m := userToModel(user)
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return fmt.Errorf("creating user: %w", err)
	}
	user.ID = m.ID
	user.CreatedAt = m.CreatedAt
	user.UpdatedAt = m.UpdatedAt
	return nil
}

func (r *UserRepo) Update(ctx context.Context, user *entity.User) error {
	m := userToModel(user)
	if err := r.db.WithContext(ctx).Save(m).Error; err != nil {
		return fmt.Errorf("updating user: %w", err)
	}
	user.UpdatedAt = m.UpdatedAt
	return nil
}

// --- Converters ---

func userToDomain(m *User) *entity.User {
	return &entity.User{
		ID:            m.ID,
		Email:         m.Email,
		PasswordHash:  m.PasswordHash,
		FirstName:     m.FirstName,
		LastName:      m.LastName,
		IsActive:      m.IsActive,
		EmailVerified: m.EmailVerified,
		LastLoginAt:   m.LastLoginAt,
		CreatedAt:     m.CreatedAt,
		UpdatedAt:     m.UpdatedAt,
	}
}

func userToModel(d *entity.User) *User {
	return &User{
		ID:            d.ID,
		Email:         d.Email,
		PasswordHash:  d.PasswordHash,
		FirstName:     d.FirstName,
		LastName:      d.LastName,
		IsActive:      d.IsActive,
		EmailVerified: d.EmailVerified,
		LastLoginAt:   d.LastLoginAt,
		CreatedAt:     d.CreatedAt,
		UpdatedAt:     d.UpdatedAt,
	}
}
