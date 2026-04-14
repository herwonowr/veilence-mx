package rbac_test

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/veilence/veilence-mx/backend/internal/repo/persistent"
	"github.com/veilence/veilence-mx/backend/internal/usecase/rbac"
)

func setupRBACTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	err = db.AutoMigrate(
		&persistent.User{},
		&persistent.Organization{},
		&persistent.Role{},
		&persistent.Permission{},
		&persistent.OrgMember{},
		&persistent.Invitation{},
	)
	require.NoError(t, err)

	// Seed system permissions
	err = rbac.SeedPermissions(db)
	require.NoError(t, err)

	t.Cleanup(func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	})
	return db
}

func createTestUser(t *testing.T, db *gorm.DB, email string) *persistent.User {
	t.Helper()
	user := &persistent.User{
		Email:     email,
		FirstName: "Test",
		LastName:  "User",
		IsActive:  true,
	}
	require.NoError(t, user.HashPassword("Password123"))
	require.NoError(t, db.Create(user).Error)
	return user
}

// --- CreateOrganization ---

func TestCreateOrganization_WithDefaultRoles(t *testing.T) {
	db := setupRBACTestDB(t)
	svc := rbac.NewService(db)
	owner := createTestUser(t, db, "owner@example.com")

	org, err := svc.CreateOrganization(owner.ID, "Test Org", "test-org", "A test organization")
	require.NoError(t, err)
	assert.NotZero(t, org.ID)
	assert.Equal(t, "Test Org", org.Name)
	assert.Equal(t, "test-org", org.Slug)
	assert.Equal(t, owner.ID, org.OwnerID)
	assert.True(t, org.IsActive)

	// Verify default roles were created
	roles, err := svc.GetOrgRoles(org.ID)
	require.NoError(t, err)
	assert.Len(t, roles, 4) // owner, admin, member, viewer

	roleNames := make([]string, 0, len(roles))
	for _, r := range roles {
		roleNames = append(roleNames, r.Name)
	}
	assert.True(t, slices.Contains(roleNames, "owner"))
	assert.True(t, slices.Contains(roleNames, "admin"))
	assert.True(t, slices.Contains(roleNames, "member"))
	assert.True(t, slices.Contains(roleNames, "viewer"))
}

func TestCreateOrganization_DuplicateSlug(t *testing.T) {
	db := setupRBACTestDB(t)
	svc := rbac.NewService(db)
	owner := createTestUser(t, db, "owner2@example.com")

	_, err := svc.CreateOrganization(owner.ID, "Org One", "my-org", "First org")
	require.NoError(t, err)

	_, err = svc.CreateOrganization(owner.ID, "Org Two", "my-org", "Duplicate slug")
	require.Error(t, err)
	assert.ErrorIs(t, err, rbac.ErrSlugTaken)
}

// --- Permission Checking ---

func TestCheckPermission_OwnerHasAll(t *testing.T) {
	db := setupRBACTestDB(t)
	svc := rbac.NewService(db)
	owner := createTestUser(t, db, "perm-owner@example.com")

	org, err := svc.CreateOrganization(owner.ID, "Perm Org", "perm-org", "")
	require.NoError(t, err)

	// Owner should have all permissions
	testPermissions := []struct {
		resource string
		action   string
	}{
		{"packages", "read"},
		{"packages", "write"},
		{"packages", "delete"},
		{"alerts", "read"},
		{"alerts", "write"},
		{"org", "read"},
		{"org", "write"},
		{"org", "delete"},
		{"members", "read"},
		{"members", "invite"},
		{"members", "remove"},
		{"audit", "read"},
	}

	for _, perm := range testPermissions {
		err := svc.CheckPermission(owner.ID, org.ID, perm.resource, perm.action)
		assert.NoError(t, err, "owner should have %s:%s", perm.resource, perm.action)
	}
}

func TestCheckPermission_ViewerReadOnly(t *testing.T) {
	db := setupRBACTestDB(t)
	svc := rbac.NewService(db)
	owner := createTestUser(t, db, "viewer-owner@example.com")
	viewer := createTestUser(t, db, "viewer@example.com")

	org, err := svc.CreateOrganization(owner.ID, "Viewer Org", "viewer-org", "")
	require.NoError(t, err)

	// Get the viewer role
	roles, err := svc.GetOrgRoles(org.ID)
	require.NoError(t, err)
	var viewerRole *persistent.Role
	for i := range roles {
		if roles[i].Name == persistent.RoleViewer {
			viewerRole = &roles[i]
			break
		}
	}
	require.NotNil(t, viewerRole)

	// Invite viewer
	_, rawToken, err := svc.InviteMember(org.ID, viewer.Email, viewerRole.ID, owner.ID)
	require.NoError(t, err)

	_, err = svc.AcceptInvitation(rawToken, viewer.ID, viewer.Email)
	require.NoError(t, err)

	// Viewer should have read permissions
	assert.NoError(t, svc.CheckPermission(viewer.ID, org.ID, "packages", "read"))
	assert.NoError(t, svc.CheckPermission(viewer.ID, org.ID, "alerts", "read"))
	assert.NoError(t, svc.CheckPermission(viewer.ID, org.ID, "releases", "read"))
	assert.NoError(t, svc.CheckPermission(viewer.ID, org.ID, "settings", "read"))

	// Viewer should NOT have write permissions
	assert.ErrorIs(t, svc.CheckPermission(viewer.ID, org.ID, "packages", "write"), rbac.ErrPermissionDenied)
	assert.ErrorIs(t, svc.CheckPermission(viewer.ID, org.ID, "org", "write"), rbac.ErrPermissionDenied)
	assert.ErrorIs(t, svc.CheckPermission(viewer.ID, org.ID, "org", "delete"), rbac.ErrPermissionDenied)
	assert.ErrorIs(t, svc.CheckPermission(viewer.ID, org.ID, "members", "invite"), rbac.ErrPermissionDenied)
}

// --- Invite & Accept ---

func TestInviteMember_AcceptInvitation(t *testing.T) {
	db := setupRBACTestDB(t)
	svc := rbac.NewService(db)
	owner := createTestUser(t, db, "invite-owner@example.com")
	member := createTestUser(t, db, "invitee@example.com")

	org, err := svc.CreateOrganization(owner.ID, "Invite Org", "invite-org", "")
	require.NoError(t, err)

	// Get the member role
	roles, err := svc.GetOrgRoles(org.ID)
	require.NoError(t, err)
	var memberRole *persistent.Role
	for i := range roles {
		if roles[i].Name == persistent.RoleMember {
			memberRole = &roles[i]
			break
		}
	}
	require.NotNil(t, memberRole)

	// Invite
	invitation, rawToken, err := svc.InviteMember(org.ID, member.Email, memberRole.ID, owner.ID)
	require.NoError(t, err)
	assert.NotEmpty(t, invitation.TokenHash)
	assert.NotEmpty(t, rawToken)

	// Accept
	membership, err := svc.AcceptInvitation(rawToken, member.ID, member.Email)
	require.NoError(t, err)
	assert.Equal(t, org.ID, membership.OrgID)
	assert.Equal(t, member.ID, membership.UserID)
	assert.Equal(t, memberRole.ID, membership.RoleID)
}

func TestInviteMember_AlreadyMember(t *testing.T) {
	db := setupRBACTestDB(t)
	svc := rbac.NewService(db)
	owner := createTestUser(t, db, "dup-owner@example.com")
	member := createTestUser(t, db, "dup-member@example.com")

	org, err := svc.CreateOrganization(owner.ID, "Dup Org", "dup-org", "")
	require.NoError(t, err)

	roles, err := svc.GetOrgRoles(org.ID)
	require.NoError(t, err)
	var memberRole *persistent.Role
	for i := range roles {
		if roles[i].Name == persistent.RoleMember {
			memberRole = &roles[i]
			break
		}
	}
	require.NotNil(t, memberRole)

	// First invitation - accept
	_, rawToken1, err := svc.InviteMember(org.ID, member.Email, memberRole.ID, owner.ID)
	require.NoError(t, err)
	_, err = svc.AcceptInvitation(rawToken1, member.ID, member.Email)
	require.NoError(t, err)

	// Second invitation - accept should fail
	_, rawToken2, err := svc.InviteMember(org.ID, member.Email, memberRole.ID, owner.ID)
	require.NoError(t, err)
	_, err = svc.AcceptInvitation(rawToken2, member.ID, member.Email)
	require.Error(t, err)
	assert.ErrorIs(t, err, rbac.ErrAlreadyMember)
}

// --- Remove Member ---

func TestRemoveMember_Success(t *testing.T) {
	db := setupRBACTestDB(t)
	svc := rbac.NewService(db)
	owner := createTestUser(t, db, "rm-owner@example.com")
	member := createTestUser(t, db, "rm-member@example.com")

	org, err := svc.CreateOrganization(owner.ID, "Remove Org", "remove-org", "")
	require.NoError(t, err)

	roles, err := svc.GetOrgRoles(org.ID)
	require.NoError(t, err)
	var memberRole *persistent.Role
	for i := range roles {
		if roles[i].Name == persistent.RoleMember {
			memberRole = &roles[i]
			break
		}
	}
	require.NotNil(t, memberRole)

	_, rawToken, err := svc.InviteMember(org.ID, member.Email, memberRole.ID, owner.ID)
	require.NoError(t, err)
	_, err = svc.AcceptInvitation(rawToken, member.ID, member.Email)
	require.NoError(t, err)

	// Remove the member
	err = svc.RemoveMember(org.ID, member.ID)
	require.NoError(t, err)

	// Verify member is removed
	_, err = svc.GetUserMembership(member.ID, org.ID)
	require.Error(t, err)
}

func TestRemoveMember_CannotRemoveOwner(t *testing.T) {
	db := setupRBACTestDB(t)
	svc := rbac.NewService(db)
	owner := createTestUser(t, db, "rm-owner2@example.com")

	org, err := svc.CreateOrganization(owner.ID, "Owner Remove Org", "owner-remove-org", "")
	require.NoError(t, err)

	err = svc.RemoveMember(org.ID, owner.ID)
	require.Error(t, err)
	assert.ErrorIs(t, err, rbac.ErrCannotRemoveOwner)
}

// --- Update Member Role ---

func TestUpdateMemberRole_Success(t *testing.T) {
	db := setupRBACTestDB(t)
	svc := rbac.NewService(db)
	owner := createTestUser(t, db, "role-owner@example.com")
	member := createTestUser(t, db, "role-member@example.com")

	org, err := svc.CreateOrganization(owner.ID, "Role Org", "role-org", "")
	require.NoError(t, err)

	roles, err := svc.GetOrgRoles(org.ID)
	require.NoError(t, err)

	var memberRole, adminRole *persistent.Role
	for i := range roles {
		switch roles[i].Name {
		case persistent.RoleMember:
			memberRole = &roles[i]
		case persistent.RoleAdmin:
			adminRole = &roles[i]
		}
	}
	require.NotNil(t, memberRole)
	require.NotNil(t, adminRole)

	// Invite as member
	_, rawToken, err := svc.InviteMember(org.ID, member.Email, memberRole.ID, owner.ID)
	require.NoError(t, err)
	_, err = svc.AcceptInvitation(rawToken, member.ID, member.Email)
	require.NoError(t, err)

	// Upgrade to admin
	updated, err := svc.UpdateMemberRole(org.ID, member.ID, adminRole.ID)
	require.NoError(t, err)
	assert.Equal(t, adminRole.ID, updated.RoleID)
}

func TestUpdateMemberRole_CannotChangeOwner(t *testing.T) {
	db := setupRBACTestDB(t)
	svc := rbac.NewService(db)
	owner := createTestUser(t, db, "role-owner2@example.com")

	org, err := svc.CreateOrganization(owner.ID, "Owner Role Org", "owner-role-org", "")
	require.NoError(t, err)

	roles, err := svc.GetOrgRoles(org.ID)
	require.NoError(t, err)
	var memberRole *persistent.Role
	for i := range roles {
		if roles[i].Name == persistent.RoleMember {
			memberRole = &roles[i]
			break
		}
	}
	require.NotNil(t, memberRole)

	_, err = svc.UpdateMemberRole(org.ID, owner.ID, memberRole.ID)
	require.Error(t, err)
	assert.ErrorIs(t, err, rbac.ErrCannotChangeOwner)
}

// --- Organization CRUD ---

func TestGetOrganization(t *testing.T) {
	db := setupRBACTestDB(t)
	svc := rbac.NewService(db)
	owner := createTestUser(t, db, "get-org@example.com")

	created, err := svc.CreateOrganization(owner.ID, "Get Org", "get-org", "desc")
	require.NoError(t, err)

	org, err := svc.GetOrganization(created.ID)
	require.NoError(t, err)
	assert.Equal(t, "Get Org", org.Name)
	assert.Equal(t, "get-org", org.Slug)
}

func TestGetOrganization_NotFound(t *testing.T) {
	db := setupRBACTestDB(t)
	svc := rbac.NewService(db)

	_, err := svc.GetOrganization(99999)
	require.Error(t, err)
	assert.ErrorIs(t, err, rbac.ErrOrgNotFound)
}

func TestGetUserOrganizations(t *testing.T) {
	db := setupRBACTestDB(t)
	svc := rbac.NewService(db)
	owner := createTestUser(t, db, "list-orgs@example.com")

	_, err := svc.CreateOrganization(owner.ID, "Org A", "org-a", "")
	require.NoError(t, err)
	_, err = svc.CreateOrganization(owner.ID, "Org B", "org-b", "")
	require.NoError(t, err)

	orgs, err := svc.GetUserOrganizations(owner.ID)
	require.NoError(t, err)
	assert.Len(t, orgs, 2)
}

func TestUpdateOrganization(t *testing.T) {
	db := setupRBACTestDB(t)
	svc := rbac.NewService(db)
	owner := createTestUser(t, db, "update-org@example.com")

	org, err := svc.CreateOrganization(owner.ID, "Old Name", "old-slug", "old desc")
	require.NoError(t, err)

	updated, err := svc.UpdateOrganization(org.ID, "New Name", "new-slug", "new desc")
	require.NoError(t, err)
	assert.Equal(t, "New Name", updated.Name)
	assert.Equal(t, "new-slug", updated.Slug)
	assert.Equal(t, "new desc", updated.Description)
}

func TestDeleteOrganization(t *testing.T) {
	db := setupRBACTestDB(t)
	svc := rbac.NewService(db)
	owner := createTestUser(t, db, "delete-org@example.com")

	org, err := svc.CreateOrganization(owner.ID, "Delete Org", "delete-org", "")
	require.NoError(t, err)

	err = svc.DeleteOrganization(org.ID)
	require.NoError(t, err)

	_, err = svc.GetOrganization(org.ID)
	require.Error(t, err)
}
