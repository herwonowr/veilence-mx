package notifications_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/veilence/veilence-mx/backend/internal/domain"
	"github.com/veilence/veilence-mx/backend/internal/models"
	"github.com/veilence/veilence-mx/backend/internal/notifications"
	"github.com/veilence/veilence-mx/backend/internal/repository"
)

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	err = db.AutoMigrate(
		&models.NotificationChannel{},
		&models.NotificationRule{},
		&models.Notification{},
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	})
	return db
}

func newService(db *gorm.DB) *notifications.Service {
	channelRepo := repository.NewNotificationChannelRepo(db)
	ruleRepo := repository.NewNotificationRuleRepo(db)
	notifRepo := repository.NewNotificationRepo(db)
	return notifications.NewService(channelRepo, ruleRepo, notifRepo)
}

// createTestChannel is a convenience helper that creates a channel and fails
// the test immediately if an error occurs.
func createTestChannel(t *testing.T, svc *notifications.Service, orgID uint, name string, chanType domain.NotificationChannelType, config string) *domain.NotificationChannel {
	t.Helper()
	ch, err := svc.CreateChannel(orgID, name, chanType, config)
	require.NoError(t, err)
	return ch
}

// createTestRule is a convenience helper that creates a rule and fails
// the test immediately if an error occurs.
func createTestRule(t *testing.T, svc *notifications.Service, orgID, channelID uint, severity string) *domain.NotificationRule {
	t.Helper()
	rule, err := svc.CreateRule(orgID, channelID, severity)
	require.NoError(t, err)
	return rule
}

// seedNotification inserts a Notification directly via GORM for testing query
// methods without going through Dispatch.
func seedNotification(t *testing.T, db *gorm.DB, orgID, userID, channelID uint, title, message string, isRead bool) *models.Notification {
	t.Helper()
	n := &models.Notification{
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

	ch, err := svc.CreateChannel(1, "Email Alerts", domain.NotificationChannelEmail, `{"to":"ops@example.com"}`)
	require.NoError(t, err)
	assert.NotZero(t, ch.ID)
	assert.Equal(t, uint(1), ch.OrgID)
	assert.Equal(t, "Email Alerts", ch.Name)
	assert.Equal(t, domain.NotificationChannelEmail, ch.Type)
	assert.Equal(t, `{"to":"ops@example.com"}`, ch.Config)
	assert.True(t, ch.IsActive, "new channels should be active by default")
}

func TestCreateChannel_Slack(t *testing.T) {
	db := setupTestDB(t)
	svc := newService(db)

	ch, err := svc.CreateChannel(1, "Slack Ops", domain.NotificationChannelSlack, `{"webhook":"https://hooks.slack.com/xxx"}`)
	require.NoError(t, err)
	assert.NotZero(t, ch.ID)
	assert.Equal(t, domain.NotificationChannelSlack, ch.Type)
}

func TestCreateChannel_Webhook(t *testing.T) {
	db := setupTestDB(t)
	svc := newService(db)

	ch, err := svc.CreateChannel(1, "Webhook", domain.NotificationChannelWebhook, `{"url":"https://example.com/hook"}`)
	require.NoError(t, err)
	assert.NotZero(t, ch.ID)
	assert.Equal(t, domain.NotificationChannelWebhook, ch.Type)
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

	createTestChannel(t, svc, 1, "Org1 Email", domain.NotificationChannelEmail, `{}`)
	createTestChannel(t, svc, 1, "Org1 Slack", domain.NotificationChannelSlack, `{}`)
	createTestChannel(t, svc, 2, "Org2 Email", domain.NotificationChannelEmail, `{}`)

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

	ch := createTestChannel(t, svc, 1, "Original", domain.NotificationChannelEmail, `{"to":"old@example.com"}`)

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

	ch := createTestChannel(t, svc, 1, "Org1 Only", domain.NotificationChannelEmail, `{}`)

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

	ch := createTestChannel(t, svc, 1, "To Delete", domain.NotificationChannelEmail, `{}`)

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

	ch := createTestChannel(t, svc, 1, "Org1 Only", domain.NotificationChannelEmail, `{}`)

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

	ch := createTestChannel(t, svc, 1, "Alerts", domain.NotificationChannelEmail, `{}`)

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

	ch := createTestChannel(t, svc, 1, "All Levels", domain.NotificationChannelSlack, `{}`)

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

	ch1 := createTestChannel(t, svc, 1, "Ch1", domain.NotificationChannelEmail, `{}`)
	ch2 := createTestChannel(t, svc, 2, "Ch2", domain.NotificationChannelEmail, `{}`)

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

	ch := createTestChannel(t, svc, 1, "Ch", domain.NotificationChannelEmail, `{}`)
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

	ch := createTestChannel(t, svc, 1, "Ch", domain.NotificationChannelEmail, `{}`)
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

	ch := createTestChannel(t, svc, 1, "Email", domain.NotificationChannelEmail, `{}`)
	createTestRule(t, svc, 1, ch.ID, "high")

	svc.Dispatch(1, "high", "Alert", "This is a high alert")

	var count int64
	db.Model(&models.Notification{}).Where("org_id = ?", 1).Count(&count)
	assert.Equal(t, int64(1), count)
}

func TestDispatch_MatchesSeverityAboveThreshold(t *testing.T) {
	db := setupTestDB(t)
	svc := newService(db)

	ch := createTestChannel(t, svc, 1, "Email", domain.NotificationChannelEmail, `{}`)
	createTestRule(t, svc, 1, ch.ID, "medium")

	svc.Dispatch(1, "critical", "Critical Issue", "Something terrible happened")

	var count int64
	db.Model(&models.Notification{}).Where("org_id = ?", 1).Count(&count)
	assert.Equal(t, int64(1), count)
}

func TestDispatch_SkipsBelowThreshold(t *testing.T) {
	db := setupTestDB(t)
	svc := newService(db)

	ch := createTestChannel(t, svc, 1, "Email", domain.NotificationChannelEmail, `{}`)
	createTestRule(t, svc, 1, ch.ID, "critical")

	svc.Dispatch(1, "low", "Info", "Low priority event")

	var count int64
	db.Model(&models.Notification{}).Where("org_id = ?", 1).Count(&count)
	assert.Equal(t, int64(0), count, "should not create notification for severity below threshold")
}

func TestDispatch_MultipleRulesMultipleChannels(t *testing.T) {
	db := setupTestDB(t)
	svc := newService(db)

	ch1 := createTestChannel(t, svc, 1, "Slack Low", domain.NotificationChannelSlack, `{}`)
	ch2 := createTestChannel(t, svc, 1, "Email High", domain.NotificationChannelEmail, `{}`)
	ch3 := createTestChannel(t, svc, 1, "Webhook Critical", domain.NotificationChannelWebhook, `{"url":"http://invalid.test"}`)

	createTestRule(t, svc, 1, ch1.ID, "low")
	createTestRule(t, svc, 1, ch2.ID, "high")
	createTestRule(t, svc, 1, ch3.ID, "critical")

	svc.Dispatch(1, "high", "High Alert", "Something important")

	var notifs []models.Notification
	db.Where("org_id = ?", 1).Find(&notifs)
	assert.Len(t, notifs, 2, "expected 2 notifications: low-threshold and high-threshold channels")

	channelIDs := map[uint]bool{}
	for _, n := range notifs {
		channelIDs[n.ChannelID] = true
	}
	assert.True(t, channelIDs[ch1.ID], "low-threshold channel should receive notification")
	assert.True(t, channelIDs[ch2.ID], "high-threshold channel should receive notification")
	assert.False(t, channelIDs[ch3.ID], "critical-only channel should NOT receive notification")
}

func TestDispatch_AllSeverityLevels(t *testing.T) {
	db := setupTestDB(t)
	svc := newService(db)

	ch := createTestChannel(t, svc, 1, "All", domain.NotificationChannelEmail, `{}`)
	createTestRule(t, svc, 1, ch.ID, "low")

	levels := []string{"low", "medium", "high", "critical"}
	for _, level := range levels {
		svc.Dispatch(1, level, fmt.Sprintf("Title %s", level), "msg")
	}

	var count int64
	db.Model(&models.Notification{}).Where("org_id = ?", 1).Count(&count)
	assert.Equal(t, int64(4), count, "low-threshold rule should fire for all severity levels")
}

func TestDispatch_NoMatchingRules(t *testing.T) {
	db := setupTestDB(t)
	svc := newService(db)

	svc.Dispatch(1, "critical", "Nobody Listening", "No rules")

	var count int64
	db.Model(&models.Notification{}).Where("org_id = ?", 1).Count(&count)
	assert.Equal(t, int64(0), count)
}

func TestDispatch_SkipsDisabledChannel(t *testing.T) {
	db := setupTestDB(t)
	svc := newService(db)

	ch := createTestChannel(t, svc, 1, "Disabled", domain.NotificationChannelEmail, `{}`)
	createTestRule(t, svc, 1, ch.ID, "low")

	_, err := svc.UpdateChannel(ch.ID, 1, ch.Name, ch.Config, false)
	require.NoError(t, err)

	svc.Dispatch(1, "critical", "Won't Arrive", "Channel is disabled")

	var count int64
	db.Model(&models.Notification{}).Where("org_id = ?", 1).Count(&count)
	assert.Equal(t, int64(0), count, "disabled channel should not produce notifications")
}

func TestDispatch_SkipsInactiveRule(t *testing.T) {
	db := setupTestDB(t)
	svc := newService(db)

	ch := createTestChannel(t, svc, 1, "Active Channel", domain.NotificationChannelEmail, `{}`)
	rule := createTestRule(t, svc, 1, ch.ID, "low")

	db.Model(&models.NotificationRule{}).Where("id = ?", rule.ID).Update("is_active", false)

	svc.Dispatch(1, "critical", "Won't Arrive", "Rule is inactive")

	var count int64
	db.Model(&models.Notification{}).Where("org_id = ?", 1).Count(&count)
	assert.Equal(t, int64(0), count, "inactive rule should not produce notifications")
}

func TestDispatch_IsolatedByOrg(t *testing.T) {
	db := setupTestDB(t)
	svc := newService(db)

	ch1 := createTestChannel(t, svc, 1, "Org1", domain.NotificationChannelEmail, `{}`)
	ch2 := createTestChannel(t, svc, 2, "Org2", domain.NotificationChannelEmail, `{}`)
	createTestRule(t, svc, 1, ch1.ID, "low")
	createTestRule(t, svc, 2, ch2.ID, "low")

	svc.Dispatch(1, "high", "Org1 Alert", "only org 1")

	var count1, count2 int64
	db.Model(&models.Notification{}).Where("org_id = ?", 1).Count(&count1)
	db.Model(&models.Notification{}).Where("org_id = ?", 2).Count(&count2)
	assert.Equal(t, int64(1), count1)
	assert.Equal(t, int64(0), count2, "other org should not receive notification")
}

func TestDispatch_NotificationFields(t *testing.T) {
	db := setupTestDB(t)
	svc := newService(db)

	ch := createTestChannel(t, svc, 1, "Email", domain.NotificationChannelEmail, `{}`)
	createTestRule(t, svc, 1, ch.ID, "low")

	svc.Dispatch(1, "critical", "My Title", "My Message Body")

	var notif models.Notification
	require.NoError(t, db.Where("org_id = ?", 1).First(&notif).Error)
	assert.Equal(t, uint(1), notif.OrgID)
	assert.Equal(t, uint(0), notif.UserID, "Dispatch creates org-wide notifications with user_id=0")
	assert.Equal(t, ch.ID, notif.ChannelID)
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
	ch := createTestChannel(t, svc, 1, "Hook", domain.NotificationChannelWebhook, config)
	createTestRule(t, svc, 1, ch.ID, "low")

	svc.Dispatch(1, "high", "Webhook Title", "Webhook Body")

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

	ch := createTestChannel(t, svc, 1, "Dead Hook", domain.NotificationChannelWebhook, `{"url":"http://127.0.0.1:1"}`)
	createTestRule(t, svc, 1, ch.ID, "low")

	svc.Dispatch(1, "critical", "Unreachable", "The webhook URL is dead")

	var count int64
	db.Model(&models.Notification{}).Where("org_id = ?", 1).Count(&count)
	assert.Equal(t, int64(1), count, "in-app notification should still be created even when webhook fails")
}

func TestDispatch_WebhookInvalidConfig(t *testing.T) {
	db := setupTestDB(t)
	svc := newService(db)

	ch := createTestChannel(t, svc, 1, "Bad Config", domain.NotificationChannelWebhook, `not-json`)
	createTestRule(t, svc, 1, ch.ID, "low")

	svc.Dispatch(1, "critical", "Bad Config", "Config is not valid JSON")

	var count int64
	db.Model(&models.Notification{}).Where("org_id = ?", 1).Count(&count)
	assert.Equal(t, int64(1), count)
}

func TestDispatch_WebhookEmptyURL(t *testing.T) {
	db := setupTestDB(t)
	svc := newService(db)

	ch := createTestChannel(t, svc, 1, "Empty URL", domain.NotificationChannelWebhook, `{"url":""}`)
	createTestRule(t, svc, 1, ch.ID, "low")

	svc.Dispatch(1, "critical", "Empty URL", "URL field is empty")

	var count int64
	db.Model(&models.Notification{}).Where("org_id = ?", 1).Count(&count)
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
	ch := createTestChannel(t, svc, 1, "Error Hook", domain.NotificationChannelWebhook, config)
	createTestRule(t, svc, 1, ch.ID, "low")

	svc.Dispatch(1, "high", "Server Error", "The hook returned 500")

	var count int64
	db.Model(&models.Notification{}).Where("org_id = ?", 1).Count(&count)
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

	var updated models.Notification
	db.First(&updated, n.ID)
	assert.True(t, updated.IsRead)
}

func TestMarkRead_OrgWideNotification(t *testing.T) {
	db := setupTestDB(t)
	svc := newService(db)

	n := seedNotification(t, db, 1, 0, 1, "Org-Wide", "for everyone", false)

	err := svc.MarkRead(n.ID, 42)
	require.NoError(t, err)

	var updated models.Notification
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

	ch := createTestChannel(t, svc, 1, "Lifecycle Email", domain.NotificationChannelEmail, `{}`)
	createTestRule(t, svc, 1, ch.ID, "low")

	svc.Dispatch(1, "critical", "Lifecycle Test", "Testing the full flow")

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
