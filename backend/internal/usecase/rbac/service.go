package rbac

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"slices"
	"time"

	"github.com/veilence/veilence-mx/backend/internal/entity"
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
	repo RBACRepository
}

// NewService creates a new RBAC service.
func NewService(repo RBACRepository) *Service {
	return &Service{repo: repo}
}

// CreateWorkspace creates a new workspace, seeds default roles, and assigns
// the creating user as the owner.
func (s *Service) CreateWorkspace(userID string, name, slug, description string) (*entity.Workspace, error) {
	ctx := ctx_bg()

	// Check slug uniqueness
	count, _ := s.repo.CountWorkspacesBySlug(ctx, slug, nil)
	if count > 0 {
		return nil, ErrSlugTaken
	}

	ws := &entity.Workspace{
		Name:        name,
		Slug:        slug,
		Description: description,
		OwnerID:     userID,
		IsActive:    true,
	}

	err := s.repo.WithTransaction(ctx, func(tx RBACRepository) error {
		if err := tx.CreateWorkspace(ctx, ws); err != nil {
			return fmt.Errorf("creating workspace: %w", err)
		}

		// Create default roles
		roles, err := s.createDefaultRoles(ctx, tx, ws.ID)
		if err != nil {
			return fmt.Errorf("creating default roles: %w", err)
		}

		// Find the owner role and assign it to the creating user
		var ownerRole *entity.Role
		for i := range roles {
			if roles[i].Name == entity.RoleOwner {
				ownerRole = &roles[i]
				break
			}
		}
		if ownerRole == nil {
			return errors.New("owner role not created")
		}

		member := &entity.WorkspaceMember{
			WorkspaceID: ws.ID,
			UserID:      userID,
			RoleID:      ownerRole.ID,
			JoinedAt:    time.Now(),
		}
		if err := tx.CreateMember(ctx, member); err != nil {
			return fmt.Errorf("creating owner membership: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	slog.Info("workspace created", "workspace_id", ws.ID, "slug", slug, "owner_id", userID)
	return ws, nil
}

// createDefaultRoles creates the four system roles (owner, admin, member, viewer)
// with their respective permissions for the given workspace.
func (s *Service) createDefaultRoles(ctx_unused interface{}, tx RBACRepository, workspaceID string) ([]entity.Role, error) {
	ctx := ctx_bg()

	// Load all system permissions
	allPerms, err := tx.FindAllPermissions(ctx)
	if err != nil {
		return nil, fmt.Errorf("loading permissions: %w", err)
	}

	permMap := make(map[string]entity.Permission)
	for _, p := range allPerms {
		key := p.Resource + ":" + p.Action
		permMap[key] = p
	}

	lookupPerms := func(keys []string) []entity.Permission {
		var perms []entity.Permission
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
		{entity.RoleOwner, "Full access to the workspace", allKeys},
		{entity.RoleAdmin, "Administrative access (cannot delete workspace)", adminKeys},
		{entity.RoleMember, "Standard member with read/write access", memberKeys},
		{entity.RoleViewer, "Read-only access", viewerKeys},
	}

	var roles []entity.Role
	for _, def := range roleDefinitions {
		role := entity.Role{
			WorkspaceID: workspaceID,
			Name:        def.Name,
			Description: def.Description,
			IsSystem:    true,
			Permissions: lookupPerms(def.PermKeys),
		}
		if err := tx.CreateRole(ctx, &role); err != nil {
			return nil, fmt.Errorf("creating role %s: %w", def.Name, err)
		}
		roles = append(roles, role)
	}

	return roles, nil
}

// GetUserWorkspaces returns all workspaces the user is a member of.
func (s *Service) GetUserWorkspaces(userID string) ([]entity.Workspace, error) {
	return s.repo.FindWorkspacesByUserID(ctx_bg(), userID)
}

// GetWorkspace returns a single workspace by ID.
func (s *Service) GetWorkspace(workspaceID string) (*entity.Workspace, error) {
	ws, err := s.repo.FindWorkspaceByID(ctx_bg(), workspaceID)
	if err != nil {
		return nil, ErrWorkspaceNotFound
	}
	return ws, nil
}

// UpdateWorkspace updates the workspace's name, slug, and description.
func (s *Service) UpdateWorkspace(workspaceID string, name, slug, description string) (*entity.Workspace, error) {
	ctx := ctx_bg()

	ws, err := s.repo.FindWorkspaceByID(ctx, workspaceID)
	if err != nil {
		return nil, ErrWorkspaceNotFound
	}

	// Check slug uniqueness if changed
	if slug != ws.Slug {
		count, _ := s.repo.CountWorkspacesBySlug(ctx, slug, &workspaceID)
		if count > 0 {
			return nil, ErrSlugTaken
		}
	}

	ws.Name = name
	ws.Slug = slug
	ws.Description = description

	if err := s.repo.UpdateWorkspace(ctx, ws); err != nil {
		return nil, fmt.Errorf("updating workspace: %w", err)
	}
	return ws, nil
}

// DeleteWorkspace soft-deletes the workspace.
func (s *Service) DeleteWorkspace(workspaceID string) error {
	if err := s.repo.SoftDeleteWorkspace(ctx_bg(), workspaceID); err != nil {
		return ErrWorkspaceNotFound
	}
	slog.Info("workspace deleted", "workspace_id", workspaceID)
	return nil
}

// GetWorkspaceMembers returns all members of a workspace with their roles and user data.
func (s *Service) GetWorkspaceMembers(workspaceID string) ([]entity.WorkspaceMember, error) {
	return s.repo.FindMembersByWorkspaceID(ctx_bg(), workspaceID)
}

// InviteMember creates an invitation for a user to join a workspace.
// Returns the invitation and the raw token (for inclusion in the invitation URL).
// Only the SHA-256 hash of the token is stored in the database.
func (s *Service) InviteMember(workspaceID string, email string, roleID string, invitedBy string) (*entity.Invitation, string, error) {
	ctx := ctx_bg()

	// Verify the role exists and belongs to this org
	role, err := s.repo.FindRoleByIDAndWorkspace(ctx, roleID, workspaceID)
	if err != nil || role == nil {
		return nil, "", ErrRoleNotFound
	}

	// Cannot invite as owner
	if role.Name == entity.RoleOwner {
		return nil, "", errors.New("cannot invite user as owner")
	}

	// Generate secure token
	rawToken, err := generateToken()
	if err != nil {
		return nil, "", fmt.Errorf("generating invitation token: %w", err)
	}

	invitation := &entity.Invitation{
		WorkspaceID: workspaceID,
		Email:       email,
		RoleID:      roleID,
		TokenHash:   hashToken(rawToken),
		InvitedBy:   invitedBy,
		ExpiresAt:   time.Now().Add(7 * 24 * time.Hour), // 7 days
	}

	if err := s.repo.CreateInvitation(ctx, invitation); err != nil {
		return nil, "", fmt.Errorf("creating invitation: %w", err)
	}

	slog.Info("invitation created", "workspace_id", workspaceID, "email", email, "invited_by", invitedBy)
	return invitation, rawToken, nil
}

// AcceptInvitation accepts a pending invitation and creates a membership.
// The userEmail is compared against the invitation email to prevent unauthorized
// acceptance. If the emails don't match, ErrInvitationEmailMismatch is returned.
func (s *Service) AcceptInvitation(token string, userID string, userEmail string) (*entity.WorkspaceMember, error) {
	ctx := ctx_bg()

	invitation, err := s.repo.FindInvitationByTokenHash(ctx, hashToken(token))
	if err != nil || invitation == nil {
		return nil, ErrInvitationNotFound
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
	existingCount, _ := s.repo.CountMembersByUserAndWorkspace(ctx, userID, invitation.WorkspaceID)
	if existingCount > 0 {
		return nil, ErrAlreadyMember
	}

	var member *entity.WorkspaceMember
	err = s.repo.WithTransaction(ctx, func(tx RBACRepository) error {
		now := time.Now()
		invitation.AcceptedAt = &now
		if err := tx.UpdateInvitation(ctx, invitation); err != nil {
			return fmt.Errorf("updating invitation: %w", err)
		}

		member = &entity.WorkspaceMember{
			WorkspaceID: invitation.WorkspaceID,
			UserID:      userID,
			RoleID:      invitation.RoleID,
			JoinedAt:    now,
		}
		if err := tx.CreateMember(ctx, member); err != nil {
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
func (s *Service) GetInvitationByToken(token string) (*entity.Invitation, error) {
	invitation, err := s.repo.FindInvitationByTokenHash(ctx_bg(), hashToken(token))
	if err != nil || invitation == nil {
		return nil, ErrInvitationNotFound
	}
	return invitation, nil
}

// ListPendingInvitations returns all pending (not accepted, not expired)
// invitations for a workspace.
func (s *Service) ListPendingInvitations(workspaceID string) ([]entity.Invitation, error) {
	return s.repo.FindPendingInvitations(ctx_bg(), workspaceID)
}

// RevokeInvitation deletes a pending invitation by ID and org. Only pending
// (not accepted) invitations can be revoked.
func (s *Service) RevokeInvitation(workspaceID, invitationID string) error {
	if err := s.repo.DeletePendingInvitation(ctx_bg(), workspaceID, invitationID); err != nil {
		return ErrInvitationNotFound
	}
	slog.Info("invitation revoked", "invitation_id", invitationID, "workspace_id", workspaceID)
	return nil
}

// RemoveMember removes a user from a workspace. The owner cannot be removed.
func (s *Service) RemoveMember(workspaceID, userID string) error {
	ctx := ctx_bg()

	// Check if user is the owner
	ws, err := s.repo.FindWorkspaceByID(ctx, workspaceID)
	if err != nil {
		return ErrWorkspaceNotFound
	}
	if ws.OwnerID == userID {
		return ErrCannotRemoveOwner
	}

	if err := s.repo.DeleteMemberByUserAndWorkspace(ctx, userID, workspaceID); err != nil {
		return ErrMemberNotFound
	}

	slog.Info("member removed", "workspace_id", workspaceID, "user_id", userID)
	return nil
}

// UpdateMemberRole changes a member's role within a workspace.
// The owner's role cannot be changed.
func (s *Service) UpdateMemberRole(workspaceID, userID, newRoleID string) (*entity.WorkspaceMember, error) {
	ctx := ctx_bg()

	// Check if user is the owner
	ws, err := s.repo.FindWorkspaceByID(ctx, workspaceID)
	if err != nil {
		return nil, ErrWorkspaceNotFound
	}
	if ws.OwnerID == userID {
		return nil, ErrCannotChangeOwner
	}

	// Verify the new role exists and belongs to this org
	role, err := s.repo.FindRoleByIDAndWorkspace(ctx, newRoleID, workspaceID)
	if err != nil || role == nil {
		return nil, ErrRoleNotFound
	}

	// Cannot assign owner role
	if role.Name == entity.RoleOwner {
		return nil, ErrCannotChangeOwner
	}

	member, err := s.repo.FindMemberByUserAndWorkspace(ctx, userID, workspaceID)
	if err != nil || member == nil {
		return nil, ErrMemberNotFound
	}

	member.RoleID = newRoleID
	member.Role = role
	if err := s.repo.UpdateMember(ctx, member); err != nil {
		return nil, fmt.Errorf("updating member role: %w", err)
	}

	slog.Info("member role updated", "workspace_id", workspaceID, "user_id", userID, "new_role_id", newRoleID)
	return member, nil
}

// CheckPermission verifies whether a user has a specific permission within a workspace.
// Returns nil if permitted, ErrPermissionDenied otherwise.
func (s *Service) CheckPermission(userID, workspaceID string, resource, action string) error {
	ok, err := s.repo.CheckUserPermission(ctx_bg(), userID, workspaceID, resource, action)
	if err != nil {
		return fmt.Errorf("checking permission: %w", err)
	}
	if !ok {
		return ErrPermissionDenied
	}
	return nil
}

// CheckRolePermission verifies whether a given role name has a specific permission
// within a workspace. This is used for API key auth where the key has an assigned
// role rather than a user membership.
// Returns nil if permitted, ErrPermissionDenied otherwise.
func (s *Service) CheckRolePermission(workspaceID string, roleName, resource, action string) error {
	ok, err := s.repo.CheckRolePermission(ctx_bg(), workspaceID, roleName, resource, action)
	if err != nil {
		return fmt.Errorf("checking role permission: %w", err)
	}
	if !ok {
		return ErrPermissionDenied
	}
	return nil
}

// GetWorkspaceRoles returns all roles for a workspace.
func (s *Service) GetWorkspaceRoles(workspaceID string) ([]entity.Role, error) {
	return s.repo.FindRolesByWorkspaceID(ctx_bg(), workspaceID)
}

// GetAllPermissions returns all system permissions.
func (s *Service) GetAllPermissions() ([]entity.Permission, error) {
	return s.repo.FindAllPermissions(ctx_bg())
}

// GetUserMembership returns the user's membership record for a workspace.
func (s *Service) GetUserMembership(userID, workspaceID string) (*entity.WorkspaceMember, error) {
	member, err := s.repo.FindMemberByUserAndWorkspace(ctx_bg(), userID, workspaceID)
	if err != nil || member == nil {
		return nil, ErrMemberNotFound
	}
	return member, nil
}

// SeedPermissions inserts all system permissions into the database if they
// don't already exist. This is idempotent and safe to call on every startup.
func SeedPermissions(repo RBACRepository) error {
	ctx := ctx_bg()
	for _, perm := range entity.SystemPermissions {
		if err := repo.SeedPermission(ctx, perm); err != nil {
			return fmt.Errorf("seeding permission %s:%s: %w", perm.Resource, perm.Action, err)
		}
	}
	slog.Info("system permissions seeded", "count", len(entity.SystemPermissions))
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

// ctx_bg returns a background context. Many RBAC methods don't receive
// a context parameter (legacy API), so we use background context internally.
func ctx_bg() context.Context {
	return context.Background()
}
