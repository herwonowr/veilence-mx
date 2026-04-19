// Package mailer provides SMTP email delivery for authentication workflows
// (password reset, email verification). It is a standalone pkg/ library that
// does NOT import internal/ - all SMTP logic is self-contained.
package mailer

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"net/smtp"
	"time"
)

// SMTPConfig holds SMTP server configuration for sending emails.
type SMTPConfig struct {
	Host     string
	Port     string
	Username string
	Password string
	From     string
}

// IsConfigured returns true if the SMTP configuration has the minimum required fields.
func (c SMTPConfig) IsConfigured() bool {
	return c.Host != "" && c.Port != "" && c.From != ""
}

// Mailer sends authentication-related emails via SMTP.
// It implements usecase.AuthEmailSender.
type Mailer struct {
	smtp        SMTPConfig
	frontendURL string
}

// New creates a new Mailer with the given SMTP configuration and frontend base URL.
func New(smtp SMTPConfig, frontendURL string) *Mailer {
	return &Mailer{
		smtp:        smtp,
		frontendURL: frontendURL,
	}
}

// SendPasswordResetEmail sends a password reset email with a link to reset the password.
func (m *Mailer) SendPasswordResetEmail(_ context.Context, email, token string) error {
	resetLink := fmt.Sprintf("%s/reset-password?token=%s", m.frontendURL, token)

	subject := "[Veilence-MX] Password Reset Request"
	body := fmt.Sprintf(
		"You requested a password reset for your Veilence-MX account.\r\n\r\n"+
			"Click the link below to reset your password:\r\n\r\n"+
			"%s\r\n\r\n"+
			"This link expires in 1 hour.\r\n\r\n"+
			"If you did not request this, you can safely ignore this email.\r\n\r\n"+
			"---\r\nSent by Veilence-MX",
		resetLink,
	)

	return m.sendEmail([]string{email}, subject, body)
}

// SendVerificationEmail sends an email verification link to the user.
func (m *Mailer) SendVerificationEmail(_ context.Context, email, token string) error {
	verifyLink := fmt.Sprintf("%s/verify-email?token=%s", m.frontendURL, token)

	subject := "[Veilence-MX] Verify Your Email Address"
	body := fmt.Sprintf(
		"Please verify your email address for your Veilence-MX account.\r\n\r\n"+
			"Click the link below to verify:\r\n\r\n"+
			"%s\r\n\r\n"+
			"This link expires in 24 hours.\r\n\r\n"+
			"If you did not create this account, you can safely ignore this email.\r\n\r\n"+
			"---\r\nSent by Veilence-MX",
		verifyLink,
	)

	return m.sendEmail([]string{email}, subject, body)
}

// sendEmail builds an RFC 2822 message and sends it via SMTP.
func (m *Mailer) sendEmail(recipients []string, subject, body string) error {
	if !m.smtp.IsConfigured() {
		return fmt.Errorf("mailer: SMTP not configured")
	}

	var msg bytes.Buffer
	msg.WriteString("From: " + m.smtp.From + "\r\n")
	msg.WriteString("To: " + recipients[0] + "\r\n")
	msg.WriteString("Subject: " + subject + "\r\n")
	msg.WriteString("MIME-Version: 1.0\r\n")
	msg.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	msg.WriteString("\r\n")
	msg.WriteString(body)

	return m.sendRawEmail(recipients, msg.Bytes())
}

// sendRawEmail sends a pre-formatted email message via SMTP.
func (m *Mailer) sendRawEmail(recipients []string, msg []byte) error {
	addr := net.JoinHostPort(m.smtp.Host, m.smtp.Port)

	var auth smtp.Auth
	if m.smtp.Username != "" {
		auth = smtp.PlainAuth("", m.smtp.Username, m.smtp.Password, m.smtp.Host)
	}

	if m.smtp.Port == "465" {
		return m.sendImplicitTLS(addr, auth, recipients, msg)
	}

	return smtp.SendMail(addr, auth, m.smtp.From, recipients, msg)
}

// sendImplicitTLS sends an email over implicit TLS (port 465).
func (m *Mailer) sendImplicitTLS(addr string, auth smtp.Auth, recipients []string, msg []byte) error {
	tlsConfig := &tls.Config{
		ServerName: m.smtp.Host,
		MinVersion: tls.VersionTLS12,
	}

	conn, err := tls.DialWithDialer(&net.Dialer{Timeout: 10 * time.Second}, "tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("mailer: connecting to SMTP server: %w", err)
	}

	client, err := smtp.NewClient(conn, m.smtp.Host)
	if err != nil {
		conn.Close()
		return fmt.Errorf("mailer: creating SMTP client: %w", err)
	}
	defer func() {
		if err := client.Close(); err != nil {
			slog.Warn("mailer: failed to close SMTP client", "error", err)
		}
	}()

	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("mailer: SMTP authentication: %w", err)
		}
	}

	if err := client.Mail(m.smtp.From); err != nil {
		return fmt.Errorf("mailer: SMTP MAIL FROM: %w", err)
	}

	for _, rcpt := range recipients {
		if err := client.Rcpt(rcpt); err != nil {
			return fmt.Errorf("mailer: SMTP RCPT TO %s: %w", rcpt, err)
		}
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("mailer: SMTP DATA: %w", err)
	}

	if _, err := w.Write(msg); err != nil {
		return fmt.Errorf("mailer: writing email body: %w", err)
	}

	if err := w.Close(); err != nil {
		return fmt.Errorf("mailer: closing email body: %w", err)
	}

	return client.Quit()
}
