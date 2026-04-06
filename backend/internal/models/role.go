package models

import (
	"time"
)

// Role represents a named set of permissions within an organization.
type Role struct {
	ID          uint         `gorm:"primarykey" json:"id"`
	OrgID       uint         `gorm:"not null;index" json:"orgId"`
	Name        string       `gorm:"not null;type:varchar(50)" json:"name"`
	Description string       `gorm:"type:text" json:"description"`
	IsSystem    bool         `gorm:"not null;default:false" json:"isSystem"`
	CreatedAt   time.Time    `json:"createdAt"`
	UpdatedAt   time.Time    `json:"updatedAt"`
	Permissions []Permission `gorm:"many2many:role_permissions" json:"permissions,omitempty"`
}

// TableName returns the table name for Role.
func (Role) TableName() string {
	return "roles"
}

// Permission represents a single resource-action permission.
type Permission struct {
	ID       uint   `gorm:"primarykey" json:"id"`
	Resource string `gorm:"not null;type:varchar(50)" json:"resource"`
	Action   string `gorm:"not null;type:varchar(50)" json:"action"`
}

// TableName returns the table name for Permission.
func (Permission) TableName() string {
	return "permissions"
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
