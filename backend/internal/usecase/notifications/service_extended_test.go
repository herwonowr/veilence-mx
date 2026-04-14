package notifications_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/veilence/veilence-mx/backend/internal/entity"
	"github.com/veilence/veilence-mx/backend/internal/repo/persistent"
	"github.com/veilence/veilence-mx/backend/internal/usecase/notifications"
)

// =====================================================================
// MarkAllRead
// =====================================================================

func TestMarkAllRead_Success(t *testing.T) {
	db := setupTestDB(t)
	svc := newService(db)

	// Create channel + in-app notifications
	ch := createTestChannel(t, svc, 1, "email-ch", entity.NotificationChannelEmail, "{}")

	// Seed multiple unread notifications
	seedNotification(t, db, 1, 10, ch.ID, "Alert 1", "body 1", false)
	seedNotification(t, db, 1, 10, ch.ID, "Alert 2", "body 2", false)
	seedNotification(t, db, 1, 10, ch.ID, "Alert 3", "body 3", false)

	// Also seed a read notification (should not be affected)
	seedNotification(t, db, 1, 10, ch.ID, "Already Read", "body read", true)

	affected, err := svc.MarkAllRead(1, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(3), affected, "should mark 3 unread as read")

	// Verify unread count is now 0
	count, err := svc.GetUnreadCount(1, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)
}

func TestMarkAllRead_NoUnread(t *testing.T) {
	db := setupTestDB(t)
	svc := newService(db)

	ch := createTestChannel(t, svc, 1, "email-ch-2", entity.NotificationChannelEmail, "{}")
	seedNotification(t, db, 1, 10, ch.ID, "Already Read", "body", true)

	affected, err := svc.MarkAllRead(1, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(0), affected, "no unread to mark")
}

func TestMarkAllRead_OrgScoped(t *testing.T) {
	db := setupTestDB(t)
	svc := newService(db)

	ch1 := createTestChannel(t, svc, 1, "org1-ch", entity.NotificationChannelEmail, "{}")
	ch2 := createTestChannel(t, svc, 2, "org2-ch", entity.NotificationChannelEmail, "{}")

	// Seed notifications for user 10 in org 1 and org 2
	seedNotification(t, db, 1, 10, ch1.ID, "Org1 Alert", "body", false)
	seedNotification(t, db, 2, 10, ch2.ID, "Org2 Alert", "body", false)

	// Mark all read only for org 1
	affected, err := svc.MarkAllRead(1, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(1), affected, "should only mark org 1 notifications")

	// Org 2 should still have unread
	count, err := svc.GetUnreadCount(2, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count, "org 2 notifications should remain unread")
}

// =====================================================================
// SMTPConfig.IsConfigured
// =====================================================================

func TestSMTPConfig_IsConfigured(t *testing.T) {
	tests := []struct {
		name       string
		host       string
		port       string
		from       string
		configured bool
	}{
		{"fully configured", "smtp.example.com", "587", "noreply@example.com", true},
		{"missing host", "", "587", "noreply@example.com", false},
		{"missing port", "smtp.example.com", "", "noreply@example.com", false},
		{"missing from", "smtp.example.com", "587", "", false},
		{"all empty", "", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := notifications.SMTPConfig{
				Host: tt.host,
				Port: tt.port,
				From: tt.from,
			}
			assert.Equal(t, tt.configured, cfg.IsConfigured())
		})
	}
}

// =====================================================================
// ListChannels — empty org
// =====================================================================

func TestListChannels_EmptyOrg(t *testing.T) {
	db := setupTestDB(t)
	svc := newService(db)

	channels, err := svc.ListChannels(99999)
	require.NoError(t, err)
	assert.Empty(t, channels)
}

// =====================================================================
// ListRules — empty org
// =====================================================================

func TestListRules_EmptyOrg(t *testing.T) {
	db := setupTestDB(t)
	svc := newService(db)

	rules, err := svc.ListRules(99999)
	require.NoError(t, err)
	assert.Empty(t, rules)
}

// =====================================================================
// Dispatch — email channel (SMTP not configured)
// =====================================================================

func TestDispatch_EmailChannel_SMTPNotConfigured(t *testing.T) {
	db := setupTestDB(t)
	// newService creates a service with empty SMTPConfig (not configured)
	svc := newService(db)

	config := `{"recipients":"test@example.com"}`
	ch := createTestChannel(t, svc, 1, "Email Ch", entity.NotificationChannelEmail, config)
	createTestRule(t, svc, 1, ch.ID, "low")

	// Dispatch — should create notification record but skip email (SMTP not configured)
	svc.Dispatch(1, "critical", "Email Test", "Test body")

	// Notification record should still be created
	var notifs []persistent.Notification
	db.Find(&notifs)
	assert.Len(t, notifs, 1, "notification record should exist even without SMTP")
}

func TestDispatch_EmailChannel_InvalidConfig(t *testing.T) {
	db := setupTestDB(t)
	svc := newService(db)

	// Config is not valid JSON
	ch := createTestChannel(t, svc, 1, "Bad Email", entity.NotificationChannelEmail, "not-json")
	createTestRule(t, svc, 1, ch.ID, "low")

	// Should not panic, just log error
	svc.Dispatch(1, "critical", "Bad Config", "Invalid config")

	var notifs []persistent.Notification
	db.Find(&notifs)
	assert.Len(t, notifs, 1)
}

func TestDispatch_EmailChannel_EmptyRecipients(t *testing.T) {
	db := setupTestDB(t)
	svc := newService(db)

	config := `{"recipients":""}`
	ch := createTestChannel(t, svc, 1, "Empty Recip", entity.NotificationChannelEmail, config)
	createTestRule(t, svc, 1, ch.ID, "low")

	// Should not panic
	svc.Dispatch(1, "critical", "Empty Recipients", "No recipients")

	var notifs []persistent.Notification
	db.Find(&notifs)
	assert.Len(t, notifs, 1)
}

// =====================================================================
// Dispatch — email channel with configured SMTP (unreachable server)
// =====================================================================

func TestDispatch_EmailChannel_SMTPConfiguredButUnreachable(t *testing.T) {
	db := setupTestDB(t)
	channelRepo := persistent.NewNotificationChannelRepo(db)
	ruleRepo := persistent.NewNotificationRuleRepo(db)
	notifRepo := persistent.NewNotificationRepo(db)

	// Create service with SMTP configured but pointed to unreachable address
	svc := notifications.NewService(channelRepo, ruleRepo, notifRepo, notifications.SMTPConfig{
		Host: "127.0.0.1",
		Port: "1",
		From: "noreply@test.example.com",
	})
	svc.AllowLocalURLs = true // Tests use localhost

	config := `{"recipients":"test@example.com"}`
	ch := createTestChannel(t, svc, 1, "SMTP Email", entity.NotificationChannelEmail, config)
	createTestRule(t, svc, 1, ch.ID, "low")

	// Should not panic, just log error about unreachable SMTP server
	svc.Dispatch(1, "critical", "SMTP Unreachable", "Testing unreachable SMTP")

	var notifs []persistent.Notification
	db.Find(&notifs)
	assert.Len(t, notifs, 1)
}

func TestDispatch_EmailChannel_MultipleRecipients(t *testing.T) {
	db := setupTestDB(t)
	channelRepo := persistent.NewNotificationChannelRepo(db)
	ruleRepo := persistent.NewNotificationRuleRepo(db)
	notifRepo := persistent.NewNotificationRepo(db)

	svc := notifications.NewService(channelRepo, ruleRepo, notifRepo, notifications.SMTPConfig{
		Host: "127.0.0.1",
		Port: "1",
		From: "noreply@test.example.com",
	})
	svc.AllowLocalURLs = true // Tests use localhost

	config := `{"recipients":"a@test.com, b@test.com, c@test.com"}`
	ch := createTestChannel(t, svc, 1, "Multi Recip", entity.NotificationChannelEmail, config)
	createTestRule(t, svc, 1, ch.ID, "low")

	// Should not panic — will fail on SMTP connect but exercises the email path
	svc.Dispatch(1, "critical", "Multi Recipients", "Testing multiple recipients")

	var notifs []persistent.Notification
	db.Find(&notifs)
	assert.Len(t, notifs, 1)
}

// =====================================================================
// Dispatch — Slack channel
// =====================================================================

func TestDispatch_SlackChannel_Success(t *testing.T) {
	db := setupTestDB(t)
	svc := newService(db)

	var receivedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := fmt.Sprintf(`{"webhookUrl":"%s"}`, server.URL)
	ch := createTestChannel(t, svc, 1, "Slack Ch", entity.NotificationChannelSlack, config)
	createTestRule(t, svc, 1, ch.ID, "low")

	svc.Dispatch(1, "high", "Slack Alert", "Something happened")

	// Should have received the Slack payload
	assert.NotEmpty(t, receivedBody)

	var payload map[string]string
	require.NoError(t, json.Unmarshal(receivedBody, &payload))
	assert.Contains(t, payload["text"], "Slack Alert")
}

func TestDispatch_SlackChannel_InvalidConfig(t *testing.T) {
	db := setupTestDB(t)
	svc := newService(db)

	ch := createTestChannel(t, svc, 1, "Bad Slack", entity.NotificationChannelSlack, "not-json")
	createTestRule(t, svc, 1, ch.ID, "low")

	// Should not panic
	svc.Dispatch(1, "critical", "Bad Slack Config", "Invalid")

	var notifs []persistent.Notification
	db.Find(&notifs)
	assert.Len(t, notifs, 1)
}

func TestDispatch_SlackChannel_EmptyURL(t *testing.T) {
	db := setupTestDB(t)
	svc := newService(db)

	ch := createTestChannel(t, svc, 1, "Empty Slack", entity.NotificationChannelSlack, `{"webhookUrl":""}`)
	createTestRule(t, svc, 1, ch.ID, "low")

	// Should not panic
	svc.Dispatch(1, "critical", "Empty Slack URL", "No URL")

	var notifs []persistent.Notification
	db.Find(&notifs)
	assert.Len(t, notifs, 1)
}

func TestDispatch_SlackChannel_ServerError(t *testing.T) {
	db := setupTestDB(t)
	svc := newService(db)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	config := fmt.Sprintf(`{"webhookUrl":"%s"}`, server.URL)
	ch := createTestChannel(t, svc, 1, "Error Slack", entity.NotificationChannelSlack, config)
	createTestRule(t, svc, 1, ch.ID, "low")

	// Should not panic — logs the error
	svc.Dispatch(1, "critical", "Slack Error", "Server returns 500")

	var notifs []persistent.Notification
	db.Find(&notifs)
	assert.Len(t, notifs, 1)
}

func TestDispatch_SlackChannel_UnreachableURL(t *testing.T) {
	db := setupTestDB(t)
	svc := newService(db)

	config := `{"webhookUrl":"http://127.0.0.1:1"}`
	ch := createTestChannel(t, svc, 1, "Dead Slack", entity.NotificationChannelSlack, config)
	createTestRule(t, svc, 1, ch.ID, "low")

	// Should not panic
	svc.Dispatch(1, "critical", "Slack Unreachable", "URL is dead")

	var notifs []persistent.Notification
	db.Find(&notifs)
	assert.Len(t, notifs, 1)
}

// =====================================================================
// Dispatch — unknown channel type
// =====================================================================

func TestDispatch_UnknownChannelType(t *testing.T) {
	db := setupTestDB(t)
	svc := newService(db)

	// Create channel with unknown type directly in DB
	ch := &persistent.NotificationChannel{
		OrgID:    1,
		Name:     "Unknown Type",
		Type:     "carrier_pigeon",
		Config:   "{}",
		IsActive: true,
	}
	require.NoError(t, db.Create(ch).Error)

	// Create rule linked to this channel
	rule := &persistent.NotificationRule{
		OrgID:     1,
		ChannelID: ch.ID,
		Severity:  "low",
		IsActive:  true,
	}
	require.NoError(t, db.Create(rule).Error)

	// Should not panic — logs a warning
	svc.Dispatch(1, "critical", "Unknown Type", "Carrier pigeon channel")

	var notifs []persistent.Notification
	db.Find(&notifs)
	assert.Len(t, notifs, 1)
}
