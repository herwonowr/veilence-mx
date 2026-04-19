package entity

import "time"

// Workspace represents a tenant workspace in the system.
type Workspace struct {
	ID             uint
	Name           string
	Slug           string
	Description    string
	OwnerID        uint
	IsActive       bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// WorkspaceMember represents a user's membership in a workspace with a specific role.
type WorkspaceMember struct {
	ID             uint
	WorkspaceID    uint
	UserID         uint
	RoleID         uint
	Role           *Role
	JoinedAt       time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
