package response

import "time"

// WorkspaceResponse is the JSON representation of a workspace.
type WorkspaceResponse struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Slug         string    `json:"slug"`
	Description  string    `json:"description"`
	OwnerID      string    `json:"ownerId"`
	IsActive     bool      `json:"isActive"`
	Role         string    `json:"role"`
	PackageCount *int64    `json:"packageCount,omitempty"`
	MemberCount  *int64    `json:"memberCount,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}
