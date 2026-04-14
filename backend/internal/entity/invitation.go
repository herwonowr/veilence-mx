package entity

import "time"

// Invitation represents a pending invitation for a user to join an organization.
type Invitation struct {
	ID             uint `json:"id"`
	OrgID          uint `json:"orgId"`
	Email          string `json:"email"`
	RoleID         uint `json:"roleId"`
	TokenHash      string `json:"-"`
	InvitedBy      uint `json:"invitedBy"`
	ExpiresAt      time.Time `json:"expiresAt"`
	AcceptedAt     *time.Time `json:"acceptedAt,omitempty"`
	CreatedAt      time.Time `json:"createdAt"`
}
