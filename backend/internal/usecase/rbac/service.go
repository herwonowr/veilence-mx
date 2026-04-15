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
	ErrOrgNotFound        = errors.New("organization not found")
	ErrMemberNotFound     = errors.New("member not found")
	ErrInvitationNotFound = errors.New("invitation not found")
	ErrInvitationExpired  = errors.New("invitation has expired")
	ErrInvitationAccepted = errors.New("invitation already accepted")
	ErrAlreadyMember      = errors.New("user is already a member of this organization")
	ErrCannotRemoveOwner  = errors.New("cannot remove the organization owner")
	ErrCannotChangeOwner  = errors.New("cannot change the owner's role")
	ErrRoleNotFound       = errors.New("role not found")
	ErrPermissionDenied   = errors.New("permission denied")
	ErrSlugTaken            = errors.New("organization slug is already taken")
	ErrInvitationEmailMismatch = errors.New("invitation email does not match accepting user")
	ErrInvitationRevoked    = errors.New("invitation has been revoked")
)

// Service provides RBAC and organization management operations.
type Service struct {
	db *gorm.DB
}

// NewService creates a new RBAC service.
func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

// CreateOrganization creates a new organization, seeds default roles, and assigns
// the creating user as the owner.
func (s *Service) CreateOrganization(userID uint, name, slug, description string) (*persistent.Organization, error) {
	// Check slug uniqueness
	var count int64
	s.db.Model(&persistent.Organization{}).Where("slug = ?", slug).Count(&count)
	if count > 0 {
		return nil, ErrSlugTaken
	}

	org := &persistent.Organization{
		Name:        name,
		Slug:        slug,
		Description: description,
		OwnerID:     userID,
		IsActive:    true,
	}

	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(org).Error; err != nil {
			return fmt.Errorf("creating organization: %w", err)
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

		member := &persistent.OrgMember{
			OrgID:    org.ID,
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

	slog.Info("organization created", "org_id", org.ID, "slug", slug, "owner_id", userID)
	return org, nil
}

// createDefaultRoles creates the four system roles (owner, admin, member, viewer)
// with their respective permissions for the given organization.
func (s *Service) createDefaultRoles(tx *gorm.DB, orgID uint) ([]persistent.Role, error) {
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
		return k == "org:delete"
	})

	// Member permissions
	memberKeys := []string{
		"packages:read", "packages:write",
		"alerts:read", "alerts:write",
		"releases:read",
		"settings:read",
		"members:read",
		"roles:read",
		"org:read",
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
		{persistent.RoleOwner, "Full access to the organization", allKeys},
		{persistent.RoleAdmin, "Administrative access (cannot delete organization)", adminKeys},
		{persistent.RoleMember, "Standard member with read/write access", memberKeys},
		{persistent.RoleViewer, "Read-only access", viewerKeys},
	}

	var roles []persistent.Role
	for _, def := range roleDefinitions {
		role := persistent.Role{
			OrgID:       orgID,
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

// GetUserOrganizations returns all organizations the user is a member of.
func (s *Service) GetUserOrganizations(userID uint) ([]persistent.Organization, error) {
	var orgs []persistent.Organization
	err := s.db.
		Joins("JOIN org_members ON org_members.org_id = organizations.id").
		Where("org_members.user_id = ? AND organizations.deleted_at IS NULL", userID).
		Find(&orgs).Error
	if err != nil {
		return nil, fmt.Errorf("listing user organizations: %w", err)
	}
	return orgs, nil
}

// GetOrganization returns a single organization by ID.
func (s *Service) GetOrganization(orgID uint) (*persistent.Organization, error) {
	var org persistent.Organization
	if err := s.db.First(&org, orgID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrOrgNotFound
		}
		return nil, fmt.Errorf("getting organization: %w", err)
	}
	return &org, nil
}

// UpdateOrganization updates the organization's name, slug, and description.
func (s *Service) UpdateOrganization(orgID uint, name, slug, description string) (*persistent.Organization, error) {
	var org persistent.Organization
	if err := s.db.First(&org, orgID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrOrgNotFound
		}
		return nil, fmt.Errorf("getting organization: %w", err)
	}

	// Check slug uniqueness if changed
	if slug != org.Slug {
		var count int64
		s.db.Model(&persistent.Organization{}).Where("slug = ? AND id != ?", slug, orgID).Count(&count)
		if count > 0 {
			return nil, ErrSlugTaken
		}
	}

	org.Name = name
	org.Slug = slug
	org.Description = description

	if err := s.db.Save(&org).Error; err != nil {
		return nil, fmt.Errorf("updating organization: %w", err)
	}
	return &org, nil
}

// DeleteOrganization soft-deletes the organization.
func (s *Service) DeleteOrganization(orgID uint) error {
	result := s.db.Delete(&persistent.Organization{}, orgID)
	if result.Error != nil {
		return fmt.Errorf("deleting organization: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrOrgNotFound
	}
	slog.Info("organization deleted", "org_id", orgID)
	return nil
}

// GetOrgMembers returns all members of an organization with their roles and user data.
func (s *Service) GetOrgMembers(orgID uint) ([]persistent.OrgMember, error) {
	var members []persistent.OrgMember
	err := s.db.
		Preload("Role").
		Preload("Role.Permissions").
		Preload("User").
		Where("org_id = ?", orgID).
		Find(&members).Error
	if err != nil {
		return nil, fmt.Errorf("listing org members: %w", err)
	}
	return members, nil
}

// InviteMember creates an invitation for a user to join an organization.
// Returns the invitation and the raw token (for inclusion in the invitation URL).
// Only the SHA-256 hash of the token is stored in the database.
func (s *Service) InviteMember(orgID uint, email string, roleID, invitedBy uint) (*persistent.Invitation, string, error) {
	// Verify the role exists and belongs to this org
	var role persistent.Role
	if err := s.db.Where("id = ? AND org_id = ?", roleID, orgID).First(&role).Error; err != nil {
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
		OrgID:     orgID,
		Email:     email,
		RoleID:    roleID,
		TokenHash: hashToken(rawToken),
		InvitedBy: invitedBy,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour), // 7 days
	}

	if err := s.db.Create(invitation).Error; err != nil {
		return nil, "", fmt.Errorf("creating invitation: %w", err)
	}

	slog.Info("invitation created", "org_id", orgID, "email", email, "invited_by", invitedBy)
	return invitation, rawToken, nil
}

// AcceptInvitation accepts a pending invitation and creates a membership.
// The userEmail is compared against the invitation email to prevent unauthorized
// acceptance. If the emails don't match, ErrInvitationEmailMismatch is returned.
func (s *Service) AcceptInvitation(token string, userID uint, userEmail string) (*persistent.OrgMember, error) {
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
	s.db.Model(&persistent.OrgMember{}).
		Where("org_id = ? AND user_id = ?", invitation.OrgID, userID).
		Count(&existingCount)
	if existingCount > 0 {
		return nil, ErrAlreadyMember
	}

	var member *persistent.OrgMember
	err := s.db.Transaction(func(tx *gorm.DB) error {
		now := time.Now()
		invitation.AcceptedAt = &now
		if err := tx.Save(&invitation).Error; err != nil {
			return fmt.Errorf("updating invitation: %w", err)
		}

		member = &persistent.OrgMember{
			OrgID:    invitation.OrgID,
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

	slog.Info("invitation accepted", "org_id", invitation.OrgID, "user_id", userID)
	return member, nil
}

// GetInvitationByToken returns invitation details by token. This allows the
// frontend to show the user what organization they are being invited to before
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
// invitations for an organization.
func (s *Service) ListPendingInvitations(orgID uint) ([]persistent.Invitation, error) {
	var invitations []persistent.Invitation
	err := s.db.
		Where("org_id = ? AND accepted_at IS NULL AND expires_at > ?", orgID, time.Now()).
		Order("created_at DESC").
		Find(&invitations).Error
	if err != nil {
		return nil, fmt.Errorf("listing pending invitations: %w", err)
	}
	return invitations, nil
}

// RevokeInvitation deletes a pending invitation by ID and org. Only pending
// (not accepted) invitations can be revoked.
func (s *Service) RevokeInvitation(orgID, invitationID uint) error {
	result := s.db.
		Where("id = ? AND org_id = ? AND accepted_at IS NULL", invitationID, orgID).
		Delete(&persistent.Invitation{})
	if result.Error != nil {
		return fmt.Errorf("revoking invitation: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrInvitationNotFound
	}

	slog.Info("invitation revoked", "invitation_id", invitationID, "org_id", orgID)
	return nil
}

// RemoveMember removes a user from an organization. The owner cannot be removed.
func (s *Service) RemoveMember(orgID, userID uint) error {
	// Check if user is the owner
	var org persistent.Organization
	if err := s.db.First(&org, orgID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrOrgNotFound
		}
		return fmt.Errorf("getting organization: %w", err)
	}
	if org.OwnerID == userID {
		return ErrCannotRemoveOwner
	}

	result := s.db.Where("org_id = ? AND user_id = ?", orgID, userID).Delete(&persistent.OrgMember{})
	if result.Error != nil {
		return fmt.Errorf("removing member: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrMemberNotFound
	}

	slog.Info("member removed", "org_id", orgID, "user_id", userID)
	return nil
}

// UpdateMemberRole changes a member's role within an organization.
// The owner's role cannot be changed.
func (s *Service) UpdateMemberRole(orgID, userID, newRoleID uint) (*persistent.OrgMember, error) {
	// Check if user is the owner
	var org persistent.Organization
	if err := s.db.First(&org, orgID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrOrgNotFound
		}
		return nil, fmt.Errorf("getting organization: %w", err)
	}
	if org.OwnerID == userID {
		return nil, ErrCannotChangeOwner
	}

	// Verify the new role exists and belongs to this org
	var role persistent.Role
	if err := s.db.Where("id = ? AND org_id = ?", newRoleID, orgID).First(&role).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRoleNotFound
		}
		return nil, fmt.Errorf("checking role: %w", err)
	}

	// Cannot assign owner role
	if role.Name == persistent.RoleOwner {
		return nil, ErrCannotChangeOwner
	}

	var member persistent.OrgMember
	if err := s.db.Where("org_id = ? AND user_id = ?", orgID, userID).First(&member).Error; err != nil {
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

	slog.Info("member role updated", "org_id", orgID, "user_id", userID, "new_role_id", newRoleID)
	return &member, nil
}

// CheckPermission verifies whether a user has a specific permission within an organization.
// Returns nil if permitted, ErrPermissionDenied otherwise.
func (s *Service) CheckPermission(userID, orgID uint, resource, action string) error {
	var count int64
	err := s.db.Model(&persistent.Permission{}).
		Joins("JOIN role_permissions ON role_permissions.permission_id = permissions.id").
		Joins("JOIN roles ON roles.id = role_permissions.role_id").
		Joins("JOIN org_members ON org_members.role_id = roles.id").
		Where("org_members.user_id = ? AND org_members.org_id = ? AND permissions.resource = ? AND permissions.action = ?",
			userID, orgID, resource, action).
		Count(&count).Error
	if err != nil {
		return fmt.Errorf("checking permission: %w", err)
	}
	if count == 0 {
		return ErrPermissionDenied
	}
	return nil
}

// GetOrgRoles returns all roles for an organization.
func (s *Service) GetOrgRoles(orgID uint) ([]persistent.Role, error) {
	var roles []persistent.Role
	err := s.db.
		Preload("Permissions").
		Where("org_id = ?", orgID).
		Find(&roles).Error
	if err != nil {
		return nil, fmt.Errorf("listing org roles: %w", err)
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

// GetUserMembership returns the user's membership record for an organization.
func (s *Service) GetUserMembership(userID, orgID uint) (*persistent.OrgMember, error) {
	var member persistent.OrgMember
	err := s.db.
		Preload("Role").
		Where("user_id = ? AND org_id = ?", userID, orgID).
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
