package v1

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	validation "github.com/veilence/veilence-mx/backend/internal/controller/restapi/v1/request"
	"github.com/veilence/veilence-mx/backend/internal/controller/restapi/v1/response"

	"github.com/veilence/veilence-mx/backend/internal/usecase/rbac"
)

type createOrgRequest struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
}

type updateOrgRequest struct {
	Name        *string `json:"name"`
	Slug        *string `json:"slug"`
	Description *string `json:"description"`
}

// CreateWorkspace handles POST /api/workspaces - creates a new workspace.
func (h *WorkspaceHandlers) CreateWorkspace(w http.ResponseWriter, r *http.Request) {
	userID := rbac.UserIDFromContext(r.Context())
	if userID == 0 {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	var req createOrgRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := validation.ValidateRequired(req.Name, "name"); err != nil {
		respondAppError(w, Validation(err.Error()))
		return
	}
	if err := validation.ValidateSlug(req.Slug); err != nil {
		respondAppError(w, Validation(err.Error()))
		return
	}

	org, err := h.RBAC.CreateWorkspace(userID, req.Name, req.Slug, req.Description)
	if err != nil {
		if errors.Is(err, rbac.ErrSlugTaken) {
			respondError(w, http.StatusConflict, "workspace slug is already taken")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to create workspace")
		return
	}

	h.Audit.LogAction(r.Context(), "create", "workspace", org.ID, fmt.Sprintf("created workspace %q (slug: %s)", req.Name, req.Slug))

	respondJSON(w, http.StatusCreated, response.WorkspaceResponse{
		ID: org.ID, Name: org.Name, Slug: org.Slug, Description: org.Description,
		OwnerID: org.OwnerID, IsActive: org.IsActive, CreatedAt: org.CreatedAt, UpdatedAt: org.UpdatedAt,
	}, nil)
}

// ListWorkspaces handles GET /api/workspaces - lists workspaces the user belongs to.
func (h *WorkspaceHandlers) ListWorkspaces(w http.ResponseWriter, r *http.Request) {
	userID := rbac.UserIDFromContext(r.Context())
	if userID == 0 {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	orgs, err := h.RBAC.GetUserWorkspaces(userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list workspaces")
		return
	}

	result := make([]response.WorkspaceResponse, len(orgs))
	for i, o := range orgs {
		result[i] = response.WorkspaceResponse{
			ID: o.ID, Name: o.Name, Slug: o.Slug, Description: o.Description,
			OwnerID: o.OwnerID, IsActive: o.IsActive, CreatedAt: o.CreatedAt, UpdatedAt: o.UpdatedAt,
		}
	}
	respondJSON(w, http.StatusOK, result, nil)
}

// GetWorkspace handles GET /api/workspaces/{workspaceId} - returns workspace details.
func (h *WorkspaceHandlers) GetWorkspace(w http.ResponseWriter, r *http.Request) {
	workspaceID := rbac.WorkspaceIDFromContext(r.Context())
	if workspaceID == 0 {
		respondError(w, http.StatusBadRequest, "workspace context required")
		return
	}

	org, err := h.RBAC.GetWorkspace(workspaceID)
	if err != nil {
		if errors.Is(err, rbac.ErrWorkspaceNotFound) {
			respondError(w, http.StatusNotFound, "workspace not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to get workspace")
		return
	}

	respondJSON(w, http.StatusOK, response.WorkspaceResponse{
		ID: org.ID, Name: org.Name, Slug: org.Slug, Description: org.Description,
		OwnerID: org.OwnerID, IsActive: org.IsActive, CreatedAt: org.CreatedAt, UpdatedAt: org.UpdatedAt,
	}, nil)
}

// UpdateWorkspace handles PUT /api/workspaces/{workspaceId} - updates workspace details.
func (h *WorkspaceHandlers) UpdateWorkspace(w http.ResponseWriter, r *http.Request) {
	workspaceID := rbac.WorkspaceIDFromContext(r.Context())
	if workspaceID == 0 {
		respondError(w, http.StatusBadRequest, "workspace context required")
		return
	}

	var req updateOrgRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Fetch existing org to fill in missing fields (support partial updates)
	existingOrg, err := h.RBAC.GetWorkspace(workspaceID)
	if err != nil {
		respondError(w, http.StatusNotFound, "workspace not found")
		return
	}

	name := existingOrg.Name
	if req.Name != nil && *req.Name != "" {
		name = *req.Name
	}
	slug := existingOrg.Slug
	if req.Slug != nil && *req.Slug != "" {
		if err := validation.ValidateSlug(*req.Slug); err != nil {
			respondAppError(w, Validation(err.Error()))
			return
		}
		slug = *req.Slug
	}
	description := existingOrg.Description
	if req.Description != nil {
		description = *req.Description
	}

	org, err := h.RBAC.UpdateWorkspace(workspaceID, name, slug, description)
	if err != nil {
		if errors.Is(err, rbac.ErrWorkspaceNotFound) {
			respondError(w, http.StatusNotFound, "workspace not found")
			return
		}
		if errors.Is(err, rbac.ErrSlugTaken) {
			respondError(w, http.StatusConflict, "workspace slug is already taken")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to update workspace")
		return
	}

	h.Audit.LogAction(r.Context(), "update", "workspace", workspaceID, fmt.Sprintf("updated workspace %q (slug: %s)", name, slug))

	respondJSON(w, http.StatusOK, response.WorkspaceResponse{
		ID: org.ID, Name: org.Name, Slug: org.Slug, Description: org.Description,
		OwnerID: org.OwnerID, IsActive: org.IsActive, CreatedAt: org.CreatedAt, UpdatedAt: org.UpdatedAt,
	}, nil)
}

// DeleteWorkspace handles DELETE /api/workspaces/{workspaceId} - soft-deletes a workspace.
func (h *WorkspaceHandlers) DeleteWorkspace(w http.ResponseWriter, r *http.Request) {
	wsIDStr := chi.URLParam(r, "workspaceId")
	workspaceID, err := strconv.ParseUint(wsIDStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid workspace ID")
		return
	}

	if err := h.RBAC.DeleteWorkspace(uint(workspaceID)); err != nil {
		if errors.Is(err, rbac.ErrWorkspaceNotFound) {
			respondError(w, http.StatusNotFound, "workspace not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to delete workspace")
		return
	}

	h.Audit.LogAction(r.Context(), "delete", "workspace", uint(workspaceID), fmt.Sprintf("deleted workspace %d", workspaceID))

	respondJSON(w, http.StatusOK, nil, nil)
}
