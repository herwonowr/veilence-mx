package entity

import "time"

// PasswordResetToken represents a token for resetting a user's password.
type PasswordResetToken struct {
	ID             uint `json:"id"`
	UserID         uint `json:"userId"`
	TokenHash      string `json:"-"`
	ExpiresAt      time.Time `json:"expiresAt"`
	UsedAt         *time.Time `json:"usedAt,omitempty"`
	CreatedAt      time.Time `json:"createdAt"`
}

// EmailVerificationToken represents a token for verifying a user's email address.
type EmailVerificationToken struct {
	ID             uint `json:"id"`
	UserID         uint `json:"userId"`
	TokenHash      string `json:"-"`
	ExpiresAt      time.Time `json:"expiresAt"`
	CreatedAt      time.Time `json:"createdAt"`
}
