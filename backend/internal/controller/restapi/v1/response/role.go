package response

import "time"

// RoleResponse is the JSON representation of an organization role.
// Mapped from persistent.Role at the handler level because the RBAC
// service currently returns persistent types (pre-existing arch compromise).
type RoleResponse struct {
	ID          uint                 `json:"id"`
	OrgID       uint                 `json:"orgId"`
	Name        string               `json:"name"`
	Description string               `json:"description"`
	IsSystem    bool                 `json:"isSystem"`
	CreatedAt   time.Time            `json:"createdAt"`
	UpdatedAt   time.Time            `json:"updatedAt"`
	Permissions []PermissionResponse `json:"permissions,omitempty"`
}

// PermissionResponse is the JSON representation of a permission.
type PermissionResponse struct {
	ID       uint   `json:"id"`
	Resource string `json:"resource"`
	Action   string `json:"action"`
}
