package repository

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/veilence/veilence-mx/backend/internal/domain"
	"github.com/veilence/veilence-mx/backend/internal/models"
)

// RoleRepo implements domain.RoleRepository using GORM.
type RoleRepo struct {
	db *gorm.DB
}

// NewRoleRepo creates a new RoleRepo.
func NewRoleRepo(db *gorm.DB) *RoleRepo {
	return &RoleRepo{db: db}
}

func (r *RoleRepo) FindByID(ctx context.Context, id uint) (*domain.Role, error) {
	var m models.Role
	if err := r.db.WithContext(ctx).Preload("Permissions").First(&m, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("role %w", domain.ErrNotFound)
		}
		return nil, fmt.Errorf("finding role: %w", err)
	}
	return roleToDomain(&m), nil
}

func (r *RoleRepo) FindByIDAndOrg(ctx context.Context, id, orgID uint) (*domain.Role, error) {
	var m models.Role
	err := r.db.WithContext(ctx).
		Preload("Permissions").
		Where("id = ? AND org_id = ?", id, orgID).
		First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("role %w", domain.ErrNotFound)
		}
		return nil, fmt.Errorf("finding role: %w", err)
	}
	return roleToDomain(&m), nil
}

func (r *RoleRepo) FindByOrgID(ctx context.Context, orgID uint) ([]domain.Role, error) {
	var ms []models.Role
	err := r.db.WithContext(ctx).
		Preload("Permissions").
		Where("org_id = ?", orgID).
		Find(&ms).Error
	if err != nil {
		return nil, fmt.Errorf("listing roles: %w", err)
	}
	result := make([]domain.Role, len(ms))
	for i := range ms {
		result[i] = *roleToDomain(&ms[i])
	}
	return result, nil
}

func (r *RoleRepo) Create(ctx context.Context, role *domain.Role) error {
	m := roleToModel(role)
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return fmt.Errorf("creating role: %w", err)
	}
	role.ID = m.ID
	role.CreatedAt = m.CreatedAt
	role.UpdatedAt = m.UpdatedAt
	return nil
}

// --- Converters ---

func roleToDomain(m *models.Role) *domain.Role {
	d := &domain.Role{
		ID:          m.ID,
		OrgID:       m.OrgID,
		Name:        m.Name,
		Description: m.Description,
		IsSystem:    m.IsSystem,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}
	if len(m.Permissions) > 0 {
		d.Permissions = make([]domain.Permission, len(m.Permissions))
		for i, p := range m.Permissions {
			d.Permissions[i] = domain.Permission{
				ID:       p.ID,
				Resource: p.Resource,
				Action:   p.Action,
			}
		}
	}
	return d
}

func roleToModel(d *domain.Role) *models.Role {
	m := &models.Role{
		ID:          d.ID,
		OrgID:       d.OrgID,
		Name:        d.Name,
		Description: d.Description,
		IsSystem:    d.IsSystem,
		CreatedAt:   d.CreatedAt,
		UpdatedAt:   d.UpdatedAt,
	}
	if len(d.Permissions) > 0 {
		m.Permissions = make([]models.Permission, len(d.Permissions))
		for i, p := range d.Permissions {
			m.Permissions[i] = models.Permission{
				ID:       p.ID,
				Resource: p.Resource,
				Action:   p.Action,
			}
		}
	}
	return m
}
