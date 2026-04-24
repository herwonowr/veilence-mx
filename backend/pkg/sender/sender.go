// Package sender provides HTTP and SMTP transport implementations
// for the notification system.
package sender

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/smtp"
	"time"
)

// SMTPConfig holds SMTP server configuration.
type SMTPConfig struct {
	Host     string
	Port     string
	Username string
	Password string
	From     string
	UseTLS   bool
}

// IsConfigured returns true if the SMTP configuration has the minimum required fields.
func (c SMTPConfig) IsConfigured() bool {
	return c.Host != "" && c.Port != "" && c.From != ""
}

// SMTPSender implements usecase.EmailNotificationSender using SMTP.
type SMTPSender struct {
	config SMTPConfig
}

// NewSMTPSender creates a new SMTP-based email sender.
func NewSMTPSender(cfg SMTPConfig) *SMTPSender {
	return &SMTPSender{config: cfg}
}

// SendNotificationEmail sends an email via the configured SMTP server.
func (s *SMTPSender) SendNotificationEmail(from string, recipients []string, subject, body string) error {
	if !s.config.IsConfigured() {
		return fmt.Errorf("SMTP not configured")
	}

	addr := net.JoinHostPort(s.config.Host, s.config.Port)

	var auth smtp.Auth
	if s.config.Username != "" {
		auth = smtp.PlainAuth("", s.config.Username, s.config.Password, s.config.Host)
	}

	// Build the raw email message
	var msg bytes.Buffer
	msg.WriteString("From: " + from + "\r\n")
	for i, r := range recipients {
		if i > 0 {
			msg.WriteString(", ")
		}
		msg.WriteString(r)
	}
	msg.WriteString("\r\nSubject: " + subject + "\r\n")
	msg.WriteString("MIME-Version: 1.0\r\n")
	msg.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	msg.WriteString("\r\n")
	msg.WriteString(body)

	msgBytes := msg.Bytes()

	// Port 465 uses implicit TLS; other ports use STARTTLS
	if s.config.Port == "465" {
		return s.sendImplicitTLS(addr, auth, from, recipients, msgBytes)
	}
	return smtp.SendMail(addr, auth, from, recipients, msgBytes)
}

func (s *SMTPSender) sendImplicitTLS(addr string, auth smtp.Auth, from string, recipients []string, msg []byte) error {
	tlsConfig := &tls.Config{
		ServerName: s.config.Host,
		MinVersion: tls.VersionTLS12,
	}

	conn, err := tls.DialWithDialer(&net.Dialer{Timeout: 10 * time.Second}, "tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("connecting to SMTP server: %w", err)
	}

	client, err := smtp.NewClient(conn, s.config.Host)
	if err != nil {
		conn.Close()
		return fmt.Errorf("creating SMTP client: %w", err)
	}
	defer client.Close()

	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP authentication: %w", err)
		}
	}

	if err := client.Mail(from); err != nil {
		return fmt.Errorf("SMTP MAIL FROM: %w", err)
	}

	for _, rcpt := range recipients {
		if err := client.Rcpt(rcpt); err != nil {
			return fmt.Errorf("SMTP RCPT TO %s: %w", rcpt, err)
		}
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("SMTP DATA: %w", err)
	}

	if _, err := w.Write(msg); err != nil {
		return fmt.Errorf("writing email body: %w", err)
	}

	if err := w.Close(); err != nil {
		return fmt.Errorf("closing email body: %w", err)
	}

	return client.Quit()
}

// HTTPWebhookSender implements usecase.WebhookSender using net/http.
type HTTPWebhookSender struct{}

// NewHTTPWebhookSender creates a new HTTP webhook sender.
func NewHTTPWebhookSender() *HTTPWebhookSender {
	return &HTTPWebhookSender{}
}

// SendWebhook posts a JSON payload to the given URL.
// If signingSecret is non-empty, it adds an HMAC-SHA256 signature header.
func (s *HTTPWebhookSender) SendWebhook(url string, payload []byte, signingSecret string) error {
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("creating webhook request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	if signingSecret != "" {
		mac := hmac.New(sha256.New, []byte(signingSecret))
		mac.Write(payload)
		sig := hex.EncodeToString(mac.Sum(nil))
		req.Header.Set("X-Signature-256", "sha256="+sig)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("sending webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned HTTP %d", resp.StatusCode)
	}
	return nil
}

// HTTPSlackSender implements usecase.SlackSender using net/http.
type HTTPSlackSender struct{}

// NewHTTPSlackSender creates a new HTTP-based Slack sender.
func NewHTTPSlackSender() *HTTPSlackSender {
	return &HTTPSlackSender{}
}

// SendSlack posts a text message to a Slack incoming webhook URL.
func (s *HTTPSlackSender) SendSlack(webhookURL string, text string) error {
	payload := struct {
		Text string `json:"text"`
	}{Text: text}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshaling slack payload: %w", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(webhookURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("sending slack notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("slack webhook returned HTTP %d", resp.StatusCode)
	}
	return nil
}
