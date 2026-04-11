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

	"gorm.io/gorm"

	"github.com/veilence/veilence-mx/backend/internal/models"
	"github.com/veilence/veilence-mx/backend/internal/notifications"
)

// Scheduler runs a background goroutine that checks every hour whether any org
// is due for its email digest, generates the digest content, and sends it via
// the existing SMTP infrastructure.
type Scheduler struct {
	db   *gorm.DB
	smtp notifications.SMTPConfig

	// checkInterval controls how often the scheduler polls for due digests.
	// Default: 1 hour. Exposed for testing.
	checkInterval time.Duration

	// nowFunc is a clock function used for determining the current time.
	// Default: time.Now. Exposed for testing.
	nowFunc func() time.Time

	// lastSentAt tracks when each org last received a digest (in-memory).
	// Key: orgID, Value: time the last digest was sent.
	lastSentAt map[uint]time.Time
}

// Config holds configuration for the digest scheduler.
type Config struct {
	// CheckInterval is how often the scheduler checks for due digests.
	// Default: 1 hour.
	CheckInterval time.Duration
}

// New creates a new digest Scheduler.
func New(db *gorm.DB, smtpCfg notifications.SMTPConfig, cfg Config) *Scheduler {
	interval := cfg.CheckInterval
	if interval <= 0 {
		interval = 1 * time.Hour
	}
	return &Scheduler{
		db:            db,
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
	// Find all orgs that have digest enabled
	type orgDigestConfig struct {
		OrgID      uint
		Frequency  string
		Recipients string
	}

	var enabledOrgs []orgDigestConfig

	// Get all org IDs that have email_digest_enabled=true
	var enabledSettings []models.Setting
	if err := s.db.WithContext(ctx).
		Where("key = ? AND value = ?", models.SettingEmailDigestEnabled, "true").
		Find(&enabledSettings).Error; err != nil {
		slog.Error("digest: failed to query enabled orgs", "error", err)
		return
	}

	for _, setting := range enabledSettings {
		orgID := setting.OrgID

		// Get frequency and recipients for this org
		var frequency, recipients string
		var freqSetting, recipSetting models.Setting

		if err := s.db.WithContext(ctx).
			Where("org_id = ? AND key = ?", orgID, models.SettingEmailDigestFrequency).
			First(&freqSetting).Error; err != nil {
			frequency = "daily" // default
		} else {
			frequency = freqSetting.Value
		}

		if err := s.db.WithContext(ctx).
			Where("org_id = ? AND key = ?", orgID, models.SettingEmailDigestRecipients).
			First(&recipSetting).Error; err != nil {
			continue // no recipients configured, skip
		} else {
			recipients = recipSetting.Value
		}

		if recipients == "" {
			continue
		}

		enabledOrgs = append(enabledOrgs, orgDigestConfig{
			OrgID:      orgID,
			Frequency:  frequency,
			Recipients: recipients,
		})
	}

	now := s.nowFunc()

	for _, org := range enabledOrgs {
		if !s.isDue(org.OrgID, org.Frequency, now) {
			continue
		}

		digest, err := s.GenerateDigest(ctx, org.OrgID, org.Frequency, now)
		if err != nil {
			slog.Error("digest: failed to generate",
				"org_id", org.OrgID,
				"error", err,
			)
			continue
		}

		if err := s.sendDigestEmail(org.Recipients, org.Frequency, digest); err != nil {
			slog.Error("digest: failed to send email",
				"org_id", org.OrgID,
				"recipients", org.Recipients,
				"error", err,
			)
			continue
		}

		s.lastSentAt[org.OrgID] = now
		slog.Info("digest: sent successfully",
			"org_id", org.OrgID,
			"frequency", org.Frequency,
			"recipients", org.Recipients,
		)
	}
}

// isDue returns true if the org's digest is due to be sent based on frequency.
func (s *Scheduler) isDue(orgID uint, frequency string, now time.Time) bool {
	lastSent, ok := s.lastSentAt[orgID]
	if !ok {
		// Never sent — send now
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
	TopAlerts []TopAlert
}

// TopAlert is a summary of an alert for the digest.
type TopAlert struct {
	ID          uint
	PackageName string
	Severity    string
	Message     string
	CreatedAt   time.Time
}

// GenerateDigest generates the digest content for an org over the given period.
func (s *Scheduler) GenerateDigest(ctx context.Context, orgID uint, frequency string, now time.Time) (*DigestContent, error) {
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
	if err := s.db.WithContext(ctx).
		Model(&models.Alert{}).
		Where("org_id = ? AND created_at >= ?", orgID, since).
		Count(&digest.NewAlertsCount).Error; err != nil {
		return nil, fmt.Errorf("counting new alerts: %w", err)
	}

	// Count packages analyzed (distinct package_id from releases created in period)
	if err := s.db.WithContext(ctx).
		Model(&models.Release{}).
		Joins("JOIN packages ON packages.id = releases.package_id").
		Where("packages.org_id = ? AND releases.created_at >= ?", orgID, since).
		Distinct("releases.package_id").
		Count(&digest.PackagesAnalyzed).Error; err != nil {
		return nil, fmt.Errorf("counting packages analyzed: %w", err)
	}

	// Classification breakdown from analyses in the period
	type classCount struct {
		Classification string
		Count          int64
	}
	var classRows []classCount
	if err := s.db.WithContext(ctx).
		Model(&models.Analysis{}).
		Select("analyses.classification, COUNT(*) as count").
		Joins("JOIN diffs ON diffs.id = analyses.diff_id").
		Joins("JOIN releases ON releases.id = diffs.release_id").
		Joins("JOIN packages ON packages.id = releases.package_id").
		Where("packages.org_id = ? AND analyses.created_at >= ?", orgID, since).
		Group("analyses.classification").
		Scan(&classRows).Error; err != nil {
		return nil, fmt.Errorf("querying classification breakdown: %w", err)
	}
	for _, row := range classRows {
		digest.ClassificationBreakdown[row.Classification] = row.Count
	}

	// Top 5 alerts by severity (critical > high > medium > low), most recent first
	type alertRow struct {
		ID          uint
		PackageName string
		Severity    string
		Message     string
		CreatedAt   time.Time
	}
	var topRows []alertRow
	if err := s.db.WithContext(ctx).
		Model(&models.Alert{}).
		Select("alerts.id, packages.name as package_name, alerts.severity, alerts.message, alerts.created_at").
		Joins("JOIN packages ON packages.id = alerts.package_id").
		Where("alerts.org_id = ? AND alerts.created_at >= ?", orgID, since).
		Order("CASE alerts.severity WHEN 'critical' THEN 0 WHEN 'high' THEN 1 WHEN 'medium' THEN 2 WHEN 'low' THEN 3 END ASC, alerts.created_at DESC").
		Limit(5).
		Scan(&topRows).Error; err != nil {
		return nil, fmt.Errorf("querying top alerts: %w", err)
	}
	for _, row := range topRows {
		digest.TopAlerts = append(digest.TopAlerts, TopAlert{
			ID:          row.ID,
			PackageName: row.PackageName,
			Severity:    row.Severity,
			Message:     row.Message,
			CreatedAt:   row.CreatedAt,
		})
	}

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
			b.WriteString(fmt.Sprintf("  %d. [%s] %s — %s\n",
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
