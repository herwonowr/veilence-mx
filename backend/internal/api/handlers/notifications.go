package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/veilence/veilence-mx/backend/internal/api/validation"
	"github.com/veilence/veilence-mx/backend/internal/apperror"
	"github.com/veilence/veilence-mx/backend/internal/domain"
	"github.com/veilence/veilence-mx/backend/internal/rbac"
)

// createChannelRequest is the request body for creating a notification channel.
type createChannelRequest struct {
	Name   string                          `json:"name"`
	Type   domain.NotificationChannelType  `json:"type"`
	Config string                          `json:"config"`
}

// updateChannelRequest is the request body for updating a notification channel.
type updateChannelRequest struct {
	Name     string `json:"name"`
	Config   string `json:"config"`
	IsActive bool   `json:"isActive"`
}

// createRuleRequest is the request body for creating a notification rule.
type createRuleRequest struct {
	ChannelID uint   `json:"channelId"`
	Severity  string `json:"severity"`
}

// ListNotificationChannels handles GET /api/orgs/{orgId}/notification-channels.
func (h *NotificationHandlers) ListNotificationChannels(w http.ResponseWriter, r *http.Request) {
	orgID := rbac.OrgIDFromContext(r.Context())
	if orgID == 0 {
		respondError(w, http.StatusBadRequest, "organization context required")
		return
	}

	channels, err := h.Notifications.ListChannels(orgID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list notification channels")
		return
	}

	respondJSON(w, http.StatusOK, channels, nil)
}

// CreateNotificationChannel handles POST /api/orgs/{orgId}/notification-channels.
func (h *NotificationHandlers) CreateNotificationChannel(w http.ResponseWriter, r *http.Request) {
	orgID := rbac.OrgIDFromContext(r.Context())
	if orgID == 0 {
		respondError(w, http.StatusBadRequest, "organization context required")
		return
	}

	var req createChannelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := validation.ValidateRequired(req.Name, "name"); err != nil {
		respondAppError(w, apperror.Validation(err.Error()))
		return
	}
	if err := validation.ValidateMaxLength(req.Name, "name", 100); err != nil {
		respondAppError(w, apperror.Validation(err.Error()))
		return
	}

	validTypes := map[domain.NotificationChannelType]bool{
		domain.NotificationChannelEmail:   true,
		domain.NotificationChannelSlack:   true,
		domain.NotificationChannelWebhook: true,
	}
	if !validTypes[req.Type] {
		respondAppError(w, apperror.Validation("type must be 'email', 'slack', or 'webhook'"))
		return
	}

	channel, err := h.Notifications.CreateChannel(orgID, req.Name, req.Type, req.Config)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to create notification channel")
		return
	}

	h.Audit.LogAction(r.Context(), "create", "notification_channel", channel.ID, fmt.Sprintf("created %s notification channel %q", req.Type, req.Name))

	respondJSON(w, http.StatusCreated, channel, nil)
}

// UpdateNotificationChannel handles PUT /api/orgs/{orgId}/notification-channels/{id}.
func (h *NotificationHandlers) UpdateNotificationChannel(w http.ResponseWriter, r *http.Request) {
	orgID := rbac.OrgIDFromContext(r.Context())
	if orgID == 0 {
		respondError(w, http.StatusBadRequest, "organization context required")
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid channel ID")
		return
	}

	var req updateChannelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := validation.ValidateRequired(req.Name, "name"); err != nil {
		respondAppError(w, apperror.Validation(err.Error()))
		return
	}
	if err := validation.ValidateMaxLength(req.Name, "name", 100); err != nil {
		respondAppError(w, apperror.Validation(err.Error()))
		return
	}

	channel, err := h.Notifications.UpdateChannel(uint(id), orgID, req.Name, req.Config, req.IsActive)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to update notification channel")
		return
	}

	h.Audit.LogAction(r.Context(), "update", "notification_channel", uint(id), fmt.Sprintf("updated notification channel %q (active: %t)", req.Name, req.IsActive))

	respondJSON(w, http.StatusOK, channel, nil)
}

// DeleteNotificationChannel handles DELETE /api/orgs/{orgId}/notification-channels/{id}.
func (h *NotificationHandlers) DeleteNotificationChannel(w http.ResponseWriter, r *http.Request) {
	orgID := rbac.OrgIDFromContext(r.Context())
	if orgID == 0 {
		respondError(w, http.StatusBadRequest, "organization context required")
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid channel ID")
		return
	}

	if err := h.Notifications.DeleteChannel(uint(id), orgID); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to delete notification channel")
		return
	}

	h.Audit.LogAction(r.Context(), "delete", "notification_channel", uint(id), fmt.Sprintf("deleted notification channel %d", id))

	respondJSON(w, http.StatusOK, nil, nil)
}

// ListNotificationRules handles GET /api/orgs/{orgId}/notification-rules.
func (h *NotificationHandlers) ListNotificationRules(w http.ResponseWriter, r *http.Request) {
	orgID := rbac.OrgIDFromContext(r.Context())
	if orgID == 0 {
		respondError(w, http.StatusBadRequest, "organization context required")
		return
	}

	rules, err := h.Notifications.ListRules(orgID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list notification rules")
		return
	}

	respondJSON(w, http.StatusOK, rules, nil)
}

// CreateNotificationRule handles POST /api/orgs/{orgId}/notification-rules.
func (h *NotificationHandlers) CreateNotificationRule(w http.ResponseWriter, r *http.Request) {
	orgID := rbac.OrgIDFromContext(r.Context())
	if orgID == 0 {
		respondError(w, http.StatusBadRequest, "organization context required")
		return
	}

	var req createRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.ChannelID == 0 {
		respondAppError(w, apperror.Validation("channelId is required"))
		return
	}

	validSeverities := map[string]bool{
		"low": true, "medium": true, "high": true, "critical": true,
	}
	if req.Severity != "" && !validSeverities[req.Severity] {
		respondAppError(w, apperror.Validation("severity must be 'low', 'medium', 'high', or 'critical'"))
		return
	}

	rule, err := h.Notifications.CreateRule(orgID, req.ChannelID, req.Severity)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to create notification rule")
		return
	}

	h.Audit.LogAction(r.Context(), "create", "notification_rule", rule.ID, fmt.Sprintf("created notification rule for channel %d (severity: %s)", req.ChannelID, req.Severity))

	respondJSON(w, http.StatusCreated, rule, nil)
}

// DeleteNotificationRule handles DELETE /api/orgs/{orgId}/notification-rules/{id}.
func (h *NotificationHandlers) DeleteNotificationRule(w http.ResponseWriter, r *http.Request) {
	orgID := rbac.OrgIDFromContext(r.Context())
	if orgID == 0 {
		respondError(w, http.StatusBadRequest, "organization context required")
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid rule ID")
		return
	}

	if err := h.Notifications.DeleteRule(uint(id), orgID); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to delete notification rule")
		return
	}

	h.Audit.LogAction(r.Context(), "delete", "notification_rule", uint(id), fmt.Sprintf("deleted notification rule %d", id))

	respondJSON(w, http.StatusOK, nil, nil)
}

// ListUserNotifications handles GET /api/notifications — lists the current
// user's notifications across all organizations.
func (h *NotificationHandlers) ListUserNotifications(w http.ResponseWriter, r *http.Request) {
	userID := rbac.UserIDFromContext(r.Context())
	if userID == 0 {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	onlyUnread := r.URL.Query().Get("unread") == "true"

	notifications, err := h.Notifications.ListNotifications(0, userID, onlyUnread)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list notifications")
		return
	}

	respondJSON(w, http.StatusOK, notifications, nil)
}

// GetUnreadCount handles GET /api/notifications/unread-count — returns the
// count of unread notifications for the current user.
func (h *NotificationHandlers) GetUnreadCount(w http.ResponseWriter, r *http.Request) {
	userID := rbac.UserIDFromContext(r.Context())
	if userID == 0 {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	count, err := h.Notifications.GetUnreadCount(0, userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to get unread count")
		return
	}

	respondJSON(w, http.StatusOK, map[string]int64{"count": count}, nil)
}

// MarkNotificationRead handles PUT /api/notifications/{id}/read — marks a
// notification as read.
func (h *NotificationHandlers) MarkNotificationRead(w http.ResponseWriter, r *http.Request) {
	userID := rbac.UserIDFromContext(r.Context())
	if userID == 0 {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid notification ID")
		return
	}

	if err := h.Notifications.MarkRead(uint(id), userID); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to mark notification as read")
		return
	}

	respondJSON(w, http.StatusOK, nil, nil)
}
