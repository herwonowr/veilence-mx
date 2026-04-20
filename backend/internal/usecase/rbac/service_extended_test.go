package rbac_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/veilence/veilence-mx/backend/internal/entity"
	"github.com/veilence/veilence-mx/backend/internal/repo/persistent"
	"github.com/veilence/veilence-mx/backend/internal/usecase/rbac"
)

// =====================================================================
// GetWorkspaceMembers
// =====================================================================

func TestGetWorkspaceMembers_ReturnsAllMembers(t *testing.T) {
	db := setupRBACTestDB(t)
	svc := rbac.NewService(persistent.NewRBACRepo(db))
	owner := createTestUser(t, db, "members-owner@example.com")
	member := createTestUser(t, db, "members-member@example.com")

	org, err := svc.CreateWorkspace(owner.ID, "Members Org", "members-org", "")
	require.NoError(t, err)

	roles, err := svc.GetWorkspaceRoles(org.ID)
	require.NoError(t, err)
	var memberRole *entity.Role
	for i := range roles {
		if roles[i].Name == entity.RoleMember {
			memberRole = &roles[i]
			break
		}
	}
	require.NotNil(t, memberRole)

	_, rawToken, err := svc.InviteMember(org.ID, member.Email, memberRole.ID, owner.ID)
	require.NoError(t, err)
	_, err = svc.AcceptInvitation(rawToken, member.ID, member.Email)
	require.NoError(t, err)

	members, err := svc.GetWorkspaceMembers(org.ID)
	require.NoError(t, err)
	assert.Len(t, members, 2, "owner + invited member")
}

func TestGetWorkspaceMembers_EmptyOrg(t *testing.T) {
	db := setupRBACTestDB(t)
	svc := rbac.NewService(persistent.NewRBACRepo(db))

	// Query members for a non-existent org - should return empty, not error
	members, err := svc.GetWorkspaceMembers("01935d5a-0000-7000-8000-00000001869f")
	require.NoError(t, err)
	assert.Empty(t, members)
}

// =====================================================================
// GetInvitationByToken
// =====================================================================

func TestGetInvitationByToken_Success(t *testing.T) {
	db := setupRBACTestDB(t)
	svc := rbac.NewService(persistent.NewRBACRepo(db))
	owner := createTestUser(t, db, "invite-token-owner@example.com")

	org, err := svc.CreateWorkspace(owner.ID, "Token Org", "token-org", "")
	require.NoError(t, err)

	roles, err := svc.GetWorkspaceRoles(org.ID)
	require.NoError(t, err)
	var memberRole *entity.Role
	for i := range roles {
		if roles[i].Name == entity.RoleMember {
			memberRole = &roles[i]
			break
		}
	}
	require.NotNil(t, memberRole)

	_, rawToken, err := svc.InviteMember(org.ID, "invitee-token@example.com", memberRole.ID, owner.ID)
	require.NoError(t, err)

	invitation, err := svc.GetInvitationByToken(rawToken)
	require.NoError(t, err)
	assert.Equal(t, org.ID, invitation.WorkspaceID)
	assert.Equal(t, "invitee-token@example.com", invitation.Email)
	assert.Nil(t, invitation.AcceptedAt)
}

func TestGetInvitationByToken_NotFound(t *testing.T) {
	db := setupRBACTestDB(t)
	svc := rbac.NewService(persistent.NewRBACRepo(db))

	_, err := svc.GetInvitationByToken("nonexistent-token")
	require.Error(t, err)
	assert.ErrorIs(t, err, rbac.ErrInvitationNotFound)
}

// =====================================================================
// ListPendingInvitations
// =====================================================================

func TestListPendingInvitations_Success(t *testing.T) {
	db := setupRBACTestDB(t)
	svc := rbac.NewService(persistent.NewRBACRepo(db))
	owner := createTestUser(t, db, "pending-owner@example.com")

	org, err := svc.CreateWorkspace(owner.ID, "Pending Org", "pending-org", "")
	require.NoError(t, err)

	roles, err := svc.GetWorkspaceRoles(org.ID)
	require.NoError(t, err)
	var memberRole *entity.Role
	for i := range roles {
		if roles[i].Name == entity.RoleMember {
			memberRole = &roles[i]
			break
		}
	}
	require.NotNil(t, memberRole)

	// Create two invitations
	_, _, err = svc.InviteMember(org.ID, "pending1@example.com", memberRole.ID, owner.ID)
	require.NoError(t, err)
	_, _, err = svc.InviteMember(org.ID, "pending2@example.com", memberRole.ID, owner.ID)
	require.NoError(t, err)

	invitations, err := svc.ListPendingInvitations(org.ID)
	require.NoError(t, err)
	assert.Len(t, invitations, 2)
}

func TestListPendingInvitations_ExcludesAccepted(t *testing.T) {
	db := setupRBACTestDB(t)
	svc := rbac.NewService(persistent.NewRBACRepo(db))
	owner := createTestUser(t, db, "exclude-owner@example.com")
	invitee := createTestUser(t, db, "exclude-invitee@example.com")

	org, err := svc.CreateWorkspace(owner.ID, "Exclude Org", "exclude-org", "")
	require.NoError(t, err)

	roles, err := svc.GetWorkspaceRoles(org.ID)
	require.NoError(t, err)
	var memberRole *entity.Role
	for i := range roles {
		if roles[i].Name == entity.RoleMember {
			memberRole = &roles[i]
			break
		}
	}
	require.NotNil(t, memberRole)

	// Create and accept one invitation
	_, rawToken, err := svc.InviteMember(org.ID, invitee.Email, memberRole.ID, owner.ID)
	require.NoError(t, err)
	_, err = svc.AcceptInvitation(rawToken, invitee.ID, invitee.Email)
	require.NoError(t, err)

	// Create a pending invitation
	_, _, err = svc.InviteMember(org.ID, "still-pending@example.com", memberRole.ID, owner.ID)
	require.NoError(t, err)

	invitations, err := svc.ListPendingInvitations(org.ID)
	require.NoError(t, err)
	assert.Len(t, invitations, 1, "only pending invitation should be listed")
	assert.Equal(t, "still-pending@example.com", invitations[0].Email)
}

// =====================================================================
// RevokeInvitation
// =====================================================================

func TestRevokeInvitation_Success(t *testing.T) {
	db := setupRBACTestDB(t)
	svc := rbac.NewService(persistent.NewRBACRepo(db))
	owner := createTestUser(t, db, "revoke-inv-owner@example.com")

	org, err := svc.CreateWorkspace(owner.ID, "Revoke Inv Org", "revoke-inv-org", "")
	require.NoError(t, err)

	roles, err := svc.GetWorkspaceRoles(org.ID)
	require.NoError(t, err)
	var memberRole *entity.Role
	for i := range roles {
		if roles[i].Name == entity.RoleMember {
			memberRole = &roles[i]
			break
		}
	}
	require.NotNil(t, memberRole)

	invitation, _, err := svc.InviteMember(org.ID, "to-revoke-inv@example.com", memberRole.ID, owner.ID)
	require.NoError(t, err)

	err = svc.RevokeInvitation(org.ID, invitation.ID)
	require.NoError(t, err)

	// Should no longer be in pending list
	invitations, err := svc.ListPendingInvitations(org.ID)
	require.NoError(t, err)
	assert.Empty(t, invitations)
}

func TestRevokeInvitation_NotFound(t *testing.T) {
	db := setupRBACTestDB(t)
	svc := rbac.NewService(persistent.NewRBACRepo(db))

	err := svc.RevokeInvitation("01935d5a-0000-7000-8000-000000000001", "01935d5a-0000-7000-8000-00000001869f")
	require.Error(t, err)
	assert.ErrorIs(t, err, rbac.ErrInvitationNotFound)
}

func TestRevokeInvitation_CannotRevokeAccepted(t *testing.T) {
	db := setupRBACTestDB(t)
	svc := rbac.NewService(persistent.NewRBACRepo(db))
	owner := createTestUser(t, db, "revoke-accepted-owner@example.com")
	invitee := createTestUser(t, db, "revoke-accepted-invitee@example.com")

	org, err := svc.CreateWorkspace(owner.ID, "Revoke Accepted Org", "revoke-accepted-org", "")
	require.NoError(t, err)

	roles, err := svc.GetWorkspaceRoles(org.ID)
	require.NoError(t, err)
	var memberRole *entity.Role
	for i := range roles {
		if roles[i].Name == entity.RoleMember {
			memberRole = &roles[i]
			break
		}
	}
	require.NotNil(t, memberRole)

	invitation, rawToken, err := svc.InviteMember(org.ID, invitee.Email, memberRole.ID, owner.ID)
	require.NoError(t, err)
	_, err = svc.AcceptInvitation(rawToken, invitee.ID, invitee.Email)
	require.NoError(t, err)

	// Cannot revoke an accepted invitation
	err = svc.RevokeInvitation(org.ID, invitation.ID)
	require.Error(t, err)
	assert.ErrorIs(t, err, rbac.ErrInvitationNotFound)
}

// =====================================================================
// GetAllPermissions
// =====================================================================

func TestGetAllPermissions(t *testing.T) {
	db := setupRBACTestDB(t)
	svc := rbac.NewService(persistent.NewRBACRepo(db))

	perms, err := svc.GetAllPermissions()
	require.NoError(t, err)
	assert.NotEmpty(t, perms)
	// Should contain at least packages:read
	found := false
	for _, p := range perms {
		if p.Resource == "packages" && p.Action == "read" {
			found = true
			break
		}
	}
	assert.True(t, found, "should contain packages:read permission")
}

// =====================================================================
// InviteMember - cannot invite as owner
// =====================================================================

func TestInviteMember_CannotInviteAsOwner(t *testing.T) {
	db := setupRBACTestDB(t)
	svc := rbac.NewService(persistent.NewRBACRepo(db))
	owner := createTestUser(t, db, "no-owner-invite@example.com")

	org, err := svc.CreateWorkspace(owner.ID, "No Owner Inv", "no-owner-inv", "")
	require.NoError(t, err)

	roles, err := svc.GetWorkspaceRoles(org.ID)
	require.NoError(t, err)
	var ownerRole *entity.Role
	for i := range roles {
		if roles[i].Name == entity.RoleOwner {
			ownerRole = &roles[i]
			break
		}
	}
	require.NotNil(t, ownerRole)

	_, _, err = svc.InviteMember(org.ID, "new-owner@example.com", ownerRole.ID, owner.ID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot invite user as owner")
}

// =====================================================================
// AcceptInvitation - email mismatch
// =====================================================================

func TestAcceptInvitation_EmailMismatch(t *testing.T) {
	db := setupRBACTestDB(t)
	svc := rbac.NewService(persistent.NewRBACRepo(db))
	owner := createTestUser(t, db, "mismatch-owner@example.com")
	wrongUser := createTestUser(t, db, "wrong-user@example.com")

	org, err := svc.CreateWorkspace(owner.ID, "Mismatch Org", "mismatch-org", "")
	require.NoError(t, err)

	roles, err := svc.GetWorkspaceRoles(org.ID)
	require.NoError(t, err)
	var memberRole *entity.Role
	for i := range roles {
		if roles[i].Name == entity.RoleMember {
			memberRole = &roles[i]
			break
		}
	}
	require.NotNil(t, memberRole)

	_, rawToken, err := svc.InviteMember(org.ID, "correct-email@example.com", memberRole.ID, owner.ID)
	require.NoError(t, err)

	// Wrong user (different email) tries to accept
	_, err = svc.AcceptInvitation(rawToken, wrongUser.ID, wrongUser.Email)
	require.Error(t, err)
	assert.ErrorIs(t, err, rbac.ErrInvitationEmailMismatch)
}

// =====================================================================
// AcceptInvitation - expired invitation
// =====================================================================

func TestAcceptInvitation_Expired(t *testing.T) {
	db := setupRBACTestDB(t)
	svc := rbac.NewService(persistent.NewRBACRepo(db))
	owner := createTestUser(t, db, "expired-inv-owner@example.com")
	invitee := createTestUser(t, db, "expired-inv@example.com")

	org, err := svc.CreateWorkspace(owner.ID, "Expired Inv Org", "expired-inv-org", "")
	require.NoError(t, err)

	roles, err := svc.GetWorkspaceRoles(org.ID)
	require.NoError(t, err)
	var memberRole *entity.Role
	for i := range roles {
		if roles[i].Name == entity.RoleMember {
			memberRole = &roles[i]
			break
		}
	}
	require.NotNil(t, memberRole)

	invitation, rawToken, err := svc.InviteMember(org.ID, invitee.Email, memberRole.ID, owner.ID)
	require.NoError(t, err)

	// Manually expire the invitation
	db.Model(&persistent.Invitation{}).Where("id = ?", invitation.ID).Update("expires_at", "2020-01-01 00:00:00")

	_, err = svc.AcceptInvitation(rawToken, invitee.ID, invitee.Email)
	require.Error(t, err)
	assert.ErrorIs(t, err, rbac.ErrInvitationExpired)
}

// =====================================================================
// UpdateWorkspace - slug uniqueness on update
// =====================================================================

func TestUpdateWorkspace_DuplicateSlugOnUpdate(t *testing.T) {
	db := setupRBACTestDB(t)
	svc := rbac.NewService(persistent.NewRBACRepo(db))
	owner := createTestUser(t, db, "dup-slug-update@example.com")

	org1, err := svc.CreateWorkspace(owner.ID, "Workspace One", "workspace-one-slug", "")
	require.NoError(t, err)

	_, err = svc.CreateWorkspace(owner.ID, "Workspace Two", "org-two-slug", "")
	require.NoError(t, err)

	// Try to update org1's slug to org2's slug
	_, err = svc.UpdateWorkspace(org1.ID, "Org One Updated", "org-two-slug", "")
	require.Error(t, err)
	assert.ErrorIs(t, err, rbac.ErrSlugTaken)
}

// =====================================================================
// UpdateMemberRole - role not found
// =====================================================================

func TestUpdateMemberRole_RoleNotFound(t *testing.T) {
	db := setupRBACTestDB(t)
	svc := rbac.NewService(persistent.NewRBACRepo(db))
	owner := createTestUser(t, db, "role-nf-owner@example.com")
	member := createTestUser(t, db, "role-nf-member@example.com")

	org, err := svc.CreateWorkspace(owner.ID, "Role NF Org", "role-nf-org", "")
	require.NoError(t, err)

	roles, err := svc.GetWorkspaceRoles(org.ID)
	require.NoError(t, err)
	var memberRole *entity.Role
	for i := range roles {
		if roles[i].Name == entity.RoleMember {
			memberRole = &roles[i]
			break
		}
	}
	require.NotNil(t, memberRole)

	_, rawToken, err := svc.InviteMember(org.ID, member.Email, memberRole.ID, owner.ID)
	require.NoError(t, err)
	_, err = svc.AcceptInvitation(rawToken, member.ID, member.Email)
	require.NoError(t, err)

	// Try to update to a non-existent role
	_, err = svc.UpdateMemberRole(org.ID, member.ID, "01935d5a-0000-7000-8000-00000001869f")
	require.Error(t, err)
	assert.ErrorIs(t, err, rbac.ErrRoleNotFound)
}

// =====================================================================
// DeleteWorkspace - not found
// =====================================================================

func TestDeleteWorkspace_NotFound(t *testing.T) {
	db := setupRBACTestDB(t)
	svc := rbac.NewService(persistent.NewRBACRepo(db))

	err := svc.DeleteWorkspace("01935d5a-0000-7000-8000-00000001869f")
	require.Error(t, err)
	assert.ErrorIs(t, err, rbac.ErrWorkspaceNotFound)
}

// =====================================================================
// RemoveMember - not found
// =====================================================================

func TestRemoveMember_NotFound(t *testing.T) {
	db := setupRBACTestDB(t)
	svc := rbac.NewService(persistent.NewRBACRepo(db))
	owner := createTestUser(t, db, "rm-nf-owner@example.com")

	org, err := svc.CreateWorkspace(owner.ID, "RM NF Org", "rm-nf-org", "")
	require.NoError(t, err)

	err = svc.RemoveMember(org.ID, "01935d5a-0000-7000-8000-00000001869f")
	require.Error(t, err)
	assert.ErrorIs(t, err, rbac.ErrMemberNotFound)
}

// =====================================================================
// GetUserMembership
// =====================================================================

func TestGetUserMembership_Success(t *testing.T) {
	db := setupRBACTestDB(t)
	svc := rbac.NewService(persistent.NewRBACRepo(db))
	owner := createTestUser(t, db, "membership-owner@example.com")

	org, err := svc.CreateWorkspace(owner.ID, "Membership Org", "membership-org", "")
	require.NoError(t, err)

	membership, err := svc.GetUserMembership(owner.ID, org.ID)
	require.NoError(t, err)
	assert.Equal(t, owner.ID, membership.UserID)
	assert.Equal(t, org.ID, membership.WorkspaceID)
	assert.NotNil(t, membership.Role)
	assert.Equal(t, entity.RoleOwner, membership.Role.Name)
}

func TestGetUserMembership_NotFound(t *testing.T) {
	db := setupRBACTestDB(t)
	svc := rbac.NewService(persistent.NewRBACRepo(db))

	_, err := svc.GetUserMembership("01935d5a-0000-7000-8000-00000001869f", "01935d5a-0000-7000-8000-00000001869e")
	require.Error(t, err)
	assert.ErrorIs(t, err, rbac.ErrMemberNotFound)
}

// =====================================================================
// CheckPermission - non-member
// =====================================================================

func TestCheckPermission_NonMember(t *testing.T) {
	db := setupRBACTestDB(t)
	svc := rbac.NewService(persistent.NewRBACRepo(db))
	owner := createTestUser(t, db, "nonmember-owner@example.com")
	outsider := createTestUser(t, db, "outsider@example.com")

	org, err := svc.CreateWorkspace(owner.ID, "NonMember Org", "nonmember-org", "")
	require.NoError(t, err)

	// Outsider (not a member) should be denied
	err = svc.CheckPermission(outsider.ID, org.ID, "packages", "read")
	assert.ErrorIs(t, err, rbac.ErrPermissionDenied)
}

// =====================================================================
// InviteMember - invalid role (wrong org)
// =====================================================================

func TestInviteMember_RoleFromDifferentOrg(t *testing.T) {
	db := setupRBACTestDB(t)
	svc := rbac.NewService(persistent.NewRBACRepo(db))
	owner := createTestUser(t, db, "cross-org-owner@example.com")

	org1, err := svc.CreateWorkspace(owner.ID, "Workspace A", "cross-org-a", "")
	require.NoError(t, err)

	org2, err := svc.CreateWorkspace(owner.ID, "Workspace B", "cross-org-b", "")
	require.NoError(t, err)

	// Get a role from org2
	roles, err := svc.GetWorkspaceRoles(org2.ID)
	require.NoError(t, err)
	var org2MemberRole *entity.Role
	for i := range roles {
		if roles[i].Name == entity.RoleMember {
			org2MemberRole = &roles[i]
			break
		}
	}
	require.NotNil(t, org2MemberRole)

	// Try to invite to org1 using org2's role
	_, _, err = svc.InviteMember(org1.ID, "cross@example.com", org2MemberRole.ID, owner.ID)
	require.Error(t, err)
	assert.ErrorIs(t, err, rbac.ErrRoleNotFound)
}
