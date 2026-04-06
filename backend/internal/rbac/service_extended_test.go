package rbac_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/veilence/veilence-mx/backend/internal/models"
	"github.com/veilence/veilence-mx/backend/internal/rbac"
)

// =====================================================================
// GetOrgMembers
// =====================================================================

func TestGetOrgMembers_ReturnsAllMembers(t *testing.T) {
	db := setupRBACTestDB(t)
	svc := rbac.NewService(db)
	owner := createTestUser(t, db, "members-owner@example.com")
	member := createTestUser(t, db, "members-member@example.com")

	org, err := svc.CreateOrganization(owner.ID, "Members Org", "members-org", "")
	require.NoError(t, err)

	roles, err := svc.GetOrgRoles(org.ID)
	require.NoError(t, err)
	var memberRole *models.Role
	for i := range roles {
		if roles[i].Name == models.RoleMember {
			memberRole = &roles[i]
			break
		}
	}
	require.NotNil(t, memberRole)

	_, rawToken, err := svc.InviteMember(org.ID, member.Email, memberRole.ID, owner.ID)
	require.NoError(t, err)
	_, err = svc.AcceptInvitation(rawToken, member.ID, member.Email)
	require.NoError(t, err)

	members, err := svc.GetOrgMembers(org.ID)
	require.NoError(t, err)
	assert.Len(t, members, 2, "owner + invited member")
}

func TestGetOrgMembers_EmptyOrg(t *testing.T) {
	db := setupRBACTestDB(t)
	svc := rbac.NewService(db)

	// Query members for a non-existent org — should return empty, not error
	members, err := svc.GetOrgMembers(99999)
	require.NoError(t, err)
	assert.Empty(t, members)
}

// =====================================================================
// GetInvitationByToken
// =====================================================================

func TestGetInvitationByToken_Success(t *testing.T) {
	db := setupRBACTestDB(t)
	svc := rbac.NewService(db)
	owner := createTestUser(t, db, "invite-token-owner@example.com")

	org, err := svc.CreateOrganization(owner.ID, "Token Org", "token-org", "")
	require.NoError(t, err)

	roles, err := svc.GetOrgRoles(org.ID)
	require.NoError(t, err)
	var memberRole *models.Role
	for i := range roles {
		if roles[i].Name == models.RoleMember {
			memberRole = &roles[i]
			break
		}
	}
	require.NotNil(t, memberRole)

	_, rawToken, err := svc.InviteMember(org.ID, "invitee-token@example.com", memberRole.ID, owner.ID)
	require.NoError(t, err)

	invitation, err := svc.GetInvitationByToken(rawToken)
	require.NoError(t, err)
	assert.Equal(t, org.ID, invitation.OrgID)
	assert.Equal(t, "invitee-token@example.com", invitation.Email)
	assert.Nil(t, invitation.AcceptedAt)
}

func TestGetInvitationByToken_NotFound(t *testing.T) {
	db := setupRBACTestDB(t)
	svc := rbac.NewService(db)

	_, err := svc.GetInvitationByToken("nonexistent-token")
	require.Error(t, err)
	assert.ErrorIs(t, err, rbac.ErrInvitationNotFound)
}

// =====================================================================
// ListPendingInvitations
// =====================================================================

func TestListPendingInvitations_Success(t *testing.T) {
	db := setupRBACTestDB(t)
	svc := rbac.NewService(db)
	owner := createTestUser(t, db, "pending-owner@example.com")

	org, err := svc.CreateOrganization(owner.ID, "Pending Org", "pending-org", "")
	require.NoError(t, err)

	roles, err := svc.GetOrgRoles(org.ID)
	require.NoError(t, err)
	var memberRole *models.Role
	for i := range roles {
		if roles[i].Name == models.RoleMember {
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
	svc := rbac.NewService(db)
	owner := createTestUser(t, db, "exclude-owner@example.com")
	invitee := createTestUser(t, db, "exclude-invitee@example.com")

	org, err := svc.CreateOrganization(owner.ID, "Exclude Org", "exclude-org", "")
	require.NoError(t, err)

	roles, err := svc.GetOrgRoles(org.ID)
	require.NoError(t, err)
	var memberRole *models.Role
	for i := range roles {
		if roles[i].Name == models.RoleMember {
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
	svc := rbac.NewService(db)
	owner := createTestUser(t, db, "revoke-inv-owner@example.com")

	org, err := svc.CreateOrganization(owner.ID, "Revoke Inv Org", "revoke-inv-org", "")
	require.NoError(t, err)

	roles, err := svc.GetOrgRoles(org.ID)
	require.NoError(t, err)
	var memberRole *models.Role
	for i := range roles {
		if roles[i].Name == models.RoleMember {
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
	svc := rbac.NewService(db)

	err := svc.RevokeInvitation(1, 99999)
	require.Error(t, err)
	assert.ErrorIs(t, err, rbac.ErrInvitationNotFound)
}

func TestRevokeInvitation_CannotRevokeAccepted(t *testing.T) {
	db := setupRBACTestDB(t)
	svc := rbac.NewService(db)
	owner := createTestUser(t, db, "revoke-accepted-owner@example.com")
	invitee := createTestUser(t, db, "revoke-accepted-invitee@example.com")

	org, err := svc.CreateOrganization(owner.ID, "Revoke Accepted Org", "revoke-accepted-org", "")
	require.NoError(t, err)

	roles, err := svc.GetOrgRoles(org.ID)
	require.NoError(t, err)
	var memberRole *models.Role
	for i := range roles {
		if roles[i].Name == models.RoleMember {
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
	svc := rbac.NewService(db)

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
// InviteMember — cannot invite as owner
// =====================================================================

func TestInviteMember_CannotInviteAsOwner(t *testing.T) {
	db := setupRBACTestDB(t)
	svc := rbac.NewService(db)
	owner := createTestUser(t, db, "no-owner-invite@example.com")

	org, err := svc.CreateOrganization(owner.ID, "No Owner Inv", "no-owner-inv", "")
	require.NoError(t, err)

	roles, err := svc.GetOrgRoles(org.ID)
	require.NoError(t, err)
	var ownerRole *models.Role
	for i := range roles {
		if roles[i].Name == models.RoleOwner {
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
// AcceptInvitation — email mismatch
// =====================================================================

func TestAcceptInvitation_EmailMismatch(t *testing.T) {
	db := setupRBACTestDB(t)
	svc := rbac.NewService(db)
	owner := createTestUser(t, db, "mismatch-owner@example.com")
	wrongUser := createTestUser(t, db, "wrong-user@example.com")

	org, err := svc.CreateOrganization(owner.ID, "Mismatch Org", "mismatch-org", "")
	require.NoError(t, err)

	roles, err := svc.GetOrgRoles(org.ID)
	require.NoError(t, err)
	var memberRole *models.Role
	for i := range roles {
		if roles[i].Name == models.RoleMember {
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
// AcceptInvitation — expired invitation
// =====================================================================

func TestAcceptInvitation_Expired(t *testing.T) {
	db := setupRBACTestDB(t)
	svc := rbac.NewService(db)
	owner := createTestUser(t, db, "expired-inv-owner@example.com")
	invitee := createTestUser(t, db, "expired-inv@example.com")

	org, err := svc.CreateOrganization(owner.ID, "Expired Inv Org", "expired-inv-org", "")
	require.NoError(t, err)

	roles, err := svc.GetOrgRoles(org.ID)
	require.NoError(t, err)
	var memberRole *models.Role
	for i := range roles {
		if roles[i].Name == models.RoleMember {
			memberRole = &roles[i]
			break
		}
	}
	require.NotNil(t, memberRole)

	invitation, rawToken, err := svc.InviteMember(org.ID, invitee.Email, memberRole.ID, owner.ID)
	require.NoError(t, err)

	// Manually expire the invitation
	db.Model(&models.Invitation{}).Where("id = ?", invitation.ID).Update("expires_at", "2020-01-01 00:00:00")

	_, err = svc.AcceptInvitation(rawToken, invitee.ID, invitee.Email)
	require.Error(t, err)
	assert.ErrorIs(t, err, rbac.ErrInvitationExpired)
}

// =====================================================================
// UpdateOrganization — slug uniqueness on update
// =====================================================================

func TestUpdateOrganization_DuplicateSlugOnUpdate(t *testing.T) {
	db := setupRBACTestDB(t)
	svc := rbac.NewService(db)
	owner := createTestUser(t, db, "dup-slug-update@example.com")

	org1, err := svc.CreateOrganization(owner.ID, "Org One", "org-one-slug", "")
	require.NoError(t, err)

	_, err = svc.CreateOrganization(owner.ID, "Org Two", "org-two-slug", "")
	require.NoError(t, err)

	// Try to update org1's slug to org2's slug
	_, err = svc.UpdateOrganization(org1.ID, "Org One Updated", "org-two-slug", "")
	require.Error(t, err)
	assert.ErrorIs(t, err, rbac.ErrSlugTaken)
}

// =====================================================================
// UpdateMemberRole — role not found
// =====================================================================

func TestUpdateMemberRole_RoleNotFound(t *testing.T) {
	db := setupRBACTestDB(t)
	svc := rbac.NewService(db)
	owner := createTestUser(t, db, "role-nf-owner@example.com")
	member := createTestUser(t, db, "role-nf-member@example.com")

	org, err := svc.CreateOrganization(owner.ID, "Role NF Org", "role-nf-org", "")
	require.NoError(t, err)

	roles, err := svc.GetOrgRoles(org.ID)
	require.NoError(t, err)
	var memberRole *models.Role
	for i := range roles {
		if roles[i].Name == models.RoleMember {
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
	_, err = svc.UpdateMemberRole(org.ID, member.ID, 99999)
	require.Error(t, err)
	assert.ErrorIs(t, err, rbac.ErrRoleNotFound)
}

// =====================================================================
// DeleteOrganization — not found
// =====================================================================

func TestDeleteOrganization_NotFound(t *testing.T) {
	db := setupRBACTestDB(t)
	svc := rbac.NewService(db)

	err := svc.DeleteOrganization(99999)
	require.Error(t, err)
	assert.ErrorIs(t, err, rbac.ErrOrgNotFound)
}

// =====================================================================
// RemoveMember — not found
// =====================================================================

func TestRemoveMember_NotFound(t *testing.T) {
	db := setupRBACTestDB(t)
	svc := rbac.NewService(db)
	owner := createTestUser(t, db, "rm-nf-owner@example.com")

	org, err := svc.CreateOrganization(owner.ID, "RM NF Org", "rm-nf-org", "")
	require.NoError(t, err)

	err = svc.RemoveMember(org.ID, 99999)
	require.Error(t, err)
	assert.ErrorIs(t, err, rbac.ErrMemberNotFound)
}

// =====================================================================
// GetUserMembership
// =====================================================================

func TestGetUserMembership_Success(t *testing.T) {
	db := setupRBACTestDB(t)
	svc := rbac.NewService(db)
	owner := createTestUser(t, db, "membership-owner@example.com")

	org, err := svc.CreateOrganization(owner.ID, "Membership Org", "membership-org", "")
	require.NoError(t, err)

	membership, err := svc.GetUserMembership(owner.ID, org.ID)
	require.NoError(t, err)
	assert.Equal(t, owner.ID, membership.UserID)
	assert.Equal(t, org.ID, membership.OrgID)
	assert.NotNil(t, membership.Role)
	assert.Equal(t, models.RoleOwner, membership.Role.Name)
}

func TestGetUserMembership_NotFound(t *testing.T) {
	db := setupRBACTestDB(t)
	svc := rbac.NewService(db)

	_, err := svc.GetUserMembership(99999, 99999)
	require.Error(t, err)
	assert.ErrorIs(t, err, rbac.ErrMemberNotFound)
}

// =====================================================================
// CheckPermission — non-member
// =====================================================================

func TestCheckPermission_NonMember(t *testing.T) {
	db := setupRBACTestDB(t)
	svc := rbac.NewService(db)
	owner := createTestUser(t, db, "nonmember-owner@example.com")
	outsider := createTestUser(t, db, "outsider@example.com")

	org, err := svc.CreateOrganization(owner.ID, "NonMember Org", "nonmember-org", "")
	require.NoError(t, err)

	// Outsider (not a member) should be denied
	err = svc.CheckPermission(outsider.ID, org.ID, "packages", "read")
	assert.ErrorIs(t, err, rbac.ErrPermissionDenied)
}

// =====================================================================
// InviteMember — invalid role (wrong org)
// =====================================================================

func TestInviteMember_RoleFromDifferentOrg(t *testing.T) {
	db := setupRBACTestDB(t)
	svc := rbac.NewService(db)
	owner := createTestUser(t, db, "cross-org-owner@example.com")

	org1, err := svc.CreateOrganization(owner.ID, "Org A", "cross-org-a", "")
	require.NoError(t, err)

	org2, err := svc.CreateOrganization(owner.ID, "Org B", "cross-org-b", "")
	require.NoError(t, err)

	// Get a role from org2
	roles, err := svc.GetOrgRoles(org2.ID)
	require.NoError(t, err)
	var org2MemberRole *models.Role
	for i := range roles {
		if roles[i].Name == models.RoleMember {
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
