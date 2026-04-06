package handlers

import (
	"net/http"

	"github.com/veilence/veilence-mx/backend/internal/rbac"
)

// ListRoles handles GET /api/orgs/{orgId}/roles — lists roles in the organization.
func (h *OrgHandlers) ListRoles(w http.ResponseWriter, r *http.Request) {
	orgID := rbac.OrgIDFromContext(r.Context())
	if orgID == 0 {
		respondError(w, http.StatusBadRequest, "organization context required")
		return
	}

	roles, err := h.RBAC.GetOrgRoles(orgID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list roles")
		return
	}

	respondJSON(w, http.StatusOK, roles, nil)
}

// ListPermissions handles GET /api/permissions — lists all available system permissions.
func (h *OrgHandlers) ListPermissions(w http.ResponseWriter, r *http.Request) {
	perms, err := h.RBAC.GetAllPermissions()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list permissions")
		return
	}

	respondJSON(w, http.StatusOK, perms, nil)
}
