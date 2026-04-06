package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/veilence/veilence-mx/backend/internal/api/validation"
	"github.com/veilence/veilence-mx/backend/internal/apperror"
	"github.com/veilence/veilence-mx/backend/internal/rbac"
)

type inviteMemberRequest struct {
	Email  string `json:"email"`
	RoleID uint   `json:"roleId"`
}

type updateMemberRoleRequest struct {
	RoleID uint `json:"roleId"`
}

// ListMembers handles GET /api/orgs/{orgId}/members — lists organization members.
func (h *OrgHandlers) ListMembers(w http.ResponseWriter, r *http.Request) {
	orgID := rbac.OrgIDFromContext(r.Context())
	if orgID == 0 {
		respondError(w, http.StatusBadRequest, "organization context required")
		return
	}

	members, err := h.RBAC.GetOrgMembers(orgID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list members")
		return
	}

	respondJSON(w, http.StatusOK, members, nil)
}

// InviteMember handles POST /api/orgs/{orgId}/invitations — invites a user to the organization.
func (h *OrgHandlers) InviteMember(w http.ResponseWriter, r *http.Request) {
	orgID := rbac.OrgIDFromContext(r.Context())
	userID := rbac.UserIDFromContext(r.Context())
	if orgID == 0 || userID == 0 {
		respondError(w, http.StatusBadRequest, "organization and user context required")
		return
	}

	var req inviteMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Email == "" {
		respondAppError(w, apperror.Validation("email is required"))
		return
	}
	if err := validation.ValidateEmail(req.Email); err != nil {
		respondAppError(w, apperror.Validation(err.Error()))
		return
	}
	if req.RoleID == 0 {
		respondAppError(w, apperror.Validation("roleId is required"))
		return
	}

	invitation, err := h.RBAC.InviteMember(orgID, req.Email, req.RoleID, userID)
	if err != nil {
		if errors.Is(err, rbac.ErrRoleNotFound) {
			respondError(w, http.StatusBadRequest, "role not found in this organization")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to create invitation")
		return
	}

	h.Audit.LogAction(r.Context(), "invite", "member", invitation.ID, fmt.Sprintf("invited %s with role %d", req.Email, req.RoleID))

	respondJSON(w, http.StatusCreated, invitation, nil)
}

// AcceptInvitation handles POST /api/orgs/{orgId}/invitations/{token}/accept — accepts an invitation.
func (h *OrgHandlers) AcceptInvitation(w http.ResponseWriter, r *http.Request) {
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

	member, err := h.RBAC.AcceptInvitation(token, userID)
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
			respondError(w, http.StatusConflict, "already a member of this organization")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to accept invitation")
		return
	}

	h.Audit.LogAction(r.Context(), "accept", "member", member.ID, fmt.Sprintf("accepted invitation for user %d", userID))

	respondJSON(w, http.StatusOK, member, nil)
}

// RemoveMember handles DELETE /api/orgs/{orgId}/members/{userId} — removes a member.
func (h *OrgHandlers) RemoveMember(w http.ResponseWriter, r *http.Request) {
	orgID := rbac.OrgIDFromContext(r.Context())
	if orgID == 0 {
		respondError(w, http.StatusBadRequest, "organization context required")
		return
	}

	targetUserIDStr := chi.URLParam(r, "userId")
	targetUserID, err := strconv.ParseUint(targetUserIDStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid user ID")
		return
	}

	if err := h.RBAC.RemoveMember(orgID, uint(targetUserID)); err != nil {
		if errors.Is(err, rbac.ErrCannotRemoveOwner) {
			respondError(w, http.StatusForbidden, "cannot remove the organization owner")
			return
		}
		if errors.Is(err, rbac.ErrMemberNotFound) {
			respondError(w, http.StatusNotFound, "member not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to remove member")
		return
	}

	h.Audit.LogAction(r.Context(), "remove", "member", uint(targetUserID), fmt.Sprintf("removed member (user %d) from organization", targetUserID))

	respondJSON(w, http.StatusOK, nil, nil)
}

// UpdateMemberRole handles PUT /api/orgs/{orgId}/members/{userId}/role — changes a member's role.
func (h *OrgHandlers) UpdateMemberRole(w http.ResponseWriter, r *http.Request) {
	orgID := rbac.OrgIDFromContext(r.Context())
	if orgID == 0 {
		respondError(w, http.StatusBadRequest, "organization context required")
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
		respondAppError(w, apperror.Validation("roleId is required"))
		return
	}

	member, err := h.RBAC.UpdateMemberRole(orgID, uint(targetUserID), req.RoleID)
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
			respondError(w, http.StatusBadRequest, "role not found in this organization")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to update member role")
		return
	}

	h.Audit.LogAction(r.Context(), "update", "member", member.ID, fmt.Sprintf("updated role for user %d to role %d", targetUserID, req.RoleID))

	respondJSON(w, http.StatusOK, member, nil)
}
