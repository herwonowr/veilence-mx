package notifications

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/veilence/veilence-mx/backend/internal/entity"
	"github.com/veilence/veilence-mx/backend/internal/usecase"
)

// SMTPConfig holds SMTP server configuration for sending email notifications.
type SMTPConfig struct {
	// Host is the SMTP server hostname (e.g. "smtp.gmail.com").
	Host string
	// Port is the SMTP server port (e.g. "587" for STARTTLS, "465" for implicit TLS).
	Port string
	// Username is the SMTP authentication username.
	Username string
	// Password is the SMTP authentication password or app-specific password.
	Password string
	// From is the sender email address.
	From string
	// UseTLS enables STARTTLS. Set to false for implicit TLS on port 465.
	UseTLS bool
}

// IsConfigured returns true if the SMTP configuration has the minimum required fields.
func (c SMTPConfig) IsConfigured() bool {
	return c.Host != "" && c.Port != "" && c.From != ""
}

// Service provides notification management and dispatch operations.
type Service struct {
	channels      usecase.NotificationChannelRepository
	rules         usecase.NotificationRuleRepository
	notifications usecase.NotificationRepository
	workspaces    usecase.UserWorkspaceLister
	smtp          SMTPConfig
	emailSender   usecase.EmailNotificationSender // nil = SMTP not configured
	webhookSender usecase.WebhookSender
	slackSender   usecase.SlackSender

	// AllowLocalURLs disables SSRF protection for webhook/Slack URLs.
	// This must ONLY be set to true in tests that use httptest.NewServer (localhost).
	AllowLocalURLs bool
}

// NewService creates a new notification service.
func NewService(
	channels usecase.NotificationChannelRepository,
	rules usecase.NotificationRuleRepository,
	notifications usecase.NotificationRepository,
	workspaces usecase.UserWorkspaceLister,
	smtpCfg SMTPConfig,
	emailSender usecase.EmailNotificationSender,
	webhookSender usecase.WebhookSender,
	slackSender usecase.SlackSender,
) *Service {
	return &Service{
		channels:      channels,
		rules:         rules,
		notifications: notifications,
		workspaces:    workspaces,
		smtp:          smtpCfg,
		emailSender:   emailSender,
		webhookSender: webhookSender,
		slackSender:   slackSender,
	}
}

// CreateChannel creates a new notification channel for a workspace.
func (s *Service) CreateChannel(workspaceID string, name string, channelType entity.NotificationChannelType, config string) (*entity.NotificationChannel, error) {
	ctx := context.Background()

	channel := &entity.NotificationChannel{
		WorkspaceID: workspaceID,
		Name:        name,
		Type:        channelType,
		Config:      config,
		IsActive:    true,
	}

	if err := s.channels.Create(ctx, channel); err != nil {
		return nil, fmt.Errorf("creating notification channel: %w", err)
	}

	slog.Info("notification channel created",
		"channel_id", channel.ID,
		"workspace_id", workspaceID,
		"type", channelType,
		"name", name,
	)
	return channel, nil
}

// ListChannels returns all notification channels for a workspace.
func (s *Service) ListChannels(workspaceID string) ([]entity.NotificationChannel, error) {
	ctx := context.Background()

	channels, err := s.channels.FindByWorkspaceID(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("listing notification channels: %w", err)
	}
	return channels, nil
}

// UpdateChannel updates an existing notification channel.
// The workspaceID parameter ensures the channel belongs to the requesting workspace.
func (s *Service) UpdateChannel(id, workspaceID string, name string, config string, isActive bool) (*entity.NotificationChannel, error) {
	ctx := context.Background()

	channel, err := s.channels.FindByIDAndWorkspace(ctx, id, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("finding notification channel: %w", err)
	}

	channel.Name = name
	channel.Config = config
	channel.IsActive = isActive

	if err := s.channels.Update(ctx, channel); err != nil {
		return nil, fmt.Errorf("updating notification channel: %w", err)
	}

	slog.Info("notification channel updated", "channel_id", id, "workspace_id", workspaceID)
	return channel, nil
}

// DeleteChannel deletes a notification channel by ID.
// The workspaceID parameter ensures the channel belongs to the requesting workspace.
func (s *Service) DeleteChannel(id, workspaceID string) error {
	ctx := context.Background()

	affected, err := s.channels.DeleteByIDAndWorkspace(ctx, id, workspaceID)
	if err != nil {
		return fmt.Errorf("deleting notification channel: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("Notification channel not found")
	}

	slog.Info("notification channel deleted", "channel_id", id, "workspace_id", workspaceID)
	return nil
}

// CreateRule creates a new notification routing rule.
// It verifies the channel belongs to the same workspace before creating the rule.
func (s *Service) CreateRule(workspaceID, channelID string, severity string) (*entity.NotificationRule, error) {
	ctx := context.Background()

	// Verify the channel belongs to the requesting workspace
	_, err := s.channels.FindByIDAndWorkspace(ctx, channelID, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("Notification channel not found in this workspace")
	}

	rule := &entity.NotificationRule{
		WorkspaceID: workspaceID,
		ChannelID:   channelID,
		Severity:    severity,
		IsActive:    true,
	}

	if err := s.rules.Create(ctx, rule); err != nil {
		return nil, fmt.Errorf("creating notification rule: %w", err)
	}

	slog.Info("notification rule created",
		"rule_id", rule.ID,
		"workspace_id", workspaceID,
		"channel_id", channelID,
		"severity", severity,
	)
	return rule, nil
}

// ListRules returns all notification rules for a workspace.
func (s *Service) ListRules(workspaceID string) ([]entity.NotificationRule, error) {
	ctx := context.Background()

	rules, err := s.rules.FindByWorkspaceID(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("listing notification rules: %w", err)
	}
	return rules, nil
}

// DeleteRule deletes a notification rule by ID.
// The workspaceID parameter ensures the rule belongs to the requesting workspace.
func (s *Service) DeleteRule(id, workspaceID string) error {
	ctx := context.Background()

	affected, err := s.rules.DeleteByIDAndWorkspace(ctx, id, workspaceID)
	if err != nil {
		return fmt.Errorf("deleting notification rule: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("Notification rule not found")
	}

	slog.Info("notification rule deleted", "rule_id", id, "workspace_id", workspaceID)
	return nil
}

// severityOrder defines the severity hierarchy for comparison.
var severityOrder = map[string]int{
	"low":      0,
	"medium":   1,
	"high":     2,
	"critical": 3,
}

// Dispatch sends a notification through all matching rules/channels for an
// workspace. It creates in-app notification records and dispatches to
// external channels (email, Slack, webhook).
func (s *Service) Dispatch(ctx context.Context, workspaceID string, severity, title, message string) {
	s.dispatchInternal(ctx, workspaceID, severity, "", "", "", title, message, "")
}

// DispatchEvent sends a notification with full structured event data through
// all matching rules/channels for a workspace.
func (s *Service) DispatchEvent(ctx context.Context, workspaceID string, evt entity.NotificationEvent) {
	s.dispatchInternal(ctx, workspaceID, evt.Severity, evt.EventType, evt.ReferenceID, evt.ReferenceType, evt.Title, evt.Message, evt.UserID)
}

// dispatchInternal is the shared implementation for Dispatch and DispatchEvent.
func (s *Service) dispatchInternal(ctx context.Context, workspaceID string, severity, eventType string, referenceID string, referenceType, title, message string, userID string) {
	// Always create one in-app notification record (ChannelID=0 means in-app,
	// not tied to any external channel). This ensures the frontend bell icon
	// always has something to show regardless of whether notification
	// rules/channels are configured.
	inAppNotification := &entity.Notification{
		WorkspaceID:   workspaceID,
		UserID:        userID, // empty string means workspace-wide, non-empty targets a specific user
		ChannelID:     "",     // in-app notification, no external channel
		Severity:      severity,
		EventType:     eventType,
		ReferenceID:   referenceID,
		ReferenceType: referenceType,
		Title:         title,
		Message:       message,
		SentAt:        time.Now(),
	}
	if err := s.notifications.Create(ctx, inAppNotification); err != nil {
		slog.Error("failed to create in-app notification record",
			"workspace_id", workspaceID,
			"error", err,
		)
		// Continue to attempt external dispatch even if in-app record fails
	}

	// Find all active rules for this workspace whose severity threshold is met
	rules, err := s.rules.FindActiveByWorkspaceID(ctx, workspaceID)
	if err != nil {
		slog.Error("failed to load notification rules", "workspace_id", workspaceID, "error", err)
		return
	}

	incomingSeverity := severityOrder[severity]

	for _, rule := range rules {
		ruleSeverity := severityOrder[rule.Severity]
		if incomingSeverity < ruleSeverity {
			continue
		}

		// Load the channel
		channel, err := s.channels.FindByID(ctx, rule.ChannelID)
		if err != nil {
			slog.Error("failed to load notification channel",
				"channel_id", rule.ChannelID,
				"error", err,
			)
			continue
		}

		if !channel.IsActive {
			continue
		}

		// Dispatch to external channel (email, Slack, webhook).
		// Best-effort for bulk sends - errors are logged inside dispatch.
		_ = s.dispatchToChannel(*channel, title, message)
	}
}

// dispatchToChannel routes the notification to the appropriate channel sender.
func (s *Service) dispatchToChannel(channel entity.NotificationChannel, title, message string) error {
	switch channel.Type {
	case entity.NotificationChannelEmail:
		return s.sendEmail(channel, title, message)
	case entity.NotificationChannelSlack:
		return s.sendSlack(channel, title, message)
	case entity.NotificationChannelWebhook:
		return s.sendWebhook(channel, title, message)
	default:
		return fmt.Errorf("unknown notification channel type: %s", channel.Type)
	}
}

// emailConfig is the expected JSON config for an email channel.
type emailConfig struct {
	// Recipients is a comma-separated list of email addresses to notify.
	Recipients string `json:"recipients"`
}

// sendEmail sends a notification email via the configured email sender.
// The channel config should contain a JSON object with a "recipients" field.
// If the email sender is not configured, it returns an error.
func (s *Service) sendEmail(channel entity.NotificationChannel, title, message string) error {
	if s.emailSender == nil {
		return fmt.Errorf("SMTP not configured")
	}

	var cfg emailConfig
	if err := json.Unmarshal([]byte(channel.Config), &cfg); err != nil {
		return fmt.Errorf("invalid email channel config: %w", err)
	}

	if cfg.Recipients == "" {
		return fmt.Errorf("email recipients is empty")
	}

	recipients := strings.Split(cfg.Recipients, ",")
	for i := range recipients {
		recipients[i] = strings.TrimSpace(recipients[i])
	}

	subject := "[Veilence-MX] " + title
	body := message + "\r\n\r\n---\r\nSent by Veilence-MX notification system\r\n"

	if err := s.emailSender.SendNotificationEmail(s.smtp.From, recipients, subject, body); err != nil {
		return fmt.Errorf("sending email: %w", err)
	}

	slog.Info("email notification sent",
		"channel_id", channel.ID,
		"recipients", cfg.Recipients,
		"title", title,
	)
	return nil
}

// ---------------------------------------------------------------------------
// SSRF prevention - URL validation
// ---------------------------------------------------------------------------

// ErrSSRFBlocked is returned when a URL targets a private/internal address.
var ErrSSRFBlocked = errors.New("URL targets a private or internal address")

// ErrInvalidSlackURL is returned when a Slack webhook URL doesn't match the expected entity.
var ErrInvalidSlackURL = errors.New("Slack webhook URL must use https://hooks.slack.com")

// ValidateWebhookURL validates that a URL is safe for outbound HTTP requests.
// It blocks private IP ranges, localhost, link-local addresses, and non-HTTP(S) schemes.
func ValidateWebhookURL(rawURL string) error {
	if rawURL == "" {
		return fmt.Errorf("URL is empty")
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	// Only allow HTTP and HTTPS schemes.
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return fmt.Errorf("URL scheme must be http or https, got %q", parsed.Scheme)
	}

	// Resolve hostname to check if it points to a private IP.
	hostname := parsed.Hostname()
	if hostname == "" {
		return fmt.Errorf("URL has no hostname")
	}

	// Block localhost by name.
	lowerHost := strings.ToLower(hostname)
	if lowerHost == "localhost" || lowerHost == "localhost." {
		return ErrSSRFBlocked
	}

	// Resolve and check IP addresses.
	ips, err := net.LookupHost(hostname)
	if err != nil {
		// If we can't resolve, block to be safe - the webhook would fail anyway.
		return fmt.Errorf("cannot resolve hostname %q: %w", hostname, err)
	}

	for _, ipStr := range ips {
		ip := net.ParseIP(ipStr)
		if ip == nil {
			continue
		}
		if isPrivateOrReservedIP(ip) {
			return ErrSSRFBlocked
		}
	}

	return nil
}

// ValidateSlackWebhookURL validates that a Slack webhook URL is safe and
// actually targets hooks.slack.com.
func ValidateSlackWebhookURL(rawURL string) error {
	if err := ValidateWebhookURL(rawURL); err != nil {
		return err
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return err
	}

	// Slack webhook URLs must be HTTPS and target hooks.slack.com.
	if strings.ToLower(parsed.Scheme) != "https" {
		return ErrInvalidSlackURL
	}
	host := strings.ToLower(parsed.Hostname())
	if host != "hooks.slack.com" {
		return ErrInvalidSlackURL
	}

	return nil
}

// ValidateChannelConfig validates URLs in a channel's config based on channel type.
// Call this when creating or updating a channel to reject SSRF targets early.
func ValidateChannelConfig(channelType entity.NotificationChannelType, config string) error {
	switch channelType {
	case entity.NotificationChannelWebhook:
		var cfg webhookConfig
		if err := json.Unmarshal([]byte(config), &cfg); err != nil {
			return fmt.Errorf("invalid webhook config JSON: %w", err)
		}
		if cfg.URL == "" {
			return fmt.Errorf("Webhook URL is required")
		}
		return ValidateWebhookURL(cfg.URL)

	case entity.NotificationChannelSlack:
		var cfg slackConfig
		if err := json.Unmarshal([]byte(config), &cfg); err != nil {
			return fmt.Errorf("invalid slack config JSON: %w", err)
		}
		if cfg.WebhookURL == "" {
			return fmt.Errorf("Slack webhook URL is required")
		}
		return ValidateSlackWebhookURL(cfg.WebhookURL)

	case entity.NotificationChannelEmail:
		// Email channels don't have URLs to validate for SSRF.
		return nil

	default:
		return nil
	}
}

// isPrivateOrReservedIP returns true if the IP is in a private, loopback,
// link-local, or other reserved range that should not be targeted by
// outbound webhook requests.
func isPrivateOrReservedIP(ip net.IP) bool {
	// Check standard private/reserved ranges.
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
		return true
	}

	// IPv4 private ranges: 10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16
	// Also: 169.254.0.0/16 (link-local, caught above), 100.64.0.0/10 (CGNAT)
	privateRanges := []struct {
		network string
	}{
		{"10.0.0.0/8"},
		{"172.16.0.0/12"},
		{"192.168.0.0/16"},
		{"100.64.0.0/10"},  // Carrier-grade NAT
		{"169.254.0.0/16"}, // Link-local (redundant with IsLinkLocalUnicast but explicit)
		{"127.0.0.0/8"},    // Loopback (redundant but explicit)
		{"0.0.0.0/8"},      // "This" network
		{"fc00::/7"},       // IPv6 unique local
		{"::1/128"},        // IPv6 loopback
		{"fe80::/10"},      // IPv6 link-local (redundant but explicit)
	}

	for _, r := range privateRanges {
		_, cidr, err := net.ParseCIDR(r.network)
		if err != nil {
			continue
		}
		if cidr.Contains(ip) {
			return true
		}
	}

	return false
}

// slackConfig is the expected JSON config for a Slack channel.
type slackConfig struct {
	// WebhookURL is the Slack incoming webhook URL.
	WebhookURL string `json:"webhookUrl"`
}

// sendSlack sends a notification to a Slack channel via incoming webhook.
func (s *Service) sendSlack(channel entity.NotificationChannel, title, message string) error {
	var cfg slackConfig
	if err := json.Unmarshal([]byte(channel.Config), &cfg); err != nil {
		return fmt.Errorf("invalid slack channel config: %w", err)
	}

	if cfg.WebhookURL == "" {
		return fmt.Errorf("slack webhook URL is empty")
	}

	// SSRF protection: validate the Slack webhook URL at dispatch time.
	if !s.AllowLocalURLs {
		if err := ValidateSlackWebhookURL(cfg.WebhookURL); err != nil {
			return fmt.Errorf("slack webhook URL blocked by SSRF policy: %w", err)
		}
	}

	// Format as a Slack mrkdwn message
	text := fmt.Sprintf(":warning: *%s*\n%s", title, message)

	if err := s.slackSender.SendSlack(cfg.WebhookURL, text); err != nil {
		return fmt.Errorf("sending slack notification: %w", err)
	}

	slog.Info("slack notification sent",
		"channel_id", channel.ID,
		"title", title,
	)
	return nil
}

// webhookPayload is the JSON payload sent to webhook notification channels.
type webhookPayload struct {
	Title   string `json:"title"`
	Message string `json:"message"`
	Channel string `json:"channel"`
}

// webhookConfig is the expected JSON config for a webhook channel.
type webhookConfig struct {
	URL    string `json:"url"`
	Secret string `json:"secret,omitempty"`
}

// sendWebhook actually POSTs to the configured webhook URL via the webhook sender.
// If a signing secret is configured, it is passed to the sender for HMAC-SHA256 signing.
func (s *Service) sendWebhook(channel entity.NotificationChannel, title, message string) error {
	var cfg webhookConfig
	if err := json.Unmarshal([]byte(channel.Config), &cfg); err != nil {
		return fmt.Errorf("invalid webhook config: %w", err)
	}

	if cfg.URL == "" {
		return fmt.Errorf("webhook URL is empty")
	}

	// SSRF protection: validate the webhook URL at dispatch time.
	if !s.AllowLocalURLs {
		if err := ValidateWebhookURL(cfg.URL); err != nil {
			return fmt.Errorf("webhook URL blocked by SSRF policy: %w", err)
		}
	}

	payload := webhookPayload{
		Title:   title,
		Message: message,
		Channel: channel.Name,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshaling webhook payload: %w", err)
	}

	if err := s.webhookSender.SendWebhook(cfg.URL, body, cfg.Secret); err != nil {
		return fmt.Errorf("sending webhook: %w", err)
	}

	slog.Info("webhook notification sent",
		"channel_id", channel.ID,
		"url", cfg.URL,
	)
	return nil
}

// ListNotifications returns notifications for a user across all workspaces.
// If onlyUnread is true, only unread notifications are returned.
// When workspaceID is 0, notifications are scoped to the workspaces the user
// actually belongs to - users with no workspaces receive zero notifications.
func (s *Service) ListNotifications(workspaceID, userID string, onlyUnread bool) ([]entity.Notification, error) {
	ctx := context.Background()

	if workspaceID == "" {
		wsIDs, err := s.workspaces.FindWorkspaceIDsByUserID(ctx, userID)
		if err != nil {
			return nil, fmt.Errorf("listing user workspaces: %w", err)
		}

		if len(wsIDs) == 0 {
			return nil, nil
		}

		notifs, err := s.notifications.FindByUserAndWorkspaceIDs(ctx, wsIDs, userID, onlyUnread)
		if err != nil {
			return nil, fmt.Errorf("listing workspace notifications: %w", err)
		}
		return notifs, nil
	}

	notifs, err := s.notifications.FindByUserAndWorkspace(ctx, workspaceID, userID, onlyUnread)
	if err != nil {
		return nil, fmt.Errorf("listing notifications: %w", err)
	}
	return notifs, nil
}

// MarkRead marks a notification as read.
// The userID parameter ensures the notification belongs to the requesting user
// (either directly assigned or workspace-wide with user_id=0).
// The notification is verified to belong to one of the user's workspaces.
func (s *Service) MarkRead(id, userID string) error {
	ctx := context.Background()

	// Verify the notification belongs to the user (either via workspace
	// membership or direct user targeting like invitation notifications).
	notif, err := s.notifications.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("Notification not found")
	}

	// Allow if the notification is directly targeted to this user
	if notif.UserID != userID {
		// Otherwise, verify the user is a member of the notification's workspace
		wsIDs, err := s.workspaces.FindWorkspaceIDsByUserID(ctx, userID)
		if err != nil {
			return fmt.Errorf("listing user workspaces: %w", err)
		}

		if !containsString(wsIDs, notif.WorkspaceID) {
			return fmt.Errorf("Notification not found")
		}
	}

	// Use the notification's actual workspace ID for the scoped update.
	affected, err := s.notifications.MarkRead(ctx, id, notif.WorkspaceID, userID)
	if err != nil {
		return fmt.Errorf("marking notification as read: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("Notification not found")
	}
	return nil
}

// GetUnreadCount returns the number of unread notifications for a user.
// When workspaceID is 0, counts are scoped to the user's actual workspaces
// plus any user-targeted notifications from other workspaces.
func (s *Service) GetUnreadCount(workspaceID, userID string) (int64, error) {
	ctx := context.Background()

	if workspaceID == "" {
		wsIDs, err := s.workspaces.FindWorkspaceIDsByUserID(ctx, userID)
		if err != nil {
			return 0, fmt.Errorf("listing user workspaces: %w", err)
		}
		if len(wsIDs) == 0 {
			return 0, nil
		}
		count, err := s.notifications.CountUnreadByWorkspaceIDs(ctx, wsIDs, userID)
		if err != nil {
			return 0, fmt.Errorf("counting workspace unread: %w", err)
		}
		return count, nil
	}

	count, err := s.notifications.CountUnread(ctx, workspaceID, userID)
	if err != nil {
		return 0, fmt.Errorf("counting unread notifications: %w", err)
	}
	return count, nil
}

// MarkAllRead marks all unread notifications as read for a user.
// If workspaceID is non-zero, only notifications for that workspace are affected.
// When workspaceID is 0, only notifications from the user's workspaces are affected.
func (s *Service) MarkAllRead(workspaceID, userID string) (int64, error) {
	ctx := context.Background()

	if workspaceID == "" {
		wsIDs, err := s.workspaces.FindWorkspaceIDsByUserID(ctx, userID)
		if err != nil {
			return 0, fmt.Errorf("listing user workspaces: %w", err)
		}
		if len(wsIDs) == 0 {
			return 0, nil
		}
		affected, err := s.notifications.MarkAllReadByWorkspaceIDs(ctx, wsIDs, userID)
		if err != nil {
			return 0, fmt.Errorf("marking workspace notifications as read: %w", err)
		}
		return affected, nil
	}

	affected, err := s.notifications.MarkAllRead(ctx, workspaceID, userID)
	if err != nil {
		return 0, fmt.Errorf("marking all notifications as read: %w", err)
	}
	return affected, nil
}

// DeleteByID deletes a single notification by ID, scoped to the user's workspaces.
// When workspaceID is 0, the notification is verified to belong to one of the user's workspaces.
func (s *Service) DeleteByID(ctx context.Context, id, workspaceID, userID string) (int64, error) {
	if workspaceID == "" {
		// Verify the notification belongs to the user (via workspace or direct targeting).
		notif, err := s.notifications.FindByID(ctx, id)
		if err != nil {
			return 0, fmt.Errorf("notification not found")
		}

		if notif.UserID != userID {
			// Not directly targeted - check workspace membership
			wsIDs, err := s.workspaces.FindWorkspaceIDsByUserID(ctx, userID)
			if err != nil {
				return 0, fmt.Errorf("listing user workspaces: %w", err)
			}

			if !containsString(wsIDs, notif.WorkspaceID) {
				return 0, fmt.Errorf("notification not found")
			}
		}

		// Use the notification's actual workspace ID for the scoped delete.
		workspaceID = notif.WorkspaceID
	}

	affected, err := s.notifications.DeleteByID(ctx, id, workspaceID, userID)
	if err != nil {
		return 0, fmt.Errorf("deleting notification: %w", err)
	}
	if affected == 0 {
		return 0, fmt.Errorf("notification not found")
	}

	slog.Info("notification deleted", "notification_id", id, "workspace_id", workspaceID, "user_id", userID)
	return affected, nil
}

// DeleteAll deletes all notifications for a user within a workspace.
// When workspaceID is 0, only notifications from the user's workspaces are deleted.
func (s *Service) DeleteAll(ctx context.Context, workspaceID, userID string) (int64, error) {
	if workspaceID == "" {
		wsIDs, err := s.workspaces.FindWorkspaceIDsByUserID(ctx, userID)
		if err != nil {
			return 0, fmt.Errorf("listing user workspaces: %w", err)
		}
		if len(wsIDs) == 0 {
			return 0, nil
		}
		affected, err := s.notifications.DeleteAllByWorkspaceIDs(ctx, wsIDs, userID)
		if err != nil {
			return 0, fmt.Errorf("deleting workspace notifications: %w", err)
		}
		slog.Info("all notifications deleted", "user_id", userID, "count", affected)
		return affected, nil
	}

	affected, err := s.notifications.DeleteAll(ctx, workspaceID, userID)
	if err != nil {
		return 0, fmt.Errorf("deleting all notifications: %w", err)
	}

	slog.Info("all notifications deleted", "workspace_id", workspaceID, "user_id", userID, "count", affected)
	return affected, nil
}

// DeleteBatch deletes multiple notifications by IDs, scoped to the user's workspaces.
// When workspaceID is 0, notifications are scoped to the user's actual workspaces.
func (s *Service) DeleteBatch(ctx context.Context, ids []string, workspaceID, userID string) (int64, error) {
	if len(ids) == 0 {
		return 0, fmt.Errorf("no notification IDs provided")
	}
	if len(ids) > 100 {
		return 0, fmt.Errorf("batch delete limited to 100 notifications at a time")
	}

	if workspaceID == "" {
		wsIDs, err := s.workspaces.FindWorkspaceIDsByUserID(ctx, userID)
		if err != nil {
			return 0, fmt.Errorf("listing user workspaces: %w", err)
		}
		if len(wsIDs) == 0 {
			return 0, nil
		}
		affected, err := s.notifications.DeleteBatchByWorkspaceIDs(ctx, ids, wsIDs, userID)
		if err != nil {
			return 0, fmt.Errorf("batch deleting notifications: %w", err)
		}
		slog.Info("notifications batch deleted", "workspace_ids", wsIDs, "user_id", userID, "requested", len(ids), "deleted", affected)
		return affected, nil
	}

	affected, err := s.notifications.DeleteBatch(ctx, ids, workspaceID, userID)
	if err != nil {
		return 0, fmt.Errorf("batch deleting notifications: %w", err)
	}

	slog.Info("notifications batch deleted", "workspace_id", workspaceID, "user_id", userID, "requested", len(ids), "deleted", affected)
	return affected, nil
}

// TestChannel sends a test notification through a specific channel to verify it works.
// Returns nil on success, an error describing the failure otherwise.
func (s *Service) TestChannel(id, workspaceID string) error {
	ctx := context.Background()

	channel, err := s.channels.FindByIDAndWorkspace(ctx, id, workspaceID)
	if err != nil {
		return fmt.Errorf("Notification channel not found in this workspace")
	}

	title := "Veilence-MX Test Notification"
	message := "This is a test notification from Veilence-MX. If you received this, your notification channel is configured correctly."

	if err := s.dispatchToChannel(*channel, title, message); err != nil {
		return fmt.Errorf("Test notification failed: %w", err)
	}

	slog.Info("test notification dispatched",
		"channel_id", id,
		"workspace_id", workspaceID,
		"type", channel.Type,
	)
	return nil
}

// GetChannel returns a notification channel by ID and workspace.
func (s *Service) GetChannel(id, workspaceID string) (*entity.NotificationChannel, error) {
	ctx := context.Background()
	return s.channels.FindByIDAndWorkspace(ctx, id, workspaceID)
}

// containsString returns true if the slice contains the given value.
func containsString(slice []string, val string) bool {
	for _, v := range slice {
		if v == val {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// Webhook HMAC-SHA256 utilities
// ---------------------------------------------------------------------------

// ComputeHMACSignature computes the HMAC-SHA256 hex digest of body using the
// given secret key. The returned string is the lowercase hex-encoded digest
// (without the "sha256=" prefix - callers add that when building the header).
func ComputeHMACSignature(secret, body []byte) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

// ValidateWebhookSignature validates that signatureHeader matches the
// HMAC-SHA256 of body computed with secret. The signatureHeader is expected in
// the form "sha256=<hex_digest>". The comparison is performed in constant time
// to prevent timing-based side-channel attacks.
func ValidateWebhookSignature(secret, body []byte, signatureHeader string) bool {
	const prefix = "sha256="
	if len(signatureHeader) <= len(prefix) {
		return false
	}
	if signatureHeader[:len(prefix)] != prefix {
		return false
	}

	receivedHex := signatureHeader[len(prefix):]
	receivedMAC, err := hex.DecodeString(receivedHex)
	if err != nil {
		return false
	}

	expectedMAC := hmac.New(sha256.New, secret)
	expectedMAC.Write(body)

	return hmac.Equal(receivedMAC, expectedMAC.Sum(nil))
}


