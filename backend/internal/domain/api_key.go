package domain

import "time"

// APIKey represents a long-lived API key for programmatic access.
type APIKey struct {
	ID         uint       `json:"id"`
	UserID     uint       `json:"userId"`
	Name       string     `json:"name"`
	KeyHash    string     `json:"-"`
	KeyPrefix  string     `json:"keyPrefix"`
	LastUsedAt *time.Time `json:"lastUsedAt,omitempty"`
	ExpiresAt  *time.Time `json:"expiresAt,omitempty"`
	IsActive   bool       `json:"isActive"`
	CreatedAt  time.Time  `json:"createdAt"`
}
