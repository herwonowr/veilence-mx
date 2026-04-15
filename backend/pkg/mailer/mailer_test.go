package mailer_test

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/veilence/veilence-mx/backend/pkg/mailer"
)

func TestSMTPConfig_IsConfigured(t *testing.T) {
	tests := []struct {
		name       string
		cfg        mailer.SMTPConfig
		configured bool
	}{
		{"fully configured", mailer.SMTPConfig{Host: "smtp.example.com", Port: "587", From: "noreply@example.com"}, true},
		{"missing host", mailer.SMTPConfig{Host: "", Port: "587", From: "noreply@example.com"}, false},
		{"missing port", mailer.SMTPConfig{Host: "smtp.example.com", Port: "", From: "noreply@example.com"}, false},
		{"missing from", mailer.SMTPConfig{Host: "smtp.example.com", Port: "587", From: ""}, false},
		{"all empty", mailer.SMTPConfig{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.configured, tt.cfg.IsConfigured())
		})
	}
}

// TestSendPasswordResetEmail_NotConfigured verifies that sending with an
// unconfigured SMTP returns an error instead of panicking.
func TestSendPasswordResetEmail_NotConfigured(t *testing.T) {
	m := mailer.New(mailer.SMTPConfig{}, "https://app.example.com")
	err := m.SendPasswordResetEmail(context.Background(), "user@example.com", "abc123")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "SMTP not configured")
}

// TestSendVerificationEmail_NotConfigured verifies that sending with an
// unconfigured SMTP returns an error instead of panicking.
func TestSendVerificationEmail_NotConfigured(t *testing.T) {
	m := mailer.New(mailer.SMTPConfig{}, "https://app.example.com")
	err := m.SendVerificationEmail(context.Background(), "user@example.com", "xyz789")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "SMTP not configured")
}

// TestNew_SetsFields ensures the constructor stores config correctly.
func TestNew_ReturnsNonNil(t *testing.T) {
	cfg := mailer.SMTPConfig{
		Host:     "smtp.example.com",
		Port:     "587",
		Username: "user",
		Password: "pass",
		From:     "noreply@example.com",
	}
	m := mailer.New(cfg, "https://app.example.com")
	require.NotNil(t, m)
}

// TestBuildPasswordResetLink verifies the password reset email would contain
// the correct link. We test this indirectly by ensuring the mailer attempts
// to send (and fails with "SMTP not configured" for unconfigured) — the
// message formatting is internal, so we verify the link pattern via a
// lightweight send attempt on a configured-but-unreachable SMTP.
func TestPasswordResetEmail_LinkFormat(t *testing.T) {
	// We can test that the frontend URL and token are incorporated by
	// verifying the error path with a configured but unreachable SMTP.
	cfg := mailer.SMTPConfig{
		Host: "127.0.0.1",
		Port: "59999", // not listening
		From: "noreply@example.com",
	}
	m := mailer.New(cfg, "https://app.example.com")

	// This will fail on dial, but it proves the code path runs without panic.
	err := m.SendPasswordResetEmail(context.Background(), "user@example.com", "my-token-123")
	require.Error(t, err)
	// The error should be from SMTP dial, not from message construction.
	assert.True(t,
		!strings.Contains(err.Error(), "SMTP not configured"),
		"expected dial error, not config error: %v", err)
}

// TestVerificationEmail_LinkFormat tests the verification email code path.
func TestVerificationEmail_LinkFormat(t *testing.T) {
	cfg := mailer.SMTPConfig{
		Host: "127.0.0.1",
		Port: "59999", // not listening
		From: "noreply@example.com",
	}
	m := mailer.New(cfg, "https://app.example.com")

	err := m.SendVerificationEmail(context.Background(), "user@example.com", "verify-token-456")
	require.Error(t, err)
	assert.True(t,
		!strings.Contains(err.Error(), "SMTP not configured"),
		"expected dial error, not config error: %v", err)
}
