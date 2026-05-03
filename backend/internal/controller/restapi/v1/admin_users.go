package v1

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/veilence/veilence-mx/backend/internal/controller/restapi/v1/response"
	"github.com/veilence/veilence-mx/backend/internal/entity"
	"github.com/veilence/veilence-mx/backend/internal/usecase"
	"github.com/veilence/veilence-mx/backend/internal/usecase/auth"
)

// AdminUserHandlers handles super-admin user management endpoints.
type AdminUserHandlers struct {
	Auth         *auth.Service
	RBACRepo     usecase.RBACRepository
	IdentityRepo usecase.UserIdentityRepository
}

// adminUpdateUserRequest is the request body for PUT /api/admin/users/{id}.
type adminUpdateUserRequest struct {
	FirstName       *string `json:"firstName"`
	LastName        *string `json:"lastName"`
	IsSuperAdmin    *bool   `json:"isSuperAdmin"`
	IsActive        *bool   `json:"isActive"`
	ConfirmPassword string  `json:"confirmPassword"`
}

// HandleListUsers returns a paginated list of all users.
// GET /api/admin/users
func (h *AdminUserHandlers) HandleListUsers(w http.ResponseWriter, r *http.Request) {
	page, limit := parsePagination(r)
	search := r.URL.Query().Get("search")

	users, total, err := h.Auth.AdminListUsers(r.Context(), page, limit, search)
	if err != nil {
		slog.Error("HandleListUsers: listing users", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to list users")
		return
	}

	totalPages := int(total) / limit
	if int(total)%limit != 0 {
		totalPages++
	}

	respondJSON(w, http.StatusOK, response.ListUsersResponse{
		Users:      response.AdminUsersFromEntities(users),
		Total:      total,
		Page:       page,
		PageSize:   limit,
		TotalPages: totalPages,
	}, nil)
}

// HandleGetUser returns a single user by ID with nested identities and workspaces.
// GET /api/admin/users/{id}
func (h *AdminUserHandlers) HandleGetUser(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")
	if userID == "" {
		respondAppError(w, Validation("user ID is required"))
		return
	}

	user, err := h.Auth.AdminGetUser(r.Context(), userID)
	if err != nil {
		if errors.Is(err, entity.ErrNotFound) {
			respondAppError(w, NotFound("user"))
			return
		}
		slog.Error("HandleGetUser: fetching user", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to fetch user")
		return
	}

	// Build detail response with identities and workspaces.
	detail := response.PlatformUserDetailResponse{
		AdminUserResponse: response.AdminUserFromEntity(user),
		Identities:        []response.LinkedIdentityResponse{},
		Workspaces:        []response.UserWorkspaceResponse{},
	}

	// Fetch linked identities.
	if h.IdentityRepo != nil {
		identities, identErr := h.IdentityRepo.FindByUserID(r.Context(), userID)
		if identErr != nil {
			slog.Error("HandleGetUser: fetching identities", "error", identErr)
		} else {
			for _, id := range identities {
				detail.Identities = append(detail.Identities, response.LinkedIdentityResponse{
					ID:             id.ID,
					Provider:       string(id.Provider),
					ProviderEmail:  id.Email,
					ProviderUserID: id.ProviderUserID,
				})
			}
		}
	}

	// Fetch workspace memberships.
	if h.RBACRepo != nil {
		wsResult, wsErr := h.RBACRepo.FindWorkspacesByUserID(r.Context(), userID, entity.WorkspaceListParams{Page: 1, Limit: 100})
		if wsErr != nil {
			slog.Error("HandleGetUser: fetching workspaces", "error", wsErr)
		} else if wsResult != nil {
			for _, ws := range wsResult.Workspaces {
				detail.Workspaces = append(detail.Workspaces, response.UserWorkspaceResponse{
					ID:   ws.ID,
					Name: ws.Name,
					Role: ws.Role,
				})
			}
		}
	}

	respondJSON(w, http.StatusOK, detail, nil)
}

// HandleUpdateUser updates a user (super-admin operation with step-up auth).
// PUT /api/admin/users/{id}
func (h *AdminUserHandlers) HandleUpdateUser(w http.ResponseWriter, r *http.Request) {
	targetUserID := chi.URLParam(r, "id")
	if targetUserID == "" {
		respondAppError(w, Validation("user ID is required"))
		return
	}

	callerID := auth.UserIDFromContext(r.Context())
	if callerID == "" {
		respondError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	var req adminUpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Step-up auth: require password confirmation from the X-Confirm-Password header
	// or the confirmPassword body field.
	confirmPassword := r.Header.Get("X-Confirm-Password")
	if confirmPassword == "" {
		confirmPassword = req.ConfirmPassword
	}
	if confirmPassword == "" {
		respondAppError(w, Validation("password confirmation required (X-Confirm-Password header or confirmPassword field)"))
		return
	}

	updates := auth.AdminUserUpdate{
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		IsSuperAdmin: req.IsSuperAdmin,
		IsActive:     req.IsActive,
	}

	updated, err := h.Auth.AdminUpdateUser(r.Context(), callerID, targetUserID, confirmPassword, updates)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidPassword) {
			respondAppError(w, Forbidden("password confirmation failed"))
			return
		}
		if errors.Is(err, auth.ErrLastSuperAdmin) {
			respondAppError(w, BadRequest("cannot demote or deactivate the last super admin"))
			return
		}
		if errors.Is(err, entity.ErrNotFound) {
			respondAppError(w, NotFound("user"))
			return
		}
		slog.Error("HandleUpdateUser: updating user", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to update user")
		return
	}

	respondJSON(w, http.StatusOK, response.AdminUserFromEntity(updated), nil)
}
