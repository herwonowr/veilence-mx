package models

import (
	"time"
)

// RefreshToken represents a stored refresh token for token rotation.
// Only the SHA-256 hash of the token is stored — the plaintext token
// is returned to the client once and never persisted.
type RefreshToken struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	UserID    uint      `gorm:"not null;index" json:"userId"`
	TokenHash string    `gorm:"uniqueIndex;not null;column:token_hash;type:varchar(255)" json:"-"`
	ExpiresAt time.Time `gorm:"not null" json:"expiresAt"`
	CreatedAt time.Time `json:"createdAt"`
}

// TableName returns the table name for RefreshToken.
func (RefreshToken) TableName() string {
	return "refresh_tokens"
}
