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
	WorkspaceID    uint
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
	WorkspaceID    uint
	ChannelID      uint
	Severity       string
	IsActive       bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// Notification represents an in-app notification sent to a user or workspace.
type Notification struct {
	ID            uint
	WorkspaceID   uint
	UserID        uint
	ChannelID     uint
	Severity      string // "critical", "high", "medium", "low"
	EventType     string // e.g. "alert.created.malicious", "diff.error"
	ReferenceID   uint   // ID of related entity (alert_id, release_id, etc.)
	ReferenceType string // "alert", "release", "package" (for click-through)
	Title         string
	Message       string
	IsRead        bool
	SentAt        time.Time
	CreatedAt     time.Time
}

// NotificationEvent carries structured event data for dispatching notifications.
type NotificationEvent struct {
	Severity      string
	EventType     string
	Title         string
	Message       string
	ReferenceID   uint
	ReferenceType string // "alert", "release", "package"
}

// Notification event type constants.
const (
	NotifEventAlertMalicious  = "alert.created.malicious"
	NotifEventAlertSuspicious = "alert.created.suspicious"
	NotifEventAnalysisError   = "analysis.error"
	NotifEventDiscoveryAdded  = "discovery.packages_added"
	NotifEventDiffError       = "diff.error"
	NotifEventStaleRemoved    = "packages.stale_removed"
)
