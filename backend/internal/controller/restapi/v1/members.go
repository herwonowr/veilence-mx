package v1

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	validation "github.com/veilence/veilence-mx/backend/internal/controller/restapi/v1/request"
	"github.com/veilence/veilence-mx/backend/internal/controller/restapi/v1/response"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/veilence/veilence-mx/backend/internal/entity"
	"github.com/veilence/veilence-mx/backend/internal/usecase/auth"
	"github.com/veilence/veilence-mx/backend/internal/usecase/rbac"
)

type inviteMemberRequest struct {
	Email  string `json:"email"`
	RoleID string `json:"roleId"`
}

type addMemberRequest struct {
	Email     string `json:"email"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	RoleID    string `json:"roleId"`
	Password  string `json:"password,omitempty"`
}

type updateMemberRoleRequest struct {
	RoleID string `json:"roleId"`
}

// invitationResponse is the response shape for invitation records.
// It provides camelCase JSON keys, a computed status field, and omits the token hash.
type invitationResponse struct {
	ID          string    `json:"id"`
	WorkspaceID string    `json:"workspaceId"`
	Email       string    `json:"email"`
	RoleID      string    `json:"roleId"`
	InvitedBy   string    `json:"invitedBy"`
	Status      string    `json:"status"`
	ExpiresAt   time.Time `json:"expiresAt"`
	CreatedAt   time.Time `json:"createdAt"`
}

// toInvitationResponse converts an entity.Invitation to a response with computed status.
func toInvitationResponse(inv entity.Invitation) invitationResponse {
	status := "pending"
	if inv.AcceptedAt != nil {
		status = "accepted"
	} else if inv.DeclinedAt != nil {
		status = "declined"
	} else if time.Now().After(inv.ExpiresAt) {
		status = "expired"
	}
	return invitationResponse{
		ID:          inv.ID,
		WorkspaceID: inv.WorkspaceID,
		Email:       inv.Email,
		RoleID:      inv.RoleID,
		InvitedBy:   inv.InvitedBy,
		Status:      status,
		ExpiresAt:   inv.ExpiresAt,
		CreatedAt:   inv.CreatedAt,
	}
}

// flatMember is the flattened response shape for workspace members.
// The frontend expects email, firstName, and lastName at the top level
// instead of nested under a "user" object.
type flatMember struct {
	ID          string                `json:"id"`
	WorkspaceID string                `json:"workspaceId"`
	UserID      string                `json:"userId"`
	RoleID      string                `json:"roleId"`
	Role        response.RoleResponse `json:"role"`
	JoinedAt    time.Time             `json:"joinedAt"`
	Email       string                `json:"email"`
	FirstName   string                `json:"firstName"`
	LastName    string                `json:"lastName"`
}

// ListMembers handles GET /api/workspaces/{workspaceId}/members - lists workspace members.
func (h *WorkspaceHandlers) ListMembers(w http.ResponseWriter, r *http.Request) {
	workspaceID := rbac.WorkspaceIDFromContext(r.Context())
	if workspaceID == "" {
		respondError(w, http.StatusBadRequest, "Workspace context required")
		return
	}

	members, err := h.RBAC.GetWorkspaceMembers(r.Context(), workspaceID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to list members")
		return
	}

	// Flatten the response: promote User fields to the top level so the
	// frontend receives {email, firstName, lastName} directly instead of
	// a nested user object.
	flat := make([]flatMember, len(members))
	for i, m := range members {
		var roleResp response.RoleResponse
		if m.Role != nil {
			perms := make([]response.PermissionResponse, len(m.Role.Permissions))
			for j, p := range m.Role.Permissions {
				perms[j] = response.PermissionResponse{
					ID:       p.ID,
					Resource: p.Resource,
					Action:   p.Action,
				}
			}
			roleResp = response.RoleResponse{
				ID:          m.Role.ID,
				WorkspaceID: m.Role.WorkspaceID,
				Name:        m.Role.Name,
				Description: m.Role.Description,
				IsSystem:    m.Role.IsSystem,
				CreatedAt:   m.Role.CreatedAt,
				UpdatedAt:   m.Role.UpdatedAt,
				Permissions: perms,
			}
		}
		var email, firstName, lastName string
		if m.User != nil {
			email = m.User.Email
			firstName = m.User.FirstName
			lastName = m.User.LastName
		}
		flat[i] = flatMember{
			ID:          m.ID,
			WorkspaceID: m.WorkspaceID,
			UserID:      m.UserID,
			RoleID:      m.RoleID,
			Role:        roleResp,
			JoinedAt:    m.JoinedAt,
			Email:       email,
			FirstName:   firstName,
			LastName:    lastName,
		}
	}

	respondJSON(w, http.StatusOK, flat, nil)
}

// AddMember handles POST /api/workspaces/{workspaceId}/members - directly adds a user to the workspace.
// This is the admin-driven flow when registration is disabled: creates a user account and workspace membership in one step.
func (h *WorkspaceHandlers) AddMember(w http.ResponseWriter, r *http.Request) {
	workspaceID := rbac.WorkspaceIDFromContext(r.Context())
	if workspaceID == "" {
		respondError(w, http.StatusBadRequest, "Workspace context required")
		return
	}

	var req addMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	req.Email = strings.TrimSpace(req.Email)
	req.FirstName = strings.TrimSpace(req.FirstName)
	req.LastName = strings.TrimSpace(req.LastName)

	if req.Email == "" {
		respondAppError(w, Validation("email is required"))
		return
	}
	if err := validation.ValidateEmail(req.Email); err != nil {
		respondAppError(w, Validation(err.Error()))
		return
	}
	if req.FirstName == "" {
		respondAppError(w, Validation("firstName is required"))
		return
	}
	if req.RoleID == "" {
		respondAppError(w, Validation("roleId is required"))
		return
	}
	if _, err := uuid.Parse(req.RoleID); err != nil {
		respondAppError(w, BadRequest("invalid roleId format"))
		return
	}

	member, userCreated, err := h.RBAC.AddUserToWorkspace(r.Context(), workspaceID, req.Email, req.FirstName, req.LastName, req.RoleID, req.Password)
	if err != nil {
		if errors.Is(err, rbac.ErrRoleNotFound) {
			respondError(w, http.StatusBadRequest, "Role not found in this workspace")
			return
		}
		if errors.Is(err, rbac.ErrAlreadyMember) {
			respondError(w, http.StatusConflict, "User is already a member of this workspace")
			return
		}
		if errors.Is(err, auth.ErrEmailDomainNotAllowed) {
			respondError(w, http.StatusForbidden, "Email domain is not allowed")
			return
		}
		respondError(w, http.StatusInternalServerError, "Failed to add member")
		return
	}

	roleName := req.RoleID
	if member.Role != nil {
		roleName = member.Role.Name
	}
	h.Audit.LogAction(r.Context(), "add", "member", member.ID, fmt.Sprintf("added %s with role %s", req.Email, roleName))

	respondJSON(w, http.StatusCreated, map[string]any{
		"member":      member,
		"userCreated": userCreated,
	}, nil)
}

// InviteMember handles POST /api/workspaces/{workspaceId}/invitations - invites a user to the workspace.
func (h *WorkspaceHandlers) InviteMember(w http.ResponseWriter, r *http.Request) {
	workspaceID := rbac.WorkspaceIDFromContext(r.Context())
	userID := rbac.UserIDFromContext(r.Context())
	if workspaceID == "" || userID == "" {
		respondError(w, http.StatusBadRequest, "Workspace and user context required")
		return
	}

	var req inviteMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Email == "" {
		respondAppError(w, Validation("email is required"))
		return
	}
	if err := validation.ValidateEmail(req.Email); err != nil {
		respondAppError(w, Validation(err.Error()))
		return
	}
	if req.RoleID == "" {
		respondAppError(w, Validation("roleId is required"))
		return
	}
	if _, err := uuid.Parse(req.RoleID); err != nil {
		respondAppError(w, BadRequest("invalid roleId format"))
		return
	}

	inviterEmail := auth.EmailFromContext(r.Context())

	invitation, rawToken, err := h.RBAC.InviteMember(r.Context(), workspaceID, req.Email, req.RoleID, userID, inviterEmail)
	if err != nil {
		if errors.Is(err, rbac.ErrInvitationsDisabled) {
			respondError(w, http.StatusForbidden, "Invitations are not available. Ask your workspace admin to add you directly.")
			return
		}
		if errors.Is(err, rbac.ErrRoleNotFound) {
			respondError(w, http.StatusBadRequest, "Role not found in this workspace")
			return
		}
		respondError(w, http.StatusInternalServerError, "Failed to create invitation")
		return
	}

	h.Audit.LogAction(r.Context(), "invite", "member", invitation.ID, fmt.Sprintf("invited %s with role %s", req.Email, invitation.RoleName))

	// Return the invitation with the raw token so the caller can construct the invitation URL.
	// The token is not stored in the model (only the hash is), so we include it explicitly.
	respondJSON(w, http.StatusCreated, map[string]any{
		"id":          invitation.ID,
		"workspaceId": invitation.WorkspaceID,
		"email":       invitation.Email,
		"roleId":      invitation.RoleID,
		"token":       rawToken,
		"invitedBy":   invitation.InvitedBy,
		"expiresAt":   invitation.ExpiresAt,
		"createdAt":   invitation.CreatedAt,
	}, nil)
}

// AcceptInvitation handles POST /api/workspaces/{workspaceId}/invitations/{token}/accept - accepts an invitation.
func (h *WorkspaceHandlers) AcceptInvitation(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	if token == "" {
		respondError(w, http.StatusBadRequest, "Invitation token is required")
		return
	}

	userID := rbac.UserIDFromContext(r.Context())
	if userID == "" {
		respondError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	userEmail := auth.EmailFromContext(r.Context())
	if userEmail == "" {
		respondError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	member, err := h.RBAC.AcceptInvitation(r.Context(), token, userID, userEmail)
	if err != nil {
		if errors.Is(err, rbac.ErrInvitationsDisabled) {
			respondError(w, http.StatusForbidden, "Invitations are not available. Ask your workspace admin to add you directly.")
			return
		}
		if errors.Is(err, rbac.ErrInvitationNotFound) {
			respondError(w, http.StatusNotFound, "Invitation not found")
			return
		}
		if errors.Is(err, rbac.ErrInvitationExpired) {
			respondError(w, http.StatusGone, "Invitation has expired")
			return
		}
		if errors.Is(err, rbac.ErrInvitationAccepted) {
			respondError(w, http.StatusConflict, "Invitation already accepted")
			return
		}
		if errors.Is(err, rbac.ErrAlreadyMember) {
			respondError(w, http.StatusConflict, "Already a member of this workspace")
			return
		}
		if errors.Is(err, rbac.ErrInvitationEmailMismatch) {
			respondError(w, http.StatusForbidden, "Invitation was sent to a different email address")
			return
		}
		respondError(w, http.StatusInternalServerError, "Failed to accept invitation")
		return
	}

	h.Audit.LogAction(r.Context(), "accept", "member", member.ID, fmt.Sprintf("accepted invitation for user %s", userEmail))

	respondJSON(w, http.StatusOK, member, nil)
}

// DeclineInvitation handles POST /api/workspaces/{workspaceId}/invitations/{token}/decline - declines an invitation by token.
func (h *WorkspaceHandlers) DeclineInvitation(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	if token == "" {
		respondError(w, http.StatusBadRequest, "Invitation token is required")
		return
	}

	userEmail := auth.EmailFromContext(r.Context())
	if userEmail == "" {
		respondError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	err := h.RBAC.DeclineInvitationByToken(r.Context(), token, userEmail)
	if err != nil {
		if errors.Is(err, rbac.ErrInvitationsDisabled) {
			respondError(w, http.StatusForbidden, "Invitations are not available. Ask your workspace admin to add you directly.")
			return
		}
		if errors.Is(err, rbac.ErrInvitationNotFound) {
			respondError(w, http.StatusNotFound, "Invitation not found")
			return
		}
		if errors.Is(err, rbac.ErrInvitationExpired) {
			respondError(w, http.StatusGone, "Invitation has expired")
			return
		}
		if errors.Is(err, rbac.ErrInvitationAccepted) {
			respondError(w, http.StatusConflict, "Invitation already accepted")
			return
		}
		if errors.Is(err, rbac.ErrInvitationDeclined) {
			respondError(w, http.StatusConflict, "Invitation already declined")
			return
		}
		if errors.Is(err, rbac.ErrInvitationEmailMismatch) {
			respondError(w, http.StatusForbidden, "Invitation was sent to a different email address")
			return
		}
		respondError(w, http.StatusInternalServerError, "Failed to decline invitation")
		return
	}

	h.Audit.LogAction(r.Context(), "decline", "invitation", token, fmt.Sprintf("declined invitation by token for %s", userEmail))

	respondJSON(w, http.StatusOK, map[string]string{"message": "invitation declined"}, nil)
}

// GetInvitationInfo handles GET /api/invitations/{token} - returns invitation details
// so the frontend can display "You've been invited to workspace" before the user accepts.
// This endpoint is public (no auth required) so unauthenticated users can see the
// invitation info and then register/log in before accepting.
func (h *WorkspaceHandlers) GetInvitationInfo(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	if token == "" {
		respondError(w, http.StatusBadRequest, "Invitation token is required")
		return
	}

	invitation, err := h.RBAC.GetInvitationByToken(r.Context(), token)
	if err != nil {
		if errors.Is(err, rbac.ErrInvitationsDisabled) {
			respondError(w, http.StatusForbidden, "Invitations are not available. Ask your workspace admin to add you directly.")
			return
		}
		if errors.Is(err, rbac.ErrInvitationNotFound) {
			respondError(w, http.StatusNotFound, "Invitation not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "Failed to get invitation")
		return
	}

	// Return limited info - don't expose internal IDs
	respondJSON(w, http.StatusOK, map[string]any{
		"email":       invitation.Email,
		"workspaceId": invitation.WorkspaceID,
		"expiresAt":   invitation.ExpiresAt,
		"accepted":    invitation.AcceptedAt != nil,
		"expired":     time.Now().After(invitation.ExpiresAt),
	}, nil)
}

// ListPendingInvitations handles GET /api/workspaces/{workspaceId}/invitations - lists
// pending invitations for the workspace.
func (h *WorkspaceHandlers) ListPendingInvitations(w http.ResponseWriter, r *http.Request) {
	workspaceID := rbac.WorkspaceIDFromContext(r.Context())
	if workspaceID == "" {
		respondError(w, http.StatusBadRequest, "Workspace context required")
		return
	}

	invitations, err := h.RBAC.ListPendingInvitations(r.Context(), workspaceID)
	if err != nil {
		if errors.Is(err, rbac.ErrInvitationsDisabled) {
			respondError(w, http.StatusForbidden, "Invitations are not available. Ask your workspace admin to add you directly.")
			return
		}
		respondError(w, http.StatusInternalServerError, "Failed to list invitations")
		return
	}

	resp := make([]invitationResponse, len(invitations))
	for i, inv := range invitations {
		resp[i] = toInvitationResponse(inv)
	}

	respondJSON(w, http.StatusOK, resp, nil)
}

// RevokeInvitation handles DELETE /api/workspaces/{workspaceId}/invitations/{id} - revokes
// a pending invitation.
func (h *WorkspaceHandlers) RevokeInvitation(w http.ResponseWriter, r *http.Request) {
	workspaceID := rbac.WorkspaceIDFromContext(r.Context())
	if workspaceID == "" {
		respondError(w, http.StatusBadRequest, "Workspace context required")
		return
	}

	id, ok := parseUUID(r, "id")
	if !ok {
		respondError(w, http.StatusBadRequest, "Invalid invitation ID")
		return
	}

	// Resolve invitation email for audit log readability.
	invEmail := id
	if invitations, err := h.RBAC.ListPendingInvitations(r.Context(), workspaceID); err == nil {
		for _, inv := range invitations {
			if inv.ID == id {
				invEmail = inv.Email
				break
			}
		}
	}

	if err := h.RBAC.RevokeInvitation(r.Context(), workspaceID, id); err != nil {
		if errors.Is(err, rbac.ErrInvitationsDisabled) {
			respondError(w, http.StatusForbidden, "Invitations are not available. Ask your workspace admin to add you directly.")
			return
		}
		if errors.Is(err, rbac.ErrInvitationNotFound) {
			respondError(w, http.StatusNotFound, "Invitation not found or already accepted")
			return
		}
		respondError(w, http.StatusInternalServerError, "Failed to revoke invitation")
		return
	}

	h.Audit.LogAction(r.Context(), "revoke", "invitation", id, fmt.Sprintf("revoked invitation for %s", invEmail))

	respondJSON(w, http.StatusOK, map[string]string{"message": "invitation revoked"}, nil)
}

// ResendInvitation handles POST /api/workspaces/{workspaceId}/invitations/{id}/resend -
// resends a pending invitation email with a fresh token and extended expiry.
func (h *WorkspaceHandlers) ResendInvitation(w http.ResponseWriter, r *http.Request) {
	workspaceID := rbac.WorkspaceIDFromContext(r.Context())
	if workspaceID == "" {
		respondError(w, http.StatusBadRequest, "Workspace context required")
		return
	}

	id, ok := parseUUID(r, "id")
	if !ok {
		respondError(w, http.StatusBadRequest, "Invalid invitation ID")
		return
	}

	invitation, _, err := h.RBAC.ResendInvitation(r.Context(), workspaceID, id)
	if err != nil {
		if errors.Is(err, rbac.ErrInvitationsDisabled) {
			respondError(w, http.StatusForbidden, "Invitations are not available. Ask your workspace admin to add you directly.")
			return
		}
		if errors.Is(err, rbac.ErrInvitationNotFound) {
			respondError(w, http.StatusNotFound, "Invitation not found")
			return
		}
		if errors.Is(err, rbac.ErrInvitationExpired) {
			respondError(w, http.StatusGone, "Invitation has expired")
			return
		}
		if errors.Is(err, rbac.ErrInvitationAccepted) {
			respondError(w, http.StatusConflict, "Invitation already accepted")
			return
		}
		respondError(w, http.StatusInternalServerError, "Failed to resend invitation")
		return
	}

	h.Audit.LogAction(r.Context(), "resend", "invitation", id, fmt.Sprintf("resent invitation to %s", invitation.Email))

	respondJSON(w, http.StatusOK, toInvitationResponse(*invitation), nil)
}

// RemoveMember handles DELETE /api/workspaces/{workspaceId}/members/{userId} - removes a member.
func (h *WorkspaceHandlers) RemoveMember(w http.ResponseWriter, r *http.Request) {
	workspaceID := rbac.WorkspaceIDFromContext(r.Context())
	if workspaceID == "" {
		respondError(w, http.StatusBadRequest, "Workspace context required")
		return
	}

	targetUserID, ok := parseUUID(r, "userId")
	if !ok {
		respondError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	// Look up member email before removal for audit log readability.
	memberEmail := targetUserID
	if members, err := h.RBAC.GetWorkspaceMembers(r.Context(), workspaceID); err == nil {
		for _, m := range members {
			if m.UserID == targetUserID && m.User != nil {
				memberEmail = m.User.Email
				break
			}
		}
	}

	if err := h.RBAC.RemoveMember(r.Context(), workspaceID, targetUserID); err != nil {
		if errors.Is(err, rbac.ErrCannotRemoveOwner) {
			respondError(w, http.StatusForbidden, "Cannot remove the workspace owner")
			return
		}
		if errors.Is(err, rbac.ErrMemberNotFound) {
			respondError(w, http.StatusNotFound, "Member not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "Failed to remove member")
		return
	}

	h.Audit.LogAction(r.Context(), "remove", "member", targetUserID, fmt.Sprintf("removed member %s from workspace", memberEmail))

	respondJSON(w, http.StatusOK, nil, nil)
}

// GetCurrentMemberRole handles GET /api/workspaces/{workspaceId}/members/me/role - returns the
// authenticated user's role in the current workspace. The role is already resolved by the
// RequireWorkspace middleware and stored in the request context.
func (h *WorkspaceHandlers) GetCurrentMemberRole(w http.ResponseWriter, r *http.Request) {
	role := rbac.MemberRoleFromContext(r.Context())
	if role == "" {
		respondError(w, http.StatusForbidden, "Workspace membership required")
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"role": role}, nil)
}

// UpdateMemberRole handles PUT /api/workspaces/{workspaceId}/members/{userId}/role - changes a member's role.
func (h *WorkspaceHandlers) UpdateMemberRole(w http.ResponseWriter, r *http.Request) {
	workspaceID := rbac.WorkspaceIDFromContext(r.Context())
	if workspaceID == "" {
		respondError(w, http.StatusBadRequest, "Workspace context required")
		return
	}

	targetUserID := chi.URLParam(r, "userId")
	if _, err := uuid.Parse(targetUserID); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	var req updateMemberRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.RoleID == "" {
		respondAppError(w, Validation("roleId is required"))
		return
	}
	if _, err := uuid.Parse(req.RoleID); err != nil {
		respondAppError(w, BadRequest("invalid roleId format"))
		return
	}

	member, err := h.RBAC.UpdateMemberRole(r.Context(), workspaceID, targetUserID, req.RoleID)
	if err != nil {
		if errors.Is(err, rbac.ErrCannotChangeOwner) {
			respondError(w, http.StatusForbidden, "Cannot change the owner's role")
			return
		}
		if errors.Is(err, rbac.ErrMemberNotFound) {
			respondError(w, http.StatusNotFound, "Member not found")
			return
		}
		if errors.Is(err, rbac.ErrRoleNotFound) {
			respondError(w, http.StatusBadRequest, "Role not found in this workspace")
			return
		}
		respondError(w, http.StatusInternalServerError, "Failed to update member role")
		return
	}

	roleName := req.RoleID // fallback
	if member.Role != nil {
		roleName = member.Role.Name
	}
	targetEmail := targetUserID // fallback
	if member.User != nil {
		targetEmail = member.User.Email
	}
	h.Audit.LogAction(r.Context(), "update", "member", member.ID, fmt.Sprintf("updated role for %s to %s", targetEmail, roleName))

	respondJSON(w, http.StatusOK, member, nil)
}

// myInvitationResponse is the response shape for the user's own invitations.
type myInvitationResponse struct {
	ID             string    `json:"id"`
	WorkspaceID    string    `json:"workspaceId"`
	WorkspaceName  string    `json:"workspaceName"`
	Email          string    `json:"email"`
	InvitedByEmail string    `json:"invitedByEmail"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"createdAt"`
	ExpiresAt      time.Time `json:"expiresAt"`
}

// ListMyInvitations handles GET /api/invitations/mine - returns all pending
// invitations for the authenticated user's email.
func (h *WorkspaceHandlers) ListMyInvitations(w http.ResponseWriter, r *http.Request) {
	userEmail := auth.EmailFromContext(r.Context())
	if userEmail == "" {
		respondError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	invitations, err := h.RBAC.ListMyInvitations(r.Context(), userEmail)
	if err != nil {
		if errors.Is(err, rbac.ErrInvitationsDisabled) {
			respondError(w, http.StatusForbidden, "Invitations are not available. Ask your workspace admin to add you directly.")
			return
		}
		respondError(w, http.StatusInternalServerError, "Failed to list invitations")
		return
	}

	resp := make([]myInvitationResponse, len(invitations))
	for i, inv := range invitations {
		resp[i] = myInvitationResponse{
			ID:             inv.ID,
			WorkspaceID:    inv.WorkspaceID,
			WorkspaceName:  inv.WorkspaceName,
			Email:          inv.Email,
			InvitedByEmail: inv.InvitedByEmail,
			Status:         "pending",
			CreatedAt:      inv.CreatedAt,
			ExpiresAt:      inv.ExpiresAt,
		}
	}

	respondJSON(w, http.StatusOK, resp, nil)
}

// AcceptInvitationByID handles POST /api/invitations/{id}/accept - accepts an
// invitation by ID for the authenticated user.
func (h *WorkspaceHandlers) AcceptInvitationByID(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(r, "id")
	if !ok {
		respondError(w, http.StatusBadRequest, "Invalid invitation ID")
		return
	}

	userID := rbac.UserIDFromContext(r.Context())
	userEmail := auth.EmailFromContext(r.Context())
	if userID == "" || userEmail == "" {
		respondError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	member, err := h.RBAC.AcceptInvitationByID(r.Context(), id, userID, userEmail)
	if err != nil {
		if errors.Is(err, rbac.ErrInvitationsDisabled) {
			respondError(w, http.StatusForbidden, "Invitations are not available. Ask your workspace admin to add you directly.")
			return
		}
		if errors.Is(err, rbac.ErrInvitationNotFound) {
			respondError(w, http.StatusNotFound, "Invitation not found")
			return
		}
		if errors.Is(err, rbac.ErrInvitationExpired) {
			respondError(w, http.StatusGone, "Invitation has expired")
			return
		}
		if errors.Is(err, rbac.ErrInvitationAccepted) {
			respondError(w, http.StatusConflict, "Invitation already accepted")
			return
		}
		if errors.Is(err, rbac.ErrInvitationDeclined) {
			respondError(w, http.StatusConflict, "Invitation has been declined")
			return
		}
		if errors.Is(err, rbac.ErrAlreadyMember) {
			respondError(w, http.StatusConflict, "Already a member of this workspace")
			return
		}
		if errors.Is(err, rbac.ErrInvitationEmailMismatch) {
			respondError(w, http.StatusForbidden, "Invitation was sent to a different email address")
			return
		}
		respondError(w, http.StatusInternalServerError, "Failed to accept invitation")
		return
	}

	h.Audit.LogAction(r.Context(), "accept", "member", member.ID, fmt.Sprintf("accepted invitation %s", id))

	respondJSON(w, http.StatusOK, member, nil)
}

// DeclineInvitationByID handles POST /api/invitations/{id}/decline - declines
// an invitation by ID for the authenticated user.
func (h *WorkspaceHandlers) DeclineInvitationByID(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(r, "id")
	if !ok {
		respondError(w, http.StatusBadRequest, "Invalid invitation ID")
		return
	}

	userEmail := auth.EmailFromContext(r.Context())
	if userEmail == "" {
		respondError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	err := h.RBAC.DeclineInvitationByID(r.Context(), id, userEmail)
	if err != nil {
		if errors.Is(err, rbac.ErrInvitationsDisabled) {
			respondError(w, http.StatusForbidden, "Invitations are not available. Ask your workspace admin to add you directly.")
			return
		}
		if errors.Is(err, rbac.ErrInvitationNotFound) {
			respondError(w, http.StatusNotFound, "Invitation not found")
			return
		}
		if errors.Is(err, rbac.ErrInvitationExpired) {
			respondError(w, http.StatusGone, "Invitation has expired")
			return
		}
		if errors.Is(err, rbac.ErrInvitationAccepted) {
			respondError(w, http.StatusConflict, "Invitation already accepted")
			return
		}
		if errors.Is(err, rbac.ErrInvitationDeclined) {
			respondError(w, http.StatusConflict, "Invitation already declined")
			return
		}
		if errors.Is(err, rbac.ErrInvitationEmailMismatch) {
			respondError(w, http.StatusForbidden, "Invitation was sent to a different email address")
			return
		}
		respondError(w, http.StatusInternalServerError, "Failed to decline invitation")
		return
	}

	h.Audit.LogAction(r.Context(), "decline", "invitation", id, fmt.Sprintf("declined invitation %s", id))

	respondJSON(w, http.StatusOK, map[string]string{"message": "invitation declined"}, nil)
}
