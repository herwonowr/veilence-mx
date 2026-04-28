package response

import (
	"time"

	"github.com/veilence/veilence-mx/backend/internal/entity"
)

// NotificationChannelResponse is the JSON representation of a notification channel.
type NotificationChannelResponse struct {
	ID          string                         `json:"id"`
	WorkspaceID string                         `json:"workspaceId"`
	Name        string                         `json:"name"`
	Type        entity.NotificationChannelType `json:"type"`
	Config      string                         `json:"config"`
	IsActive    bool                           `json:"isActive"`
	CreatedAt   time.Time                      `json:"createdAt"`
	UpdatedAt   time.Time                      `json:"updatedAt"`
}

// NotificationChannelFromEntity maps a domain NotificationChannel to a response DTO.
func NotificationChannelFromEntity(c *entity.NotificationChannel) NotificationChannelResponse {
	return NotificationChannelResponse{
		ID:          c.ID,
		WorkspaceID: c.WorkspaceID,
		Name:        c.Name,
		Type:        c.Type,
		Config:      c.Config,
		IsActive:    c.IsActive,
		CreatedAt:   c.CreatedAt,
		UpdatedAt:   c.UpdatedAt,
	}
}

// NotificationChannelsFromEntities maps a slice of domain NotificationChannels to response DTOs.
func NotificationChannelsFromEntities(channels []entity.NotificationChannel) []NotificationChannelResponse {
	result := make([]NotificationChannelResponse, len(channels))
	for i := range channels {
		result[i] = NotificationChannelFromEntity(&channels[i])
	}
	return result
}

// NotificationRuleResponse is the JSON representation of a notification rule.
type NotificationRuleResponse struct {
	ID          string    `json:"id"`
	WorkspaceID string    `json:"workspaceId"`
	ChannelID   string    `json:"channelId"`
	Severity    string    `json:"severity"`
	IsActive    bool      `json:"isActive"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// NotificationRuleFromEntity maps a domain NotificationRule to a response DTO.
func NotificationRuleFromEntity(r *entity.NotificationRule) NotificationRuleResponse {
	return NotificationRuleResponse{
		ID:          r.ID,
		WorkspaceID: r.WorkspaceID,
		ChannelID:   r.ChannelID,
		Severity:    r.Severity,
		IsActive:    r.IsActive,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}
}

// NotificationRulesFromEntities maps a slice of domain NotificationRules to response DTOs.
func NotificationRulesFromEntities(rules []entity.NotificationRule) []NotificationRuleResponse {
	result := make([]NotificationRuleResponse, len(rules))
	for i := range rules {
		result[i] = NotificationRuleFromEntity(&rules[i])
	}
	return result
}

// NotificationResponse is the JSON representation of an in-app notification.
type NotificationResponse struct {
	ID            string    `json:"id"`
	WorkspaceID   string    `json:"workspaceId"`
	UserID        string    `json:"userId"`
	ChannelID     string    `json:"channelId"`
	Severity      string    `json:"severity"`
	EventType     string    `json:"eventType"`
	ReferenceID   string    `json:"referenceId"`
	ReferenceType string    `json:"referenceType"`
	Title         string    `json:"title"`
	Message       string    `json:"message"`
	IsRead        bool      `json:"isRead"`
	SentAt        time.Time `json:"sentAt"`
	CreatedAt     time.Time `json:"createdAt"`
}

// NotificationFromEntity maps a domain Notification to a response DTO.
func NotificationFromEntity(n *entity.Notification) NotificationResponse {
	return NotificationResponse{
		ID:            n.ID,
		WorkspaceID:   n.WorkspaceID,
		UserID:        n.UserID,
		ChannelID:     n.ChannelID,
		Severity:      n.Severity,
		EventType:     n.EventType,
		ReferenceID:   n.ReferenceID,
		ReferenceType: n.ReferenceType,
		Title:         n.Title,
		Message:       n.Message,
		IsRead:        n.IsRead,
		SentAt:        n.SentAt,
		CreatedAt:     n.CreatedAt,
	}
}

// NotificationsFromEntities maps a slice of domain Notifications to response DTOs.
func NotificationsFromEntities(ns []entity.Notification) []NotificationResponse {
	result := make([]NotificationResponse, len(ns))
	for i := range ns {
		result[i] = NotificationFromEntity(&ns[i])
	}
	return result
}
