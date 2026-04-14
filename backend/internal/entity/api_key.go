package entity

import "time"

// APIKeyScope defines the permission level for an API key.
type APIKeyScope string

const (
	// APIKeyScopeRead allows read-only access (GET requests).
	APIKeyScopeRead APIKeyScope = "read"
	// APIKeyScopeWrite allows read and write access (GET, POST, PUT, PATCH requests).
	APIKeyScopeWrite APIKeyScope = "write"
	// APIKeyScopeAdmin allows full access including admin-level operations.
	APIKeyScopeAdmin APIKeyScope = "admin"
)

// ValidAPIKeyScopes is the set of valid API key scopes.
var ValidAPIKeyScopes = []APIKeyScope{APIKeyScopeRead, APIKeyScopeWrite, APIKeyScopeAdmin}

// IsValidAPIKeyScope returns true if the given scope string is a valid API key scope.
func IsValidAPIKeyScope(s string) bool {
	for _, v := range ValidAPIKeyScopes {
		if string(v) == s {
			return true
		}
	}
	return false
}

// ScopeAllows returns true if the scope grants access for the given HTTP method.
// read: GET, HEAD, OPTIONS
// write: read + POST, PUT, PATCH (but not DELETE on critical resources)
// admin: all methods
func (s APIKeyScope) ScopeAllows(method string) bool {
	switch s {
	case APIKeyScopeAdmin:
		return true
	case APIKeyScopeWrite:
		return method != "DELETE"
	case APIKeyScopeRead:
		return method == "GET" || method == "HEAD" || method == "OPTIONS"
	default:
		return false
	}
}

// APIKey represents a long-lived API key for programmatic access.
type APIKey struct {
	ID             uint `json:"id"`
	UserID         uint `json:"userId"`
	Name           string `json:"name"`
	KeyHash        string `json:"-"`
	KeyPrefix      string `json:"keyPrefix"`
	Scope          APIKeyScope `json:"scope"`
	LastUsedAt     *time.Time `json:"lastUsedAt,omitempty"`
	ExpiresAt      *time.Time `json:"expiresAt,omitempty"`
	IsActive       bool `json:"isActive"`
	CreatedAt      time.Time `json:"createdAt"`
}
