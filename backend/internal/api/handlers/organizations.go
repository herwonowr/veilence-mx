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

// CreateOrganization handles POST /api/orgs — creates a new organization.
func (h *OrgHandlers) CreateOrganization(w http.ResponseWriter, r *http.Request) {
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
		respondAppError(w, apperror.Validation(err.Error()))
		return
	}
	if err := validation.ValidateSlug(req.Slug); err != nil {
		respondAppError(w, apperror.Validation(err.Error()))
		return
	}

	org, err := h.RBAC.CreateOrganization(userID, req.Name, req.Slug, req.Description)
	if err != nil {
		if errors.Is(err, rbac.ErrSlugTaken) {
			respondError(w, http.StatusConflict, "organization slug is already taken")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to create organization")
		return
	}

	h.Audit.LogAction(r.Context(), "create", "org", org.ID, fmt.Sprintf("created organization %q (slug: %s)", req.Name, req.Slug))

	respondJSON(w, http.StatusCreated, org, nil)
}

// ListOrganizations handles GET /api/orgs — lists organizations the user belongs to.
func (h *OrgHandlers) ListOrganizations(w http.ResponseWriter, r *http.Request) {
	userID := rbac.UserIDFromContext(r.Context())
	if userID == 0 {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	orgs, err := h.RBAC.GetUserOrganizations(userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list organizations")
		return
	}

	respondJSON(w, http.StatusOK, orgs, nil)
}

// GetOrganization handles GET /api/orgs/{orgId} — returns organization details.
func (h *OrgHandlers) GetOrganization(w http.ResponseWriter, r *http.Request) {
	orgID := rbac.OrgIDFromContext(r.Context())
	if orgID == 0 {
		respondError(w, http.StatusBadRequest, "organization context required")
		return
	}

	org, err := h.RBAC.GetOrganization(orgID)
	if err != nil {
		if errors.Is(err, rbac.ErrOrgNotFound) {
			respondError(w, http.StatusNotFound, "organization not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to get organization")
		return
	}

	respondJSON(w, http.StatusOK, org, nil)
}

// UpdateOrganization handles PUT /api/orgs/{orgId} — updates organization details.
func (h *OrgHandlers) UpdateOrganization(w http.ResponseWriter, r *http.Request) {
	orgID := rbac.OrgIDFromContext(r.Context())
	if orgID == 0 {
		respondError(w, http.StatusBadRequest, "organization context required")
		return
	}

	var req updateOrgRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Fetch existing org to fill in missing fields (support partial updates)
	existingOrg, err := h.RBAC.GetOrganization(orgID)
	if err != nil {
		respondError(w, http.StatusNotFound, "organization not found")
		return
	}

	name := existingOrg.Name
	if req.Name != nil && *req.Name != "" {
		name = *req.Name
	}
	slug := existingOrg.Slug
	if req.Slug != nil && *req.Slug != "" {
		if err := validation.ValidateSlug(*req.Slug); err != nil {
			respondAppError(w, apperror.Validation(err.Error()))
			return
		}
		slug = *req.Slug
	}
	description := existingOrg.Description
	if req.Description != nil {
		description = *req.Description
	}

	org, err := h.RBAC.UpdateOrganization(orgID, name, slug, description)
	if err != nil {
		if errors.Is(err, rbac.ErrOrgNotFound) {
			respondError(w, http.StatusNotFound, "organization not found")
			return
		}
		if errors.Is(err, rbac.ErrSlugTaken) {
			respondError(w, http.StatusConflict, "organization slug is already taken")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to update organization")
		return
	}

	h.Audit.LogAction(r.Context(), "update", "org", orgID, fmt.Sprintf("updated organization %q (slug: %s)", name, slug))

	respondJSON(w, http.StatusOK, org, nil)
}

// DeleteOrganization handles DELETE /api/orgs/{orgId} — soft-deletes an organization.
func (h *OrgHandlers) DeleteOrganization(w http.ResponseWriter, r *http.Request) {
	orgIDStr := chi.URLParam(r, "orgId")
	orgID, err := strconv.ParseUint(orgIDStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid organization ID")
		return
	}

	if err := h.RBAC.DeleteOrganization(uint(orgID)); err != nil {
		if errors.Is(err, rbac.ErrOrgNotFound) {
			respondError(w, http.StatusNotFound, "organization not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to delete organization")
		return
	}

	h.Audit.LogAction(r.Context(), "delete", "org", uint(orgID), fmt.Sprintf("deleted organization %d", orgID))

	respondJSON(w, http.StatusOK, nil, nil)
}
