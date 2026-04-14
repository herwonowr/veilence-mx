package entity

import "time"

// Session represents an active user session with metadata for session management.
type Session struct {
	ID             uint `json:"id"`
	UserID         uint `json:"userId"`
	TokenHash      string `json:"-"`
	IPAddress      string `json:"iPAddress"`
	UserAgent      string `json:"userAgent"`
	CreatedAt      time.Time `json:"createdAt"`
	LastActive     time.Time `json:"lastActive"`
	ExpiresAt      time.Time `json:"expiresAt"`
}
