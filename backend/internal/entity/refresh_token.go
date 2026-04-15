package entity

import "time"

// RefreshToken represents a stored refresh token for token rotation.
// Only the SHA-256 hash of the token is stored -- the plaintext token
// is returned to the client once and never persisted.
type RefreshToken struct {
	ID             uint
	UserID         uint
	TokenHash      string
	ExpiresAt      time.Time
	CreatedAt      time.Time
}
