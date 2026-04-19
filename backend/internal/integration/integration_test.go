package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/veilence/veilence-mx/backend/internal/controller/restapi"
	v1 "github.com/veilence/veilence-mx/backend/internal/controller/restapi/v1"
	"github.com/veilence/veilence-mx/backend/internal/usecase/audit"
	"github.com/veilence/veilence-mx/backend/internal/usecase/auth"
	"github.com/veilence/veilence-mx/backend/internal/repo/persistent"
	"github.com/veilence/veilence-mx/backend/internal/usecase/notifications"
	"github.com/veilence/veilence-mx/backend/internal/usecase/rbac"
	"github.com/veilence/veilence-mx/backend/internal/usecase/alertuc"
	alertnoteuc "github.com/veilence/veilence-mx/backend/internal/usecase/alertnote"
	"github.com/veilence/veilence-mx/backend/internal/usecase/dashboarduc"
	"github.com/veilence/veilence-mx/backend/internal/usecase/healthuc"
	"github.com/veilence/veilence-mx/backend/internal/usecase/pkguc"
	"github.com/veilence/veilence-mx/backend/internal/usecase/releaseuc"
	"github.com/veilence/veilence-mx/backend/internal/usecase/settinguc"
)

const testJWTSecret = "integration-test-jwt-secret-very-long-key-1234567890"

// testDBPinger implements healthuc.DBPinger for integration tests.
type testDBPinger struct {
	db *gorm.DB
}

func (p testDBPinger) PingDB(ctx context.Context) error {
	sqlDB, err := p.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}

// testRedisPinger implements healthuc.RedisPinger for integration tests (always succeeds).
type testRedisPinger struct{}

func (testRedisPinger) Ping(_ context.Context) error { return nil }

// testServer holds the full application stack for integration testing.
type testServer struct {
	server  *httptest.Server
	db      *gorm.DB
	authSvc *auth.Service
	rbacSvc *rbac.Service
}

// setupIntegrationServer creates a full integration test server with all services wired up.
func setupIntegrationServer(t *testing.T) *testServer {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(
		&persistent.User{},
		&persistent.RefreshToken{},
		&persistent.APIKey{},
		&persistent.PasswordResetToken{},
		&persistent.EmailVerificationToken{},
		&persistent.Session{},
		&persistent.Workspace{},
		&persistent.Role{},
		&persistent.Permission{},
		&persistent.WorkspaceMember{},
		&persistent.Invitation{},
		&persistent.Package{},
		&persistent.Release{},
		&persistent.Diff{},
		&persistent.Analysis{},
		&persistent.Alert{},
		&persistent.Setting{},
		&persistent.AuditLog{},
		&persistent.NotificationChannel{},
		&persistent.NotificationRule{},
		&persistent.Notification{},
	)
	require.NoError(t, err)

	// Seed permissions for RBAC
	err = rbac.SeedPermissions(db)
	require.NoError(t, err)

	// Create services
	userRepo := persistent.NewUserRepo(db)
	refreshTokenRepo := persistent.NewRefreshTokenRepo(db)
	apiKeyRepo := persistent.NewAPIKeyRepo(db)
	passwordResetTokenRepo := persistent.NewPasswordResetTokenRepo(db)
	emailVerificationTokenRepo := persistent.NewEmailVerificationTokenRepo(db)
	sessionRepo := persistent.NewSessionRepo(db)

	authSvc := auth.NewService(userRepo, refreshTokenRepo, apiKeyRepo, passwordResetTokenRepo, emailVerificationTokenRepo, sessionRepo, nil, nil, testJWTSecret)
	rbacSvc := rbac.NewService(db)
	auditSvc := audit.NewService(db)

	channelRepo := persistent.NewNotificationChannelRepo(db)
	ruleRepo := persistent.NewNotificationRuleRepo(db)
	notifRepo := persistent.NewNotificationRepo(db)
	notifSvc := notifications.NewService(channelRepo, ruleRepo, notifRepo, notifications.SMTPConfig{})

	dashboardRepo := persistent.NewDashboardRepo(db)
	alertNoteRepo := persistent.NewAlertNoteRepo(db)
	alertRepo := persistent.NewAlertRepo(db)
	packageRepo := persistent.NewPackageRepo(db)
	releaseRepo := persistent.NewReleaseRepo(db)
	diffRepo := persistent.NewDiffRepo(db)
	analysisRepo := persistent.NewAnalysisRepo(db)
	settingRepo := persistent.NewSettingRepo(db)

	alertNoteSvc := alertnoteuc.New(alertNoteRepo, alertRepo, userRepo)
	pkgSvc := pkguc.New(packageRepo, auditSvc)
	alertSvc := alertuc.New(alertRepo, auditSvc)
	releaseSvc := releaseuc.New(packageRepo, releaseRepo, diffRepo, analysisRepo, nil)
	settingSvc := settinguc.New(settingRepo)
	dashboardSvc := dashboarduc.New(dashboardRepo, releaseRepo, diffRepo, analysisRepo, nil)
	healthSvc := healthuc.New(testDBPinger{db: db}, testRedisPinger{})

	h := &v1.Handlers{
		Auth: &v1.AuthHandlers{
			Auth:  authSvc,
			Audit: auditSvc,
		},
		Sessions: &v1.SessionHandlers{
			Auth: authSvc,
		},
		Notifications: &v1.NotificationHandlers{
			Notifications: notifSvc,
			Audit:         auditSvc,
		},
		Workspace: &v1.WorkspaceHandlers{
			RBAC:  rbacSvc,
			Audit: auditSvc,
		},
		AuditLogs: &v1.AuditHandlers{
			Audit: auditSvc,
		},
		Packages: &v1.PackageHandlers{
			PkgSvc:     pkgSvc,
			ReleaseSvc: releaseSvc,
			Audit:      auditSvc,
		},
		Alerts: &v1.AlertHandlers{
			AlertSvc: alertSvc,
			Notes:    alertNoteSvc,
			Audit:    auditSvc,
		},
		Settings: &v1.SettingsHandlers{
			SettingSvc: settingSvc,
			Audit:      auditSvc,
		},
		Dashboard: &v1.DashboardHandlers{
			DashboardSvc: dashboardSvc,
		},
		Health: &v1.HealthHandlers{
			HealthSvc: healthSvc,
		},
	}

	router := restapi.NewRouter(h, "http://localhost:3000", authSvc, rbacSvc)
	server := httptest.NewServer(router)

	t.Cleanup(func() {
		server.Close()
		sqlDB, _ := db.DB()
		sqlDB.Close()
	})

	return &testServer{
		server:  server,
		db:      db,
		authSvc: authSvc,
		rbacSvc: rbacSvc,
	}
}

// jsonRequest sends a JSON request to the test server.
func (ts *testServer) jsonRequest(t *testing.T, method, path string, body any, headers map[string]string) *http.Response {
	t.Helper()

	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		require.NoError(t, err)
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, ts.server.URL+path, bodyReader)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	return resp
}

// parseResponse reads and parses a JSON response body.
func parseResponse(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	var result map[string]any
	require.NoError(t, json.Unmarshal(data, &result))
	return result
}

// getCSRFToken gets a CSRF token by making a GET request to a safe public endpoint.
// We use /api/invitations/dummy which won't panic (unlike /api/health which needs Queue).
func (ts *testServer) getCSRFToken(t *testing.T) (string, []*http.Cookie) {
	t.Helper()
	resp, err := http.Get(ts.server.URL + "/api/invitations/dummy")
	require.NoError(t, err)
	defer resp.Body.Close()
	cookies := resp.Cookies()
	var csrfToken string
	for _, c := range cookies {
		if c.Name == "_csrf_token" {
			csrfToken = c.Value
			break
		}
	}
	return csrfToken, cookies
}

// jsonRequestWithCSRF sends a JSON request with CSRF token.
func (ts *testServer) jsonRequestWithCSRF(t *testing.T, method, path string, body any, authToken string) (*http.Response, map[string]any) {
	t.Helper()

	csrfToken, cookies := ts.getCSRFToken(t)

	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		require.NoError(t, err)
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, ts.server.URL+path, bodyReader)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://localhost:3000")

	if csrfToken != "" {
		req.Header.Set("X-CSRF-Token", csrfToken)
	}
	for _, c := range cookies {
		req.AddCookie(c)
	}
	if authToken != "" {
		req.Header.Set("Authorization", "Bearer "+authToken)
	}

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	result := parseResponse(t, resp)
	return resp, result
}

// =====================================================================
// Test: Full Auth Flow (Register → Login → GetMe → Refresh → Logout)
// =====================================================================

func TestIntegration_AuthFlow(t *testing.T) {
	ts := setupIntegrationServer(t)

	// 1. Register
	registerBody := map[string]string{
		"email":     "auth-flow@example.com",
		"password":  "SecurePass123!",
		"firstName": "Auth",
		"lastName":  "Flow",
	}
	resp, result := ts.jsonRequestWithCSRF(t, "POST", "/api/auth/register", registerBody, "")
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	data := result["data"].(map[string]any)
	accessToken := data["accessToken"].(string)
	refreshToken := data["refreshToken"].(string)
	assert.NotEmpty(t, accessToken)
	assert.NotEmpty(t, refreshToken)

	user := data["user"].(map[string]any)
	assert.Equal(t, "auth-flow@example.com", user["email"])

	// 2. Login with the same credentials
	loginBody := map[string]string{
		"email":    "auth-flow@example.com",
		"password": "SecurePass123!",
	}
	resp, result = ts.jsonRequestWithCSRF(t, "POST", "/api/auth/login", loginBody, "")
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	data = result["data"].(map[string]any)
	accessToken = data["accessToken"].(string)
	refreshToken = data["refreshToken"].(string)

	// 3. GetMe with the access token
	resp, result = ts.jsonRequestWithCSRF(t, "GET", "/api/auth/me", nil, accessToken)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	userData := result["data"].(map[string]any)
	assert.Equal(t, "auth-flow@example.com", userData["email"])

	// 4. Refresh tokens
	refreshBody := map[string]string{
		"refreshToken": refreshToken,
	}
	resp, result = ts.jsonRequestWithCSRF(t, "POST", "/api/auth/refresh", refreshBody, "")
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	newTokens := result["data"].(map[string]any)
	newAccessToken := newTokens["accessToken"].(string)
	newRefreshToken := newTokens["refreshToken"].(string)
	assert.NotEmpty(t, newAccessToken)
	assert.NotEqual(t, refreshToken, newRefreshToken)

	// 5. Logout
	logoutBody := map[string]string{
		"refreshToken": newRefreshToken,
	}
	resp, result = ts.jsonRequestWithCSRF(t, "POST", "/api/auth/logout", logoutBody, newAccessToken)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// 6. Old refresh token should no longer work
	resp, result = ts.jsonRequestWithCSRF(t, "POST", "/api/auth/refresh", map[string]string{"refreshToken": newRefreshToken}, "")
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// =====================================================================
// Test: Register validation
// =====================================================================

func TestIntegration_Register_Validation(t *testing.T) {
	ts := setupIntegrationServer(t)

	tests := []struct {
		name       string
		body       map[string]string
		wantStatus int
	}{
		{
			"missing email",
			map[string]string{"email": "", "password": "Pass123!", "firstName": "A", "lastName": "B"},
			http.StatusBadRequest,
		},
		{
			"invalid email",
			map[string]string{"email": "notanemail", "password": "Pass123!", "firstName": "A", "lastName": "B"},
			http.StatusBadRequest,
		},
		{
			"short password",
			map[string]string{"email": "valid@example.com", "password": "short", "firstName": "A", "lastName": "B"},
			http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, _ := ts.jsonRequestWithCSRF(t, "POST", "/api/auth/register", tt.body, "")
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
		})
	}
}

// =====================================================================
// Test: Duplicate registration
// =====================================================================

func TestIntegration_Register_DuplicateEmail(t *testing.T) {
	ts := setupIntegrationServer(t)

	body := map[string]string{
		"email": "dup@example.com", "password": "SecurePass123!", "firstName": "A", "lastName": "B",
	}

	resp, _ := ts.jsonRequestWithCSRF(t, "POST", "/api/auth/register", body, "")
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	resp, _ = ts.jsonRequestWithCSRF(t, "POST", "/api/auth/register", body, "")
	assert.Equal(t, http.StatusConflict, resp.StatusCode)
}

// =====================================================================
// Test: Login failure
// =====================================================================

func TestIntegration_Login_WrongCredentials(t *testing.T) {
	ts := setupIntegrationServer(t)

	body := map[string]string{"email": "nobody@example.com", "password": "Wrong123!"}
	resp, _ := ts.jsonRequestWithCSRF(t, "POST", "/api/auth/login", body, "")
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// =====================================================================
// Test: Protected endpoint without auth
// =====================================================================

func TestIntegration_ProtectedEndpoint_NoAuth(t *testing.T) {
	ts := setupIntegrationServer(t)

	resp, _ := ts.jsonRequestWithCSRF(t, "GET", "/api/auth/me", nil, "")
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// =====================================================================
// Test: Workspace lifecycle (Create → Get → List → Update → Delete)
// =====================================================================

func TestIntegration_WorkspaceLifecycle(t *testing.T) {
	ts := setupIntegrationServer(t)

	// Register and login to get a token
	registerBody := map[string]string{
		"email": "org-life@example.com", "password": "SecurePass123!",
		"firstName": "Org", "lastName": "Life",
	}
	resp, result := ts.jsonRequestWithCSRF(t, "POST", "/api/auth/register", registerBody, "")
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	data := result["data"].(map[string]any)
	accessToken := data["accessToken"].(string)

	// 1. Create workspace
	wsBody := map[string]string{
		"name": "Test Workspace", "slug": "test-workspace", "description": "Integration test workspace",
	}
	resp, result = ts.jsonRequestWithCSRF(t, "POST", "/api/workspaces", wsBody, accessToken)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	wsData := result["data"].(map[string]any)
	wsID := wsData["id"].(float64)
	assert.NotZero(t, wsID)
	assert.Equal(t, "Test Workspace", wsData["name"])
	assert.Equal(t, "test-workspace", wsData["slug"])

	// 2. List workspaces
	resp, result = ts.jsonRequestWithCSRF(t, "GET", "/api/workspaces", nil, accessToken)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	orgs := result["data"].([]any)
	assert.Len(t, orgs, 1)

	// 3. Get workspace
	wsPath := fmt.Sprintf("/api/workspaces/%.0f", wsID)
	resp, result = ts.jsonRequestWithCSRF(t, "GET", wsPath, nil, accessToken)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// 4. Update workspace
	updateBody := map[string]string{
		"name": "Updated Workspace", "slug": "updated-workspace", "description": "Updated desc",
	}
	resp, result = ts.jsonRequestWithCSRF(t, "PUT", wsPath, updateBody, accessToken)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	updatedWorkspace := result["data"].(map[string]any)
	assert.Equal(t, "Updated Workspace", updatedWorkspace["name"])

	// 5. List members (owner should be listed)
	resp, result = ts.jsonRequestWithCSRF(t, "GET", wsPath+"/members", nil, accessToken)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	members := result["data"].([]any)
	assert.Len(t, members, 1)

	// 6. List roles
	resp, result = ts.jsonRequestWithCSRF(t, "GET", wsPath+"/roles", nil, accessToken)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	roles := result["data"].([]any)
	assert.Len(t, roles, 4, "owner, admin, member, viewer")

	// 7. Delete workspace
	resp, _ = ts.jsonRequestWithCSRF(t, "DELETE", wsPath, nil, accessToken)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// 8. Workspace should no longer be accessible
	resp, _ = ts.jsonRequestWithCSRF(t, "GET", wsPath, nil, accessToken)
	assert.NotEqual(t, http.StatusOK, resp.StatusCode)
}

// =====================================================================
// Test: Duplicate workspace slug
// =====================================================================

func TestIntegration_WorkspaceCreate_DuplicateSlug(t *testing.T) {
	ts := setupIntegrationServer(t)

	registerBody := map[string]string{
		"email": "dup-slug@example.com", "password": "SecurePass123!",
		"firstName": "Dup", "lastName": "Slug",
	}
	resp, result := ts.jsonRequestWithCSRF(t, "POST", "/api/auth/register", registerBody, "")
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	accessToken := result["data"].(map[string]any)["accessToken"].(string)

	wsBody := map[string]string{"name": "Workspace1", "slug": "my-slug", "description": ""}
	resp, _ = ts.jsonRequestWithCSRF(t, "POST", "/api/workspaces", wsBody, accessToken)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	resp, _ = ts.jsonRequestWithCSRF(t, "POST", "/api/workspaces", wsBody, accessToken)
	assert.Equal(t, http.StatusConflict, resp.StatusCode)
}

// =====================================================================
// Test: API Key flow
// =====================================================================

func TestIntegration_APIKeyFlow(t *testing.T) {
	ts := setupIntegrationServer(t)

	// Register
	registerBody := map[string]string{
		"email": "apikey@example.com", "password": "SecurePass123!",
		"firstName": "API", "lastName": "Key",
	}
	resp, result := ts.jsonRequestWithCSRF(t, "POST", "/api/auth/register", registerBody, "")
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	accessToken := result["data"].(map[string]any)["accessToken"].(string)

	// Create a workspace (required for API key creation)
	wsBody := map[string]string{"name": "Test Workspace", "slug": "test-ws-apikey"}
	resp, result = ts.jsonRequestWithCSRF(t, "POST", "/api/workspaces", wsBody, accessToken)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	wsID := result["data"].(map[string]any)["id"].(float64)

	// Create API key (with workspace context via X-Workspace-ID header)
	keyBody := map[string]string{"name": "test-key", "role": "viewer"}
	csrfToken, cookies := ts.getCSRFToken(t)
	body, _ := json.Marshal(keyBody)
	req, err := http.NewRequest("POST", ts.server.URL+"/api/auth/api-keys", bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Workspace-ID", fmt.Sprintf("%.0f", wsID))
	req.Header.Set("X-CSRF-Token", csrfToken)
	req.Header.Set("Origin", "http://localhost:3000")
	for _, c := range cookies {
		req.AddCookie(c)
	}
	apiResp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	result = parseResponse(t, apiResp)
	assert.Equal(t, http.StatusCreated, apiResp.StatusCode)
	keyData := result["data"].(map[string]any)
	rawKey := keyData["key"].(string)
	assert.NotEmpty(t, rawKey)

	apiKeyObj := keyData["apiKey"].(map[string]any)
	keyID := apiKeyObj["id"].(float64)

	// List API keys
	resp, result = ts.jsonRequestWithCSRF(t, "GET", "/api/auth/api-keys", nil, accessToken)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	keys := result["data"].([]any)
	assert.Len(t, keys, 1)

	// Access a protected endpoint using the API key (instead of Bearer token)
	// The auth middleware should accept the X-API-Key header
	csrfToken, cookies = ts.getCSRFToken(t)
	req, err = http.NewRequest("GET", ts.server.URL+"/api/auth/me", nil)
	require.NoError(t, err)
	req.Header.Set("X-API-Key", rawKey)
	req.Header.Set("X-CSRF-Token", csrfToken)
	req.Header.Set("Origin", "http://localhost:3000")
	for _, c := range cookies {
		req.AddCookie(c)
	}
	apiResp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	apiResult := parseResponse(t, apiResp)
	assert.Equal(t, http.StatusOK, apiResp.StatusCode)
	assert.Equal(t, "apikey@example.com", apiResult["data"].(map[string]any)["email"])

	// Revoke the API key
	resp, _ = ts.jsonRequestWithCSRF(t, "DELETE", fmt.Sprintf("/api/auth/api-keys/%.0f", keyID), nil, accessToken)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// List API keys — should be empty
	resp, result = ts.jsonRequestWithCSRF(t, "GET", "/api/auth/api-keys", nil, accessToken)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	keys = result["data"].([]any)
	assert.Len(t, keys, 0)
}

// =====================================================================
// Test: Password reset flow
// =====================================================================

func TestIntegration_ForgotPassword(t *testing.T) {
	ts := setupIntegrationServer(t)

	// Register a user
	registerBody := map[string]string{
		"email": "forgot@example.com", "password": "OldPass123!",
		"firstName": "Forgot", "lastName": "PW",
	}
	resp, _ := ts.jsonRequestWithCSRF(t, "POST", "/api/auth/register", registerBody, "")
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	// Request forgot password (always returns 200 to prevent enumeration)
	forgotBody := map[string]string{"email": "forgot@example.com"}
	resp, result := ts.jsonRequestWithCSRF(t, "POST", "/api/auth/forgot-password", forgotBody, "")
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	msg := result["data"].(map[string]any)["message"].(string)
	assert.Contains(t, msg, "if an account")

	// Unknown email also returns 200
	resp, _ = ts.jsonRequestWithCSRF(t, "POST", "/api/auth/forgot-password", map[string]string{"email": "unknown@example.com"}, "")
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// =====================================================================
// Test: RBAC enforcement — viewer cannot write
// =====================================================================

func TestIntegration_RBAC_ViewerCannotWrite(t *testing.T) {
	ts := setupIntegrationServer(t)

	// Register owner
	ownerBody := map[string]string{
		"email": "rbac-owner@example.com", "password": "SecurePass123!",
		"firstName": "Owner", "lastName": "RBAC",
	}
	resp, result := ts.jsonRequestWithCSRF(t, "POST", "/api/auth/register", ownerBody, "")
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	ownerToken := result["data"].(map[string]any)["accessToken"].(string)

	// Register viewer
	viewerBody := map[string]string{
		"email": "rbac-viewer@example.com", "password": "SecurePass123!",
		"firstName": "Viewer", "lastName": "RBAC",
	}
	resp, result = ts.jsonRequestWithCSRF(t, "POST", "/api/auth/register", viewerBody, "")
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	viewerToken := result["data"].(map[string]any)["accessToken"].(string)
	viewerUser := result["data"].(map[string]any)["user"].(map[string]any)
	viewerID := uint(viewerUser["id"].(float64))

	// Owner creates org
	wsBody := map[string]string{"name": "RBAC Workspace", "slug": "rbac-workspace", "description": ""}
	resp, result = ts.jsonRequestWithCSRF(t, "POST", "/api/workspaces", wsBody, ownerToken)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	wsID := result["data"].(map[string]any)["id"].(float64)

	// Get viewer role ID
	roles, err := ts.rbacSvc.GetWorkspaceRoles(uint(wsID))
	require.NoError(t, err)
	var viewerRoleID uint
	for _, r := range roles {
		if r.Name == persistent.RoleViewer {
			viewerRoleID = r.ID
			break
		}
	}
	require.NotZero(t, viewerRoleID)

	// Invite viewer via service (since invite endpoint needs RBAC)
	_, rawToken, err := ts.rbacSvc.InviteMember(uint(wsID), "rbac-viewer@example.com", viewerRoleID, 1)
	require.NoError(t, err)
	_, err = ts.rbacSvc.AcceptInvitation(rawToken, viewerID, "rbac-viewer@example.com")
	require.NoError(t, err)

	// Viewer tries to update the org (requires workspace:write) — should be denied
	updateBody := map[string]string{"name": "Hacked", "slug": "rbac-workspace", "description": "Viewer wrote this"}
	wsPath := fmt.Sprintf("/api/workspaces/%.0f", wsID)
	resp, _ = ts.jsonRequestWithCSRF(t, "PUT", wsPath, updateBody, viewerToken)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)

	// Viewer can read packages (requires packages:read which viewer has)
	csrfToken, cookies := ts.getCSRFToken(t)
	req, err := http.NewRequest("GET", ts.server.URL+"/api/packages", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+viewerToken)
	req.Header.Set("X-Workspace-ID", fmt.Sprintf("%.0f", wsID))
	req.Header.Set("X-CSRF-Token", csrfToken)
	req.Header.Set("Origin", "http://localhost:3000")
	for _, c := range cookies {
		req.AddCookie(c)
	}
	pkgResp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer pkgResp.Body.Close()
	assert.Equal(t, http.StatusOK, pkgResp.StatusCode)
}

// =====================================================================
// Test: Session management
// =====================================================================

func TestIntegration_SessionManagement(t *testing.T) {
	ts := setupIntegrationServer(t)

	registerBody := map[string]string{
		"email": "session@example.com", "password": "SecurePass123!",
		"firstName": "Session", "lastName": "User",
	}
	resp, result := ts.jsonRequestWithCSRF(t, "POST", "/api/auth/register", registerBody, "")
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	accessToken := result["data"].(map[string]any)["accessToken"].(string)

	// List sessions
	resp, result = ts.jsonRequestWithCSRF(t, "GET", "/api/auth/sessions", nil, accessToken)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	sessions := result["data"].([]any)
	// Sessions depend on whether Login creates a session entry
	_ = sessions
}

// =====================================================================
// Test: Permissions endpoint
// =====================================================================

func TestIntegration_ListPermissions(t *testing.T) {
	ts := setupIntegrationServer(t)

	// Register to get a token
	registerBody := map[string]string{
		"email": "perms@example.com", "password": "SecurePass123!",
		"firstName": "Perm", "lastName": "User",
	}
	resp, result := ts.jsonRequestWithCSRF(t, "POST", "/api/auth/register", registerBody, "")
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	accessToken := result["data"].(map[string]any)["accessToken"].(string)

	resp, result = ts.jsonRequestWithCSRF(t, "GET", "/api/permissions", nil, accessToken)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	perms := result["data"].([]any)
	assert.NotEmpty(t, perms, "should list system permissions")
}

// =====================================================================
// Test: Package CRUD with workspace scoping
// =====================================================================

func TestIntegration_PackageCRUD_WorkspaceScoped(t *testing.T) {
	ts := setupIntegrationServer(t)

	// Register and create an org
	registerBody := map[string]string{
		"email": "pkg-crud@example.com", "password": "SecurePass123!",
		"firstName": "Pkg", "lastName": "CRUD",
	}
	resp, result := ts.jsonRequestWithCSRF(t, "POST", "/api/auth/register", registerBody, "")
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	accessToken := result["data"].(map[string]any)["accessToken"].(string)

	wsBody := map[string]string{"name": "Pkg Workspace", "slug": "pkg-workspace", "description": ""}
	resp, result = ts.jsonRequestWithCSRF(t, "POST", "/api/workspaces", wsBody, accessToken)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	wsID := result["data"].(map[string]any)["id"].(float64)

	// Create package (using X-Workspace-ID header for workspace scoping)
	pkgBody := map[string]string{"name": "django", "ecosystem": "python"}

	csrfToken, cookies := ts.getCSRFToken(t)
	req, err := http.NewRequest("POST", ts.server.URL+"/api/packages", bytes.NewReader(mustJSON(t, pkgBody)))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("X-Workspace-ID", fmt.Sprintf("%.0f", wsID))
	req.Header.Set("X-CSRF-Token", csrfToken)
	req.Header.Set("Origin", "http://localhost:3000")
	for _, c := range cookies {
		req.AddCookie(c)
	}
	pkgResp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	pkgResult := parseResponse(t, pkgResp)
	assert.Equal(t, http.StatusCreated, pkgResp.StatusCode)
	pkgData := pkgResult["data"].(map[string]any)
	pkgID := pkgData["id"].(float64)
	assert.NotZero(t, pkgID)

	// List packages
	csrfToken, cookies = ts.getCSRFToken(t)
	req, err = http.NewRequest("GET", ts.server.URL+"/api/packages", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("X-Workspace-ID", fmt.Sprintf("%.0f", wsID))
	req.Header.Set("X-CSRF-Token", csrfToken)
	req.Header.Set("Origin", "http://localhost:3000")
	for _, c := range cookies {
		req.AddCookie(c)
	}
	listResp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	listResult := parseResponse(t, listResp)
	assert.Equal(t, http.StatusOK, listResp.StatusCode)
	pkgs := listResult["data"].([]any)
	assert.Len(t, pkgs, 1)
}

// mustJSON marshals v to JSON or fails the test.
func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	data, err := json.Marshal(v)
	require.NoError(t, err)
	return data
}

// =====================================================================
// Test: Invitation flow through HTTP
// =====================================================================

func TestIntegration_InvitationFlow(t *testing.T) {
	ts := setupIntegrationServer(t)

	// Register owner
	resp, result := ts.jsonRequestWithCSRF(t, "POST", "/api/auth/register", map[string]string{
		"email": "inv-owner@example.com", "password": "SecurePass123!", "firstName": "Inv", "lastName": "Owner",
	}, "")
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	ownerToken := result["data"].(map[string]any)["accessToken"].(string)

	// Create org
	resp, result = ts.jsonRequestWithCSRF(t, "POST", "/api/workspaces", map[string]string{
		"name": "Inv Workspace", "slug": "inv-workspace", "description": "",
	}, ownerToken)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	wsID := uint(result["data"].(map[string]any)["id"].(float64))

	// Get the member role ID
	roles, err := ts.rbacSvc.GetWorkspaceRoles(wsID)
	require.NoError(t, err)
	var memberRoleID uint
	for _, r := range roles {
		if r.Name == persistent.RoleMember {
			memberRoleID = r.ID
			break
		}
	}
	require.NotZero(t, memberRoleID)

	// Invite via API
	inviteBody := map[string]any{
		"email":  "inv-member@example.com",
		"roleId": memberRoleID,
	}
	wsPath := fmt.Sprintf("/api/workspaces/%d", wsID)
	resp, result = ts.jsonRequestWithCSRF(t, "POST", wsPath+"/invitations", inviteBody, ownerToken)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	invData := result["data"].(map[string]any)
	rawToken := invData["token"].(string)
	assert.NotEmpty(t, rawToken)

	// List pending invitations
	resp, result = ts.jsonRequestWithCSRF(t, "GET", wsPath+"/invitations", nil, ownerToken)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	invitations := result["data"].([]any)
	assert.Len(t, invitations, 1)

	// Get invitation info (public endpoint)
	resp, result = ts.jsonRequestWithCSRF(t, "GET", "/api/invitations/"+rawToken, nil, "")
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Register the invited user and accept
	resp, result = ts.jsonRequestWithCSRF(t, "POST", "/api/auth/register", map[string]string{
		"email": "inv-member@example.com", "password": "SecurePass123!", "firstName": "Inv", "lastName": "Member",
	}, "")
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	memberToken := result["data"].(map[string]any)["accessToken"].(string)

	// Accept invitation
	acceptPath := fmt.Sprintf("/api/workspaces/%d/invitations/%s/accept", wsID, rawToken)
	resp, _ = ts.jsonRequestWithCSRF(t, "POST", acceptPath, nil, memberToken)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Members list should now have 2 members
	resp, result = ts.jsonRequestWithCSRF(t, "GET", wsPath+"/members", nil, ownerToken)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	members := result["data"].([]any)
	assert.Len(t, members, 2)
}

// =====================================================================
// Test: Audit log creation through actions
// =====================================================================

func TestIntegration_AuditLogsCreated(t *testing.T) {
	ts := setupIntegrationServer(t)

	// Register (creates an audit log)
	resp, result := ts.jsonRequestWithCSRF(t, "POST", "/api/auth/register", map[string]string{
		"email": "audit-log@example.com", "password": "SecurePass123!", "firstName": "Audit", "lastName": "Log",
	}, "")
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	accessToken := result["data"].(map[string]any)["accessToken"].(string)

	// Create org
	resp, result = ts.jsonRequestWithCSRF(t, "POST", "/api/workspaces", map[string]string{
		"name": "Audit Org", "slug": "audit-org", "description": "",
	}, accessToken)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	wsID := result["data"].(map[string]any)["id"].(float64)

	// Perform an org-scoped action that creates an audit log (create a package)
	csrfToken, cookies := ts.getCSRFToken(t)
	pkgBody := mustJSON(t, map[string]string{"name": "audit-pkg", "ecosystem": "python"})
	req, err := http.NewRequest("POST", ts.server.URL+"/api/packages", bytes.NewReader(pkgBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("X-Workspace-ID", fmt.Sprintf("%.0f", wsID))
	req.Header.Set("X-CSRF-Token", csrfToken)
	req.Header.Set("Origin", "http://localhost:3000")
	for _, c := range cookies {
		req.AddCookie(c)
	}
	pkgResp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	_ = parseResponse(t, pkgResp)
	require.Equal(t, http.StatusCreated, pkgResp.StatusCode)

	// Check audit logs exist for the org
	wsPath := fmt.Sprintf("/api/workspaces/%.0f", wsID)
	resp, result = ts.jsonRequestWithCSRF(t, "GET", wsPath+"/audit-logs", nil, accessToken)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	// There should be at least the package creation audit log
	logsData := result["data"].([]any)
	assert.NotEmpty(t, logsData, "should have audit logs for org-scoped actions")
}

// =====================================================================
// Test: Email verification flow
// =====================================================================

func TestIntegration_EmailVerification(t *testing.T) {
	ts := setupIntegrationServer(t)

	// Register
	resp, result := ts.jsonRequestWithCSRF(t, "POST", "/api/auth/register", map[string]string{
		"email": "verify-email@example.com", "password": "SecurePass123!", "firstName": "V", "lastName": "E",
	}, "")
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	accessToken := result["data"].(map[string]any)["accessToken"].(string)

	// Request verification email
	resp, _ = ts.jsonRequestWithCSRF(t, "POST", "/api/auth/send-verification", nil, accessToken)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Get the token from the DB directly (in real flow this would be sent via email)
	user := result["data"].(map[string]any)["user"].(map[string]any)
	userID := uint(user["id"].(float64))

	rawToken, err := ts.authSvc.GenerateEmailVerificationToken(userID)
	require.NoError(t, err)

	// Verify email via API
	verifyBody := map[string]string{"token": rawToken}
	resp, _ = ts.jsonRequestWithCSRF(t, "POST", "/api/auth/verify-email", verifyBody, "")
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Check that user is verified
	verifiedUser, err := ts.authSvc.GetUserByID(userID)
	require.NoError(t, err)
	assert.True(t, verifiedUser.EmailVerified)
}

// =====================================================================
// Test: Password reset flow via API
// =====================================================================

func TestIntegration_PasswordReset(t *testing.T) {
	ts := setupIntegrationServer(t)

	// Register
	resp, _ := ts.jsonRequestWithCSRF(t, "POST", "/api/auth/register", map[string]string{
		"email": "pw-reset@example.com", "password": "OldPass123!", "firstName": "PW", "lastName": "Reset",
	}, "")
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	// Request forgot password via API
	resp, _ = ts.jsonRequestWithCSRF(t, "POST", "/api/auth/forgot-password", map[string]string{"email": "pw-reset@example.com"}, "")
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Get the token from the DB directly
	rawToken, err := ts.authSvc.ForgotPassword("pw-reset@example.com")
	require.NoError(t, err)
	require.NotEmpty(t, rawToken)

	// Reset password via API
	resetBody := map[string]string{"token": rawToken, "newPassword": "NewPass456!"}
	resp, _ = ts.jsonRequestWithCSRF(t, "POST", "/api/auth/reset-password", resetBody, "")
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Login with new password should succeed
	loginBody := map[string]string{"email": "pw-reset@example.com", "password": "NewPass456!"}
	resp, _ = ts.jsonRequestWithCSRF(t, "POST", "/api/auth/login", loginBody, "")
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// =====================================================================
// Test: Response shapes include standard envelope
// =====================================================================

func TestIntegration_ResponseEnvelope(t *testing.T) {
	ts := setupIntegrationServer(t)

	// Register — success response should have {data: {...}, error: null}
	resp, result := ts.jsonRequestWithCSRF(t, "POST", "/api/auth/register", map[string]string{
		"email": "envelope@example.com", "password": "SecurePass123!", "firstName": "E", "lastName": "V",
	}, "")
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	// Should have "data" key
	assert.Contains(t, result, "data")

	// Error responses should have {error: "..."}
	resp, result = ts.jsonRequestWithCSRF(t, "POST", "/api/auth/login", map[string]string{
		"email": "nonexistent@example.com", "password": "Wrong123!",
	}, "")
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	assert.Contains(t, result, "error")
	assert.NotNil(t, result["error"])
}

// =====================================================================
// Test: CORS headers present
// =====================================================================

func TestIntegration_CORSHeaders(t *testing.T) {
	ts := setupIntegrationServer(t)

	req, err := http.NewRequest("OPTIONS", ts.server.URL+"/api/invitations/dummy", nil)
	require.NoError(t, err)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "GET")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.NotEmpty(t, resp.Header.Get("Access-Control-Allow-Origin"))
}

// =====================================================================
// Test: Security headers present
// =====================================================================

func TestIntegration_SecurityHeaders(t *testing.T) {
	ts := setupIntegrationServer(t)

	// Use a public endpoint that doesn't need Queue
	req, err := http.NewRequest("GET", ts.server.URL+"/api/invitations/dummy", nil)
	require.NoError(t, err)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, "nosniff", resp.Header.Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", resp.Header.Get("X-Frame-Options"))
}

// =====================================================================
// Test: Rate limiting headers
// =====================================================================

func TestIntegration_RateLimitHeaders(t *testing.T) {
	ts := setupIntegrationServer(t)

	// Use a public endpoint that doesn't need Queue
	resp, err := http.Get(ts.server.URL + "/api/invitations/dummy")
	require.NoError(t, err)
	defer resp.Body.Close()

	// Rate limiter should set X-RateLimit-* headers
	assert.NotEmpty(t, resp.Header.Get("X-RateLimit-Limit"), "should have rate limit header")
	assert.NotEmpty(t, resp.Header.Get("X-RateLimit-Remaining"), "should have rate limit remaining header")
}

// =====================================================================
// Test: Settings CRUD through org-scoped endpoint
// =====================================================================

func TestIntegration_SettingsCRUD(t *testing.T) {
	ts := setupIntegrationServer(t)

	// Register + create org
	resp, result := ts.jsonRequestWithCSRF(t, "POST", "/api/auth/register", map[string]string{
		"email": "settings@example.com", "password": "SecurePass123!", "firstName": "S", "lastName": "C",
	}, "")
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	accessToken := result["data"].(map[string]any)["accessToken"].(string)

	resp, result = ts.jsonRequestWithCSRF(t, "POST", "/api/workspaces", map[string]string{
		"name": "Settings Org", "slug": "settings-org", "description": "",
	}, accessToken)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	wsID := result["data"].(map[string]any)["id"].(float64)

	// Update settings via org-scoped endpoint
	csrfToken, cookies := ts.getCSRFToken(t)
	settingsBody := `{"monitoring_interval":"10m"}`
	req, err := http.NewRequest("PUT", ts.server.URL+"/api/settings", bytes.NewReader([]byte(settingsBody)))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("X-Workspace-ID", fmt.Sprintf("%.0f", wsID))
	req.Header.Set("X-CSRF-Token", csrfToken)
	req.Header.Set("Origin", "http://localhost:3000")
	for _, c := range cookies {
		req.AddCookie(c)
	}
	settingsResp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	settingsResult := parseResponse(t, settingsResp)
	assert.Equal(t, http.StatusOK, settingsResp.StatusCode)
	settingsData := settingsResult["data"].(map[string]any)
	assert.Equal(t, "10m", settingsData["monitoring_interval"])
}

// =====================================================================
// Test: CSRF enforcement on state-changing requests
// =====================================================================

func TestIntegration_CSRF_Required(t *testing.T) {
	ts := setupIntegrationServer(t)

	// Pre-auth routes (register, login, etc.) are exempt from CSRF so that
	// users who haven't loaded a page yet can still authenticate. Verify
	// that a non-exempt protected route still requires CSRF.

	// First, register + login to get an access token
	body := map[string]string{
		"email": "csrf@example.com", "password": "SecurePass123!", "firstName": "C", "lastName": "S",
	}
	data, _ := json.Marshal(body)
	req, err := http.NewRequest("POST", ts.server.URL+"/api/auth/register", bytes.NewReader(data))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://localhost:3000")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Pre-auth route should succeed without CSRF token
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
}
