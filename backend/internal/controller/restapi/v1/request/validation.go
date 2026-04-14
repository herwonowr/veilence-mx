package request

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

// Common validation errors.
var (
	ErrRequired  = errors.New("is required")
	ErrTooLong   = errors.New("exceeds maximum length")
	ErrBadFormat = errors.New("has invalid format")
)

// emailRegex is a basic regex for validating email addresses. It checks for
// the presence of a local part, an @ symbol, and a domain with at least one dot.
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// slugRegex matches valid slug strings: lowercase alphanumeric characters
// and hyphens, between 3 and 100 characters long.
var slugRegex = regexp.MustCompile(`^[a-z0-9][a-z0-9\-]{1,98}[a-z0-9]$`)

// ValidateEmail validates that the given string is a well-formed email address.
// It trims whitespace and checks against a standard email regex pattern.
func ValidateEmail(email string) error {
	email = strings.TrimSpace(email)
	if email == "" {
		return fmt.Errorf("email %w", ErrRequired)
	}
	if !emailRegex.MatchString(email) {
		return fmt.Errorf("email %w", ErrBadFormat)
	}
	return nil
}

// MaxPasswordLength is the maximum password length accepted, matching bcrypt's
// 72-byte input limit. Passwords longer than this are silently truncated by
// bcrypt, which can lead to surprising authentication behavior.
const MaxPasswordLength = 72

// ValidatePassword validates that the given password meets security
// requirements: at least 8 characters, at most 72 characters (bcrypt limit),
// containing at least one uppercase letter, one lowercase letter, and one digit.
func ValidatePassword(password string) error {
	if password == "" {
		return fmt.Errorf("password %w", ErrRequired)
	}
	if len(password) < 8 {
		return errors.New("password must be at least 8 characters")
	}
	if len(password) > MaxPasswordLength {
		return fmt.Errorf("password %w (%d characters max)", ErrTooLong, MaxPasswordLength)
	}

	var hasUpper, hasLower, hasDigit bool
	for _, ch := range password {
		switch {
		case unicode.IsUpper(ch):
			hasUpper = true
		case unicode.IsLower(ch):
			hasLower = true
		case unicode.IsDigit(ch):
			hasDigit = true
		}
	}

	if !hasUpper {
		return errors.New("password must contain at least one uppercase letter")
	}
	if !hasLower {
		return errors.New("password must contain at least one lowercase letter")
	}
	if !hasDigit {
		return errors.New("password must contain at least one digit")
	}

	return nil
}

// ValidateSlug validates that the given string is a valid URL slug: lowercase
// alphanumeric characters and hyphens, between 3 and 100 characters, starting
// and ending with an alphanumeric character.
func ValidateSlug(slug string) error {
	if slug == "" {
		return fmt.Errorf("slug %w", ErrRequired)
	}
	if len(slug) < 3 {
		return errors.New("slug must be at least 3 characters")
	}
	if len(slug) > 100 {
		return errors.New("slug must be at most 100 characters")
	}
	if !slugRegex.MatchString(slug) {
		return fmt.Errorf("slug %w: must be lowercase alphanumeric and hyphens", ErrBadFormat)
	}
	return nil
}

// ValidateRequired validates that the given field is not empty after trimming.
// The name parameter is used in the error message to identify which field
// failed validation.
func ValidateRequired(field, name string) error {
	if strings.TrimSpace(field) == "" {
		return fmt.Errorf("%s %w", name, ErrRequired)
	}
	return nil
}

// ValidateMaxLength validates that the given field does not exceed the
// specified maximum length. The name parameter is used in the error message.
func ValidateMaxLength(field, name string, max int) error {
	if len(field) > max {
		return fmt.Errorf("%s %w (%d characters max)", name, ErrTooLong, max)
	}
	return nil
}

// ValidateOneOf validates that the given value is one of the allowed values.
// The name parameter is used in the error message.
func ValidateOneOf(value, name string, allowed []string) error {
	for _, a := range allowed {
		if value == a {
			return nil
		}
	}
	return fmt.Errorf("%s must be one of: %s", name, strings.Join(allowed, ", "))
}

// ValidateRequiredUint validates that the given uint field is non-zero.
// The name parameter is used in the error message.
func ValidateRequiredUint(value uint, name string) error {
	if value == 0 {
		return fmt.Errorf("%s %w", name, ErrRequired)
	}
	return nil
}
