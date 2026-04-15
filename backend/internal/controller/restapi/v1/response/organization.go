package response

import "time"

// OrganizationResponse is the JSON representation of an organization.
// Mapped from persistent.Organization at the handler level because the RBAC
// service currently returns persistent types (pre-existing arch compromise).
type OrganizationResponse struct {
	ID          uint      `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description string    `json:"description"`
	OwnerID     uint      `json:"ownerId"`
	IsActive    bool      `json:"isActive"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}
