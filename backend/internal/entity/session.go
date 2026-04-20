package entity

import "time"

// Session represents an active user session with metadata for session management.
type Session struct {
	ID             string
	UserID         string
	TokenHash      string
	IPAddress      string
	UserAgent      string
	CreatedAt      time.Time
	LastActive     time.Time
	ExpiresAt      time.Time
}
