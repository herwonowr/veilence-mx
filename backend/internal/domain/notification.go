package domain

import "time"

// NotificationChannelType represents the type of a notification channel.
type NotificationChannelType string

const (
	// NotificationChannelEmail represents an email notification channel.
	NotificationChannelEmail NotificationChannelType = "email"
	// NotificationChannelSlack represents a Slack notification channel.
	NotificationChannelSlack NotificationChannelType = "slack"
	// NotificationChannelWebhook represents a webhook notification channel.
	NotificationChannelWebhook NotificationChannelType = "webhook"
)

// NotificationChannel represents a configured channel for sending notifications.
type NotificationChannel struct {
	ID        uint                    `json:"id"`
	OrgID     uint                    `json:"orgId"`
	Name      string                  `json:"name"`
	Type      NotificationChannelType `json:"type"`
	Config    string                  `json:"config"`
	IsActive  bool                    `json:"isActive"`
	CreatedAt time.Time               `json:"createdAt"`
	UpdatedAt time.Time               `json:"updatedAt"`
}

// NotificationRule defines a routing rule that maps severity levels to channels.
type NotificationRule struct {
	ID        uint      `json:"id"`
	OrgID     uint      `json:"orgId"`
	ChannelID uint      `json:"channelId"`
	Severity  string    `json:"severity"`
	IsActive  bool      `json:"isActive"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// Notification represents an in-app notification sent to a user or org.
type Notification struct {
	ID        uint      `json:"id"`
	OrgID     uint      `json:"orgId"`
	UserID    uint      `json:"userId"`
	ChannelID uint      `json:"channelId"`
	Title     string    `json:"title"`
	Message   string    `json:"message"`
	IsRead    bool      `json:"isRead"`
	SentAt    time.Time `json:"sentAt"`
	CreatedAt time.Time `json:"createdAt"`
}
