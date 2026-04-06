package domain

import "time"

// Organization represents a tenant organization in the system.
type Organization struct {
	ID          uint      `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description string    `json:"description"`
	OwnerID     uint      `json:"ownerId"`
	IsActive    bool      `json:"isActive"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// OrgMember represents a user's membership in an organization with a specific role.
type OrgMember struct {
	ID        uint      `json:"id"`
	OrgID     uint      `json:"orgId"`
	UserID    uint      `json:"userId"`
	RoleID    uint      `json:"roleId"`
	Role      *Role     `json:"role,omitempty"`
	JoinedAt  time.Time `json:"joinedAt"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
