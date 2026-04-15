package entity

import "time"

// Invitation represents a pending invitation for a user to join an organization.
type Invitation struct {
	ID             uint
	OrgID          uint
	Email          string
	RoleID         uint
	TokenHash      string
	InvitedBy      uint
	ExpiresAt      time.Time
	AcceptedAt     *time.Time
	CreatedAt      time.Time
}
