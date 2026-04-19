package audit_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/veilence/veilence-mx/backend/internal/repo/persistent"
	"github.com/veilence/veilence-mx/backend/internal/usecase/audit"
)

// =====================================================================
// GetAuditLog
// =====================================================================

func TestGetAuditLog_Success(t *testing.T) {
	db, svc := setupTestDB(t)

	ctx := context.Background()
	svc.LogAction(ctx, "create", "package", 42, "created pkg lodash")

	var logs []persistent.AuditLog
	db.Find(&logs)
	require.Len(t, logs, 1)

	entry, err := svc.GetAuditLog(logs[0].ID)
	require.NoError(t, err)
	assert.Equal(t, "create", entry.Action)
	assert.Equal(t, "package", entry.Resource)
	assert.Equal(t, uint(42), entry.ResourceID)
}

func TestGetAuditLog_NotFound(t *testing.T) {
	_, svc := setupTestDB(t)

	_, err := svc.GetAuditLog(99999)
	require.Error(t, err)
}

// =====================================================================
// WithHTTPRequest + LogAction with IP/UA extraction
// =====================================================================

func TestLogAction_WithHTTPRequest_ExtractsIPAndUA(t *testing.T) {
	db, svc := setupTestDB(t)

	// Build context with an HTTP request containing IP and User-Agent
	req := httptest.NewRequest("POST", "/api/packages", nil)
	req.RemoteAddr = "192.168.1.100:54321"
	req.Header.Set("User-Agent", "TestAgent/2.0")

	ctx := audit.WithHTTPRequest(context.Background(), req)
	svc.LogAction(ctx, "create", "package", 1, "test with request")

	var logs []persistent.AuditLog
	db.Find(&logs)
	require.Len(t, logs, 1)
	assert.Equal(t, "192.168.1.100:54321", logs[0].IPAddress)
	assert.Equal(t, "TestAgent/2.0", logs[0].UserAgent)
}

func TestLogAuthEvent_WithHTTPRequest_ExtractsIPAndUA(t *testing.T) {
	db, svc := setupTestDB(t)

	req := httptest.NewRequest("POST", "/api/auth/login", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	req.Header.Set("User-Agent", "LoginBrowser/1.0")

	ctx := audit.WithHTTPRequest(context.Background(), req)
	svc.LogAuthEvent(ctx, "login", 42, "user logged in")

	var logs []persistent.AuditLog
	db.Find(&logs)
	require.Len(t, logs, 1)
	assert.Equal(t, "10.0.0.1:12345", logs[0].IPAddress)
	assert.Equal(t, "LoginBrowser/1.0", logs[0].UserAgent)
}

// =====================================================================
// RequestCaptureMiddleware
// =====================================================================

func TestRequestCaptureMiddleware(t *testing.T) {
	db, svc := setupTestDB(t)

	// Handler that uses the audit service to log an action
	handler := audit.RequestCaptureMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		svc.LogAction(r.Context(), "test", "middleware", 1, "via middleware")
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "172.16.0.1:9999"
	req.Header.Set("User-Agent", "MiddlewareTest/1.0")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	var logs []persistent.AuditLog
	db.Find(&logs)
	require.Len(t, logs, 1)
	assert.Equal(t, "172.16.0.1:9999", logs[0].IPAddress)
	assert.Equal(t, "MiddlewareTest/1.0", logs[0].UserAgent)
}

// =====================================================================
// CorrelationMiddleware + CorrelationIDFromContext
// =====================================================================

func TestCorrelationMiddleware_GeneratesID(t *testing.T) {
	db, svc := setupTestDB(t)

	handler := audit.CorrelationMiddleware(audit.RequestCaptureMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// The correlation ID should be in the context
		corrID := audit.CorrelationIDFromContext(r.Context())
		assert.NotEmpty(t, corrID, "correlation ID should be generated")

		svc.LogAction(r.Context(), "correlated", "test", 1, "with correlation")
		w.WriteHeader(http.StatusOK)
	})))

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// Correlation ID should be in the response header
	corrHeader := rr.Header().Get("X-Correlation-ID")
	assert.NotEmpty(t, corrHeader)

	// And it should be stored in the audit log
	var logs []persistent.AuditLog
	db.Find(&logs)
	require.Len(t, logs, 1)
	assert.Equal(t, corrHeader, logs[0].CorrelationID)
}

func TestCorrelationMiddleware_PropagatesExistingID(t *testing.T) {
	handler := audit.CorrelationMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		corrID := audit.CorrelationIDFromContext(r.Context())
		assert.Equal(t, "my-custom-correlation-id", corrID)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Correlation-ID", "my-custom-correlation-id")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, "my-custom-correlation-id", rr.Header().Get("X-Correlation-ID"))
}

func TestCorrelationIDFromContext_NoValue(t *testing.T) {
	ctx := context.Background()
	corrID := audit.CorrelationIDFromContext(ctx)
	assert.Empty(t, corrID)
}

// =====================================================================
// ListAuditLogs - pagination
// =====================================================================

func TestListAuditLogs_Pagination(t *testing.T) {
	_, svc := setupTestDB(t)

	ctx := context.Background()
	for i := 0; i < 25; i++ {
		svc.LogAction(ctx, "create", "package", uint(i+1), "pkg")
	}

	// Page 1, limit 10
	logs, total, err := svc.ListAuditLogs(0, audit.AuditLogFilters{}, 1, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(25), total)
	assert.Len(t, logs, 10)

	// Page 3, limit 10 (only 5 left)
	logs, total, err = svc.ListAuditLogs(0, audit.AuditLogFilters{}, 3, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(25), total)
	assert.Len(t, logs, 5)
}

func TestListAuditLogs_FilterByUserID(t *testing.T) {
	_, svc := setupTestDB(t)

	ctx := context.Background()
	svc.LogAuthEvent(ctx, "login", 1, "user 1 logged in")
	svc.LogAuthEvent(ctx, "login", 2, "user 2 logged in")
	svc.LogAuthEvent(ctx, "logout", 1, "user 1 logged out")

	logs, total, err := svc.ListAuditLogs(0, audit.AuditLogFilters{UserID: 1}, 1, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, logs, 2)
}

// =====================================================================
// CaptureState/LogChange - multiple fields
// =====================================================================

func TestCaptureState_MultipleFieldChanges(t *testing.T) {
	db, svc := setupTestDB(t)

	ctx := context.Background()
	before := map[string]any{
		"name":   "Before Name",
		"url":    "https://old.example.com",
		"active": true,
	}

	auditCtx := svc.CaptureState(ctx, "channel", 5, before)

	after := map[string]any{
		"name":   "After Name",
		"url":    "https://new.example.com",
		"active": true, // unchanged
	}

	auditCtx.LogChange("update", after)

	var logs []persistent.AuditLog
	db.Find(&logs)
	require.Len(t, logs, 1)
	assert.Contains(t, logs[0].Details, "Before Name")
	assert.Contains(t, logs[0].Details, "After Name")
	assert.Contains(t, logs[0].Details, "old.example.com")
	assert.Contains(t, logs[0].Details, "new.example.com")
	// "active" didn't change, so it shouldn't appear
	assert.NotContains(t, logs[0].Details, `"active"`)
}
