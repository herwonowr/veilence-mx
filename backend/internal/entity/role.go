package entity

import "time"

// Role represents a named set of permissions within an organization.
type Role struct {
	ID             uint `json:"id"`
	OrgID          uint `json:"orgId"`
	Name           string `json:"name"`
	Description    string `json:"description"`
	IsSystem       bool `json:"isSystem"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
	Permissions    []Permission `json:"permissions,omitempty"`
}

// Permission represents a single resource-action permission.
type Permission struct {
	ID             uint `json:"id"`
	Resource       string `json:"resource"`
	Action         string `json:"action"`
}

// System role names.
const (
	RoleOwner  = "owner"
	RoleAdmin  = "admin"
	RoleMember = "member"
	RoleViewer = "viewer"
)

// SystemPermissions defines all permissions available in the system.
var SystemPermissions = []Permission{
	{Resource: "packages", Action: "read"},
	{Resource: "packages", Action: "write"},
	{Resource: "packages", Action: "delete"},
	{Resource: "alerts", Action: "read"},
	{Resource: "alerts", Action: "write"},
	{Resource: "releases", Action: "read"},
	{Resource: "settings", Action: "read"},
	{Resource: "settings", Action: "write"},
	{Resource: "members", Action: "read"},
	{Resource: "members", Action: "invite"},
	{Resource: "members", Action: "remove"},
	{Resource: "roles", Action: "read"},
	{Resource: "roles", Action: "write"},
	{Resource: "org", Action: "read"},
	{Resource: "org", Action: "write"},
	{Resource: "org", Action: "delete"},
	{Resource: "api_keys", Action: "read"},
	{Resource: "api_keys", Action: "write"},
	{Resource: "audit", Action: "read"},
	{Resource: "notifications", Action: "read"},
	{Resource: "notifications", Action: "create"},
	{Resource: "notifications", Action: "update"},
	{Resource: "notifications", Action: "delete"},
}
