package entity

import "time"

// Session represents an active user session with metadata for session management.
type Session struct {
	ID             uint
	UserID         uint
	TokenHash      string
	IPAddress      string
	UserAgent      string
	CreatedAt      time.Time
	LastActive     time.Time
	ExpiresAt      time.Time
}
