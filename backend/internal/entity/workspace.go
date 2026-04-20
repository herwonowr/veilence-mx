package entity

import "time"

// Workspace represents a tenant workspace in the system.
type Workspace struct {
	ID             string
	Name           string
	Slug           string
	Description    string
	OwnerID        string
	IsActive       bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// WorkspaceMember represents a user's membership in a workspace with a specific role.
type WorkspaceMember struct {
	ID             string
	WorkspaceID    string
	UserID         string
	RoleID         string
	Role           *Role
	User           *User
	JoinedAt       time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
