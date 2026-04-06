package repository

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/veilence/veilence-mx/backend/internal/domain"
	"github.com/veilence/veilence-mx/backend/internal/models"
)

// PermissionRepo implements domain.PermissionRepository using GORM.
type PermissionRepo struct {
	db *gorm.DB
}

// NewPermissionRepo creates a new PermissionRepo.
func NewPermissionRepo(db *gorm.DB) *PermissionRepo {
	return &PermissionRepo{db: db}
}

func (r *PermissionRepo) FindAll(ctx context.Context) ([]domain.Permission, error) {
	var ms []models.Permission
	if err := r.db.WithContext(ctx).Find(&ms).Error; err != nil {
		return nil, fmt.Errorf("listing permissions: %w", err)
	}
	result := make([]domain.Permission, len(ms))
	for i := range ms {
		result[i] = domain.Permission{
			ID:       ms[i].ID,
			Resource: ms[i].Resource,
			Action:   ms[i].Action,
		}
	}
	return result, nil
}

func (r *PermissionRepo) FindOrCreate(ctx context.Context, perm *domain.Permission) error {
	m := &models.Permission{
		Resource: perm.Resource,
		Action:   perm.Action,
	}
	result := r.db.WithContext(ctx).
		Where("resource = ? AND action = ?", m.Resource, m.Action).
		FirstOrCreate(m)
	if result.Error != nil {
		return fmt.Errorf("seeding permission %s:%s: %w", perm.Resource, perm.Action, result.Error)
	}
	perm.ID = m.ID
	return nil
}

func (r *PermissionRepo) CheckUserPermission(ctx context.Context, userID, orgID uint, resource, action string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.Permission{}).
		Joins("JOIN role_permissions ON role_permissions.permission_id = permissions.id").
		Joins("JOIN roles ON roles.id = role_permissions.role_id").
		Joins("JOIN org_members ON org_members.role_id = roles.id").
		Where("org_members.user_id = ? AND org_members.org_id = ? AND permissions.resource = ? AND permissions.action = ?",
			userID, orgID, resource, action).
		Count(&count).Error
	if err != nil {
		return false, fmt.Errorf("checking permission: %w", err)
	}
	return count > 0, nil
}
