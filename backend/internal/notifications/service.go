package notifications

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/veilence/veilence-mx/backend/internal/domain"
)

// Service provides notification management and dispatch operations.
type Service struct {
	channels      domain.NotificationChannelRepository
	rules         domain.NotificationRuleRepository
	notifications domain.NotificationRepository
}

// NewService creates a new notification service.
func NewService(channels domain.NotificationChannelRepository, rules domain.NotificationRuleRepository, notifications domain.NotificationRepository) *Service {
	return &Service{
		channels:      channels,
		rules:         rules,
		notifications: notifications,
	}
}

// CreateChannel creates a new notification channel for an organization.
func (s *Service) CreateChannel(orgID uint, name string, channelType domain.NotificationChannelType, config string) (*domain.NotificationChannel, error) {
	ctx := context.Background()

	channel := &domain.NotificationChannel{
		OrgID:    orgID,
		Name:     name,
		Type:     channelType,
		Config:   config,
		IsActive: true,
	}

	if err := s.channels.Create(ctx, channel); err != nil {
		return nil, fmt.Errorf("creating notification channel: %w", err)
	}

	slog.Info("notification channel created",
		"channel_id", channel.ID,
		"org_id", orgID,
		"type", channelType,
		"name", name,
	)
	return channel, nil
}

// ListChannels returns all notification channels for an organization.
func (s *Service) ListChannels(orgID uint) ([]domain.NotificationChannel, error) {
	ctx := context.Background()

	channels, err := s.channels.FindByOrgID(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("listing notification channels: %w", err)
	}
	return channels, nil
}

// UpdateChannel updates an existing notification channel.
// The orgID parameter ensures the channel belongs to the requesting organization.
func (s *Service) UpdateChannel(id, orgID uint, name string, config string, isActive bool) (*domain.NotificationChannel, error) {
	ctx := context.Background()

	channel, err := s.channels.FindByIDAndOrg(ctx, id, orgID)
	if err != nil {
		return nil, fmt.Errorf("finding notification channel: %w", err)
	}

	channel.Name = name
	channel.Config = config
	channel.IsActive = isActive

	if err := s.channels.Update(ctx, channel); err != nil {
		return nil, fmt.Errorf("updating notification channel: %w", err)
	}

	slog.Info("notification channel updated", "channel_id", id, "org_id", orgID)
	return channel, nil
}

// DeleteChannel deletes a notification channel by ID.
// The orgID parameter ensures the channel belongs to the requesting organization.
func (s *Service) DeleteChannel(id, orgID uint) error {
	ctx := context.Background()

	affected, err := s.channels.DeleteByIDAndOrg(ctx, id, orgID)
	if err != nil {
		return fmt.Errorf("deleting notification channel: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("notification channel not found")
	}

	slog.Info("notification channel deleted", "channel_id", id, "org_id", orgID)
	return nil
}

// CreateRule creates a new notification routing rule.
func (s *Service) CreateRule(orgID, channelID uint, severity string) (*domain.NotificationRule, error) {
	ctx := context.Background()

	rule := &domain.NotificationRule{
		OrgID:     orgID,
		ChannelID: channelID,
		Severity:  severity,
		IsActive:  true,
	}

	if err := s.rules.Create(ctx, rule); err != nil {
		return nil, fmt.Errorf("creating notification rule: %w", err)
	}

	slog.Info("notification rule created",
		"rule_id", rule.ID,
		"org_id", orgID,
		"channel_id", channelID,
		"severity", severity,
	)
	return rule, nil
}

// ListRules returns all notification rules for an organization.
func (s *Service) ListRules(orgID uint) ([]domain.NotificationRule, error) {
	ctx := context.Background()

	rules, err := s.rules.FindByOrgID(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("listing notification rules: %w", err)
	}
	return rules, nil
}

// DeleteRule deletes a notification rule by ID.
// The orgID parameter ensures the rule belongs to the requesting organization.
func (s *Service) DeleteRule(id, orgID uint) error {
	ctx := context.Background()

	affected, err := s.rules.DeleteByIDAndOrg(ctx, id, orgID)
	if err != nil {
		return fmt.Errorf("deleting notification rule: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("notification rule not found")
	}

	slog.Info("notification rule deleted", "rule_id", id, "org_id", orgID)
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
// organization. It creates in-app notification records and dispatches to
// external channels (email, Slack, webhook).
func (s *Service) Dispatch(orgID uint, severity, title, message string) {
	ctx := context.Background()

	// Find all active rules for this org whose severity threshold is met
	rules, err := s.rules.FindActiveByOrgID(ctx, orgID)
	if err != nil {
		slog.Error("failed to load notification rules", "org_id", orgID, "error", err)
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

		// Create in-app notification record
		notification := &domain.Notification{
			OrgID:     orgID,
			UserID:    0, // org-wide
			ChannelID: channel.ID,
			Title:     title,
			Message:   message,
			SentAt:    time.Now(),
		}
		if err := s.notifications.Create(ctx, notification); err != nil {
			slog.Error("failed to create notification record",
				"org_id", orgID,
				"channel_id", channel.ID,
				"error", err,
			)
			continue
		}

		// Dispatch to external channel
		s.dispatchToChannel(*channel, title, message)
	}
}

// dispatchToChannel routes the notification to the appropriate channel sender.
func (s *Service) dispatchToChannel(channel domain.NotificationChannel, title, message string) {
	switch channel.Type {
	case domain.NotificationChannelEmail:
		s.sendEmail(channel, title, message)
	case domain.NotificationChannelSlack:
		s.sendSlack(channel, title, message)
	case domain.NotificationChannelWebhook:
		s.sendWebhook(channel, title, message)
	default:
		slog.Warn("unknown notification channel type",
			"channel_id", channel.ID,
			"type", channel.Type,
		)
	}
}

// sendEmail is a stub implementation that logs the email instead of sending it.
func (s *Service) sendEmail(channel domain.NotificationChannel, title, message string) {
	slog.Info("[STUB] email notification",
		"channel_id", channel.ID,
		"channel_name", channel.Name,
		"title", title,
		"message", message,
		"config", channel.Config,
	)
}

// sendSlack is a stub implementation that logs the Slack message instead of sending it.
func (s *Service) sendSlack(channel domain.NotificationChannel, title, message string) {
	slog.Info("[STUB] slack notification",
		"channel_id", channel.ID,
		"channel_name", channel.Name,
		"title", title,
		"message", message,
		"config", channel.Config,
	)
}

// webhookPayload is the JSON payload sent to webhook notification channels.
type webhookPayload struct {
	Title   string `json:"title"`
	Message string `json:"message"`
	Channel string `json:"channel"`
}

// webhookConfig is the expected JSON config for a webhook channel.
type webhookConfig struct {
	URL string `json:"url"`
}

// sendWebhook actually POSTs to the configured webhook URL.
func (s *Service) sendWebhook(channel domain.NotificationChannel, title, message string) {
	var cfg webhookConfig
	if err := json.Unmarshal([]byte(channel.Config), &cfg); err != nil {
		slog.Error("invalid webhook config",
			"channel_id", channel.ID,
			"error", err,
		)
		return
	}

	if cfg.URL == "" {
		slog.Error("webhook URL is empty", "channel_id", channel.ID)
		return
	}

	payload := webhookPayload{
		Title:   title,
		Message: message,
		Channel: channel.Name,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		slog.Error("failed to marshal webhook payload",
			"channel_id", channel.ID,
			"error", err,
		)
		return
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(cfg.URL, "application/json", bytes.NewReader(body))
	if err != nil {
		slog.Error("failed to send webhook",
			"channel_id", channel.ID,
			"url", cfg.URL,
			"error", err,
		)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		slog.Error("webhook returned error status",
			"channel_id", channel.ID,
			"url", cfg.URL,
			"status", resp.StatusCode,
		)
		return
	}

	slog.Info("webhook notification sent",
		"channel_id", channel.ID,
		"url", cfg.URL,
		"status", resp.StatusCode,
	)
}

// ListNotifications returns notifications for a user across all organizations.
// If onlyUnread is true, only unread notifications are returned.
func (s *Service) ListNotifications(orgID, userID uint, onlyUnread bool) ([]domain.Notification, error) {
	ctx := context.Background()

	notifs, err := s.notifications.FindByUserAndOrg(ctx, orgID, userID, onlyUnread)
	if err != nil {
		return nil, fmt.Errorf("listing notifications: %w", err)
	}
	return notifs, nil
}

// MarkRead marks a notification as read.
// The userID parameter ensures the notification belongs to the requesting user
// (either directly assigned or org-wide with user_id=0).
func (s *Service) MarkRead(id, userID uint) error {
	ctx := context.Background()

	affected, err := s.notifications.MarkRead(ctx, id, userID)
	if err != nil {
		return fmt.Errorf("marking notification as read: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("notification not found")
	}
	return nil
}

// GetUnreadCount returns the number of unread notifications for a user.
func (s *Service) GetUnreadCount(orgID, userID uint) (int64, error) {
	ctx := context.Background()

	count, err := s.notifications.CountUnread(ctx, orgID, userID)
	if err != nil {
		return 0, fmt.Errorf("counting unread notifications: %w", err)
	}
	return count, nil
}
