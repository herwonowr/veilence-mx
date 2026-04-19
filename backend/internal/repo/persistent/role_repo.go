package persistent

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/veilence/veilence-mx/backend/internal/entity"
)

// RoleRepo implements entity.RoleRepository using GORM.
type RoleRepo struct {
	db *gorm.DB
}

// NewRoleRepo creates a new RoleRepo.
func NewRoleRepo(db *gorm.DB) *RoleRepo {
	return &RoleRepo{db: db}
}

func (r *RoleRepo) FindByID(ctx context.Context, id uint) (*entity.Role, error) {
	var m Role
	if err := r.db.WithContext(ctx).Preload("Permissions").First(&m, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("role %w", entity.ErrNotFound)
		}
		return nil, fmt.Errorf("finding role: %w", err)
	}
	return roleToDomain(&m), nil
}

func (r *RoleRepo) FindByIDAndWorkspace(ctx context.Context, id, workspaceID uint) (*entity.Role, error) {
	var m Role
	err := r.db.WithContext(ctx).
		Preload("Permissions").
		Where("id = ? AND workspace_id = ?", id, workspaceID).
		First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("role %w", entity.ErrNotFound)
		}
		return nil, fmt.Errorf("finding role: %w", err)
	}
	return roleToDomain(&m), nil
}

func (r *RoleRepo) FindByWorkspaceID(ctx context.Context, workspaceID uint) ([]entity.Role, error) {
	var ms []Role
	err := r.db.WithContext(ctx).
		Preload("Permissions").
		Where("workspace_id = ?", workspaceID).
		Find(&ms).Error
	if err != nil {
		return nil, fmt.Errorf("listing roles: %w", err)
	}
	result := make([]entity.Role, len(ms))
	for i := range ms {
		result[i] = *roleToDomain(&ms[i])
	}
	return result, nil
}

func (r *RoleRepo) Create(ctx context.Context, role *entity.Role) error {
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

func roleToDomain(m *Role) *entity.Role {
	d := &entity.Role{
		ID:          m.ID,
		WorkspaceID:       m.WorkspaceID,
		Name:        m.Name,
		Description: m.Description,
		IsSystem:    m.IsSystem,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}
	if len(m.Permissions) > 0 {
		d.Permissions = make([]entity.Permission, len(m.Permissions))
		for i, p := range m.Permissions {
			d.Permissions[i] = entity.Permission{
				ID:       p.ID,
				Resource: p.Resource,
				Action:   p.Action,
			}
		}
	}
	return d
}

func roleToModel(d *entity.Role) *Role {
	m := &Role{
		ID:          d.ID,
		WorkspaceID:       d.WorkspaceID,
		Name:        d.Name,
		Description: d.Description,
		IsSystem:    d.IsSystem,
		CreatedAt:   d.CreatedAt,
		UpdatedAt:   d.UpdatedAt,
	}
	if len(d.Permissions) > 0 {
		m.Permissions = make([]Permission, len(d.Permissions))
		for i, p := range d.Permissions {
			m.Permissions[i] = Permission{
				ID:       p.ID,
				Resource: p.Resource,
				Action:   p.Action,
			}
		}
	}
	return m
}
