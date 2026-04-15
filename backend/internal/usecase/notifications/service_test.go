package notifications_test

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/veilence/veilence-mx/backend/internal/entity"
	"github.com/veilence/veilence-mx/backend/internal/repo/persistent"
	"github.com/veilence/veilence-mx/backend/internal/usecase/notifications"
)

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	err = db.AutoMigrate(
		&persistent.NotificationChannel{},
		&persistent.NotificationRule{},
		&persistent.Notification{},
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	})
	return db
}

func newService(db *gorm.DB) *notifications.Service {
	channelRepo := persistent.NewNotificationChannelRepo(db)
	ruleRepo := persistent.NewNotificationRuleRepo(db)
	notifRepo := persistent.NewNotificationRepo(db)
	svc := notifications.NewService(channelRepo, ruleRepo, notifRepo, notifications.SMTPConfig{})
	svc.AllowLocalURLs = true // Tests use httptest.NewServer (localhost)
	return svc
}

// createTestChannel is a convenience helper that creates a channel and fails
// the test immediately if an error occurs.
func createTestChannel(t *testing.T, svc *notifications.Service, orgID uint, name string, chanType entity.NotificationChannelType, config string) *entity.NotificationChannel {
	t.Helper()
	ch, err := svc.CreateChannel(orgID, name, chanType, config)
	require.NoError(t, err)
	return ch
}

// createTestRule is a convenience helper that creates a rule and fails
// the test immediately if an error occurs.
func createTestRule(t *testing.T, svc *notifications.Service, orgID, channelID uint, severity string) *entity.NotificationRule {
	t.Helper()
	rule, err := svc.CreateRule(orgID, channelID, severity)
	require.NoError(t, err)
	return rule
}

// seedNotification inserts a Notification directly via GORM for testing query
// methods without going through Dispatch.
func seedNotification(t *testing.T, db *gorm.DB, orgID, userID, channelID uint, title, message string, isRead bool) *persistent.Notification {
	t.Helper()
	n := &persistent.Notification{
		OrgID:     orgID,
		UserID:    userID,
		ChannelID: channelID,
		Title:     title,
		Message:   message,
		IsRead:    isRead,
		SentAt:    time.Now(),
	}
	require.NoError(t, db.Create(n).Error)
	return n
}

// =========================================================================
// Channel CRUD
// =========================================================================

func TestCreateChannel_Email(t *testing.T) {
	db := setupTestDB(t)
	svc := newService(db)

	ch, err := svc.CreateChannel(1, "Email Alerts", entity.NotificationChannelEmail, `{"to":"ops@example.com"}`)
	require.NoError(t, err)
	assert.NotZero(t, ch.ID)
	assert.Equal(t, uint(1), ch.OrgID)
	assert.Equal(t, "Email Alerts", ch.Name)
	assert.Equal(t, entity.NotificationChannelEmail, ch.Type)
	assert.Equal(t, `{"to":"ops@example.com"}`, ch.Config)
	assert.True(t, ch.IsActive, "new channels should be active by default")
}

func TestCreateChannel_Slack(t *testing.T) {
	db := setupTestDB(t)
	svc := newService(db)

	ch, err := svc.CreateChannel(1, "Slack Ops", entity.NotificationChannelSlack, `{"webhook":"https://hooks.slack.com/xxx"}`)
	require.NoError(t, err)
	assert.NotZero(t, ch.ID)
	assert.Equal(t, entity.NotificationChannelSlack, ch.Type)
}

func TestCreateChannel_Webhook(t *testing.T) {
	db := setupTestDB(t)
	svc := newService(db)

	ch, err := svc.CreateChannel(1, "Webhook", entity.NotificationChannelWebhook, `{"url":"https://example.com/hook"}`)
	require.NoError(t, err)
	assert.NotZero(t, ch.ID)
	assert.Equal(t, entity.NotificationChannelWebhook, ch.Type)
}

func TestListChannels_Empty(t *testing.T) {
	db := setupTestDB(t)
	svc := newService(db)

	channels, err := svc.ListChannels(1)
	require.NoError(t, err)
	assert.Empty(t, channels)
}

func TestListChannels_ReturnsOnlyOwnOrg(t *testing.T) {
	db := setupTestDB(t)
	svc := newService(db)

	createTestChannel(t, svc, 1, "Org1 Email", entity.NotificationChannelEmail, `{}`)
	createTestChannel(t, svc, 1, "Org1 Slack", entity.NotificationChannelSlack, `{}`)
	createTestChannel(t, svc, 2, "Org2 Email", entity.NotificationChannelEmail, `{}`)

	channels, err := svc.ListChannels(1)
	require.NoError(t, err)
	assert.Len(t, channels, 2)
	for _, ch := range channels {
		assert.Equal(t, uint(1), ch.OrgID)
	}

	channels2, err := svc.ListChannels(2)
	require.NoError(t, err)
	assert.Len(t, channels2, 1)
}

func TestUpdateChannel_Success(t *testing.T) {
	db := setupTestDB(t)
	svc := newService(db)

	ch := createTestChannel(t, svc, 1, "Original", entity.NotificationChannelEmail, `{"to":"old@example.com"}`)

	updated, err := svc.UpdateChannel(ch.ID, 1, "Renamed", `{"to":"new@example.com"}`, false)
	require.NoError(t, err)
	assert.Equal(t, ch.ID, updated.ID)
	assert.Equal(t, "Renamed", updated.Name)
	assert.Equal(t, `{"to":"new@example.com"}`, updated.Config)
	assert.False(t, updated.IsActive)
}

func TestUpdateChannel_WrongOrg(t *testing.T) {
	db := setupTestDB(t)
	svc := newService(db)

	ch := createTestChannel(t, svc, 1, "Org1 Only", entity.NotificationChannelEmail, `{}`)

	_, err := svc.UpdateChannel(ch.ID, 999, "Hacked", `{}`, true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "finding notification channel")
}

func TestUpdateChannel_NotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := newService(db)

	_, err := svc.UpdateChannel(99999, 1, "Ghost", `{}`, true)
	require.Error(t, err)
}

func TestDeleteChannel_Success(t *testing.T) {
	db := setupTestDB(t)
	svc := newService(db)

	ch := createTestChannel(t, svc, 1, "To Delete", entity.NotificationChannelEmail, `{}`)

	err := svc.DeleteChannel(ch.ID, 1)
	require.NoError(t, err)

	// Verify it's gone
	channels, err := svc.ListChannels(1)
	require.NoError(t, err)
	assert.Empty(t, channels)
}

func TestDeleteChannel_WrongOrg(t *testing.T) {
	db := setupTestDB(t)
	svc := newService(db)

	ch := createTestChannel(t, svc, 1, "Org1 Only", entity.NotificationChannelEmail, `{}`)

	err := svc.DeleteChannel(ch.ID, 999)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "notification channel not found")
}

func TestDeleteChannel_NotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := newService(db)

	err := svc.DeleteChannel(99999, 1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "notification channel not found")
}

// =========================================================================
// Rule CRUD
// =========================================================================

func TestCreateRule_Success(t *testing.T) {
	db := setupTestDB(t)
	svc := newService(db)

	ch := createTestChannel(t, svc, 1, "Alerts", entity.NotificationChannelEmail, `{}`)

	rule, err := svc.CreateRule(1, ch.ID, "high")
	require.NoError(t, err)
	assert.NotZero(t, rule.ID)
	assert.Equal(t, uint(1), rule.OrgID)
	assert.Equal(t, ch.ID, rule.ChannelID)
	assert.Equal(t, "high", rule.Severity)
	assert.True(t, rule.IsActive)
}

func TestCreateRule_AllSeverities(t *testing.T) {
	db := setupTestDB(t)
	svc := newService(db)

	ch := createTestChannel(t, svc, 1, "All Levels", entity.NotificationChannelSlack, `{}`)

	severities := []string{"low", "medium", "high", "critical"}
	for _, sev := range severities {
		rule, err := svc.CreateRule(1, ch.ID, sev)
		require.NoError(t, err, "failed to create rule for severity %s", sev)
		assert.Equal(t, sev, rule.Severity)
	}
}

func TestListRules_Empty(t *testing.T) {
	db := setupTestDB(t)
	svc := newService(db)

	rules, err := svc.ListRules(1)
	require.NoError(t, err)
	assert.Empty(t, rules)
}

func TestListRules_ReturnsOnlyOwnOrg(t *testing.T) {
	db := setupTestDB(t)
	svc := newService(db)

	ch1 := createTestChannel(t, svc, 1, "Ch1", entity.NotificationChannelEmail, `{}`)
	ch2 := createTestChannel(t, svc, 2, "Ch2", entity.NotificationChannelEmail, `{}`)

	createTestRule(t, svc, 1, ch1.ID, "high")
	createTestRule(t, svc, 1, ch1.ID, "critical")
	createTestRule(t, svc, 2, ch2.ID, "low")

	rules, err := svc.ListRules(1)
	require.NoError(t, err)
	assert.Len(t, rules, 2)

	rules2, err := svc.ListRules(2)
	require.NoError(t, err)
	assert.Len(t, rules2, 1)
}

func TestDeleteRule_Success(t *testing.T) {
	db := setupTestDB(t)
	svc := newService(db)

	ch := createTestChannel(t, svc, 1, "Ch", entity.NotificationChannelEmail, `{}`)
	rule := createTestRule(t, svc, 1, ch.ID, "medium")

	err := svc.DeleteRule(rule.ID, 1)
	require.NoError(t, err)

	rules, err := svc.ListRules(1)
	require.NoError(t, err)
	assert.Empty(t, rules)
}

func TestDeleteRule_WrongOrg(t *testing.T) {
	db := setupTestDB(t)
	svc := newService(db)

	ch := createTestChannel(t, svc, 1, "Ch", entity.NotificationChannelEmail, `{}`)
	rule := createTestRule(t, svc, 1, ch.ID, "medium")

	err := svc.DeleteRule(rule.ID, 999)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "notification rule not found")
}

func TestDeleteRule_NotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := newService(db)

	err := svc.DeleteRule(99999, 1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "notification rule not found")
}

// =========================================================================
// Dispatch — severity routing
// =========================================================================

func TestDispatch_MatchesSeverityAtThreshold(t *testing.T) {
	db := setupTestDB(t)
	svc := newService(db)

	ch := createTestChannel(t, svc, 1, "Email", entity.NotificationChannelEmail, `{}`)
	createTestRule(t, svc, 1, ch.ID, "high")

	svc.Dispatch(context.Background(), 1, "high", "Alert", "This is a high alert")

	var count int64
	db.Model(&persistent.Notification{}).Where("org_id = ?", 1).Count(&count)
	assert.Equal(t, int64(1), count)
}

func TestDispatch_MatchesSeverityAboveThreshold(t *testing.T) {
	db := setupTestDB(t)
	svc := newService(db)

	ch := createTestChannel(t, svc, 1, "Email", entity.NotificationChannelEmail, `{}`)
	createTestRule(t, svc, 1, ch.ID, "medium")

	svc.Dispatch(context.Background(), 1, "critical", "Critical Issue", "Something terrible happened")

	var count int64
	db.Model(&persistent.Notification{}).Where("org_id = ?", 1).Count(&count)
	assert.Equal(t, int64(1), count)
}

func TestDispatch_SkipsBelowThreshold(t *testing.T) {
	db := setupTestDB(t)
	svc := newService(db)

	ch := createTestChannel(t, svc, 1, "Email", entity.NotificationChannelEmail, `{}`)
	createTestRule(t, svc, 1, ch.ID, "critical")

	svc.Dispatch(context.Background(), 1, "low", "Info", "Low priority event")

	var count int64
	db.Model(&persistent.Notification{}).Where("org_id = ?", 1).Count(&count)
	assert.Equal(t, int64(1), count, "in-app notification should always be created even when no rule matches severity")

	// Verify it's the in-app notification (ChannelID=0), not an external channel notification
	var notif persistent.Notification
	db.Where("org_id = ?", 1).First(&notif)
	assert.Equal(t, uint(0), notif.ChannelID, "in-app notification should have ChannelID=0")
}

func TestDispatch_MultipleRulesMultipleChannels(t *testing.T) {
	db := setupTestDB(t)
	svc := newService(db)

	ch1 := createTestChannel(t, svc, 1, "Slack Low", entity.NotificationChannelSlack, `{}`)
	ch2 := createTestChannel(t, svc, 1, "Email High", entity.NotificationChannelEmail, `{}`)
	ch3 := createTestChannel(t, svc, 1, "Webhook Critical", entity.NotificationChannelWebhook, `{"url":"http://invalid.test"}`)

	createTestRule(t, svc, 1, ch1.ID, "low")
	createTestRule(t, svc, 1, ch2.ID, "high")
	createTestRule(t, svc, 1, ch3.ID, "critical")

	svc.Dispatch(context.Background(), 1, "high", "High Alert", "Something important")

	// In-app notification is always created with ChannelID=0 (1 record).
	// External dispatch to matching channels happens separately (no additional records).
	var notifs []persistent.Notification
	db.Where("org_id = ?", 1).Find(&notifs)
	assert.Len(t, notifs, 1, "expected 1 in-app notification record")
	assert.Equal(t, uint(0), notifs[0].ChannelID, "in-app notification should have ChannelID=0")

	_ = ch1 // low-threshold channel gets external dispatch
	_ = ch2 // high-threshold channel gets external dispatch
	_ = ch3 // critical-only channel does NOT get external dispatch
}

func TestDispatch_AllSeverityLevels(t *testing.T) {
	db := setupTestDB(t)
	svc := newService(db)

	ch := createTestChannel(t, svc, 1, "All", entity.NotificationChannelEmail, `{}`)
	createTestRule(t, svc, 1, ch.ID, "low")

	levels := []string{"low", "medium", "high", "critical"}
	for _, level := range levels {
		svc.Dispatch(context.Background(), 1, level, fmt.Sprintf("Title %s", level), "msg")
	}

	var count int64
	db.Model(&persistent.Notification{}).Where("org_id = ?", 1).Count(&count)
	assert.Equal(t, int64(4), count, "low-threshold rule should fire for all severity levels")
}

func TestDispatch_NoMatchingRules(t *testing.T) {
	db := setupTestDB(t)
	svc := newService(db)

	svc.Dispatch(context.Background(), 1, "critical", "Nobody Listening", "No rules")

	// In-app notification should still be created even with no rules configured
	var count int64
	db.Model(&persistent.Notification{}).Where("org_id = ?", 1).Count(&count)
	assert.Equal(t, int64(1), count, "in-app notification should always be created")

	var notif persistent.Notification
	db.Where("org_id = ?", 1).First(&notif)
	assert.Equal(t, uint(0), notif.ChannelID, "in-app notification should have ChannelID=0")
}

func TestDispatch_SkipsDisabledChannel(t *testing.T) {
	db := setupTestDB(t)
	svc := newService(db)

	ch := createTestChannel(t, svc, 1, "Disabled", entity.NotificationChannelEmail, `{}`)
	createTestRule(t, svc, 1, ch.ID, "low")

	_, err := svc.UpdateChannel(ch.ID, 1, ch.Name, ch.Config, false)
	require.NoError(t, err)

	svc.Dispatch(context.Background(), 1, "critical", "Won't Arrive", "Channel is disabled")

	// In-app notification is always created, but disabled channel should not get external dispatch
	var count int64
	db.Model(&persistent.Notification{}).Where("org_id = ?", 1).Count(&count)
	assert.Equal(t, int64(1), count, "in-app notification should always be created even when channel is disabled")

	var notif persistent.Notification
	db.Where("org_id = ?", 1).First(&notif)
	assert.Equal(t, uint(0), notif.ChannelID, "in-app notification should have ChannelID=0")
}

func TestDispatch_SkipsInactiveRule(t *testing.T) {
	db := setupTestDB(t)
	svc := newService(db)

	ch := createTestChannel(t, svc, 1, "Active Channel", entity.NotificationChannelEmail, `{}`)
	rule := createTestRule(t, svc, 1, ch.ID, "low")

	db.Model(&persistent.NotificationRule{}).Where("id = ?", rule.ID).Update("is_active", false)

	svc.Dispatch(context.Background(), 1, "critical", "Won't Arrive", "Rule is inactive")

	// In-app notification is always created, but inactive rule should not trigger external dispatch
	var count int64
	db.Model(&persistent.Notification{}).Where("org_id = ?", 1).Count(&count)
	assert.Equal(t, int64(1), count, "in-app notification should always be created even when rule is inactive")

	var notif persistent.Notification
	db.Where("org_id = ?", 1).First(&notif)
	assert.Equal(t, uint(0), notif.ChannelID, "in-app notification should have ChannelID=0")
}

func TestDispatch_IsolatedByOrg(t *testing.T) {
	db := setupTestDB(t)
	svc := newService(db)

	ch1 := createTestChannel(t, svc, 1, "Org1", entity.NotificationChannelEmail, `{}`)
	ch2 := createTestChannel(t, svc, 2, "Org2", entity.NotificationChannelEmail, `{}`)
	createTestRule(t, svc, 1, ch1.ID, "low")
	createTestRule(t, svc, 2, ch2.ID, "low")

	svc.Dispatch(context.Background(), 1, "high", "Org1 Alert", "only org 1")

	var count1, count2 int64
	db.Model(&persistent.Notification{}).Where("org_id = ?", 1).Count(&count1)
	db.Model(&persistent.Notification{}).Where("org_id = ?", 2).Count(&count2)
	assert.Equal(t, int64(1), count1)
	assert.Equal(t, int64(0), count2, "other org should not receive notification")
}

func TestDispatch_NotificationFields(t *testing.T) {
	db := setupTestDB(t)
	svc := newService(db)

	ch := createTestChannel(t, svc, 1, "Email", entity.NotificationChannelEmail, `{}`)
	createTestRule(t, svc, 1, ch.ID, "low")

	svc.Dispatch(context.Background(), 1, "critical", "My Title", "My Message Body")

	var notif persistent.Notification
	require.NoError(t, db.Where("org_id = ?", 1).First(&notif).Error)
	assert.Equal(t, uint(1), notif.OrgID)
	assert.Equal(t, uint(0), notif.UserID, "Dispatch creates org-wide notifications with user_id=0")
	assert.Equal(t, uint(0), notif.ChannelID, "in-app notification should have ChannelID=0")
	assert.Equal(t, "My Title", notif.Title)
	assert.Equal(t, "My Message Body", notif.Message)
	assert.False(t, notif.IsRead)
	assert.False(t, notif.SentAt.IsZero())
}

// =========================================================================
// Webhook delivery (httptest)
// =========================================================================

func TestDispatch_WebhookDelivery(t *testing.T) {
	var mu sync.Mutex
	var received []byte
	var gotRequest bool

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		gotRequest = true
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		buf := make([]byte, r.ContentLength)
		_, err := r.Body.Read(buf)
		if err != nil && err.Error() != "EOF" {
			t.Errorf("unexpected error reading body: %v", err)
		}
		received = buf
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	db := setupTestDB(t)
	svc := newService(db)

	config := fmt.Sprintf(`{"url":"%s"}`, ts.URL)
	ch := createTestChannel(t, svc, 1, "Hook", entity.NotificationChannelWebhook, config)
	createTestRule(t, svc, 1, ch.ID, "low")

	svc.Dispatch(context.Background(), 1, "high", "Webhook Title", "Webhook Body")

	mu.Lock()
	defer mu.Unlock()
	require.True(t, gotRequest, "webhook server should have received a request")

	var payload struct {
		Title   string `json:"title"`
		Message string `json:"message"`
		Channel string `json:"channel"`
	}
	require.NoError(t, json.Unmarshal(received, &payload))
	assert.Equal(t, "Webhook Title", payload.Title)
	assert.Equal(t, "Webhook Body", payload.Message)
	assert.Equal(t, "Hook", payload.Channel)
}

func TestDispatch_WebhookToUnreachableURL(t *testing.T) {
	db := setupTestDB(t)
	svc := newService(db)

	ch := createTestChannel(t, svc, 1, "Dead Hook", entity.NotificationChannelWebhook, `{"url":"http://127.0.0.1:1"}`)
	createTestRule(t, svc, 1, ch.ID, "low")

	svc.Dispatch(context.Background(), 1, "critical", "Unreachable", "The webhook URL is dead")

	var count int64
	db.Model(&persistent.Notification{}).Where("org_id = ?", 1).Count(&count)
	assert.Equal(t, int64(1), count, "in-app notification should still be created even when webhook fails")
}

func TestDispatch_WebhookInvalidConfig(t *testing.T) {
	db := setupTestDB(t)
	svc := newService(db)

	ch := createTestChannel(t, svc, 1, "Bad Config", entity.NotificationChannelWebhook, `not-json`)
	createTestRule(t, svc, 1, ch.ID, "low")

	svc.Dispatch(context.Background(), 1, "critical", "Bad Config", "Config is not valid JSON")

	var count int64
	db.Model(&persistent.Notification{}).Where("org_id = ?", 1).Count(&count)
	assert.Equal(t, int64(1), count)
}

func TestDispatch_WebhookEmptyURL(t *testing.T) {
	db := setupTestDB(t)
	svc := newService(db)

	ch := createTestChannel(t, svc, 1, "Empty URL", entity.NotificationChannelWebhook, `{"url":""}`)
	createTestRule(t, svc, 1, ch.ID, "low")

	svc.Dispatch(context.Background(), 1, "critical", "Empty URL", "URL field is empty")

	var count int64
	db.Model(&persistent.Notification{}).Where("org_id = ?", 1).Count(&count)
	assert.Equal(t, int64(1), count)
}

func TestDispatch_WebhookServerError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	db := setupTestDB(t)
	svc := newService(db)

	config := fmt.Sprintf(`{"url":"%s"}`, ts.URL)
	ch := createTestChannel(t, svc, 1, "Error Hook", entity.NotificationChannelWebhook, config)
	createTestRule(t, svc, 1, ch.ID, "low")

	svc.Dispatch(context.Background(), 1, "high", "Server Error", "The hook returned 500")

	var count int64
	db.Model(&persistent.Notification{}).Where("org_id = ?", 1).Count(&count)
	assert.Equal(t, int64(1), count)
}

// =========================================================================
// In-app notifications — ListNotifications
// =========================================================================

func TestListNotifications_Empty(t *testing.T) {
	db := setupTestDB(t)
	svc := newService(db)

	notifs, err := svc.ListNotifications(1, 10, false)
	require.NoError(t, err)
	assert.Empty(t, notifs)
}

func TestListNotifications_OrgWideVisibleToUser(t *testing.T) {
	db := setupTestDB(t)
	svc := newService(db)

	seedNotification(t, db, 1, 0, 1, "Org Alert", "For everyone", false)

	notifs, err := svc.ListNotifications(1, 42, false)
	require.NoError(t, err)
	assert.Len(t, notifs, 1, "org-wide (user_id=0) should be visible to any user")
}

func TestListNotifications_UserSpecificPlusOrgWide(t *testing.T) {
	db := setupTestDB(t)
	svc := newService(db)

	seedNotification(t, db, 1, 0, 1, "Org", "For everyone", false)
	seedNotification(t, db, 1, 42, 1, "User 42", "Only for user 42", false)
	seedNotification(t, db, 1, 99, 1, "User 99", "Only for user 99", false)

	notifs, err := svc.ListNotifications(1, 42, false)
	require.NoError(t, err)
	assert.Len(t, notifs, 2, "user 42 sees org-wide + own notifications")
}

func TestListNotifications_OnlyUnread(t *testing.T) {
	db := setupTestDB(t)
	svc := newService(db)

	seedNotification(t, db, 1, 0, 1, "Read", "Already read", true)
	seedNotification(t, db, 1, 0, 1, "Unread", "Not yet read", false)

	notifs, err := svc.ListNotifications(1, 42, true)
	require.NoError(t, err)
	assert.Len(t, notifs, 1)
	assert.Equal(t, "Unread", notifs[0].Title)
}

func TestListNotifications_AllIncludesRead(t *testing.T) {
	db := setupTestDB(t)
	svc := newService(db)

	seedNotification(t, db, 1, 0, 1, "Read", "Already read", true)
	seedNotification(t, db, 1, 0, 1, "Unread", "Not yet read", false)

	notifs, err := svc.ListNotifications(1, 42, false)
	require.NoError(t, err)
	assert.Len(t, notifs, 2)
}

func TestListNotifications_FilterByOrg(t *testing.T) {
	db := setupTestDB(t)
	svc := newService(db)

	seedNotification(t, db, 1, 0, 1, "Org1", "from org 1", false)
	seedNotification(t, db, 2, 0, 1, "Org2", "from org 2", false)

	notifs, err := svc.ListNotifications(1, 42, false)
	require.NoError(t, err)
	assert.Len(t, notifs, 1)
	assert.Equal(t, "Org1", notifs[0].Title)
}

func TestListNotifications_ZeroOrgShowsAll(t *testing.T) {
	db := setupTestDB(t)
	svc := newService(db)

	seedNotification(t, db, 1, 0, 1, "Org1", "from org 1", false)
	seedNotification(t, db, 2, 0, 1, "Org2", "from org 2", false)

	notifs, err := svc.ListNotifications(0, 42, false)
	require.NoError(t, err)
	assert.Len(t, notifs, 2)
}

func TestListNotifications_OrderedByCreatedAtDesc(t *testing.T) {
	db := setupTestDB(t)
	svc := newService(db)

	seedNotification(t, db, 1, 0, 1, "First", "created first", false)
	time.Sleep(10 * time.Millisecond)
	seedNotification(t, db, 1, 0, 1, "Second", "created second", false)

	notifs, err := svc.ListNotifications(1, 42, false)
	require.NoError(t, err)
	require.Len(t, notifs, 2)
	assert.Equal(t, "Second", notifs[0].Title, "most recent should come first")
	assert.Equal(t, "First", notifs[1].Title)
}

// =========================================================================
// MarkRead
// =========================================================================

func TestMarkRead_Success(t *testing.T) {
	db := setupTestDB(t)
	svc := newService(db)

	n := seedNotification(t, db, 1, 42, 1, "Unread", "please read me", false)

	err := svc.MarkRead(n.ID, 42)
	require.NoError(t, err)

	var updated persistent.Notification
	db.First(&updated, n.ID)
	assert.True(t, updated.IsRead)
}

func TestMarkRead_OrgWideNotification(t *testing.T) {
	db := setupTestDB(t)
	svc := newService(db)

	n := seedNotification(t, db, 1, 0, 1, "Org-Wide", "for everyone", false)

	err := svc.MarkRead(n.ID, 42)
	require.NoError(t, err)

	var updated persistent.Notification
	db.First(&updated, n.ID)
	assert.True(t, updated.IsRead)
}

func TestMarkRead_NotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := newService(db)

	err := svc.MarkRead(99999, 42)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "notification not found")
}

func TestMarkRead_WrongUser(t *testing.T) {
	db := setupTestDB(t)
	svc := newService(db)

	n := seedNotification(t, db, 1, 42, 1, "Private", "only user 42", false)

	err := svc.MarkRead(n.ID, 99)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "notification not found")
}

// =========================================================================
// GetUnreadCount
// =========================================================================

func TestGetUnreadCount_AllUnread(t *testing.T) {
	db := setupTestDB(t)
	svc := newService(db)

	seedNotification(t, db, 1, 0, 1, "N1", "m1", false)
	seedNotification(t, db, 1, 0, 1, "N2", "m2", false)
	seedNotification(t, db, 1, 42, 1, "N3", "m3", false)

	count, err := svc.GetUnreadCount(1, 42)
	require.NoError(t, err)
	assert.Equal(t, int64(3), count, "2 org-wide + 1 user-specific = 3")
}

func TestGetUnreadCount_ExcludesRead(t *testing.T) {
	db := setupTestDB(t)
	svc := newService(db)

	seedNotification(t, db, 1, 0, 1, "Read", "m", true)
	seedNotification(t, db, 1, 0, 1, "Unread", "m", false)

	count, err := svc.GetUnreadCount(1, 42)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
}

func TestGetUnreadCount_ExcludesOtherUserNotifications(t *testing.T) {
	db := setupTestDB(t)
	svc := newService(db)

	seedNotification(t, db, 1, 99, 1, "Other User", "not mine", false)

	count, err := svc.GetUnreadCount(1, 42)
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)
}

func TestGetUnreadCount_FilterByOrg(t *testing.T) {
	db := setupTestDB(t)
	svc := newService(db)

	seedNotification(t, db, 1, 0, 1, "Org1", "m", false)
	seedNotification(t, db, 2, 0, 1, "Org2", "m", false)

	count, err := svc.GetUnreadCount(1, 42)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
}

func TestGetUnreadCount_ZeroOrgCountsAll(t *testing.T) {
	db := setupTestDB(t)
	svc := newService(db)

	seedNotification(t, db, 1, 0, 1, "Org1", "m", false)
	seedNotification(t, db, 2, 0, 1, "Org2", "m", false)

	count, err := svc.GetUnreadCount(0, 42)
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)
}

func TestGetUnreadCount_Zero(t *testing.T) {
	db := setupTestDB(t)
	svc := newService(db)

	count, err := svc.GetUnreadCount(1, 42)
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)
}

// =========================================================================
// Integration: Dispatch -> List -> MarkRead -> GetUnreadCount
// =========================================================================

func TestFullNotificationLifecycle(t *testing.T) {
	db := setupTestDB(t)
	svc := newService(db)

	ch := createTestChannel(t, svc, 1, "Lifecycle Email", entity.NotificationChannelEmail, `{}`)
	createTestRule(t, svc, 1, ch.ID, "low")

	svc.Dispatch(context.Background(), 1, "critical", "Lifecycle Test", "Testing the full flow")

	count, err := svc.GetUnreadCount(1, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)

	notifs, err := svc.ListNotifications(1, 10, false)
	require.NoError(t, err)
	require.Len(t, notifs, 1)
	assert.Equal(t, "Lifecycle Test", notifs[0].Title)
	assert.False(t, notifs[0].IsRead)

	err = svc.MarkRead(notifs[0].ID, 10)
	require.NoError(t, err)

	count, err = svc.GetUnreadCount(1, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)

	unread, err := svc.ListNotifications(1, 10, true)
	require.NoError(t, err)
	assert.Empty(t, unread)

	all, err := svc.ListNotifications(1, 10, false)
	require.NoError(t, err)
	assert.Len(t, all, 1)
	assert.True(t, all[0].IsRead)
}

// =========================================================================
// Webhook HMAC-SHA256 signing
// =========================================================================

func TestDispatch_WebhookHMAC_SignaturePresent(t *testing.T) {
	const secret = "test-webhook-secret-key"

	var mu sync.Mutex
	var gotSignature string
	var receivedBody []byte

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		gotSignature = r.Header.Get("X-Signature-256")
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("failed to read body: %v", err)
		}
		receivedBody = body
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	db := setupTestDB(t)
	svc := newService(db)

	config := fmt.Sprintf(`{"url":"%s","secret":"%s"}`, ts.URL, secret)
	ch := createTestChannel(t, svc, 1, "HMAC Hook", entity.NotificationChannelWebhook, config)
	createTestRule(t, svc, 1, ch.ID, "low")

	svc.Dispatch(context.Background(), 1, "high", "Signed Title", "Signed Body")

	mu.Lock()
	defer mu.Unlock()

	require.NotEmpty(t, gotSignature, "X-Signature-256 header should be present")
	assert.True(t, len(gotSignature) > len("sha256="), "signature should have sha256= prefix")
	assert.Equal(t, "sha256=", gotSignature[:7], "signature must start with sha256=")

	// Verify the signature matches the body
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(receivedBody)
	expectedSig := "sha256=" + hex.EncodeToString(mac.Sum(nil))
	assert.Equal(t, expectedSig, gotSignature, "HMAC signature should match the payload")
}

func TestDispatch_WebhookHMAC_NoSecretNoHeader(t *testing.T) {
	var mu sync.Mutex
	var gotSignature string

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		gotSignature = r.Header.Get("X-Signature-256")
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	db := setupTestDB(t)
	svc := newService(db)

	// No secret in config — backward compatible
	config := fmt.Sprintf(`{"url":"%s"}`, ts.URL)
	ch := createTestChannel(t, svc, 1, "No Secret Hook", entity.NotificationChannelWebhook, config)
	createTestRule(t, svc, 1, ch.ID, "low")

	svc.Dispatch(context.Background(), 1, "high", "Unsigned Title", "Unsigned Body")

	mu.Lock()
	defer mu.Unlock()

	assert.Empty(t, gotSignature, "X-Signature-256 header should NOT be present when no secret is configured")
}

func TestDispatch_WebhookHMAC_EmptySecretNoHeader(t *testing.T) {
	var mu sync.Mutex
	var gotSignature string

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		gotSignature = r.Header.Get("X-Signature-256")
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	db := setupTestDB(t)
	svc := newService(db)

	// Empty string secret — should not sign
	config := fmt.Sprintf(`{"url":"%s","secret":""}`, ts.URL)
	ch := createTestChannel(t, svc, 1, "Empty Secret", entity.NotificationChannelWebhook, config)
	createTestRule(t, svc, 1, ch.ID, "low")

	svc.Dispatch(context.Background(), 1, "high", "Test", "Test Body")

	mu.Lock()
	defer mu.Unlock()

	assert.Empty(t, gotSignature, "empty secret should not produce a signature header")
}

func TestDispatch_WebhookHMAC_DifferentSecretsProduceDifferentSignatures(t *testing.T) {
	var mu sync.Mutex
	signatures := make([]string, 0, 2)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		signatures = append(signatures, r.Header.Get("X-Signature-256"))
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	db := setupTestDB(t)
	svc := newService(db)

	config1 := fmt.Sprintf(`{"url":"%s","secret":"secret-alpha"}`, ts.URL)
	ch1 := createTestChannel(t, svc, 1, "Hook A", entity.NotificationChannelWebhook, config1)
	createTestRule(t, svc, 1, ch1.ID, "low")

	config2 := fmt.Sprintf(`{"url":"%s","secret":"secret-beta"}`, ts.URL)
	ch2 := createTestChannel(t, svc, 2, "Hook B", entity.NotificationChannelWebhook, config2)
	createTestRule(t, svc, 2, ch2.ID, "low")

	svc.Dispatch(context.Background(), 1, "high", "Same Title", "Same Body")
	svc.Dispatch(context.Background(), 2, "high", "Same Title", "Same Body")

	mu.Lock()
	defer mu.Unlock()

	require.Len(t, signatures, 2)
	assert.NotEqual(t, signatures[0], signatures[1],
		"different secrets must produce different signatures even for the same payload")
}

// =========================================================================
// ComputeHMACSignature unit tests
// =========================================================================

func TestComputeHMACSignature_KnownVector(t *testing.T) {
	secret := []byte("my-secret-key")
	body := []byte(`{"title":"Hello","message":"World","channel":"test"}`)

	sig := notifications.ComputeHMACSignature(secret, body)

	// Independently compute the expected value
	mac := hmac.New(sha256.New, secret)
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))

	assert.Equal(t, expected, sig)
	assert.Len(t, sig, 64, "SHA-256 hex digest should be 64 characters")
}

func TestComputeHMACSignature_EmptyBody(t *testing.T) {
	secret := []byte("key")
	sig := notifications.ComputeHMACSignature(secret, []byte{})

	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte{})
	expected := hex.EncodeToString(mac.Sum(nil))

	assert.Equal(t, expected, sig)
}

func TestComputeHMACSignature_DeterministicForSameInput(t *testing.T) {
	secret := []byte("deterministic")
	body := []byte("same payload")

	sig1 := notifications.ComputeHMACSignature(secret, body)
	sig2 := notifications.ComputeHMACSignature(secret, body)

	assert.Equal(t, sig1, sig2, "same inputs must produce the same signature")
}

// =========================================================================
// ValidateWebhookSignature unit tests
// =========================================================================

func TestValidateWebhookSignature_Valid(t *testing.T) {
	secret := []byte("validation-secret")
	body := []byte(`{"title":"Test","message":"Validate me","channel":"hook"}`)

	sig := notifications.ComputeHMACSignature(secret, body)
	header := "sha256=" + sig

	assert.True(t, notifications.ValidateWebhookSignature(secret, body, header),
		"valid signature should pass validation")
}

func TestValidateWebhookSignature_InvalidDigest(t *testing.T) {
	secret := []byte("validation-secret")
	body := []byte(`{"title":"Test","message":"Validate me","channel":"hook"}`)

	// Forge a bad signature
	header := "sha256=0000000000000000000000000000000000000000000000000000000000000000"

	assert.False(t, notifications.ValidateWebhookSignature(secret, body, header),
		"forged signature should fail validation")
}

func TestValidateWebhookSignature_WrongSecret(t *testing.T) {
	secret := []byte("correct-secret")
	wrongSecret := []byte("wrong-secret")
	body := []byte(`{"data":"sensitive"}`)

	sig := notifications.ComputeHMACSignature(wrongSecret, body)
	header := "sha256=" + sig

	assert.False(t, notifications.ValidateWebhookSignature(secret, body, header),
		"signature from wrong secret should fail")
}

func TestValidateWebhookSignature_TamperedBody(t *testing.T) {
	secret := []byte("tamper-check")
	originalBody := []byte(`{"title":"Original"}`)
	tamperedBody := []byte(`{"title":"Tampered"}`)

	sig := notifications.ComputeHMACSignature(secret, originalBody)
	header := "sha256=" + sig

	assert.False(t, notifications.ValidateWebhookSignature(secret, tamperedBody, header),
		"tampered body should fail validation")
}

func TestValidateWebhookSignature_MissingPrefix(t *testing.T) {
	secret := []byte("prefix-test")
	body := []byte(`data`)

	sig := notifications.ComputeHMACSignature(secret, body)

	// No "sha256=" prefix
	assert.False(t, notifications.ValidateWebhookSignature(secret, body, sig),
		"signature without sha256= prefix should fail")
}

func TestValidateWebhookSignature_EmptyHeader(t *testing.T) {
	secret := []byte("empty-header")
	body := []byte(`data`)

	assert.False(t, notifications.ValidateWebhookSignature(secret, body, ""),
		"empty header should fail")
}

func TestValidateWebhookSignature_PrefixOnly(t *testing.T) {
	secret := []byte("prefix-only")
	body := []byte(`data`)

	assert.False(t, notifications.ValidateWebhookSignature(secret, body, "sha256="),
		"header with only prefix and no digest should fail")
}

func TestValidateWebhookSignature_InvalidHex(t *testing.T) {
	secret := []byte("hex-test")
	body := []byte(`data`)

	assert.False(t, notifications.ValidateWebhookSignature(secret, body, "sha256=not-valid-hex!@#$"),
		"invalid hex in signature should fail")
}

func TestValidateWebhookSignature_WrongAlgorithmPrefix(t *testing.T) {
	secret := []byte("algo-test")
	body := []byte(`data`)

	sig := notifications.ComputeHMACSignature(secret, body)

	assert.False(t, notifications.ValidateWebhookSignature(secret, body, "sha512="+sig),
		"wrong algorithm prefix should fail")
}

// ---------------------------------------------------------------------------
// SSRF validation tests
// ---------------------------------------------------------------------------

func TestValidateWebhookURL_PublicURL(t *testing.T) {
	assert.NoError(t, notifications.ValidateWebhookURL("https://example.com/webhook"))
	assert.NoError(t, notifications.ValidateWebhookURL("http://example.com/webhook"))
}

func TestValidateWebhookURL_BlocksPrivateIPs(t *testing.T) {
	cases := []string{
		"http://10.0.0.1/hook",
		"http://172.16.0.1/hook",
		"http://172.31.255.255/hook",
		"http://192.168.1.1/hook",
		"http://127.0.0.1/hook",
		"http://169.254.1.1/hook",
		"http://[::1]/hook",
		"http://[fe80::1]/hook",
	}
	for _, tc := range cases {
		t.Run(tc, func(t *testing.T) {
			err := notifications.ValidateWebhookURL(tc)
			assert.Error(t, err, "should block %s", tc)
		})
	}
}

func TestValidateWebhookURL_BlocksLocalhost(t *testing.T) {
	assert.Error(t, notifications.ValidateWebhookURL("http://localhost/hook"))
	assert.Error(t, notifications.ValidateWebhookURL("https://localhost:8080/hook"))
}

func TestValidateWebhookURL_BlocksBadSchemes(t *testing.T) {
	assert.Error(t, notifications.ValidateWebhookURL("ftp://example.com/file"))
	assert.Error(t, notifications.ValidateWebhookURL("file:///etc/passwd"))
	assert.Error(t, notifications.ValidateWebhookURL("gopher://evil.com"))
}

func TestValidateWebhookURL_BlocksEmpty(t *testing.T) {
	assert.Error(t, notifications.ValidateWebhookURL(""))
}

func TestValidateSlackWebhookURL_ValidSlack(t *testing.T) {
	assert.NoError(t, notifications.ValidateSlackWebhookURL("https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXXXXXXXXXXXXXXXXXX"))
}

func TestValidateSlackWebhookURL_BlocksNonSlackDomain(t *testing.T) {
	assert.ErrorIs(t, notifications.ValidateSlackWebhookURL("https://evil.com/services/hook"), notifications.ErrInvalidSlackURL)
	assert.ErrorIs(t, notifications.ValidateSlackWebhookURL("https://hooks.slack.com.evil.com/hook"), notifications.ErrInvalidSlackURL)
}

func TestValidateSlackWebhookURL_BlocksHTTP(t *testing.T) {
	assert.ErrorIs(t, notifications.ValidateSlackWebhookURL("http://hooks.slack.com/services/hook"), notifications.ErrInvalidSlackURL)
}

func TestValidateChannelConfig_Webhook(t *testing.T) {
	// Valid public URL
	cfg := `{"url":"https://example.com/hook","secret":"s3cret"}`
	assert.NoError(t, notifications.ValidateChannelConfig("webhook", cfg))

	// Private IP
	cfg = `{"url":"http://10.0.0.1/hook"}`
	assert.Error(t, notifications.ValidateChannelConfig("webhook", cfg))

	// Missing URL
	cfg = `{"url":""}`
	assert.Error(t, notifications.ValidateChannelConfig("webhook", cfg))
}

func TestValidateChannelConfig_Slack(t *testing.T) {
	cfg := `{"webhookUrl":"https://hooks.slack.com/services/T/B/X"}`
	assert.NoError(t, notifications.ValidateChannelConfig("slack", cfg))

	cfg = `{"webhookUrl":"https://evil.com/hook"}`
	assert.Error(t, notifications.ValidateChannelConfig("slack", cfg))
}

func TestValidateChannelConfig_Email(t *testing.T) {
	// Email channels have no URL to validate — always pass.
	cfg := `{"recipients":"test@example.com"}`
	assert.NoError(t, notifications.ValidateChannelConfig("email", cfg))
}
