package models

import "time"

// PasswordResetToken represents a token for resetting a user's password.
type PasswordResetToken struct {
	ID        uint       `gorm:"primarykey" json:"id"`
	UserID    uint       `gorm:"not null;index" json:"userId"`
	TokenHash string     `gorm:"not null;uniqueIndex" json:"-"`
	ExpiresAt time.Time  `gorm:"not null" json:"expiresAt"`
	UsedAt    *time.Time `json:"usedAt,omitempty"`
	CreatedAt time.Time  `json:"createdAt"`
}

// TableName returns the table name for PasswordResetToken.
func (PasswordResetToken) TableName() string {
	return "password_reset_tokens"
}

// EmailVerificationToken represents a token for verifying a user's email address.
type EmailVerificationToken struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	UserID    uint      `gorm:"not null;index" json:"userId"`
	TokenHash string    `gorm:"not null;uniqueIndex" json:"-"`
	ExpiresAt time.Time `gorm:"not null" json:"expiresAt"`
	CreatedAt time.Time `json:"createdAt"`
}

// TableName returns the table name for EmailVerificationToken.
func (EmailVerificationToken) TableName() string {
	return "email_verification_tokens"
}
