package models

import (
	"time"
)

// Invitation represents a pending invitation for a user to join an organization.
type Invitation struct {
	ID         uint       `gorm:"primarykey" json:"id"`
	OrgID      uint       `gorm:"not null;index" json:"orgId"`
	Email      string     `gorm:"not null;type:varchar(255)" json:"email"`
	RoleID     uint       `gorm:"not null" json:"roleId"`
	TokenHash  string     `gorm:"uniqueIndex;not null;type:varchar(255);column:token_hash" json:"-"`
	InvitedBy  uint       `gorm:"not null" json:"invitedBy"`
	ExpiresAt  time.Time  `gorm:"not null" json:"expiresAt"`
	AcceptedAt *time.Time `json:"acceptedAt,omitempty"`
	CreatedAt  time.Time  `json:"createdAt"`
}

// TableName returns the table name for Invitation.
func (Invitation) TableName() string {
	return "invitations"
}
