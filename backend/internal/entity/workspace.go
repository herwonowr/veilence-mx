package entity

import "time"

// Workspace represents a tenant workspace in the system.
type Workspace struct {
	ID          string
	Name        string
	Slug        string
	Description string
	OwnerID     string
	IsActive    bool
	Role        string // Contextual field populated by queries, not always present.
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// WorkspaceListParams holds pagination, search, and sort parameters for listing
// workspaces a user belongs to.
type WorkspaceListParams struct {
	Page   int
	Limit  int
	Search string
}

// WorkspaceListResult holds a page of workspaces together with the total count
// for pagination metadata.
type WorkspaceListResult struct {
	Workspaces []Workspace
	Total      int64
}

// WorkspaceMember represents a user's membership in a workspace with a specific role.
type WorkspaceMember struct {
	ID          string
	WorkspaceID string
	UserID      string
	RoleID      string
	Role        *Role
	User        *User
	JoinedAt    time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
