package audit_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/veilence/veilence-mx/backend/internal/repo/persistent"
	"github.com/veilence/veilence-mx/backend/internal/usecase/audit"
)

func setupTestDB(t *testing.T) (*gorm.DB, *audit.Service) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Discard,
	})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&persistent.AuditLog{}))
	repo := persistent.NewAuditLogRepo(db)
	svc := audit.NewService(repo)
	return db, svc
}

func TestService_LogAction(t *testing.T) {
	db, svc := setupTestDB(t)

	ctx := context.Background()
	svc.LogAction(ctx, "create", "package", 1, "added package react")

	var logs []persistent.AuditLog
	db.Find(&logs)
	require.Len(t, logs, 1)
	assert.Equal(t, "create", logs[0].Action)
	assert.Equal(t, "package", logs[0].Resource)
	assert.Equal(t, uint(1), logs[0].ResourceID)
	assert.Contains(t, logs[0].Details, "react")
}

func TestService_LogAuthEvent(t *testing.T) {
	db, svc := setupTestDB(t)

	ctx := context.Background()

	// Test successful login
	svc.LogAuthEvent(ctx, "login", 42, "user test@example.com logged in")

	var logs []persistent.AuditLog
	db.Find(&logs)
	require.Len(t, logs, 1)
	assert.Equal(t, "login", logs[0].Action)
	assert.Equal(t, "auth", logs[0].Resource)
	assert.Equal(t, uint(42), logs[0].UserID)
	assert.Equal(t, uint(0), logs[0].WorkspaceID, "auth events should not be org-scoped")
}

func TestService_LogAuthEvent_FailedLogin(t *testing.T) {
	db, svc := setupTestDB(t)

	ctx := context.Background()

	// Failed login (no user ID)
	svc.LogAuthEvent(ctx, "login_failed", 0, "failed login for unknown@example.com")

	var logs []persistent.AuditLog
	db.Find(&logs)
	require.Len(t, logs, 1)
	assert.Equal(t, "login_failed", logs[0].Action)
	assert.Equal(t, uint(0), logs[0].UserID)
}

func TestService_LogAuthEvent_Logout(t *testing.T) {
	db, svc := setupTestDB(t)

	ctx := context.Background()
	svc.LogAuthEvent(ctx, "logout", 42, "user logged out")

	var logs []persistent.AuditLog
	db.Find(&logs)
	require.Len(t, logs, 1)
	assert.Equal(t, "logout", logs[0].Action)
	assert.Equal(t, uint(42), logs[0].UserID)
}

func TestService_CaptureState_LogChange(t *testing.T) {
	db, svc := setupTestDB(t)

	ctx := context.Background()

	// Capture before state
	before := map[string]any{
		"name":   "Old Name",
		"slug":   "old-slug",
		"status": "active",
	}

	auditCtx := svc.CaptureState(ctx, "org", 1, before)

	// After state with changes
	after := map[string]any{
		"name":   "New Name",
		"slug":   "old-slug", // unchanged
		"status": "inactive",
	}

	auditCtx.LogChange("update", after)

	var logs []persistent.AuditLog
	db.Find(&logs)
	require.Len(t, logs, 1)
	assert.Equal(t, "update", logs[0].Action)
	assert.Equal(t, "org", logs[0].Resource)
	assert.Equal(t, uint(1), logs[0].ResourceID)

	// Details should contain the changes (not the unchanged field)
	assert.Contains(t, logs[0].Details, "name")
	assert.Contains(t, logs[0].Details, "Old Name")
	assert.Contains(t, logs[0].Details, "New Name")
	assert.Contains(t, logs[0].Details, "status")
	// slug did not change so should not be in details
	assert.NotContains(t, logs[0].Details, `"slug"`)
}

func TestService_CaptureState_NoChanges(t *testing.T) {
	db, svc := setupTestDB(t)

	ctx := context.Background()

	before := map[string]any{
		"name": "Same Name",
	}
	auditCtx := svc.CaptureState(ctx, "org", 1, before)
	auditCtx.LogChange("update", map[string]any{
		"name": "Same Name",
	})

	// No log entry should be created when there are no changes
	var count int64
	db.Model(&persistent.AuditLog{}).Count(&count)
	assert.Equal(t, int64(0), count, "no audit log should be created for unchanged data")
}

func TestService_ListAuditLogs_Filters(t *testing.T) {
	_, svc := setupTestDB(t)

	ctx := context.Background()

	// Create multiple log entries
	svc.LogAction(ctx, "create", "package", 1, "created package")
	svc.LogAction(ctx, "update", "package", 1, "updated package")
	svc.LogAction(ctx, "delete", "package", 1, "deleted package")

	// Filter by action
	logs, total, err := svc.ListAuditLogs(0, audit.AuditLogFilters{Action: "create"}, 1, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, logs, 1)
	assert.Equal(t, "create", logs[0].Action)

	// Filter by resource
	logs, total, err = svc.ListAuditLogs(0, audit.AuditLogFilters{Resource: "package"}, 1, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)
	assert.Len(t, logs, 3)
}

func TestService_AuditLogCoverage_AuthEvents(t *testing.T) {
	db, svc := setupTestDB(t)

	ctx := context.Background()

	// Simulate all auth events that should be logged
	authEvents := []struct {
		action  string
		userID  uint
		details string
	}{
		{"login", 1, "user logged in"},
		{"login_failed", 0, "failed login for test@example.com"},
		{"logout", 1, "user logged out"},
	}

	for _, event := range authEvents {
		svc.LogAuthEvent(ctx, event.action, event.userID, event.details)
	}

	var logs []persistent.AuditLog
	db.Find(&logs)
	assert.Len(t, logs, len(authEvents), "all auth events should be logged")

	// Verify each event was logged
	for i, event := range authEvents {
		assert.Equal(t, event.action, logs[i].Action)
		assert.Equal(t, "auth", logs[i].Resource)
	}
}
