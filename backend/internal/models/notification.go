package models

import (
	"time"
)

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
	ID        uint                    `gorm:"primarykey" json:"id"`
	OrgID     uint                    `gorm:"not null;index" json:"orgId"`
	Name      string                  `gorm:"not null;type:varchar(100)" json:"name"`
	Type      NotificationChannelType `gorm:"not null;type:varchar(20)" json:"type"`
	Config    string                  `gorm:"type:text" json:"config"` // JSON blob
	IsActive  bool                    `gorm:"not null;default:true" json:"isActive"`
	CreatedAt time.Time               `json:"createdAt"`
	UpdatedAt time.Time               `json:"updatedAt"`
}

// TableName returns the table name for NotificationChannel.
func (NotificationChannel) TableName() string {
	return "notification_channels"
}

// NotificationRule defines a routing rule that maps severity levels to channels.
type NotificationRule struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	OrgID     uint      `gorm:"not null;index" json:"orgId"`
	ChannelID uint      `gorm:"not null;index" json:"channelId"`
	Severity  string    `gorm:"type:varchar(20)" json:"severity"` // min severity to trigger
	IsActive  bool      `gorm:"not null;default:true" json:"isActive"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// TableName returns the table name for NotificationRule.
func (NotificationRule) TableName() string {
	return "notification_rules"
}

// Notification represents an in-app notification sent to a user or org.
type Notification struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	OrgID     uint      `gorm:"not null;index" json:"orgId"`
	UserID    uint      `gorm:"index" json:"userId"` // 0 for org-wide
	ChannelID uint      `gorm:"index" json:"channelId"`
	Title     string    `gorm:"not null;type:varchar(255)" json:"title"`
	Message   string    `gorm:"type:text" json:"message"`
	IsRead    bool      `gorm:"not null;default:false" json:"isRead"`
	SentAt    time.Time `json:"sentAt"`
	CreatedAt time.Time `json:"createdAt"`
}

// TableName returns the table name for Notification.
func (Notification) TableName() string {
	return "notifications"
}
