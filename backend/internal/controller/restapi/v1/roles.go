package v1

import (
	"net/http"

	"github.com/veilence/veilence-mx/backend/internal/controller/restapi/v1/response"
	"github.com/veilence/veilence-mx/backend/internal/usecase/rbac"
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

	result := make([]response.RoleResponse, len(roles))
	for i, role := range roles {
		var perms []response.PermissionResponse
		if role.Permissions != nil {
			perms = make([]response.PermissionResponse, len(role.Permissions))
			for j, p := range role.Permissions {
				perms[j] = response.PermissionResponse{ID: p.ID, Resource: p.Resource, Action: p.Action}
			}
		}
		result[i] = response.RoleResponse{
			ID: role.ID, OrgID: role.OrgID, Name: role.Name, Description: role.Description,
			IsSystem: role.IsSystem, CreatedAt: role.CreatedAt, UpdatedAt: role.UpdatedAt,
			Permissions: perms,
		}
	}
	respondJSON(w, http.StatusOK, result, nil)
}

// ListPermissions handles GET /api/permissions — lists all available system permissions.
func (h *OrgHandlers) ListPermissions(w http.ResponseWriter, r *http.Request) {
	perms, err := h.RBAC.GetAllPermissions()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list permissions")
		return
	}

	result := make([]response.PermissionResponse, len(perms))
	for i, p := range perms {
		result[i] = response.PermissionResponse{ID: p.ID, Resource: p.Resource, Action: p.Action}
	}
	respondJSON(w, http.StatusOK, result, nil)
}
