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
func (r *RBACRepo) CountWorkspacesBySlug(ctx context.Context, slug string, excludeID *string) (int64, error) {
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
func (r *RBACRepo) FindWorkspaceByID(ctx context.Context, id string) (*entity.Workspace, error) {
	var model Workspace
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&model).Error; err != nil {
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

// SoftDeleteWorkspace soft-deletes a workspace and cascade-deletes all related data.
// Deletes are performed leaf-first to respect foreign key constraints.
func (r *RBACRepo) SoftDeleteWorkspace(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Verify workspace exists
		var ws Workspace
		if err := tx.Where("id = ?", id).First(&ws).Error; err != nil {
			return fmt.Errorf("workspace not found")
		}

		// Collect package IDs for this workspace
		var pkgIDs []string
		tx.Model(&Package{}).Where("workspace_id = ?", id).Pluck("id", &pkgIDs)

		if len(pkgIDs) > 0 {
			// Collect release IDs for these packages
			var releaseIDs []string
			tx.Model(&Release{}).Where("package_id IN ?", pkgIDs).Pluck("id", &releaseIDs)

			// Collect diff IDs for these releases
			var diffIDs []string
			if len(releaseIDs) > 0 {
				tx.Model(&Diff{}).Where("release_id IN ?", releaseIDs).Pluck("id", &diffIDs)
			}

			// Alert notes (depend on alerts)
			if err := tx.Where("workspace_id = ?", id).Delete(&AlertNote{}).Error; err != nil {
				return fmt.Errorf("RBACRepo.SoftDeleteWorkspace: deleting alert notes: %w", err)
			}

			// Alerts (depend on packages)
			if err := tx.Where("workspace_id = ?", id).Delete(&Alert{}).Error; err != nil {
				return fmt.Errorf("RBACRepo.SoftDeleteWorkspace: deleting alerts: %w", err)
			}

			// Analyses (depend on diffs)
			if len(diffIDs) > 0 {
				if err := tx.Where("diff_id IN ?", diffIDs).Delete(&Analysis{}).Error; err != nil {
					return fmt.Errorf("RBACRepo.SoftDeleteWorkspace: deleting analyses: %w", err)
				}
			}

			// Diffs (depend on releases)
			if len(releaseIDs) > 0 {
				if err := tx.Where("release_id IN ?", releaseIDs).Delete(&Diff{}).Error; err != nil {
					return fmt.Errorf("RBACRepo.SoftDeleteWorkspace: deleting diffs: %w", err)
				}
			}

			// Releases (depend on packages)
			if err := tx.Where("package_id IN ?", pkgIDs).Delete(&Release{}).Error; err != nil {
				return fmt.Errorf("RBACRepo.SoftDeleteWorkspace: deleting releases: %w", err)
			}

			// Packages
			if err := tx.Where("workspace_id = ?", id).Delete(&Package{}).Error; err != nil {
				return fmt.Errorf("RBACRepo.SoftDeleteWorkspace: deleting packages: %w", err)
			}
		}

		// Notification rules (depend on channels)
		if err := tx.Where("workspace_id = ?", id).Delete(&NotificationRule{}).Error; err != nil {
			return fmt.Errorf("RBACRepo.SoftDeleteWorkspace: deleting notification rules: %w", err)
		}

		// Notification channels
		if err := tx.Where("workspace_id = ?", id).Delete(&NotificationChannel{}).Error; err != nil {
			return fmt.Errorf("RBACRepo.SoftDeleteWorkspace: deleting notification channels: %w", err)
		}

		// Notifications
		if err := tx.Where("workspace_id = ?", id).Delete(&Notification{}).Error; err != nil {
			return fmt.Errorf("RBACRepo.SoftDeleteWorkspace: deleting notifications: %w", err)
		}

		// Settings
		if err := tx.Where("workspace_id = ?", id).Delete(&Setting{}).Error; err != nil {
			return fmt.Errorf("RBACRepo.SoftDeleteWorkspace: deleting settings: %w", err)
		}

		// Audit logs
		if err := tx.Where("workspace_id = ?", id).Delete(&AuditLog{}).Error; err != nil {
			return fmt.Errorf("RBACRepo.SoftDeleteWorkspace: deleting audit logs: %w", err)
		}

		// Collect role IDs for this workspace
		var roleIDs []string
		tx.Model(&Role{}).Where("workspace_id = ?", id).Pluck("id", &roleIDs)

		// Role permissions (join table)
		if len(roleIDs) > 0 {
			if err := tx.Exec("DELETE FROM role_permissions WHERE role_id IN ?", roleIDs).Error; err != nil {
				return fmt.Errorf("RBACRepo.SoftDeleteWorkspace: deleting role permissions: %w", err)
			}
		}

		// Workspace members (depend on workspace + roles)
		if err := tx.Where("workspace_id = ?", id).Delete(&WorkspaceMember{}).Error; err != nil {
			return fmt.Errorf("RBACRepo.SoftDeleteWorkspace: deleting workspace members: %w", err)
		}

		// Invitations
		if err := tx.Where("workspace_id = ?", id).Delete(&Invitation{}).Error; err != nil {
			return fmt.Errorf("RBACRepo.SoftDeleteWorkspace: deleting invitations: %w", err)
		}

		// Roles
		if len(roleIDs) > 0 {
			if err := tx.Where("id IN ?", roleIDs).Delete(&Role{}).Error; err != nil {
				return fmt.Errorf("RBACRepo.SoftDeleteWorkspace: deleting roles: %w", err)
			}
		}

		// API keys scoped to workspace
		if err := tx.Where("workspace_id = ?", id).Delete(&APIKey{}).Error; err != nil {
			return fmt.Errorf("RBACRepo.SoftDeleteWorkspace: deleting api keys: %w", err)
		}

		// Finally, soft-delete the workspace itself
		result := tx.Where("id = ?", id).Delete(&Workspace{})
		if result.Error != nil {
			return fmt.Errorf("RBACRepo.SoftDeleteWorkspace: %w", result.Error)
		}
		if result.RowsAffected == 0 {
			return fmt.Errorf("workspace not found")
		}
		return nil
	})
}

// FindWorkspacesByUserID returns a paginated, optionally filtered list
// of workspaces the user is a member of.
func (r *RBACRepo) FindWorkspacesByUserID(ctx context.Context, userID string, params entity.WorkspaceListParams) (*entity.WorkspaceListResult, error) {
	q := r.db.WithContext(ctx).Model(&Workspace{}).
		Joins("JOIN workspace_members ON workspace_members.workspace_id = workspaces.id").
		Joins("JOIN roles ON roles.id = workspace_members.role_id").
		Where("workspace_members.user_id = ? AND workspaces.deleted_at IS NULL", userID)

	if params.Search != "" {
		like := "%" + params.Search + "%"
		q = q.Where("(workspaces.name ILIKE ? OR workspaces.slug ILIKE ?)", like, like)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("RBACRepo.FindWorkspacesByUserID: count: %w", err)
	}

	offset := (params.Page - 1) * params.Limit
	// Local struct to capture the joined role name - GORM ignores gorm:"-" fields during scan
	type result struct {
		Workspace
		RoleName string
	}
	var models []result
	if err := q.Select("workspaces.*, roles.name as role_name").
		Order("workspaces.name ASC").Offset(offset).Limit(params.Limit).
		Find(&models).Error; err != nil {
		return nil, fmt.Errorf("RBACRepo.FindWorkspacesByUserID: query: %w", err)
	}

	workspaces := make([]entity.Workspace, len(models))
	for i, m := range models {
		ws := workspaceModelToEntity(m.Workspace)
		ws.Role = m.RoleName
		workspaces[i] = *ws
	}
	return &entity.WorkspaceListResult{Workspaces: workspaces, Total: total}, nil
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
func (r *RBACRepo) FindRoleByIDAndWorkspace(ctx context.Context, roleID, workspaceID string) (*entity.Role, error) {
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
func (r *RBACRepo) FindRolesByWorkspaceID(ctx context.Context, workspaceID string) ([]entity.Role, error) {
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
func (r *RBACRepo) FindMembersByWorkspaceID(ctx context.Context, workspaceID string) ([]entity.WorkspaceMember, error) {
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
func (r *RBACRepo) FindMemberByUserAndWorkspace(ctx context.Context, userID, workspaceID string) (*entity.WorkspaceMember, error) {
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
func (r *RBACRepo) CountMembersByUserAndWorkspace(ctx context.Context, userID, workspaceID string) (int64, error) {
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
func (r *RBACRepo) DeleteMemberByUserAndWorkspace(ctx context.Context, userID, workspaceID string) error {
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
	if err := r.db.WithContext(ctx).Model(&Invitation{}).Where("id = ? AND workspace_id = ?", invitation.ID, invitation.WorkspaceID).Updates(map[string]interface{}{
		"accepted_at": invitation.AcceptedAt,
		"declined_at": invitation.DeclinedAt,
		"token_hash":  invitation.TokenHash,
		"expires_at":  invitation.ExpiresAt,
	}).Error; err != nil {
		return fmt.Errorf("RBACRepo.UpdateInvitation: %w", err)
	}
	return nil
}

// FindInvitationByID returns a single invitation by ID scoped to a workspace.
func (r *RBACRepo) FindInvitationByID(ctx context.Context, workspaceID, invitationID string) (*entity.Invitation, error) {
	var m Invitation
	if err := r.db.WithContext(ctx).
		Where("id = ? AND workspace_id = ?", invitationID, workspaceID).
		First(&m).Error; err != nil {
		return nil, fmt.Errorf("RBACRepo.FindInvitationByID: %w", err)
	}
	return invitationModelToEntity(m), nil
}

// FindPendingInvitations returns pending invitations for a workspace.
func (r *RBACRepo) FindPendingInvitations(ctx context.Context, workspaceID string) ([]entity.Invitation, error) {
	var models []Invitation
	err := r.db.WithContext(ctx).
		Where("workspace_id = ? AND accepted_at IS NULL AND declined_at IS NULL AND expires_at > ?", workspaceID, time.Now()).
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
func (r *RBACRepo) DeletePendingInvitation(ctx context.Context, workspaceID, invitationID string) error {
	result := r.db.WithContext(ctx).
		Where("id = ? AND workspace_id = ? AND accepted_at IS NULL AND declined_at IS NULL", invitationID, workspaceID).
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
func (r *RBACRepo) CheckUserPermission(ctx context.Context, userID, workspaceID string, resource, action string) (bool, error) {
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
func (r *RBACRepo) CheckRolePermission(ctx context.Context, workspaceID string, roleName, resource, action string) (bool, error) {
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

// FindInvitationByIDGlobal returns an invitation by ID without workspace scoping.
func (r *RBACRepo) FindInvitationByIDGlobal(ctx context.Context, invitationID string) (*entity.Invitation, error) {
	var m Invitation
	if err := r.db.WithContext(ctx).Where("id = ?", invitationID).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("RBACRepo.FindInvitationByIDGlobal: %w", err)
	}
	return invitationModelToEntity(m), nil
}

// FindPendingInvitationsByEmail returns all pending (not accepted, not declined,
// not expired) invitations for a given email address, with workspace name and
// inviter email joined.
func (r *RBACRepo) FindPendingInvitationsByEmail(ctx context.Context, email string) ([]entity.Invitation, error) {
	var models []Invitation
	err := r.db.WithContext(ctx).
		Where("invitations.email = ? AND invitations.accepted_at IS NULL AND invitations.declined_at IS NULL AND invitations.expires_at > ?", email, time.Now()).
		Order("invitations.created_at DESC").
		Find(&models).Error
	if err != nil {
		return nil, fmt.Errorf("RBACRepo.FindPendingInvitationsByEmail: %w", err)
	}

	// Collect workspace IDs and inviter IDs for batch lookups
	wsIDs := make([]string, 0, len(models))
	inviterIDs := make([]string, 0, len(models))
	for _, m := range models {
		wsIDs = append(wsIDs, m.WorkspaceID)
		inviterIDs = append(inviterIDs, m.InvitedBy)
	}

	// Batch load workspace names
	wsNameMap := make(map[string]string)
	if len(wsIDs) > 0 {
		var workspaces []Workspace
		r.db.WithContext(ctx).Where("id IN ?", wsIDs).Find(&workspaces)
		for _, ws := range workspaces {
			wsNameMap[ws.ID] = ws.Name
		}
	}

	// Batch load inviter emails
	inviterEmailMap := make(map[string]string)
	if len(inviterIDs) > 0 {
		var users []User
		r.db.WithContext(ctx).Where("id IN ?", inviterIDs).Find(&users)
		for _, u := range users {
			inviterEmailMap[u.ID] = u.Email
		}
	}

	invitations := make([]entity.Invitation, len(models))
	for i, m := range models {
		inv := invitationModelToEntity(m)
		inv.WorkspaceName = wsNameMap[m.WorkspaceID]
		inv.InvitedByEmail = inviterEmailMap[m.InvitedBy]
		invitations[i] = *inv
	}
	return invitations, nil
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
	if m.Role.ID != "" {
		role := roleModelToEntity(m.Role)
		member.Role = role
	}
	// Convert user
	if m.User.ID != "" {
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

// FindWorkspaceIDsByUserID returns the workspace IDs the user is a member of.
// Implements usecase.UserWorkspaceLister.
func (r *RBACRepo) FindWorkspaceIDsByUserID(ctx context.Context, userID string) ([]string, error) {
	var ids []string
	err := r.db.WithContext(ctx).Model(&WorkspaceMember{}).
		Where("user_id = ?", userID).
		Pluck("workspace_id", &ids).Error
	if err != nil {
		return nil, fmt.Errorf("listing workspace IDs for user: %w", err)
	}
	return ids, nil
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
		DeclinedAt:  m.DeclinedAt,
		CreatedAt:   m.CreatedAt,
	}
}
