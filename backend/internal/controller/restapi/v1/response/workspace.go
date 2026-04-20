package response

import "time"

// WorkspaceResponse is the JSON representation of a workspace.
// Mapped from persistent.Workspace at the handler level because the RBAC
// service currently returns persistent types (pre-existing arch compromise).
type WorkspaceResponse struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description string    `json:"description"`
	OwnerID     string    `json:"ownerId"`
	IsActive    bool      `json:"isActive"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}
