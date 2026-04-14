package entity

import "time"

// RefreshToken represents a stored refresh token for token rotation.
// Only the SHA-256 hash of the token is stored -- the plaintext token
// is returned to the client once and never persisted.
type RefreshToken struct {
	ID             uint `json:"id"`
	UserID         uint `json:"userId"`
	TokenHash      string `json:"-"`
	ExpiresAt      time.Time `json:"expiresAt"`
	CreatedAt      time.Time `json:"createdAt"`
}
