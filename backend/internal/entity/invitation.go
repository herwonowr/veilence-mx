package entity

import "time"

// Invitation represents a pending invitation for a user to join a workspace.
type Invitation struct {
	ID             string
	WorkspaceID    string
	Email          string
	RoleID         string
	TokenHash      string
	InvitedBy      string
	ExpiresAt      time.Time
	AcceptedAt     *time.Time
	DeclinedAt     *time.Time
	CreatedAt      time.Time
	WorkspaceName  string // populated by joins, not persisted
	RoleName       string // populated by joins, not persisted
	InvitedByEmail string // populated by joins, not persisted
}
