package entity

import "time"

// PasswordResetToken represents a token for resetting a user's password.
type PasswordResetToken struct {
	ID             uint
	UserID         uint
	TokenHash      string
	ExpiresAt      time.Time
	UsedAt         *time.Time
	CreatedAt      time.Time
}

// EmailVerificationToken represents a token for verifying a user's email address.
type EmailVerificationToken struct {
	ID             uint
	UserID         uint
	TokenHash      string
	ExpiresAt      time.Time
	CreatedAt      time.Time
}
