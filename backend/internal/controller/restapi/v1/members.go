package v1

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	validation "github.com/veilence/veilence-mx/backend/internal/controller/restapi/v1/request"
	
	"github.com/veilence/veilence-mx/backend/internal/entity"
	"github.com/veilence/veilence-mx/backend/internal/usecase/auth"
	"github.com/veilence/veilence-mx/backend/internal/usecase/rbac"
)

type inviteMemberRequest struct {
	Email  string `json:"email"`
	RoleID uint   `json:"roleId"`
}

type updateMemberRoleRequest struct {
	RoleID uint `json:"roleId"`
}

// flatMember is the flattened response shape for workspace members.
// The frontend expects email, firstName, and lastName at the top level
// instead of nested under a "user" object.
type flatMember struct {
	ID        uint        `json:"id"`
	WorkspaceID     uint        `json:"workspaceId"`
	UserID    uint        `json:"userId"`
	RoleID    uint        `json:"roleId"`
	Role      entity.Role `json:"role"`
	JoinedAt  time.Time   `json:"joinedAt"`
	Email     string      `json:"email"`
	FirstName string      `json:"firstName"`
	LastName  string      `json:"lastName"`
}

// ListMembers handles GET /api/workspaces/{workspaceId}/members — lists workspace members.
func (h *WorkspaceHandlers) ListMembers(w http.ResponseWriter, r *http.Request) {
	workspaceID := rbac.WorkspaceIDFromContext(r.Context())
	if workspaceID == 0 {
		respondError(w, http.StatusBadRequest, "workspace context required")
		return
	}

	members, err := h.RBAC.GetWorkspaceMembers(workspaceID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list members")
		return
	}

	// Flatten the response: promote User fields to the top level so the
	// frontend receives {email, firstName, lastName} directly instead of
	// a nested user object.
	flat := make([]flatMember, len(members))
	for i, m := range members {
		role := entity.Role{}
		if m.Role != nil {
			role = *m.Role
		}
		var email, firstName, lastName string
		if m.User != nil {
			email = m.User.Email
			firstName = m.User.FirstName
			lastName = m.User.LastName
		}
		flat[i] = flatMember{
			ID:        m.ID,
			WorkspaceID:     m.WorkspaceID,
			UserID:    m.UserID,
			RoleID:    m.RoleID,
			Role:      role,
			JoinedAt:  m.JoinedAt,
			Email:     email,
			FirstName: firstName,
			LastName:  lastName,
		}
	}

	respondJSON(w, http.StatusOK, flat, nil)
}

// InviteMember handles POST /api/workspaces/{workspaceId}/invitations — invites a user to the workspace.
func (h *WorkspaceHandlers) InviteMember(w http.ResponseWriter, r *http.Request) {
	workspaceID := rbac.WorkspaceIDFromContext(r.Context())
	userID := rbac.UserIDFromContext(r.Context())
	if workspaceID == 0 || userID == 0 {
		respondError(w, http.StatusBadRequest, "workspace and user context required")
		return
	}

	var req inviteMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
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
	if req.RoleID == 0 {
		respondAppError(w, Validation("roleId is required"))
		return
	}

	invitation, rawToken, err := h.RBAC.InviteMember(workspaceID, req.Email, req.RoleID, userID)
	if err != nil {
		if errors.Is(err, rbac.ErrRoleNotFound) {
			respondError(w, http.StatusBadRequest, "role not found in this workspace")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to create invitation")
		return
	}

	h.Audit.LogAction(r.Context(), "invite", "member", invitation.ID, fmt.Sprintf("invited %s with role %d", req.Email, req.RoleID))

	// Return the invitation with the raw token so the caller can construct the invitation URL.
	// The token is not stored in the model (only the hash is), so we include it explicitly.
	respondJSON(w, http.StatusCreated, map[string]any{
		"id":        invitation.ID,
		  "workspaceId":     invitation.WorkspaceID,
		"email":     invitation.Email,
		"roleId":    invitation.RoleID,
		"token":     rawToken,
		"invitedBy": invitation.InvitedBy,
		"expiresAt": invitation.ExpiresAt,
		"createdAt": invitation.CreatedAt,
	}, nil)
}

// AcceptInvitation handles POST /api/workspaces/{workspaceId}/invitations/{token}/accept — accepts an invitation.
func (h *WorkspaceHandlers) AcceptInvitation(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	if token == "" {
		respondError(w, http.StatusBadRequest, "invitation token is required")
		return
	}

	userID := rbac.UserIDFromContext(r.Context())
	if userID == 0 {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	userEmail := auth.EmailFromContext(r.Context())
	if userEmail == "" {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	member, err := h.RBAC.AcceptInvitation(token, userID, userEmail)
	if err != nil {
		if errors.Is(err, rbac.ErrInvitationNotFound) {
			respondError(w, http.StatusNotFound, "invitation not found")
			return
		}
		if errors.Is(err, rbac.ErrInvitationExpired) {
			respondError(w, http.StatusGone, "invitation has expired")
			return
		}
		if errors.Is(err, rbac.ErrInvitationAccepted) {
			respondError(w, http.StatusConflict, "invitation already accepted")
			return
		}
		if errors.Is(err, rbac.ErrAlreadyMember) {
			respondError(w, http.StatusConflict, "already a member of this workspace")
			return
		}
		if errors.Is(err, rbac.ErrInvitationEmailMismatch) {
			respondError(w, http.StatusForbidden, "invitation was sent to a different email address")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to accept invitation")
		return
	}

	h.Audit.LogAction(r.Context(), "accept", "member", member.ID, fmt.Sprintf("accepted invitation for user %d", userID))

	respondJSON(w, http.StatusOK, member, nil)
}

// GetInvitationInfo handles GET /api/invitations/{token} — returns invitation details
// so the frontend can display "You've been invited to {org}" before the user accepts.
// This endpoint is public (no auth required) so unauthenticated users can see the
// invitation info and then register/log in before accepting.
func (h *WorkspaceHandlers) GetInvitationInfo(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	if token == "" {
		respondError(w, http.StatusBadRequest, "invitation token is required")
		return
	}

	invitation, err := h.RBAC.GetInvitationByToken(token)
	if err != nil {
		if errors.Is(err, rbac.ErrInvitationNotFound) {
			respondError(w, http.StatusNotFound, "invitation not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to get invitation")
		return
	}

	// Return limited info — don't expose internal IDs
	respondJSON(w, http.StatusOK, map[string]any{
		"email":     invitation.Email,
		  "workspaceId":     invitation.WorkspaceID,
		"expiresAt": invitation.ExpiresAt,
		"accepted":  invitation.AcceptedAt != nil,
		"expired":   time.Now().After(invitation.ExpiresAt),
	}, nil)
}

// ListPendingInvitations handles GET /api/workspaces/{workspaceId}/invitations — lists
// pending invitations for the workspace.
func (h *WorkspaceHandlers) ListPendingInvitations(w http.ResponseWriter, r *http.Request) {
	workspaceID := rbac.WorkspaceIDFromContext(r.Context())
	if workspaceID == 0 {
		respondError(w, http.StatusBadRequest, "workspace context required")
		return
	}

	invitations, err := h.RBAC.ListPendingInvitations(workspaceID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list invitations")
		return
	}

	respondJSON(w, http.StatusOK, invitations, nil)
}

// RevokeInvitation handles DELETE /api/workspaces/{workspaceId}/invitations/{id} — revokes
// a pending invitation.
func (h *WorkspaceHandlers) RevokeInvitation(w http.ResponseWriter, r *http.Request) {
	workspaceID := rbac.WorkspaceIDFromContext(r.Context())
	if workspaceID == 0 {
		respondError(w, http.StatusBadRequest, "workspace context required")
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid invitation ID")
		return
	}

	if err := h.RBAC.RevokeInvitation(workspaceID, uint(id)); err != nil {
		if errors.Is(err, rbac.ErrInvitationNotFound) {
			respondError(w, http.StatusNotFound, "invitation not found or already accepted")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to revoke invitation")
		return
	}

	h.Audit.LogAction(r.Context(), "revoke", "invitation", uint(id), fmt.Sprintf("revoked invitation %d", id))

	respondJSON(w, http.StatusOK, map[string]string{"message": "invitation revoked"}, nil)
}

// RemoveMember handles DELETE /api/workspaces/{workspaceId}/members/{userId} — removes a member.
func (h *WorkspaceHandlers) RemoveMember(w http.ResponseWriter, r *http.Request) {
	workspaceID := rbac.WorkspaceIDFromContext(r.Context())
	if workspaceID == 0 {
		respondError(w, http.StatusBadRequest, "workspace context required")
		return
	}

	targetUserIDStr := chi.URLParam(r, "userId")
	targetUserID, err := strconv.ParseUint(targetUserIDStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid user ID")
		return
	}

	if err := h.RBAC.RemoveMember(workspaceID, uint(targetUserID)); err != nil {
		if errors.Is(err, rbac.ErrCannotRemoveOwner) {
			respondError(w, http.StatusForbidden, "cannot remove the workspace owner")
			return
		}
		if errors.Is(err, rbac.ErrMemberNotFound) {
			respondError(w, http.StatusNotFound, "member not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to remove member")
		return
	}

	h.Audit.LogAction(r.Context(), "remove", "member", uint(targetUserID), fmt.Sprintf("removed member (user %d) from workspace", targetUserID))

	respondJSON(w, http.StatusOK, nil, nil)
}

// GetCurrentMemberRole handles GET /api/workspaces/{workspaceId}/members/me/role — returns the
// authenticated user's role in the current workspace. The role is already resolved by the
// RequireWorkspace middleware and stored in the request context.
func (h *WorkspaceHandlers) GetCurrentMemberRole(w http.ResponseWriter, r *http.Request) {
	role := rbac.MemberRoleFromContext(r.Context())
	if role == "" {
		respondError(w, http.StatusForbidden, "workspace membership required")
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"role": role}, nil)
}

// UpdateMemberRole handles PUT /api/workspaces/{workspaceId}/members/{userId}/role — changes a member's role.
func (h *WorkspaceHandlers) UpdateMemberRole(w http.ResponseWriter, r *http.Request) {
	workspaceID := rbac.WorkspaceIDFromContext(r.Context())
	if workspaceID == 0 {
		respondError(w, http.StatusBadRequest, "workspace context required")
		return
	}

	targetUserIDStr := chi.URLParam(r, "userId")
	targetUserID, err := strconv.ParseUint(targetUserIDStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid user ID")
		return
	}

	var req updateMemberRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.RoleID == 0 {
		respondAppError(w, Validation("roleId is required"))
		return
	}

	member, err := h.RBAC.UpdateMemberRole(workspaceID, uint(targetUserID), req.RoleID)
	if err != nil {
		if errors.Is(err, rbac.ErrCannotChangeOwner) {
			respondError(w, http.StatusForbidden, "cannot change the owner's role")
			return
		}
		if errors.Is(err, rbac.ErrMemberNotFound) {
			respondError(w, http.StatusNotFound, "member not found")
			return
		}
		if errors.Is(err, rbac.ErrRoleNotFound) {
			respondError(w, http.StatusBadRequest, "role not found in this workspace")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to update member role")
		return
	}

	h.Audit.LogAction(r.Context(), "update", "member", member.ID, fmt.Sprintf("updated role for user %d to role %d", targetUserID, req.RoleID))

	respondJSON(w, http.StatusOK, member, nil)
}
