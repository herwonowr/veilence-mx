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

func (r *UserRepo) FindByID(ctx context.Context, id string) (*entity.User, error) {
	var m User
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&m).Error; err != nil {
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

func (r *UserRepo) CountAll(ctx context.Context) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&User{}).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("counting all users: %w", err)
	}
	return count, nil
}

func (r *UserRepo) FindAll(ctx context.Context, page, limit int, search string) ([]entity.User, int64, error) {
	var ms []User
	var total int64

	q := r.db.WithContext(ctx).Model(&User{})
	if search != "" {
		pattern := "%" + search + "%"
		q = q.Where("email ILIKE ? OR first_name ILIKE ? OR last_name ILIKE ?", pattern, pattern, pattern)
	}

	if err := q.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("UserRepo.FindAll: counting: %w", err)
	}

	offset := (page - 1) * limit
	if err := q.Order("created_at DESC").Offset(offset).Limit(limit).Find(&ms).Error; err != nil {
		return nil, 0, fmt.Errorf("UserRepo.FindAll: %w", err)
	}

	result := make([]entity.User, len(ms))
	for i := range ms {
		result[i] = *userToDomain(&ms[i])
	}
	return result, total, nil
}

func (r *UserRepo) CountSuperAdmins(ctx context.Context) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&User{}).Where("is_super_admin = ? AND is_active = ?", true, true).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("UserRepo.CountSuperAdmins: %w", err)
	}
	return count, nil
}

// --- Converters ---

func userToDomain(m *User) *entity.User {
	return &entity.User{
		ID:                 m.ID,
		Email:              m.Email,
		PasswordHash:       m.PasswordHash,
		FirstName:          m.FirstName,
		LastName:           m.LastName,
		IsActive:           m.IsActive,
		IsSuperAdmin:       m.IsSuperAdmin,
		DeactivatedAt:      m.DeactivatedAt,
		EmailVerified:      m.EmailVerified,
		MustChangePassword: m.MustChangePassword,
		AuthProvider:       entity.AuthProvider(m.AuthProvider),
		LastLoginAt:        m.LastLoginAt,
		CreatedAt:          m.CreatedAt,
		UpdatedAt:          m.UpdatedAt,
	}
}

func userToModel(d *entity.User) *User {
	return &User{
		ID:                 d.ID,
		Email:              d.Email,
		PasswordHash:       d.PasswordHash,
		FirstName:          d.FirstName,
		LastName:           d.LastName,
		IsActive:           d.IsActive,
		IsSuperAdmin:       d.IsSuperAdmin,
		DeactivatedAt:      d.DeactivatedAt,
		EmailVerified:      d.EmailVerified,
		MustChangePassword: d.MustChangePassword,
		AuthProvider:       string(d.AuthProvider),
		LastLoginAt:        d.LastLoginAt,
		CreatedAt:          d.CreatedAt,
		UpdatedAt:          d.UpdatedAt,
	}
}
