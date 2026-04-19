// Package digest provides an email digest scheduler that periodically sends
// summary emails (daily or weekly) to configured recipients for each org.
package digest

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/veilence/veilence-mx/backend/internal/entity"
	"github.com/veilence/veilence-mx/backend/internal/usecase/notifications"
)

// Scheduler runs a background goroutine that checks every hour whether any org
// is due for its email digest, generates the digest content, and sends it via
// the existing SMTP infrastructure.
type Scheduler struct {
	repo DigestRepository
	smtp notifications.SMTPConfig

	// checkInterval controls how often the scheduler polls for due digests.
	// Default: 1 hour. Exposed for testing.
	checkInterval time.Duration

	// nowFunc is a clock function used for determining the current time.
	// Default: time.Now. Exposed for testing.
	nowFunc func() time.Time

	// lastSentAt tracks when each org last received a digest (in-memory).
	// Key: workspaceID, Value: time the last digest was sent.
	lastSentAt map[uint]time.Time
}

// Config holds configuration for the digest scheduler.
type Config struct {
	// CheckInterval is how often the scheduler checks for due digests.
	// Default: 1 hour.
	CheckInterval time.Duration
}

// New creates a new digest Scheduler.
func New(repo DigestRepository, smtpCfg notifications.SMTPConfig, cfg Config) *Scheduler {
	interval := cfg.CheckInterval
	if interval <= 0 {
		interval = 1 * time.Hour
	}
	return &Scheduler{
		repo:          repo,
		smtp:          smtpCfg,
		checkInterval: interval,
		nowFunc:       time.Now,
		lastSentAt:    make(map[uint]time.Time),
	}
}

// Start runs the digest scheduler loop. It blocks until the context is cancelled.
func (s *Scheduler) Start(ctx context.Context) {
	slog.Info("email digest scheduler started", "check_interval", s.checkInterval)

	ticker := time.NewTicker(s.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("email digest scheduler stopped")
			return
		case <-ticker.C:
			s.tick(ctx)
		}
	}
}

// tick checks all orgs and sends digests for any that are due.
func (s *Scheduler) tick(ctx context.Context) {
	enabledOrgs, err := s.repo.FindEnabledDigestConfigs(ctx)
	if err != nil {
		slog.Error("digest: failed to query enabled orgs", "error", err)
		return
	}

	now := s.nowFunc()

	for _, org := range enabledOrgs {
		if !s.isDue(org.WorkspaceID, org.Frequency, now) {
			continue
		}

		digest, err := s.GenerateDigest(ctx, org.WorkspaceID, org.Frequency, now)
		if err != nil {
			slog.Error("digest: failed to generate",
				"workspace_id", org.WorkspaceID,
				"error", err,
			)
			continue
		}

		if err := s.sendDigestEmail(org.Recipients, org.Frequency, digest); err != nil {
			slog.Error("digest: failed to send email",
				"workspace_id", org.WorkspaceID,
				"recipients", org.Recipients,
				"error", err,
			)
			continue
		}

		s.lastSentAt[org.WorkspaceID] = now
		slog.Info("digest: sent successfully",
			"workspace_id", org.WorkspaceID,
			"frequency", org.Frequency,
			"recipients", org.Recipients,
		)
	}
}

// isDue returns true if the org's digest is due to be sent based on frequency.
func (s *Scheduler) isDue(workspaceID uint, frequency string, now time.Time) bool {
	lastSent, ok := s.lastSentAt[workspaceID]
	if !ok {
		// Never sent - send now
		return true
	}

	switch frequency {
	case "weekly":
		return now.Sub(lastSent) >= 7*24*time.Hour
	default: // "daily"
		return now.Sub(lastSent) >= 24*time.Hour
	}
}

// DigestContent holds the data for a digest email.
type DigestContent struct {
	// Period describes the time window (e.g., "last 24 hours" or "last 7 days").
	Period string
	// NewAlertsCount is the total number of new alerts in the period.
	NewAlertsCount int64
	// PackagesAnalyzed is the number of unique packages with new releases analyzed.
	PackagesAnalyzed int64
	// ClassificationBreakdown maps classification -> count.
	ClassificationBreakdown map[string]int64
	// TopAlerts contains up to 5 most severe recent alerts.
	TopAlerts []entity.DigestTopAlert
}

// GenerateDigest generates the digest content for an org over the given period.
func (s *Scheduler) GenerateDigest(ctx context.Context, workspaceID uint, frequency string, now time.Time) (*DigestContent, error) {
	var since time.Time
	var period string
	switch frequency {
	case "weekly":
		since = now.Add(-7 * 24 * time.Hour)
		period = "last 7 days"
	default:
		since = now.Add(-24 * time.Hour)
		period = "last 24 hours"
	}

	digest := &DigestContent{
		Period:                  period,
		ClassificationBreakdown: make(map[string]int64),
	}

	// Count new alerts in the period
	alertCount, err := s.repo.CountAlertsSince(ctx, workspaceID, since)
	if err != nil {
		return nil, fmt.Errorf("counting new alerts: %w", err)
	}
	digest.NewAlertsCount = alertCount

	// Count packages analyzed
	pkgCount, err := s.repo.CountPackagesAnalyzedSince(ctx, workspaceID, since)
	if err != nil {
		return nil, fmt.Errorf("counting packages analyzed: %w", err)
	}
	digest.PackagesAnalyzed = pkgCount

	// Classification breakdown
	breakdown, err := s.repo.GetClassificationBreakdownSince(ctx, workspaceID, since)
	if err != nil {
		return nil, fmt.Errorf("querying classification breakdown: %w", err)
	}
	digest.ClassificationBreakdown = breakdown

	// Top 5 alerts
	topAlerts, err := s.repo.GetTopAlertsSince(ctx, workspaceID, since, 5)
	if err != nil {
		return nil, fmt.Errorf("querying top alerts: %w", err)
	}
	digest.TopAlerts = topAlerts

	return digest, nil
}

// FormatDigestText renders the digest content as plain text for email.
func FormatDigestText(d *DigestContent) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("Veilence-MX Digest (%s)\n", d.Period))
	b.WriteString(strings.Repeat("=", 50))
	b.WriteString("\n\n")

	b.WriteString(fmt.Sprintf("New Alerts:        %d\n", d.NewAlertsCount))
	b.WriteString(fmt.Sprintf("Packages Analyzed: %d\n", d.PackagesAnalyzed))
	b.WriteString("\n")

	if len(d.ClassificationBreakdown) > 0 {
		b.WriteString("Classification Breakdown:\n")
		for class, count := range d.ClassificationBreakdown {
			b.WriteString(fmt.Sprintf("  %-12s %d\n", class, count))
		}
		b.WriteString("\n")
	}

	if len(d.TopAlerts) > 0 {
		b.WriteString("Top Alerts:\n")
		for i, alert := range d.TopAlerts {
			b.WriteString(fmt.Sprintf("  %d. [%s] %s - %s\n",
				i+1, strings.ToUpper(alert.Severity), alert.PackageName, truncate(alert.Message, 80)))
		}
		b.WriteString("\n")
	}

	if d.NewAlertsCount == 0 && len(d.TopAlerts) == 0 {
		b.WriteString("No new alerts in this period. All clear!\n\n")
	}

	b.WriteString("---\nSent by Veilence-MX email digest\n")
	return b.String()
}

// sendDigestEmail sends the digest email via SMTP to the configured recipients.
func (s *Scheduler) sendDigestEmail(recipientsList, frequency string, digest *DigestContent) error {
	if !s.smtp.IsConfigured() {
		return fmt.Errorf("SMTP not configured")
	}

	recipients := parseRecipients(recipientsList)
	if len(recipients) == 0 {
		return fmt.Errorf("no valid recipients")
	}

	subject := fmt.Sprintf("Veilence-MX %s Digest", capitalize(frequency))
	body := FormatDigestText(digest)

	var msg bytes.Buffer
	msg.WriteString("From: " + s.smtp.From + "\r\n")
	msg.WriteString("To: " + strings.Join(recipients, ", ") + "\r\n")
	msg.WriteString("Subject: [Veilence-MX] " + subject + "\r\n")
	msg.WriteString("MIME-Version: 1.0\r\n")
	msg.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	msg.WriteString("\r\n")
	msg.WriteString(body)

	// Reuse the notification service's SMTP sending logic by delegating
	// to a helper. For simplicity, we use net/smtp directly here.
	return notifications.SendRawEmail(s.smtp, recipients, msg.Bytes())
}

// parseRecipients splits a comma-separated list of email addresses and trims whitespace.
func parseRecipients(s string) []string {
	parts := strings.Split(s, ",")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// truncate shortens a string to max length, appending "..." if truncated.
func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

// capitalize returns the string with the first letter uppercased.
func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
