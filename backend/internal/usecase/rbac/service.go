package rbac

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"slices"
	"time"

	"gorm.io/gorm"

	"github.com/veilence/veilence-mx/backend/internal/repo/persistent"
)

// Common errors returned by the RBAC service.
var (
	ErrWorkspaceNotFound        = errors.New("workspace not found")
	ErrMemberNotFound     = errors.New("member not found")
	ErrInvitationNotFound = errors.New("invitation not found")
	ErrInvitationExpired  = errors.New("invitation has expired")
	ErrInvitationAccepted = errors.New("invitation already accepted")
	ErrAlreadyMember      = errors.New("user is already a member of this workspace")
	ErrCannotRemoveOwner  = errors.New("cannot remove the workspace owner")
	ErrCannotChangeOwner  = errors.New("cannot change the owner's role")
	ErrRoleNotFound       = errors.New("role not found")
	ErrPermissionDenied   = errors.New("permission denied")
	ErrSlugTaken            = errors.New("workspace slug is already taken")
	ErrInvitationEmailMismatch = errors.New("invitation email does not match accepting user")
	ErrInvitationRevoked    = errors.New("invitation has been revoked")
)

// Service provides RBAC and workspace management operations.
type Service struct {
	db *gorm.DB
}

// NewService creates a new RBAC service.
func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

// CreateWorkspace creates a new workspace, seeds default roles, and assigns
// the creating user as the owner.
func (s *Service) CreateWorkspace(userID uint, name, slug, description string) (*persistent.Workspace, error) {
	// Check slug uniqueness
	var count int64
	s.db.Model(&persistent.Workspace{}).Where("slug = ?", slug).Count(&count)
	if count > 0 {
		return nil, ErrSlugTaken
	}

	org := &persistent.Workspace{
		Name:        name,
		Slug:        slug,
		Description: description,
		OwnerID:     userID,
		IsActive:    true,
	}

	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(org).Error; err != nil {
			return fmt.Errorf("creating workspace: %w", err)
		}

		// Create default roles
		roles, err := s.createDefaultRoles(tx, org.ID)
		if err != nil {
			return fmt.Errorf("creating default roles: %w", err)
		}

		// Find the owner role and assign it to the creating user
		var ownerRole *persistent.Role
		for i := range roles {
			if roles[i].Name == persistent.RoleOwner {
				ownerRole = &roles[i]
				break
			}
		}
		if ownerRole == nil {
			return errors.New("owner role not created")
		}

		member := &persistent.WorkspaceMember{
			WorkspaceID:    org.ID,
			UserID:   userID,
			RoleID:   ownerRole.ID,
			JoinedAt: time.Now(),
		}
		if err := tx.Create(member).Error; err != nil {
			return fmt.Errorf("creating owner membership: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	slog.Info("workspace created", "workspace_id", org.ID, "slug", slug, "owner_id", userID)
	return org, nil
}

// createDefaultRoles creates the four system roles (owner, admin, member, viewer)
// with their respective permissions for the given workspace.
func (s *Service) createDefaultRoles(tx *gorm.DB, workspaceID uint) ([]persistent.Role, error) {
	// Load all system permissions
	var allPerms []persistent.Permission
	if err := tx.Find(&allPerms).Error; err != nil {
		return nil, fmt.Errorf("loading permissions: %w", err)
	}

	permMap := make(map[string]persistent.Permission)
	for _, p := range allPerms {
		key := p.Resource + ":" + p.Action
		permMap[key] = p
	}

	lookupPerms := func(keys []string) []persistent.Permission {
		var perms []persistent.Permission
		for _, key := range keys {
			if p, ok := permMap[key]; ok {
				perms = append(perms, p)
			}
		}
		return perms
	}

	// All permission keys
	allKeys := slices.Collect(maps.Keys(permMap))

	// Admin gets all except org:delete
	adminKeys := slices.DeleteFunc(slices.Clone(allKeys), func(k string) bool {
		return k == "workspace:delete"
	})

	// Member permissions
	memberKeys := []string{
		"packages:read", "packages:write",
		"alerts:read", "alerts:write",
		"releases:read",
		"settings:read",
		"members:read",
		"roles:read",
		"workspace:read",
		"api_keys:read", "api_keys:write",
		"notifications:read", "notifications:create", "notifications:update", "notifications:delete",
	}

	// Viewer permissions
	viewerKeys := []string{
		"packages:read",
		"alerts:read",
		"releases:read",
		"settings:read",
		"notifications:read",
	}

	roleDefinitions := []struct {
		Name        string
		Description string
		PermKeys    []string
	}{
		{persistent.RoleOwner, "Full access to the workspace", allKeys},
		{persistent.RoleAdmin, "Administrative access (cannot delete workspace)", adminKeys},
		{persistent.RoleMember, "Standard member with read/write access", memberKeys},
		{persistent.RoleViewer, "Read-only access", viewerKeys},
	}

	var roles []persistent.Role
	for _, def := range roleDefinitions {
		role := persistent.Role{
			WorkspaceID:       workspaceID,
			Name:        def.Name,
			Description: def.Description,
			IsSystem:    true,
			Permissions: lookupPerms(def.PermKeys),
		}
		if err := tx.Create(&role).Error; err != nil {
			return nil, fmt.Errorf("creating role %s: %w", def.Name, err)
		}
		roles = append(roles, role)
	}

	return roles, nil
}

// GetUserWorkspaces returns all workspaces the user is a member of.
func (s *Service) GetUserWorkspaces(userID uint) ([]persistent.Workspace, error) {
	var orgs []persistent.Workspace
	err := s.db.
		Joins("JOIN workspace_members ON workspace_members.workspace_id = workspaces.id").
		Where("workspace_members.user_id = ? AND workspaces.deleted_at IS NULL", userID).
		Find(&orgs).Error
	if err != nil {
		return nil, fmt.Errorf("listing user workspaces: %w", err)
	}
	return orgs, nil
}

// GetWorkspace returns a single workspace by ID.
func (s *Service) GetWorkspace(workspaceID uint) (*persistent.Workspace, error) {
	var org persistent.Workspace
	if err := s.db.First(&org, workspaceID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrWorkspaceNotFound
		}
		return nil, fmt.Errorf("getting workspace: %w", err)
	}
	return &org, nil
}

// UpdateWorkspace updates the workspace's name, slug, and description.
func (s *Service) UpdateWorkspace(workspaceID uint, name, slug, description string) (*persistent.Workspace, error) {
	var org persistent.Workspace
	if err := s.db.First(&org, workspaceID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrWorkspaceNotFound
		}
		return nil, fmt.Errorf("getting workspace: %w", err)
	}

	// Check slug uniqueness if changed
	if slug != org.Slug {
		var count int64
		s.db.Model(&persistent.Workspace{}).Where("slug = ? AND id != ?", slug, workspaceID).Count(&count)
		if count > 0 {
			return nil, ErrSlugTaken
		}
	}

	org.Name = name
	org.Slug = slug
	org.Description = description

	if err := s.db.Save(&org).Error; err != nil {
		return nil, fmt.Errorf("updating workspace: %w", err)
	}
	return &org, nil
}

// DeleteWorkspace soft-deletes the workspace.
func (s *Service) DeleteWorkspace(workspaceID uint) error {
	result := s.db.Delete(&persistent.Workspace{}, workspaceID)
	if result.Error != nil {
		return fmt.Errorf("deleting workspace: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrWorkspaceNotFound
	}
	slog.Info("workspace deleted", "workspace_id", workspaceID)
	return nil
}

// GetWorkspaceMembers returns all members of a workspace with their roles and user data.
func (s *Service) GetWorkspaceMembers(workspaceID uint) ([]persistent.WorkspaceMember, error) {
	var members []persistent.WorkspaceMember
	err := s.db.
		Preload("Role").
		Preload("Role.Permissions").
		Preload("User").
		Where("workspace_id = ?", workspaceID).
		Find(&members).Error
	if err != nil {
		return nil, fmt.Errorf("listing workspace members: %w", err)
	}
	return members, nil
}

// InviteMember creates an invitation for a user to join a workspace.
// Returns the invitation and the raw token (for inclusion in the invitation URL).
// Only the SHA-256 hash of the token is stored in the database.
func (s *Service) InviteMember(workspaceID uint, email string, roleID, invitedBy uint) (*persistent.Invitation, string, error) {
	// Verify the role exists and belongs to this org
	var role persistent.Role
	if err := s.db.Where("id = ? AND workspace_id = ?", roleID, workspaceID).First(&role).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, "", ErrRoleNotFound
		}
		return nil, "", fmt.Errorf("checking role: %w", err)
	}

	// Cannot invite as owner
	if role.Name == persistent.RoleOwner {
		return nil, "", errors.New("cannot invite user as owner")
	}

	// Generate secure token
	rawToken, err := generateToken()
	if err != nil {
		return nil, "", fmt.Errorf("generating invitation token: %w", err)
	}

	invitation := &persistent.Invitation{
		WorkspaceID:     workspaceID,
		Email:     email,
		RoleID:    roleID,
		TokenHash: hashToken(rawToken),
		InvitedBy: invitedBy,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour), // 7 days
	}

	if err := s.db.Create(invitation).Error; err != nil {
		return nil, "", fmt.Errorf("creating invitation: %w", err)
	}

	slog.Info("invitation created", "workspace_id", workspaceID, "email", email, "invited_by", invitedBy)
	return invitation, rawToken, nil
}

// AcceptInvitation accepts a pending invitation and creates a membership.
// The userEmail is compared against the invitation email to prevent unauthorized
// acceptance. If the emails don't match, ErrInvitationEmailMismatch is returned.
func (s *Service) AcceptInvitation(token string, userID uint, userEmail string) (*persistent.WorkspaceMember, error) {
	var invitation persistent.Invitation
	if err := s.db.Where("token_hash = ?", hashToken(token)).First(&invitation).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvitationNotFound
		}
		return nil, fmt.Errorf("finding invitation: %w", err)
	}

	if invitation.AcceptedAt != nil {
		return nil, ErrInvitationAccepted
	}

	if time.Now().After(invitation.ExpiresAt) {
		return nil, ErrInvitationExpired
	}

	// Verify the accepting user's email matches the invitation email
	if invitation.Email != userEmail {
		return nil, ErrInvitationEmailMismatch
	}

	// Check if user is already a member
	var existingCount int64
	s.db.Model(&persistent.WorkspaceMember{}).
		Where("workspace_id = ? AND user_id = ?", invitation.WorkspaceID, userID).
		Count(&existingCount)
	if existingCount > 0 {
		return nil, ErrAlreadyMember
	}

	var member *persistent.WorkspaceMember
	err := s.db.Transaction(func(tx *gorm.DB) error {
		now := time.Now()
		invitation.AcceptedAt = &now
		if err := tx.Save(&invitation).Error; err != nil {
			return fmt.Errorf("updating invitation: %w", err)
		}

		member = &persistent.WorkspaceMember{
			WorkspaceID:    invitation.WorkspaceID,
			UserID:   userID,
			RoleID:   invitation.RoleID,
			JoinedAt: now,
		}
		if err := tx.Create(member).Error; err != nil {
			return fmt.Errorf("creating membership: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	slog.Info("invitation accepted", "workspace_id", invitation.WorkspaceID, "user_id", userID)
	return member, nil
}

// GetInvitationByToken returns invitation details by token. This allows the
// frontend to show the user what workspace they are being invited to before
// accepting. Does not require authentication.
func (s *Service) GetInvitationByToken(token string) (*persistent.Invitation, error) {
	var invitation persistent.Invitation
	if err := s.db.Where("token_hash = ?", hashToken(token)).First(&invitation).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvitationNotFound
		}
		return nil, fmt.Errorf("finding invitation: %w", err)
	}

	return &invitation, nil
}

// ListPendingInvitations returns all pending (not accepted, not expired)
// invitations for a workspace.
func (s *Service) ListPendingInvitations(workspaceID uint) ([]persistent.Invitation, error) {
	var invitations []persistent.Invitation
	err := s.db.
		Where("workspace_id = ? AND accepted_at IS NULL AND expires_at > ?", workspaceID, time.Now()).
		Order("created_at DESC").
		Find(&invitations).Error
	if err != nil {
		return nil, fmt.Errorf("listing pending invitations: %w", err)
	}
	return invitations, nil
}

// RevokeInvitation deletes a pending invitation by ID and org. Only pending
// (not accepted) invitations can be revoked.
func (s *Service) RevokeInvitation(workspaceID, invitationID uint) error {
	result := s.db.
		Where("id = ? AND workspace_id = ? AND accepted_at IS NULL", invitationID, workspaceID).
		Delete(&persistent.Invitation{})
	if result.Error != nil {
		return fmt.Errorf("revoking invitation: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrInvitationNotFound
	}

	slog.Info("invitation revoked", "invitation_id", invitationID, "workspace_id", workspaceID)
	return nil
}

// RemoveMember removes a user from a workspace. The owner cannot be removed.
func (s *Service) RemoveMember(workspaceID, userID uint) error {
	// Check if user is the owner
	var org persistent.Workspace
	if err := s.db.First(&org, workspaceID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrWorkspaceNotFound
		}
		return fmt.Errorf("getting workspace: %w", err)
	}
	if org.OwnerID == userID {
		return ErrCannotRemoveOwner
	}

	result := s.db.Where("workspace_id = ? AND user_id = ?", workspaceID, userID).Delete(&persistent.WorkspaceMember{})
	if result.Error != nil {
		return fmt.Errorf("removing member: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrMemberNotFound
	}

	slog.Info("member removed", "workspace_id", workspaceID, "user_id", userID)
	return nil
}

// UpdateMemberRole changes a member's role within a workspace.
// The owner's role cannot be changed.
func (s *Service) UpdateMemberRole(workspaceID, userID, newRoleID uint) (*persistent.WorkspaceMember, error) {
	// Check if user is the owner
	var org persistent.Workspace
	if err := s.db.First(&org, workspaceID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrWorkspaceNotFound
		}
		return nil, fmt.Errorf("getting workspace: %w", err)
	}
	if org.OwnerID == userID {
		return nil, ErrCannotChangeOwner
	}

	// Verify the new role exists and belongs to this org
	var role persistent.Role
	if err := s.db.Where("id = ? AND workspace_id = ?", newRoleID, workspaceID).First(&role).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRoleNotFound
		}
		return nil, fmt.Errorf("checking role: %w", err)
	}

	// Cannot assign owner role
	if role.Name == persistent.RoleOwner {
		return nil, ErrCannotChangeOwner
	}

	var member persistent.WorkspaceMember
	if err := s.db.Where("workspace_id = ? AND user_id = ?", workspaceID, userID).First(&member).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrMemberNotFound
		}
		return nil, fmt.Errorf("finding member: %w", err)
	}

	member.RoleID = newRoleID
	if err := s.db.Save(&member).Error; err != nil {
		return nil, fmt.Errorf("updating member role: %w", err)
	}

	// Reload with role
	s.db.Preload("Role").First(&member, member.ID)

	slog.Info("member role updated", "workspace_id", workspaceID, "user_id", userID, "new_role_id", newRoleID)
	return &member, nil
}

// CheckPermission verifies whether a user has a specific permission within a workspace.
// Returns nil if permitted, ErrPermissionDenied otherwise.
func (s *Service) CheckPermission(userID, workspaceID uint, resource, action string) error {
	var count int64
	err := s.db.Model(&persistent.Permission{}).
		Joins("JOIN role_permissions ON role_permissions.permission_id = permissions.id").
		Joins("JOIN roles ON roles.id = role_permissions.role_id").
		Joins("JOIN workspace_members ON workspace_members.role_id = roles.id").
		Where("workspace_members.user_id = ? AND workspace_members.workspace_id = ? AND permissions.resource = ? AND permissions.action = ?",
			userID, workspaceID, resource, action).
		Count(&count).Error
	if err != nil {
		return fmt.Errorf("checking permission: %w", err)
	}
	if count == 0 {
		return ErrPermissionDenied
	}
	return nil
}

// GetWorkspaceRoles returns all roles for a workspace.
func (s *Service) GetWorkspaceRoles(workspaceID uint) ([]persistent.Role, error) {
	var roles []persistent.Role
	err := s.db.
		Preload("Permissions").
		Where("workspace_id = ?", workspaceID).
		Find(&roles).Error
	if err != nil {
		return nil, fmt.Errorf("listing workspace roles: %w", err)
	}
	return roles, nil
}

// GetAllPermissions returns all system permissions.
func (s *Service) GetAllPermissions() ([]persistent.Permission, error) {
	var perms []persistent.Permission
	if err := s.db.Find(&perms).Error; err != nil {
		return nil, fmt.Errorf("listing permissions: %w", err)
	}
	return perms, nil
}

// GetUserMembership returns the user's membership record for a workspace.
func (s *Service) GetUserMembership(userID, workspaceID uint) (*persistent.WorkspaceMember, error) {
	var member persistent.WorkspaceMember
	err := s.db.
		Preload("Role").
		Where("user_id = ? AND workspace_id = ?", userID, workspaceID).
		First(&member).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrMemberNotFound
		}
		return nil, fmt.Errorf("getting user membership: %w", err)
	}
	return &member, nil
}

// SeedPermissions inserts all system permissions into the database if they
// don't already exist. This is idempotent and safe to call on every startup.
func SeedPermissions(db *gorm.DB) error {
	for _, perm := range persistent.SystemPermissions {
		result := db.Where("resource = ? AND action = ?", perm.Resource, perm.Action).FirstOrCreate(&perm)
		if result.Error != nil {
			return fmt.Errorf("seeding permission %s:%s: %w", perm.Resource, perm.Action, result.Error)
		}
	}
	slog.Info("system permissions seeded", "count", len(persistent.SystemPermissions))
	return nil
}

// generateToken creates a cryptographically secure random token.
func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// hashToken returns the SHA-256 hex digest of a token.
// Invitation tokens are high-entropy random values, so SHA-256 is sufficient
// (unlike passwords, they don't need bcrypt's slow hashing).
func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}
