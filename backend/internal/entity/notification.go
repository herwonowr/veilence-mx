package entity

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
	ID             uint
	OrgID          uint
	Name           string
	Type           NotificationChannelType
	Config         string
	IsActive       bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// NotificationRule defines a routing rule that maps severity levels to channels.
type NotificationRule struct {
	ID             uint
	OrgID          uint
	ChannelID      uint
	Severity       string
	IsActive       bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// Notification represents an in-app notification sent to a user or org.
type Notification struct {
	ID             uint
	OrgID          uint
	UserID         uint
	ChannelID      uint
	Title          string
	Message        string
	IsRead         bool
	SentAt         time.Time
	CreatedAt      time.Time
}
