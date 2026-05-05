package entity

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	// PasswordMinLength is the minimum acceptable password length.
	PasswordMinLength = 8
	// PasswordMaxLength is the maximum acceptable password length.
	// bcrypt silently truncates at 72 bytes, so we cap there.
	PasswordMaxLength = 72
)

// ValidatePassword checks that a plaintext password meets domain-level
// requirements: length bounds.
// For context-aware validation (e.g. checking against user email), use
// ValidatePasswordWithContext instead.
func ValidatePassword(password string) error {
	return ValidatePasswordWithContext(password, "")
}

// ValidatePasswordWithContext validates a password with optional contextual
// data. If email is non-empty, the password is checked against the email's
// local part to prevent trivially guessable passwords.
func ValidatePasswordWithContext(password, email string) error {
	n := utf8.RuneCountInString(password)
	if n < PasswordMinLength {
		return &ValidationError{Message: fmt.Sprintf("password must be at least %d characters", PasswordMinLength)}
	}
	if n > PasswordMaxLength {
		return &ValidationError{Message: fmt.Sprintf("password must be at most %d characters", PasswordMaxLength)}
	}

	// Check against context-specific words (email local part).
	if email != "" {
		localPart := email
		if idx := strings.Index(email, "@"); idx > 0 {
			localPart = email[:idx]
		}
		if len(localPart) >= 3 && strings.Contains(strings.ToLower(password), strings.ToLower(localPart)) {
			return &ValidationError{Message: "password must not contain your email address"}
		}
	}

	return nil
}

// UserFilters holds optional query filters for listing users.
type UserFilters struct {
	Search *string
	Status *string // "active" or "inactive"
	Role   *string // "super_admin" or "user"
}

// User represents an authenticated user of the system.
type User struct {
	ID                 string
	Email              string
	PasswordHash       string
	FirstName          string
	LastName           string
	IsActive           bool
	IsSuperAdmin       bool
	DeactivatedAt      *time.Time
	EmailVerified      bool
	LastLoginAt        *time.Time
	MustChangePassword bool
	AuthProvider       AuthProvider // "local", "saml", "google", "github" - tracks initial account creation method
	CreatedAt          time.Time
	UpdatedAt          time.Time
}
