package v1

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/veilence/veilence-mx/backend/internal/controller/restapi/v1/response"
	"github.com/veilence/veilence-mx/backend/internal/entity"
	"github.com/veilence/veilence-mx/backend/internal/usecase"
	"github.com/veilence/veilence-mx/backend/internal/usecase/audit"
	"github.com/veilence/veilence-mx/backend/internal/usecase/auth"
)

// AdminUserHandlers handles super-admin user management endpoints.
type AdminUserHandlers struct {
	Auth         *auth.Service
	RBACRepo     usecase.RBACRepository
	IdentityRepo usecase.UserIdentityRepository
	Audit        *audit.Service
}

// adminUpdateUserRequest is the request body for PUT /api/admin/users/{id}.
type adminUpdateUserRequest struct {
	FirstName       *string `json:"firstName"`
	LastName        *string `json:"lastName"`
	IsSuperAdmin    *bool   `json:"isSuperAdmin"`
	IsActive        *bool   `json:"isActive"`
	ConfirmPassword string  `json:"confirmPassword"`
}

// HandleListUsers returns a paginated, filterable, sortable list of all users.
// GET /api/admin/users
func (h *AdminUserHandlers) HandleListUsers(w http.ResponseWriter, r *http.Request) {
	page, limit := parsePagination(r)
	sortOrder := parseSort(r, map[string]string{
		"firstName": "first_name",
		"lastName":  "last_name",
		"email":     "email",
		"isActive":  "is_active",
		"createdAt": "created_at",
	}, "created_at DESC")

	var filters entity.UserFilters
	if search := r.URL.Query().Get("search"); search != "" {
		filters.Search = &search
	}
	if status := r.URL.Query().Get("status"); status != "" {
		validStatuses := map[string]bool{"active": true, "inactive": true}
		if !validStatuses[status] {
			respondAppError(w, Validation("invalid status filter"))
			return
		}
		filters.Status = &status
	}
	if role := r.URL.Query().Get("role"); role != "" {
		validRoles := map[string]bool{"super_admin": true, "user": true}
		if !validRoles[role] {
			respondAppError(w, Validation("invalid role filter"))
			return
		}
		filters.Role = &role
	}

	users, total, err := h.Auth.AdminListUsers(r.Context(), page, limit, sortOrder, filters)
	if err != nil {
		slog.Error("HandleListUsers: listing users", "error", err)
		respondAppError(w, Internal("failed to list users"))
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
		respondAppError(w, Internal("failed to fetch user"))
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
		respondAppError(w, Unauthorized("authentication required"))
		return
	}

	var req adminUpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondAppError(w, BadRequest("invalid request body"))
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
		if errors.Is(err, auth.ErrSelfModification) {
			respondAppError(w, BadRequest("cannot modify your own account"))
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
		respondAppError(w, Internal("failed to update user"))
		return
	}

	// Audit log the admin user update.
	var changes []string
	if req.FirstName != nil {
		changes = append(changes, fmt.Sprintf("firstName=%s", *req.FirstName))
	}
	if req.LastName != nil {
		changes = append(changes, fmt.Sprintf("lastName=%s", *req.LastName))
	}
	if req.IsSuperAdmin != nil {
		changes = append(changes, fmt.Sprintf("isSuperAdmin=%t", *req.IsSuperAdmin))
	}
	if req.IsActive != nil {
		changes = append(changes, fmt.Sprintf("isActive=%t", *req.IsActive))
	}
	h.Audit.LogAction(r.Context(), "admin.update", "user", targetUserID,
		fmt.Sprintf("admin updated user %s: %s", updated.Email, strings.Join(changes, ", ")))

	// Return full detail (identities + workspaces) consistent with HandleGetUser.
	detail := response.PlatformUserDetailResponse{
		AdminUserResponse: response.AdminUserFromEntity(updated),
		Identities:        []response.LinkedIdentityResponse{},
		Workspaces:        []response.UserWorkspaceResponse{},
	}

	if h.IdentityRepo != nil {
		identities, identErr := h.IdentityRepo.FindByUserID(r.Context(), targetUserID)
		if identErr != nil {
			slog.Error("HandleUpdateUser: fetching identities", "error", identErr)
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

	if h.RBACRepo != nil {
		wsResult, wsErr := h.RBACRepo.FindWorkspacesByUserID(r.Context(), targetUserID, entity.WorkspaceListParams{Page: 1, Limit: 100})
		if wsErr != nil {
			slog.Error("HandleUpdateUser: fetching workspaces", "error", wsErr)
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
