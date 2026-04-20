package entity

import "time"

// APIKeyRole defines the RBAC role assigned to an API key.
// Uses the same role constants as workspace roles (owner, admin, member, viewer).
type APIKeyRole string

const (
	// APIKeyRoleViewer allows read-only access.
	APIKeyRoleViewer APIKeyRole = "viewer"
	// APIKeyRoleMember allows standard read/write access.
	APIKeyRoleMember APIKeyRole = "member"
	// APIKeyRoleAdmin allows administrative access.
	APIKeyRoleAdmin APIKeyRole = "admin"
)

// ValidAPIKeyRoles is the set of valid API key roles.
// Note: "owner" is intentionally excluded - API keys cannot have owner-level access.
var ValidAPIKeyRoles = []APIKeyRole{APIKeyRoleViewer, APIKeyRoleMember, APIKeyRoleAdmin}

// IsValidAPIKeyRole returns true if the given role string is a valid API key role.
func IsValidAPIKeyRole(s string) bool {
	for _, v := range ValidAPIKeyRoles {
		if string(v) == s {
			return true
		}
	}
	return false
}

// RoleHierarchy maps role names to numeric levels for comparison.
// Higher number = more privileged.
var RoleHierarchy = map[string]int{
	"viewer": 1,
	"member": 2,
	"admin":  3,
	"owner":  4,
}

// RoleLevel returns the numeric privilege level for a role name.
// Returns 0 for unknown roles.
func RoleLevel(role string) int {
	return RoleHierarchy[role]
}

// APIKey represents a long-lived API key for programmatic access.
type APIKey struct {
	ID          string
	UserID      string
	WorkspaceID string
	Name        string
	KeyHash     string
	KeyPrefix   string
	Role        APIKeyRole
	LastUsedAt  *time.Time
	ExpiresAt   *time.Time
	IsActive    bool
	CreatedAt   time.Time
}
