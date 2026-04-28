package request

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/veilence/veilence-mx/backend/internal/entity"
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

// ValidatePassword validates that the given password meets security
// requirements per NIST SP 800-63B: length bounds only, no composition rules.
// Uses entity.PasswordMaxLength as the single source of truth for max length.
func ValidatePassword(password string) error {
	if password == "" {
		return fmt.Errorf("password %w", ErrRequired)
	}
	n := utf8.RuneCountInString(password)
	if n < entity.PasswordMinLength {
		return fmt.Errorf("password must be at least %d characters", entity.PasswordMinLength)
	}
	if n > entity.PasswordMaxLength {
		return fmt.Errorf("password %w (%d characters max)", ErrTooLong, entity.PasswordMaxLength)
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
