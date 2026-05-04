package v1

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/google/uuid"
	validation "github.com/veilence/veilence-mx/backend/internal/controller/restapi/v1/request"
	"github.com/veilence/veilence-mx/backend/internal/controller/restapi/v1/response"

	"github.com/veilence/veilence-mx/backend/internal/entity"
	"github.com/veilence/veilence-mx/backend/internal/usecase/notifications"
	"github.com/veilence/veilence-mx/backend/internal/usecase/rbac"
)

// createChannelRequest is the request body for creating a notification channel.
type createChannelRequest struct {
	Name   string                         `json:"name"`
	Type   entity.NotificationChannelType `json:"type"`
	Config string                         `json:"config"`
}

// updateChannelRequest is the request body for updating a notification channel.
type updateChannelRequest struct {
	Name     string `json:"name"`
	Config   string `json:"config"`
	IsActive bool   `json:"isActive"`
}

// createRuleRequest is the request body for creating a notification rule.
type createRuleRequest struct {
	ChannelID string `json:"channelId"`
	Severity  string `json:"severity"`
}

// batchDeleteNotificationsRequest is the request body for batch deleting notifications.
type batchDeleteNotificationsRequest struct {
	IDs []string `json:"ids"`
}

// ListNotificationChannels handles GET /api/workspaces/{workspaceId}/notification-channels.
func (h *NotificationHandlers) ListNotificationChannels(w http.ResponseWriter, r *http.Request) {
	workspaceID := rbac.WorkspaceIDFromContext(r.Context())
	if workspaceID == "" {
		respondAppError(w, BadRequest("workspace context required"))
		return
	}

	channels, err := h.Notifications.ListChannels(workspaceID)
	if err != nil {
		respondAppError(w, Internal("failed to list notification channels"))
		return
	}

	respondJSON(w, http.StatusOK, response.NotificationChannelsFromEntities(channels), nil)
}

// CreateNotificationChannel handles POST /api/workspaces/{workspaceId}/notification-channels.
func (h *NotificationHandlers) CreateNotificationChannel(w http.ResponseWriter, r *http.Request) {
	workspaceID := rbac.WorkspaceIDFromContext(r.Context())
	if workspaceID == "" {
		respondAppError(w, BadRequest("workspace context required"))
		return
	}

	var req createChannelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondAppError(w, BadRequest("invalid request body"))
		return
	}

	if err := validation.ValidateRequired(req.Name, "name"); err != nil {
		respondAppError(w, ValidationFromErr(err))
		return
	}
	if err := validation.ValidateMaxLength(req.Name, "name", 100); err != nil {
		respondAppError(w, ValidationFromErr(err))
		return
	}

	validTypes := map[entity.NotificationChannelType]bool{
		entity.NotificationChannelEmail:   true,
		entity.NotificationChannelSlack:   true,
		entity.NotificationChannelWebhook: true,
	}
	if !validTypes[req.Type] {
		respondAppError(w, Validation("type must be 'email', 'slack', or 'webhook'"))
		return
	}

	// SSRF protection: validate URLs in channel config before persisting.
	if err := notifications.ValidateChannelConfig(req.Type, req.Config); err != nil {
		respondAppError(w, Validation(fmt.Sprintf("invalid channel config: %s", err.Error())))
		return
	}

	channel, err := h.Notifications.CreateChannel(workspaceID, req.Name, req.Type, req.Config)
	if err != nil {
		respondAppError(w, Internal("failed to create notification channel"))
		return
	}

	h.Audit.LogAction(r.Context(), "create", "notification_channel", channel.ID, fmt.Sprintf("created %s notification channel %q", req.Type, req.Name))

	respondJSON(w, http.StatusCreated, response.NotificationChannelFromEntity(channel), nil)
}

// UpdateNotificationChannel handles PUT /api/workspaces/{workspaceId}/notification-channels/{id}.
func (h *NotificationHandlers) UpdateNotificationChannel(w http.ResponseWriter, r *http.Request) {
	workspaceID := rbac.WorkspaceIDFromContext(r.Context())
	if workspaceID == "" {
		respondAppError(w, BadRequest("workspace context required"))
		return
	}

	id, ok := parseUUID(r, "id")
	if !ok {
		respondAppError(w, BadRequest("invalid channel ID"))
		return
	}

	var req updateChannelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondAppError(w, BadRequest("invalid request body"))
		return
	}

	if err := validation.ValidateRequired(req.Name, "name"); err != nil {
		respondAppError(w, ValidationFromErr(err))
		return
	}
	if err := validation.ValidateMaxLength(req.Name, "name", 100); err != nil {
		respondAppError(w, ValidationFromErr(err))
		return
	}

	// SSRF protection: fetch the existing channel to know its type, then validate
	// the new config's URLs before persisting.
	existingChannel, err := h.Notifications.GetChannel(id, workspaceID)
	if err != nil {
		respondAppError(w, NotFound("notification channel not found"))
		return
	}
	if err := notifications.ValidateChannelConfig(existingChannel.Type, req.Config); err != nil {
		respondAppError(w, Validation(fmt.Sprintf("invalid channel config: %s", err.Error())))
		return
	}

	channel, err := h.Notifications.UpdateChannel(id, workspaceID, req.Name, req.Config, req.IsActive)
	if err != nil {
		respondAppError(w, Internal("failed to update notification channel"))
		return
	}

	h.Audit.LogAction(r.Context(), "update", "notification_channel", id, fmt.Sprintf("updated notification channel %q (active: %t)", req.Name, req.IsActive))

	respondJSON(w, http.StatusOK, response.NotificationChannelFromEntity(channel), nil)
}

// DeleteNotificationChannel handles DELETE /api/workspaces/{workspaceId}/notification-channels/{id}.
func (h *NotificationHandlers) DeleteNotificationChannel(w http.ResponseWriter, r *http.Request) {
	workspaceID := rbac.WorkspaceIDFromContext(r.Context())
	if workspaceID == "" {
		respondAppError(w, BadRequest("workspace context required"))
		return
	}

	id, ok := parseUUID(r, "id")
	if !ok {
		respondAppError(w, BadRequest("invalid channel ID"))
		return
	}

	// Fetch channel name before deletion for audit log readability.
	channelName := id
	if ch, err := h.Notifications.GetChannel(id, workspaceID); err == nil {
		channelName = ch.Name
	}

	if err := h.Notifications.DeleteChannel(id, workspaceID); err != nil {
		if errors.Is(err, entity.ErrNotFound) {
			respondAppError(w, NotFound("notification channel"))
			return
		}
		respondAppError(w, Internal("failed to delete notification channel"))
		return
	}

	h.Audit.LogAction(r.Context(), "delete", "notification_channel", id, fmt.Sprintf("deleted notification channel %q", channelName))

	respondJSON(w, http.StatusOK, nil, nil)
}

// ListNotificationRules handles GET /api/workspaces/{workspaceId}/notification-rules.
func (h *NotificationHandlers) ListNotificationRules(w http.ResponseWriter, r *http.Request) {
	workspaceID := rbac.WorkspaceIDFromContext(r.Context())
	if workspaceID == "" {
		respondAppError(w, BadRequest("workspace context required"))
		return
	}

	rules, err := h.Notifications.ListRules(workspaceID)
	if err != nil {
		respondAppError(w, Internal("failed to list notification rules"))
		return
	}

	respondJSON(w, http.StatusOK, response.NotificationRulesFromEntities(rules), nil)
}

// CreateNotificationRule handles POST /api/workspaces/{workspaceId}/notification-rules.
func (h *NotificationHandlers) CreateNotificationRule(w http.ResponseWriter, r *http.Request) {
	workspaceID := rbac.WorkspaceIDFromContext(r.Context())
	if workspaceID == "" {
		respondAppError(w, BadRequest("workspace context required"))
		return
	}

	var req createRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondAppError(w, BadRequest("invalid request body"))
		return
	}

	if req.ChannelID == "" {
		respondAppError(w, Validation("channelId is required"))
		return
	}
	if _, err := uuid.Parse(req.ChannelID); err != nil {
		respondAppError(w, BadRequest("invalid channelId format"))
		return
	}

	validSeverities := map[string]bool{
		"low": true, "medium": true, "high": true, "critical": true,
	}
	if req.Severity != "" && !validSeverities[req.Severity] {
		respondAppError(w, Validation("severity must be 'low', 'medium', 'high', or 'critical'"))
		return
	}

	rule, err := h.Notifications.CreateRule(workspaceID, req.ChannelID, req.Severity)
	if err != nil {
		respondAppError(w, Internal("failed to create notification rule"))
		return
	}

	// Resolve channel name for audit log readability.
	channelName := req.ChannelID
	if ch, err := h.Notifications.GetChannel(req.ChannelID, workspaceID); err == nil {
		channelName = ch.Name
	}

	h.Audit.LogAction(r.Context(), "create", "notification_rule", rule.ID, fmt.Sprintf("created notification rule for channel %q (severity: %s)", channelName, req.Severity))

	respondJSON(w, http.StatusCreated, response.NotificationRuleFromEntity(rule), nil)
}

// DeleteNotificationRule handles DELETE /api/workspaces/{workspaceId}/notification-rules/{id}.
func (h *NotificationHandlers) DeleteNotificationRule(w http.ResponseWriter, r *http.Request) {
	workspaceID := rbac.WorkspaceIDFromContext(r.Context())
	if workspaceID == "" {
		respondAppError(w, BadRequest("workspace context required"))
		return
	}

	id, ok := parseUUID(r, "id")
	if !ok {
		respondAppError(w, BadRequest("invalid rule ID"))
		return
	}

	if err := h.Notifications.DeleteRule(id, workspaceID); err != nil {
		respondAppError(w, Internal("failed to delete notification rule"))
		return
	}

	h.Audit.LogAction(r.Context(), "delete", "notification_rule", id, fmt.Sprintf("deleted notification rule %s", id))

	respondJSON(w, http.StatusOK, nil, nil)
}

// ListUserNotifications handles GET /api/notifications - lists the current
// user's notifications scoped to the workspace from the authenticated context.
func (h *NotificationHandlers) ListUserNotifications(w http.ResponseWriter, r *http.Request) {
	userID := rbac.UserIDFromContext(r.Context())
	if userID == "" {
		respondAppError(w, Unauthorized("authentication required"))
		return
	}

	workspaceID := rbac.WorkspaceIDFromContext(r.Context())
	if workspaceID == "" {
		respondAppError(w, BadRequest("workspace context required"))
		return
	}

	onlyUnread := r.URL.Query().Get("unread") == "true"

	notifications, err := h.Notifications.ListNotifications(workspaceID, userID, onlyUnread)
	if err != nil {
		respondAppError(w, Internal("failed to list notifications"))
		return
	}

	respondJSON(w, http.StatusOK, response.NotificationsFromEntities(notifications), nil)
}

// GetUnreadCount handles GET /api/notifications/unread-count - returns the
// count of unread notifications for the current user.
func (h *NotificationHandlers) GetUnreadCount(w http.ResponseWriter, r *http.Request) {
	userID := rbac.UserIDFromContext(r.Context())
	if userID == "" {
		respondAppError(w, Unauthorized("authentication required"))
		return
	}

	workspaceID := rbac.WorkspaceIDFromContext(r.Context())
	if workspaceID == "" {
		respondAppError(w, BadRequest("workspace context required"))
		return
	}

	count, err := h.Notifications.GetUnreadCount(workspaceID, userID)
	if err != nil {
		respondAppError(w, Internal("failed to get unread count"))
		return
	}

	respondJSON(w, http.StatusOK, map[string]int64{"count": count}, nil)
}

// MarkNotificationRead handles PUT /api/notifications/{id}/read - marks a
// notification as read.
func (h *NotificationHandlers) MarkNotificationRead(w http.ResponseWriter, r *http.Request) {
	userID := rbac.UserIDFromContext(r.Context())
	if userID == "" {
		respondAppError(w, Unauthorized("authentication required"))
		return
	}

	id, ok := parseUUID(r, "id")
	if !ok {
		respondAppError(w, BadRequest("invalid notification ID"))
		return
	}

	if err := h.Notifications.MarkRead(id, userID); err != nil {
		respondAppError(w, Internal("failed to mark notification as read"))
		return
	}

	respondJSON(w, http.StatusOK, nil, nil)
}

// MarkAllNotificationsRead handles PUT /api/notifications/read-all - marks all
// unread notifications as read for the current user in the current workspace.
func (h *NotificationHandlers) MarkAllNotificationsRead(w http.ResponseWriter, r *http.Request) {
	userID := rbac.UserIDFromContext(r.Context())
	if userID == "" {
		respondAppError(w, Unauthorized("authentication required"))
		return
	}

	workspaceID := rbac.WorkspaceIDFromContext(r.Context())
	if workspaceID == "" {
		respondAppError(w, BadRequest("workspace context required"))
		return
	}

	affected, err := h.Notifications.MarkAllRead(workspaceID, userID)
	if err != nil {
		respondAppError(w, Internal("failed to mark all notifications as read"))
		return
	}

	respondJSON(w, http.StatusOK, map[string]int64{"updated": affected}, nil)
}

// TestNotificationChannel handles POST /api/workspaces/{workspaceId}/notification-channels/{id}/test.
// Sends a test payload to verify the channel works.
func (h *NotificationHandlers) TestNotificationChannel(w http.ResponseWriter, r *http.Request) {
	workspaceID := rbac.WorkspaceIDFromContext(r.Context())
	if workspaceID == "" {
		respondAppError(w, BadRequest("workspace context required"))
		return
	}

	id, ok := parseUUID(r, "id")
	if !ok {
		respondAppError(w, BadRequest("invalid channel ID"))
		return
	}

	if err := h.Notifications.TestChannel(id, workspaceID); err != nil {
		slog.Error("notification channel test failed", "channel_id", id, "error", err)
		if strings.Contains(err.Error(), "not found") {
			respondAppError(w, NotFound("notification channel not found"))
		} else {
			respondAppError(w, BadGateway("failed to send test notification"))
		}
		return
	}

	// Resolve channel name for audit log readability.
	testChannelName := id
	if ch, err := h.Notifications.GetChannel(id, workspaceID); err == nil {
		testChannelName = ch.Name
	}

	h.Audit.LogAction(r.Context(), "test", "notification_channel", id, fmt.Sprintf("sent test notification to channel %q", testChannelName))

	respondJSON(w, http.StatusOK, map[string]string{"message": "test notification sent"}, nil)
}

// DeleteNotification handles DELETE /api/notifications/{id} - deletes a single
// notification for the current user.
func (h *NotificationHandlers) DeleteNotification(w http.ResponseWriter, r *http.Request) {
	userID := rbac.UserIDFromContext(r.Context())
	if userID == "" {
		respondAppError(w, Unauthorized("authentication required"))
		return
	}

	id, ok := parseUUID(r, "id")
	if !ok {
		respondAppError(w, BadRequest("invalid notification ID"))
		return
	}

	workspaceID := rbac.WorkspaceIDFromContext(r.Context())
	if workspaceID == "" {
		respondAppError(w, BadRequest("workspace context required"))
		return
	}

	// Use workspace from authenticated context
	if _, err := h.Notifications.DeleteByID(r.Context(), id, workspaceID, userID); err != nil {
		respondAppError(w, NotFound("notification not found"))
		return
	}

	h.Audit.LogAction(r.Context(), "delete", "notification", id, fmt.Sprintf("deleted notification %s", id))

	w.WriteHeader(http.StatusNoContent)
}

// DeleteAllNotifications handles DELETE /api/notifications/all - deletes all
// notifications for the current user.
func (h *NotificationHandlers) DeleteAllNotifications(w http.ResponseWriter, r *http.Request) {
	userID := rbac.UserIDFromContext(r.Context())
	if userID == "" {
		respondAppError(w, Unauthorized("authentication required"))
		return
	}

	workspaceID := rbac.WorkspaceIDFromContext(r.Context())
	if workspaceID == "" {
		respondAppError(w, BadRequest("workspace context required"))
		return
	}

	affected, err := h.Notifications.DeleteAll(r.Context(), workspaceID, userID)
	if err != nil {
		respondAppError(w, Internal("failed to delete notifications"))
		return
	}

	h.Audit.LogAction(r.Context(), "delete", "notification", "", fmt.Sprintf("deleted all notifications (%d)", affected))

	w.WriteHeader(http.StatusNoContent)
}

// DeleteBatchNotifications handles DELETE /api/notifications - deletes
// multiple notifications by IDs for the current user.
func (h *NotificationHandlers) DeleteBatchNotifications(w http.ResponseWriter, r *http.Request) {
	userID := rbac.UserIDFromContext(r.Context())
	if userID == "" {
		respondAppError(w, Unauthorized("authentication required"))
		return
	}

	var req batchDeleteNotificationsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondAppError(w, BadRequest("invalid request body"))
		return
	}

	if len(req.IDs) == 0 {
		respondAppError(w, Validation("ids is required and must not be empty"))
		return
	}
	for _, id := range req.IDs {
		if _, err := uuid.Parse(id); err != nil {
			respondAppError(w, BadRequest(fmt.Sprintf("invalid notification ID: %s", id)))
			return
		}
	}

	workspaceID := rbac.WorkspaceIDFromContext(r.Context())
	if workspaceID == "" {
		respondAppError(w, BadRequest("workspace context required"))
		return
	}

	if _, err := h.Notifications.DeleteBatch(r.Context(), req.IDs, workspaceID, userID); err != nil {
		slog.Error("failed to batch delete notifications", "error", err)
		respondAppError(w, BadRequest("failed to delete notifications"))
		return
	}

	h.Audit.LogAction(r.Context(), "delete", "notification", "", fmt.Sprintf("batch deleted notifications (requested %d)", len(req.IDs)))

	w.WriteHeader(http.StatusNoContent)
}
