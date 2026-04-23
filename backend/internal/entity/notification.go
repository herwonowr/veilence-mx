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
	ID             string
	WorkspaceID    string
	Name           string
	Type           NotificationChannelType
	Config         string
	IsActive       bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// NotificationRule defines a routing rule that maps severity levels to channels.
type NotificationRule struct {
	ID             string
	WorkspaceID    string
	ChannelID      string
	Severity       string
	IsActive       bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// Notification represents an in-app notification sent to a user or workspace.
type Notification struct {
	ID            string
	WorkspaceID   string
	UserID        string
	ChannelID     string
	Severity      string // "critical", "high", "medium", "low"
	EventType     string // e.g. "alert.created.malicious", "diff.error"
	ReferenceID   string // ID of related entity (alert_id, release_id, etc.)
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
	ReferenceID   string
	ReferenceType string // "alert", "release", "package"
	UserID        string // optional - target a specific user instead of workspace-wide
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
