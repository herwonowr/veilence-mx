package entity

import "time"

// Invitation represents a pending invitation for a user to join a workspace.
type Invitation struct {
	ID             uint
	WorkspaceID    uint
	Email          string
	RoleID         uint
	TokenHash      string
	InvitedBy      uint
	ExpiresAt      time.Time
	AcceptedAt     *time.Time
	CreatedAt      time.Time
}
