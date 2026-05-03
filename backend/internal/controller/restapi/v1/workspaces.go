package v1

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	validation "github.com/veilence/veilence-mx/backend/internal/controller/restapi/v1/request"
	"github.com/veilence/veilence-mx/backend/internal/controller/restapi/v1/response"
	"github.com/veilence/veilence-mx/backend/internal/entity"
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
	if userID == "" {
		respondError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	var req createOrgRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
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

	ws, err := h.RBAC.CreateWorkspace(r.Context(), userID, req.Name, req.Slug, req.Description)
	if err != nil {
		if errors.Is(err, rbac.ErrSlugTaken) {
			respondError(w, http.StatusConflict, "Workspace slug is already taken")
			return
		}
		respondError(w, http.StatusInternalServerError, "Failed to create workspace")
		return
	}

	h.Audit.LogAction(r.Context(), "create", "workspace", ws.ID, fmt.Sprintf("created workspace %q (slug: %s)", req.Name, req.Slug))

	respondJSON(w, http.StatusCreated, response.WorkspaceResponse{
		ID: ws.ID, Name: ws.Name, Slug: ws.Slug, Description: ws.Description,
		OwnerID: ws.OwnerID, IsActive: ws.IsActive, CreatedAt: ws.CreatedAt, UpdatedAt: ws.UpdatedAt,
	}, nil)
}

// ListWorkspaces handles GET /api/workspaces - lists workspaces the user belongs to.
// Supports pagination via ?page=&limit= and search via ?search= query params.
func (h *WorkspaceHandlers) ListWorkspaces(w http.ResponseWriter, r *http.Request) {
	userID := rbac.UserIDFromContext(r.Context())
	if userID == "" {
		respondError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	page, limit := parsePagination(r)
	search := r.URL.Query().Get("search")

	res, err := h.RBAC.GetUserWorkspaces(r.Context(), userID, entity.WorkspaceListParams{
		Page:   page,
		Limit:  limit,
		Search: search,
	})
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to list workspaces")
		return
	}

	result := make([]response.WorkspaceResponse, len(res.Workspaces))

	for i, o := range res.Workspaces {
		result[i] = response.WorkspaceResponse{
			ID: o.ID, Name: o.Name, Slug: o.Slug, Description: o.Description,
			OwnerID: o.OwnerID, IsActive: o.IsActive, Role: o.Role, CreatedAt: o.CreatedAt, UpdatedAt: o.UpdatedAt,
		}

		// Populate package count
		if pkgCount, err := h.PkgSvc.CountPackages(r.Context(), o.ID); err == nil {
			result[i].PackageCount = &pkgCount
		}

		// Populate member count
		if members, err := h.RBAC.GetWorkspaceMembers(r.Context(), o.ID); err == nil {
			memberCount := int64(len(members))
			result[i].MemberCount = &memberCount
		}
	}
	respondJSON(w, http.StatusOK, result, &Meta{Page: page, Limit: limit, Total: res.Total})
}

// GetWorkspace handles GET /api/workspaces/{workspaceId} - returns workspace details.
func (h *WorkspaceHandlers) GetWorkspace(w http.ResponseWriter, r *http.Request) {
	workspaceID := rbac.WorkspaceIDFromContext(r.Context())
	if workspaceID == "" {
		respondError(w, http.StatusBadRequest, "Workspace context required")
		return
	}

	ws, err := h.RBAC.GetWorkspace(r.Context(), workspaceID)
	if err != nil {
		if errors.Is(err, rbac.ErrWorkspaceNotFound) {
			respondError(w, http.StatusNotFound, "Workspace not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "Failed to get workspace")
		return
	}

	respondJSON(w, http.StatusOK, response.WorkspaceResponse{
		ID: ws.ID, Name: ws.Name, Slug: ws.Slug, Description: ws.Description,
		OwnerID: ws.OwnerID, IsActive: ws.IsActive, CreatedAt: ws.CreatedAt, UpdatedAt: ws.UpdatedAt,
	}, nil)
}

// UpdateWorkspace handles PUT /api/workspaces/{workspaceId} - updates workspace details.
func (h *WorkspaceHandlers) UpdateWorkspace(w http.ResponseWriter, r *http.Request) {
	workspaceID := rbac.WorkspaceIDFromContext(r.Context())
	if workspaceID == "" {
		respondError(w, http.StatusBadRequest, "Workspace context required")
		return
	}

	var req updateOrgRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Fetch existing workspace to fill in missing fields (support partial updates)
	existingOrg, err := h.RBAC.GetWorkspace(r.Context(), workspaceID)
	if err != nil {
		respondError(w, http.StatusNotFound, "Workspace not found")
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

	ws, err := h.RBAC.UpdateWorkspace(r.Context(), workspaceID, name, slug, description)
	if err != nil {
		if errors.Is(err, rbac.ErrWorkspaceNotFound) {
			respondError(w, http.StatusNotFound, "Workspace not found")
			return
		}
		if errors.Is(err, rbac.ErrSlugTaken) {
			respondError(w, http.StatusConflict, "Workspace slug is already taken")
			return
		}
		respondError(w, http.StatusInternalServerError, "Failed to update workspace")
		return
	}

	h.Audit.LogAction(r.Context(), "update", "workspace", workspaceID, fmt.Sprintf("updated workspace %q (slug: %s)", name, slug))

	respondJSON(w, http.StatusOK, response.WorkspaceResponse{
		ID: ws.ID, Name: ws.Name, Slug: ws.Slug, Description: ws.Description,
		OwnerID: ws.OwnerID, IsActive: ws.IsActive, CreatedAt: ws.CreatedAt, UpdatedAt: ws.UpdatedAt,
	}, nil)
}

// DeleteWorkspace handles DELETE /api/workspaces/{workspaceId} - soft-deletes a workspace.
func (h *WorkspaceHandlers) DeleteWorkspace(w http.ResponseWriter, r *http.Request) {
	workspaceID := rbac.WorkspaceIDFromContext(r.Context())
	if workspaceID == "" {
		respondError(w, http.StatusBadRequest, "Workspace context required")
		return
	}

	// Fetch workspace name before deletion for audit log readability.
	wsForAudit, _ := h.RBAC.GetWorkspace(r.Context(), workspaceID)

	if err := h.RBAC.DeleteWorkspace(r.Context(), workspaceID); err != nil {
		if errors.Is(err, rbac.ErrWorkspaceNotFound) {
			respondError(w, http.StatusNotFound, "Workspace not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "Failed to delete workspace")
		return
	}

	wsName := workspaceID
	if wsForAudit != nil {
		wsName = wsForAudit.Name
	}
	h.Audit.LogAction(r.Context(), "delete", "workspace", workspaceID, fmt.Sprintf("deleted workspace %q", wsName))

	respondJSON(w, http.StatusOK, nil, nil)
}
