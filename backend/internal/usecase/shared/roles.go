package shared

import (
	"context"
	"fmt"
	"maps"
	"slices"

	"github.com/veilence/veilence-mx/backend/internal/entity"
	"github.com/veilence/veilence-mx/backend/internal/usecase"
)

// CreateDefaultRoles creates the four system roles (owner, admin, member, viewer)
// with their respective permissions for the given workspace. Both the setup and
// rbac packages call this to avoid duplicating role definitions.
func CreateDefaultRoles(ctx context.Context, repo usecase.RBACRepository, workspaceID string) ([]entity.Role, error) {
	allPerms, err := repo.FindAllPermissions(ctx)
	if err != nil {
		return nil, fmt.Errorf("loading permissions: %w", err)
	}

	permMap := make(map[string]entity.Permission)
	for _, p := range allPerms {
		key := p.Resource + ":" + p.Action
		permMap[key] = p
	}

	lookupPerms := func(keys []string) []entity.Permission {
		var perms []entity.Permission
		for _, key := range keys {
			if p, ok := permMap[key]; ok {
				perms = append(perms, p)
			}
		}
		return perms
	}

	// All permission keys
	allKeys := slices.Collect(maps.Keys(permMap))

	// Admin gets all except workspace:delete
	adminKeys := slices.DeleteFunc(slices.Clone(allKeys), func(k string) bool {
		return k == "workspace:delete"
	})

	// Member permissions - read-all plus write access to packages/alerts
	memberKeys := []string{
		"packages:read", "packages:write",
		"alerts:read", "alerts:write",
		"releases:read",
		"settings:read",
		"members:read",
		"roles:read",
		"workspace:read",
		"audit:read",
		"api_keys:read", "api_keys:write",
		"notifications:read",
	}

	// Viewer permissions - read-only access to everything
	viewerKeys := []string{
		"packages:read",
		"alerts:read",
		"releases:read",
		"settings:read",
		"workspace:read",
		"members:read",
		"roles:read",
		"audit:read",
		"api_keys:read",
		"notifications:read",
	}

	roleDefinitions := []struct {
		Name        string
		Description string
		PermKeys    []string
	}{
		{entity.RoleOwner, "Full access to the workspace", allKeys},
		{entity.RoleAdmin, "Administrative access (cannot delete workspace)", adminKeys},
		{entity.RoleMember, "Standard member with read/write access", memberKeys},
		{entity.RoleViewer, "Read-only access", viewerKeys},
	}

	var roles []entity.Role
	for _, def := range roleDefinitions {
		role := entity.Role{
			WorkspaceID: workspaceID,
			Name:        def.Name,
			Description: def.Description,
			IsSystem:    true,
			Permissions: lookupPerms(def.PermKeys),
		}
		if err := repo.CreateRole(ctx, &role); err != nil {
			return nil, fmt.Errorf("creating role %s: %w", def.Name, err)
		}
		roles = append(roles, role)
	}

	return roles, nil
}
