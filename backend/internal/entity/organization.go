package entity

import "time"

// Organization represents a tenant organization in the system.
type Organization struct {
	ID             uint
	Name           string
	Slug           string
	Description    string
	OwnerID        uint
	IsActive       bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// OrgMember represents a user's membership in an organization with a specific role.
type OrgMember struct {
	ID             uint
	OrgID          uint
	UserID         uint
	RoleID         uint
	Role           *Role
	JoinedAt       time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
