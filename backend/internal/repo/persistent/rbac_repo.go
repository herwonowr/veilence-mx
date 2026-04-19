package persistent

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/veilence/veilence-mx/backend/internal/entity"
	"github.com/veilence/veilence-mx/backend/internal/usecase"
)

// RBACRepo implements rbac.RBACRepository using GORM.
type RBACRepo struct {
	db *gorm.DB
}

// NewRBACRepo creates a new RBACRepo.
func NewRBACRepo(db *gorm.DB) *RBACRepo {
	return &RBACRepo{db: db}
}

// CountWorkspacesBySlug counts workspaces with the given slug, optionally excluding one.
func (r *RBACRepo) CountWorkspacesBySlug(ctx context.Context, slug string, excludeID *uint) (int64, error) {
	var count int64
	q := r.db.WithContext(ctx).Model(&Workspace{}).Where("slug = ?", slug)
	if excludeID != nil {
		q = q.Where("id != ?", *excludeID)
	}
	if err := q.Count(&count).Error; err != nil {
		return 0, fmt.Errorf("RBACRepo.CountWorkspacesBySlug: %w", err)
	}
	return count, nil
}

// CreateWorkspace persists a new workspace.
func (r *RBACRepo) CreateWorkspace(ctx context.Context, ws *entity.Workspace) error {
	model := Workspace{
		Name:        ws.Name,
		Slug:        ws.Slug,
		Description: ws.Description,
		OwnerID:     ws.OwnerID,
		IsActive:    ws.IsActive,
	}
	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		return fmt.Errorf("RBACRepo.CreateWorkspace: %w", err)
	}
	ws.ID = model.ID
	ws.CreatedAt = model.CreatedAt
	ws.UpdatedAt = model.UpdatedAt
	return nil
}

// FindWorkspaceByID returns a workspace by ID.
func (r *RBACRepo) FindWorkspaceByID(ctx context.Context, id uint) (*entity.Workspace, error) {
	var model Workspace
	if err := r.db.WithContext(ctx).First(&model, id).Error; err != nil {
		return nil, fmt.Errorf("RBACRepo.FindWorkspaceByID: %w", err)
	}
	return workspaceModelToEntity(model), nil
}

// UpdateWorkspace updates a workspace.
func (r *RBACRepo) UpdateWorkspace(ctx context.Context, ws *entity.Workspace) error {
	if err := r.db.WithContext(ctx).Model(&Workspace{}).Where("id = ?", ws.ID).Updates(map[string]interface{}{
		"name":        ws.Name,
		"slug":        ws.Slug,
		"description": ws.Description,
	}).Error; err != nil {
		return fmt.Errorf("RBACRepo.UpdateWorkspace: %w", err)
	}
	return nil
}

// SoftDeleteWorkspace soft-deletes a workspace.
func (r *RBACRepo) SoftDeleteWorkspace(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Delete(&Workspace{}, id)
	if result.Error != nil {
		return fmt.Errorf("RBACRepo.SoftDeleteWorkspace: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("workspace not found")
	}
	return nil
}

// FindWorkspacesByUserID returns all workspaces the user is a member of.
func (r *RBACRepo) FindWorkspacesByUserID(ctx context.Context, userID uint) ([]entity.Workspace, error) {
	var models []Workspace
	err := r.db.WithContext(ctx).
		Joins("JOIN workspace_members ON workspace_members.workspace_id = workspaces.id").
		Where("workspace_members.user_id = ? AND workspaces.deleted_at IS NULL", userID).
		Find(&models).Error
	if err != nil {
		return nil, fmt.Errorf("RBACRepo.FindWorkspacesByUserID: %w", err)
	}

	result := make([]entity.Workspace, len(models))
	for i, m := range models {
		result[i] = *workspaceModelToEntity(m)
	}
	return result, nil
}

// FindAllPermissions returns all system permissions.
func (r *RBACRepo) FindAllPermissions(ctx context.Context) ([]entity.Permission, error) {
	var models []Permission
	if err := r.db.WithContext(ctx).Find(&models).Error; err != nil {
		return nil, fmt.Errorf("RBACRepo.FindAllPermissions: %w", err)
	}

	perms := make([]entity.Permission, len(models))
	for i, m := range models {
		perms[i] = entity.Permission{ID: m.ID, Resource: m.Resource, Action: m.Action}
	}
	return perms, nil
}

// CreateRole persists a new role with its permissions.
func (r *RBACRepo) CreateRole(ctx context.Context, role *entity.Role) error {
	// Convert entity permissions to persistent permissions for the association
	var permModels []Permission
	for _, p := range role.Permissions {
		permModels = append(permModels, Permission{ID: p.ID, Resource: p.Resource, Action: p.Action})
	}

	model := Role{
		WorkspaceID: role.WorkspaceID,
		Name:        role.Name,
		Description: role.Description,
		IsSystem:    role.IsSystem,
		Permissions: permModels,
	}
	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		return fmt.Errorf("RBACRepo.CreateRole: %w", err)
	}
	role.ID = model.ID
	role.CreatedAt = model.CreatedAt
	role.UpdatedAt = model.UpdatedAt
	return nil
}

// FindRoleByIDAndWorkspace returns a role by ID scoped to a workspace.
func (r *RBACRepo) FindRoleByIDAndWorkspace(ctx context.Context, roleID, workspaceID uint) (*entity.Role, error) {
	var model Role
	if err := r.db.WithContext(ctx).Where("id = ? AND workspace_id = ?", roleID, workspaceID).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("RBACRepo.FindRoleByIDAndWorkspace: %w", err)
	}
	return roleModelToEntity(model), nil
}

// FindRolesByWorkspaceID returns all roles for a workspace with permissions.
func (r *RBACRepo) FindRolesByWorkspaceID(ctx context.Context, workspaceID uint) ([]entity.Role, error) {
	var models []Role
	err := r.db.WithContext(ctx).
		Preload("Permissions").
		Where("workspace_id = ?", workspaceID).
		Find(&models).Error
	if err != nil {
		return nil, fmt.Errorf("RBACRepo.FindRolesByWorkspaceID: %w", err)
	}

	roles := make([]entity.Role, len(models))
	for i, m := range models {
		roles[i] = *roleModelToEntity(m)
	}
	return roles, nil
}

// CreateMember persists a new workspace member.
func (r *RBACRepo) CreateMember(ctx context.Context, member *entity.WorkspaceMember) error {
	model := WorkspaceMember{
		WorkspaceID: member.WorkspaceID,
		UserID:      member.UserID,
		RoleID:      member.RoleID,
		JoinedAt:    member.JoinedAt,
	}
	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		return fmt.Errorf("RBACRepo.CreateMember: %w", err)
	}
	member.ID = model.ID
	member.CreatedAt = model.CreatedAt
	return nil
}

// FindMembersByWorkspaceID returns all members with their roles and user data.
func (r *RBACRepo) FindMembersByWorkspaceID(ctx context.Context, workspaceID uint) ([]entity.WorkspaceMember, error) {
	var models []WorkspaceMember
	err := r.db.WithContext(ctx).
		Preload("Role").
		Preload("Role.Permissions").
		Preload("User").
		Where("workspace_id = ?", workspaceID).
		Find(&models).Error
	if err != nil {
		return nil, fmt.Errorf("RBACRepo.FindMembersByWorkspaceID: %w", err)
	}

	members := make([]entity.WorkspaceMember, len(models))
	for i, m := range models {
		members[i] = memberModelToEntity(m)
	}
	return members, nil
}

// FindMemberByUserAndWorkspace returns a member by user and workspace.
func (r *RBACRepo) FindMemberByUserAndWorkspace(ctx context.Context, userID, workspaceID uint) (*entity.WorkspaceMember, error) {
	var model WorkspaceMember
	err := r.db.WithContext(ctx).
		Preload("Role").
		Preload("Role.Permissions").
		Where("user_id = ? AND workspace_id = ?", userID, workspaceID).
		First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("RBACRepo.FindMemberByUserAndWorkspace: %w", err)
	}
	m := memberModelToEntity(model)
	return &m, nil
}

// CountMembersByUserAndWorkspace counts members.
func (r *RBACRepo) CountMembersByUserAndWorkspace(ctx context.Context, userID, workspaceID uint) (int64, error) {
	var count int64
	r.db.WithContext(ctx).Model(&WorkspaceMember{}).
		Where("workspace_id = ? AND user_id = ?", workspaceID, userID).
		Count(&count)
	return count, nil
}

// UpdateMember updates a member record.
func (r *RBACRepo) UpdateMember(ctx context.Context, member *entity.WorkspaceMember) error {
	if err := r.db.WithContext(ctx).Model(&WorkspaceMember{}).Where("id = ?", member.ID).Update("role_id", member.RoleID).Error; err != nil {
		return fmt.Errorf("RBACRepo.UpdateMember: %w", err)
	}
	return nil
}

// DeleteMemberByUserAndWorkspace removes a member.
func (r *RBACRepo) DeleteMemberByUserAndWorkspace(ctx context.Context, userID, workspaceID uint) error {
	result := r.db.WithContext(ctx).Where("workspace_id = ? AND user_id = ?", workspaceID, userID).Delete(&WorkspaceMember{})
	if result.Error != nil {
		return fmt.Errorf("RBACRepo.DeleteMemberByUserAndWorkspace: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("member not found")
	}
	return nil
}

// CreateInvitation persists a new invitation.
func (r *RBACRepo) CreateInvitation(ctx context.Context, invitation *entity.Invitation) error {
	model := Invitation{
		WorkspaceID: invitation.WorkspaceID,
		Email:       invitation.Email,
		RoleID:      invitation.RoleID,
		TokenHash:   invitation.TokenHash,
		InvitedBy:   invitation.InvitedBy,
		ExpiresAt:   invitation.ExpiresAt,
	}
	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		return fmt.Errorf("RBACRepo.CreateInvitation: %w", err)
	}
	invitation.ID = model.ID
	invitation.CreatedAt = model.CreatedAt
	return nil
}

// FindInvitationByTokenHash returns an invitation by token hash.
func (r *RBACRepo) FindInvitationByTokenHash(ctx context.Context, tokenHash string) (*entity.Invitation, error) {
	var model Invitation
	if err := r.db.WithContext(ctx).Where("token_hash = ?", tokenHash).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("RBACRepo.FindInvitationByTokenHash: %w", err)
	}
	return invitationModelToEntity(model), nil
}

// UpdateInvitation updates an invitation.
func (r *RBACRepo) UpdateInvitation(ctx context.Context, invitation *entity.Invitation) error {
	if err := r.db.WithContext(ctx).Model(&Invitation{}).Where("id = ?", invitation.ID).Updates(map[string]interface{}{
		"accepted_at": invitation.AcceptedAt,
	}).Error; err != nil {
		return fmt.Errorf("RBACRepo.UpdateInvitation: %w", err)
	}
	return nil
}

// FindPendingInvitations returns pending invitations for a workspace.
func (r *RBACRepo) FindPendingInvitations(ctx context.Context, workspaceID uint) ([]entity.Invitation, error) {
	var models []Invitation
	err := r.db.WithContext(ctx).
		Where("workspace_id = ? AND accepted_at IS NULL AND expires_at > ?", workspaceID, time.Now()).
		Order("created_at DESC").
		Find(&models).Error
	if err != nil {
		return nil, fmt.Errorf("RBACRepo.FindPendingInvitations: %w", err)
	}

	invitations := make([]entity.Invitation, len(models))
	for i, m := range models {
		invitations[i] = *invitationModelToEntity(m)
	}
	return invitations, nil
}

// DeletePendingInvitation deletes a pending invitation.
func (r *RBACRepo) DeletePendingInvitation(ctx context.Context, workspaceID, invitationID uint) error {
	result := r.db.WithContext(ctx).
		Where("id = ? AND workspace_id = ? AND accepted_at IS NULL", invitationID, workspaceID).
		Delete(&Invitation{})
	if result.Error != nil {
		return fmt.Errorf("RBACRepo.DeletePendingInvitation: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("invitation not found")
	}
	return nil
}

// CheckUserPermission checks if a user has a permission in a workspace.
func (r *RBACRepo) CheckUserPermission(ctx context.Context, userID, workspaceID uint, resource, action string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&Permission{}).
		Joins("JOIN role_permissions ON role_permissions.permission_id = permissions.id").
		Joins("JOIN roles ON roles.id = role_permissions.role_id").
		Joins("JOIN workspace_members ON workspace_members.role_id = roles.id").
		Where("workspace_members.user_id = ? AND workspace_members.workspace_id = ? AND permissions.resource = ? AND permissions.action = ?",
			userID, workspaceID, resource, action).
		Count(&count).Error
	if err != nil {
		return false, fmt.Errorf("RBACRepo.CheckUserPermission: %w", err)
	}
	return count > 0, nil
}

// CheckRolePermission checks if a role has a permission in a workspace.
func (r *RBACRepo) CheckRolePermission(ctx context.Context, workspaceID uint, roleName, resource, action string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&Permission{}).
		Joins("JOIN role_permissions ON role_permissions.permission_id = permissions.id").
		Joins("JOIN roles ON roles.id = role_permissions.role_id").
		Where("roles.workspace_id = ? AND roles.name = ? AND permissions.resource = ? AND permissions.action = ?",
			workspaceID, roleName, resource, action).
		Count(&count).Error
	if err != nil {
		return false, fmt.Errorf("RBACRepo.CheckRolePermission: %w", err)
	}
	return count > 0, nil
}

// SeedPermission ensures a permission exists.
func (r *RBACRepo) SeedPermission(ctx context.Context, perm entity.Permission) error {
	model := Permission{Resource: perm.Resource, Action: perm.Action}
	result := r.db.WithContext(ctx).Where("resource = ? AND action = ?", perm.Resource, perm.Action).FirstOrCreate(&model)
	if result.Error != nil {
		return fmt.Errorf("RBACRepo.SeedPermission: %w", result.Error)
	}
	return nil
}

// WithTransaction runs fn within a database transaction.
func (r *RBACRepo) WithTransaction(ctx context.Context, fn func(tx usecase.RBACRepository) error) error {
	return r.db.WithContext(ctx).Transaction(func(gormTx *gorm.DB) error {
		txRepo := &RBACRepo{db: gormTx}
		return fn(txRepo)
	})
}

// --- Conversion helpers ---

func workspaceModelToEntity(m Workspace) *entity.Workspace {
	return &entity.Workspace{
		ID:          m.ID,
		Name:        m.Name,
		Slug:        m.Slug,
		Description: m.Description,
		OwnerID:     m.OwnerID,
		IsActive:    m.IsActive,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}
}

func roleModelToEntity(m Role) *entity.Role {
	var perms []entity.Permission
	for _, p := range m.Permissions {
		perms = append(perms, entity.Permission{ID: p.ID, Resource: p.Resource, Action: p.Action})
	}
	return &entity.Role{
		ID:          m.ID,
		WorkspaceID: m.WorkspaceID,
		Name:        m.Name,
		Description: m.Description,
		IsSystem:    m.IsSystem,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
		Permissions: perms,
	}
}

func memberModelToEntity(m WorkspaceMember) entity.WorkspaceMember {
	member := entity.WorkspaceMember{
		ID:          m.ID,
		WorkspaceID: m.WorkspaceID,
		UserID:      m.UserID,
		RoleID:      m.RoleID,
		JoinedAt:    m.JoinedAt,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}
	// Convert role
	if m.Role.ID > 0 {
		role := roleModelToEntity(m.Role)
		member.Role = role
	}
	// Convert user
	if m.User.ID > 0 {
		member.User = &entity.User{
			ID:            m.User.ID,
			Email:         m.User.Email,
			FirstName:     m.User.FirstName,
			LastName:      m.User.LastName,
			IsActive:      m.User.IsActive,
			EmailVerified: m.User.EmailVerified,
		}
	}
	return member
}

func invitationModelToEntity(m Invitation) *entity.Invitation {
	return &entity.Invitation{
		ID:          m.ID,
		WorkspaceID: m.WorkspaceID,
		Email:       m.Email,
		RoleID:      m.RoleID,
		TokenHash:   m.TokenHash,
		InvitedBy:   m.InvitedBy,
		ExpiresAt:   m.ExpiresAt,
		AcceptedAt:  m.AcceptedAt,
		CreatedAt:   m.CreatedAt,
	}
}
