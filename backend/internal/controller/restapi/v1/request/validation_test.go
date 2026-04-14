package request_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	validation "github.com/veilence/veilence-mx/backend/internal/controller/restapi/v1/request"
)

// --- ValidateEmail ---

func TestValidateEmail_Valid(t *testing.T) {
	validEmails := []string{
		"user@example.com",
		"alice.bob@company.org",
		"test+tag@domain.co.uk",
		"user123@test.io",
	}

	for _, email := range validEmails {
		t.Run(email, func(t *testing.T) {
			err := validation.ValidateEmail(email)
			assert.NoError(t, err)
		})
	}
}

func TestValidateEmail_Invalid(t *testing.T) {
	invalidEmails := []string{
		"",
		"  ",
		"notanemail",
		"missing@tld",
		"@domain.com",
		"user@",
		"user@@domain.com",
	}

	for _, email := range invalidEmails {
		t.Run(email, func(t *testing.T) {
			err := validation.ValidateEmail(email)
			assert.Error(t, err)
		})
	}
}

// --- ValidatePassword ---

func TestValidatePassword_Valid(t *testing.T) {
	validPasswords := []string{
		"Password1",
		"MyP@ssw0rd",
		"Abcdefg1",
		"UPPER1lower",
		"Complex1Password!@#",
	}

	for _, pw := range validPasswords {
		t.Run(pw, func(t *testing.T) {
			err := validation.ValidatePassword(pw)
			assert.NoError(t, err)
		})
	}
}

func TestValidatePassword_TooShort(t *testing.T) {
	err := validation.ValidatePassword("Pass1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "at least 8 characters")
}

func TestValidatePassword_NoUppercase(t *testing.T) {
	err := validation.ValidatePassword("password1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "uppercase")
}

func TestValidatePassword_NoLowercase(t *testing.T) {
	err := validation.ValidatePassword("PASSWORD1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "lowercase")
}

func TestValidatePassword_NoDigit(t *testing.T) {
	err := validation.ValidatePassword("PasswordOnly")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "digit")
}

func TestValidatePassword_Empty(t *testing.T) {
	err := validation.ValidatePassword("")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "required")
}

func TestValidatePassword_TooLong(t *testing.T) {
	// 73 characters exceeds bcrypt's 72-byte limit
	longPassword := "Aa1" + strings.Repeat("x", 70)
	err := validation.ValidatePassword(longPassword)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds maximum length")
}

func TestValidatePassword_ExactMaxLength(t *testing.T) {
	// Exactly 72 characters should be accepted
	exactPassword := "Aa1" + strings.Repeat("x", 69)
	err := validation.ValidatePassword(exactPassword)
	assert.NoError(t, err)
}

// --- ValidateSlug ---

func TestValidateSlug_Valid(t *testing.T) {
	validSlugs := []string{
		"my-org",
		"test123",
		"a-b-c",
		"company-name-2024",
		"abc",
	}

	for _, slug := range validSlugs {
		t.Run(slug, func(t *testing.T) {
			err := validation.ValidateSlug(slug)
			assert.NoError(t, err)
		})
	}
}

func TestValidateSlug_Invalid(t *testing.T) {
	tests := []struct {
		name string
		slug string
	}{
		{"empty", ""},
		{"too short", "ab"},
		{"uppercase", "MyOrg"},
		{"spaces", "my org"},
		{"special chars", "my_org!"},
		{"starts with hyphen", "-my-org"},
		{"ends with hyphen", "my-org-"},
		{"too long", strings.Repeat("a", 101)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validation.ValidateSlug(tt.slug)
			assert.Error(t, err)
		})
	}
}

// --- ValidateRequired ---

func TestValidateRequired_Valid(t *testing.T) {
	err := validation.ValidateRequired("hello", "name")
	assert.NoError(t, err)
}

func TestValidateRequired_Empty(t *testing.T) {
	err := validation.ValidateRequired("", "name")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "name")
	assert.Contains(t, err.Error(), "required")
}

func TestValidateRequired_Whitespace(t *testing.T) {
	err := validation.ValidateRequired("   ", "name")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "required")
}

// --- ValidateMaxLength ---

func TestValidateMaxLength_WithinLimit(t *testing.T) {
	err := validation.ValidateMaxLength("hello", "name", 10)
	assert.NoError(t, err)
}

func TestValidateMaxLength_ExactLimit(t *testing.T) {
	err := validation.ValidateMaxLength("hello", "name", 5)
	assert.NoError(t, err)
}

func TestValidateMaxLength_ExceedsLimit(t *testing.T) {
	err := validation.ValidateMaxLength("hello world", "name", 5)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "name")
	assert.Contains(t, err.Error(), "max")
}

func TestValidateMaxLength_Empty(t *testing.T) {
	err := validation.ValidateMaxLength("", "name", 5)
	assert.NoError(t, err)
}
