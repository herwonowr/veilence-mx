package entity

import (
	"fmt"
	"time"
)

const (
	// PasswordMinLength is the minimum acceptable password length.
	PasswordMinLength = 8
	// PasswordMaxLength is the maximum acceptable password length.
	// bcrypt silently truncates at 72 bytes, so we cap there.
	PasswordMaxLength = 128
)

// ValidatePassword checks that a plaintext password meets length requirements.
// This is a domain-level guard - controller-layer validation may enforce
// additional complexity rules, but these length bounds are always enforced.
func ValidatePassword(password string) error {
	n := len(password)
	if n < PasswordMinLength {
		return fmt.Errorf("%w: password must be at least %d characters", ErrValidation, PasswordMinLength)
	}
	if n > PasswordMaxLength {
		return fmt.Errorf("%w: password must be at most %d characters", ErrValidation, PasswordMaxLength)
	}
	return nil
}

// User represents an authenticated user of the system.
type User struct {
	ID             string
	Email          string
	PasswordHash   string
	FirstName      string
	LastName       string
	IsActive       bool
	EmailVerified  bool
	LastLoginAt    *time.Time
	MustChangePassword bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
