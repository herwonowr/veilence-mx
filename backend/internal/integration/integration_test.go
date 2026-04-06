package integration_test

import (
	"bytes"
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

	"github.com/veilence/veilence-mx/backend/internal/api"
	"github.com/veilence/veilence-mx/backend/internal/api/handlers"
	"github.com/veilence/veilence-mx/backend/internal/audit"
	"github.com/veilence/veilence-mx/backend/internal/auth"
	"github.com/veilence/veilence-mx/backend/internal/models"
	"github.com/veilence/veilence-mx/backend/internal/notifications"
	"github.com/veilence/veilence-mx/backend/internal/rbac"
	"github.com/veilence/veilence-mx/backend/internal/repository"
)

const testJWTSecret = "integration-test-jwt-secret-very-long-key-1234567890"

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
		&models.User{},
		&models.RefreshToken{},
		&models.APIKey{},
		&models.PasswordResetToken{},
		&models.EmailVerificationToken{},
		&models.Session{},
		&models.Organization{},
		&models.Role{},
		&models.Permission{},
		&models.OrgMember{},
		&models.Invitation{},
		&models.Package{},
		&models.Release{},
		&models.Diff{},
		&models.Analysis{},
		&models.Alert{},
		&models.Setting{},
		&models.AuditLog{},
		&models.NotificationChannel{},
		&models.NotificationRule{},
		&models.Notification{},
	)
	require.NoError(t, err)

	// Seed permissions for RBAC
	err = rbac.SeedPermissions(db)
	require.NoError(t, err)

	// Create services
	userRepo := repository.NewUserRepo(db)
	refreshTokenRepo := repository.NewRefreshTokenRepo(db)
	apiKeyRepo := repository.NewAPIKeyRepo(db)
	passwordResetTokenRepo := repository.NewPasswordResetTokenRepo(db)
	emailVerificationTokenRepo := repository.NewEmailVerificationTokenRepo(db)
	sessionRepo := repository.NewSessionRepo(db)

	authSvc := auth.NewService(userRepo, refreshTokenRepo, apiKeyRepo, passwordResetTokenRepo, emailVerificationTokenRepo, sessionRepo, testJWTSecret)
	rbacSvc := rbac.NewService(db)
	auditSvc := audit.NewService(db)

	channelRepo := repository.NewNotificationChannelRepo(db)
	ruleRepo := repository.NewNotificationRuleRepo(db)
	notifRepo := repository.NewNotificationRepo(db)
	notifSvc := notifications.NewService(channelRepo, ruleRepo, notifRepo, notifications.SMTPConfig{})

	dashboardRepo := repository.NewDashboardRepo(db)

	h := &handlers.Handlers{
		Auth: &handlers.AuthHandlers{
			Auth:  authSvc,
			Audit: auditSvc,
		},
		Sessions: &handlers.SessionHandlers{
			Auth: authSvc,
		},
		Notifications: &handlers.NotificationHandlers{
			Notifications: notifSvc,
			Audit:         auditSvc,
		},
		Org: &handlers.OrgHandlers{
			RBAC:  rbacSvc,
			Audit: auditSvc,
		},
		AuditLogs: &handlers.AuditHandlers{
			Audit: auditSvc,
		},
		Packages: &handlers.PackageHandlers{
			DB:    db,
			Audit: auditSvc,
		},
		Alerts: &handlers.AlertHandlers{
			DB:    db,
			Audit: auditSvc,
		},
		Settings: &handlers.SettingsHandlers{
			DB:    db,
			Audit: auditSvc,
		},
		Dashboard: &handlers.DashboardHandlers{
			DB:        db,
			Dashboard: dashboardRepo,
		},
		Health: &handlers.HealthHandlers{
			DB: db,
		},
	}

	router := api.NewRouter(h, "http://localhost:3000", authSvc, rbacSvc)
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
// Test: Organization lifecycle (Create → Get → List → Update → Delete)
// =====================================================================

func TestIntegration_OrgLifecycle(t *testing.T) {
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

	// 1. Create organization
	orgBody := map[string]string{
		"name": "Test Org", "slug": "test-org", "description": "Integration test org",
	}
	resp, result = ts.jsonRequestWithCSRF(t, "POST", "/api/orgs", orgBody, accessToken)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	orgData := result["data"].(map[string]any)
	orgID := orgData["id"].(float64)
	assert.NotZero(t, orgID)
	assert.Equal(t, "Test Org", orgData["name"])
	assert.Equal(t, "test-org", orgData["slug"])

	// 2. List organizations
	resp, result = ts.jsonRequestWithCSRF(t, "GET", "/api/orgs", nil, accessToken)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	orgs := result["data"].([]any)
	assert.Len(t, orgs, 1)

	// 3. Get organization
	orgPath := fmt.Sprintf("/api/orgs/%.0f", orgID)
	resp, result = ts.jsonRequestWithCSRF(t, "GET", orgPath, nil, accessToken)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// 4. Update organization
	updateBody := map[string]string{
		"name": "Updated Org", "slug": "updated-org", "description": "Updated desc",
	}
	resp, result = ts.jsonRequestWithCSRF(t, "PUT", orgPath, updateBody, accessToken)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	updatedOrg := result["data"].(map[string]any)
	assert.Equal(t, "Updated Org", updatedOrg["name"])

	// 5. List members (owner should be listed)
	resp, result = ts.jsonRequestWithCSRF(t, "GET", orgPath+"/members", nil, accessToken)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	members := result["data"].([]any)
	assert.Len(t, members, 1)

	// 6. List roles
	resp, result = ts.jsonRequestWithCSRF(t, "GET", orgPath+"/roles", nil, accessToken)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	roles := result["data"].([]any)
	assert.Len(t, roles, 4, "owner, admin, member, viewer")

	// 7. Delete organization
	resp, _ = ts.jsonRequestWithCSRF(t, "DELETE", orgPath, nil, accessToken)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// 8. Organization should no longer be accessible
	resp, _ = ts.jsonRequestWithCSRF(t, "GET", orgPath, nil, accessToken)
	assert.NotEqual(t, http.StatusOK, resp.StatusCode)
}

// =====================================================================
// Test: Duplicate org slug
// =====================================================================

func TestIntegration_OrgCreate_DuplicateSlug(t *testing.T) {
	ts := setupIntegrationServer(t)

	registerBody := map[string]string{
		"email": "dup-slug@example.com", "password": "SecurePass123!",
		"firstName": "Dup", "lastName": "Slug",
	}
	resp, result := ts.jsonRequestWithCSRF(t, "POST", "/api/auth/register", registerBody, "")
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	accessToken := result["data"].(map[string]any)["accessToken"].(string)

	orgBody := map[string]string{"name": "Org1", "slug": "my-slug", "description": ""}
	resp, _ = ts.jsonRequestWithCSRF(t, "POST", "/api/orgs", orgBody, accessToken)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	resp, _ = ts.jsonRequestWithCSRF(t, "POST", "/api/orgs", orgBody, accessToken)
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

	// Create API key
	keyBody := map[string]string{"name": "test-key", "scope": "read"}
	resp, result = ts.jsonRequestWithCSRF(t, "POST", "/api/auth/api-keys", keyBody, accessToken)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
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
	csrfToken, cookies := ts.getCSRFToken(t)
	req, err := http.NewRequest("GET", ts.server.URL+"/api/auth/me", nil)
	require.NoError(t, err)
	req.Header.Set("X-API-Key", rawKey)
	req.Header.Set("X-CSRF-Token", csrfToken)
	req.Header.Set("Origin", "http://localhost:3000")
	for _, c := range cookies {
		req.AddCookie(c)
	}
	apiResp, err := http.DefaultClient.Do(req)
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
	orgBody := map[string]string{"name": "RBAC Org", "slug": "rbac-org", "description": ""}
	resp, result = ts.jsonRequestWithCSRF(t, "POST", "/api/orgs", orgBody, ownerToken)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	orgID := result["data"].(map[string]any)["id"].(float64)

	// Get viewer role ID
	roles, err := ts.rbacSvc.GetOrgRoles(uint(orgID))
	require.NoError(t, err)
	var viewerRoleID uint
	for _, r := range roles {
		if r.Name == models.RoleViewer {
			viewerRoleID = r.ID
			break
		}
	}
	require.NotZero(t, viewerRoleID)

	// Invite viewer via service (since invite endpoint needs RBAC)
	_, rawToken, err := ts.rbacSvc.InviteMember(uint(orgID), "rbac-viewer@example.com", viewerRoleID, 1)
	require.NoError(t, err)
	_, err = ts.rbacSvc.AcceptInvitation(rawToken, viewerID, "rbac-viewer@example.com")
	require.NoError(t, err)

	// Viewer tries to update the org (requires org:write) — should be denied
	updateBody := map[string]string{"name": "Hacked", "slug": "rbac-org", "description": "Viewer wrote this"}
	orgPath := fmt.Sprintf("/api/orgs/%.0f", orgID)
	resp, _ = ts.jsonRequestWithCSRF(t, "PUT", orgPath, updateBody, viewerToken)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)

	// Viewer can read packages (requires packages:read which viewer has)
	csrfToken, cookies := ts.getCSRFToken(t)
	req, err := http.NewRequest("GET", ts.server.URL+"/api/packages", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+viewerToken)
	req.Header.Set("X-Org-ID", fmt.Sprintf("%.0f", orgID))
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
// Test: Package CRUD with org scoping
// =====================================================================

func TestIntegration_PackageCRUD_OrgScoped(t *testing.T) {
	ts := setupIntegrationServer(t)

	// Register and create an org
	registerBody := map[string]string{
		"email": "pkg-crud@example.com", "password": "SecurePass123!",
		"firstName": "Pkg", "lastName": "CRUD",
	}
	resp, result := ts.jsonRequestWithCSRF(t, "POST", "/api/auth/register", registerBody, "")
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	accessToken := result["data"].(map[string]any)["accessToken"].(string)

	orgBody := map[string]string{"name": "Pkg Org", "slug": "pkg-org", "description": ""}
	resp, result = ts.jsonRequestWithCSRF(t, "POST", "/api/orgs", orgBody, accessToken)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	orgID := result["data"].(map[string]any)["id"].(float64)

	// Create package (using X-Org-ID header for org scoping)
	pkgBody := map[string]string{"name": "django", "registry": "pypi"}

	csrfToken, cookies := ts.getCSRFToken(t)
	req, err := http.NewRequest("POST", ts.server.URL+"/api/packages", bytes.NewReader(mustJSON(t, pkgBody)))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("X-Org-ID", fmt.Sprintf("%.0f", orgID))
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
	req.Header.Set("X-Org-ID", fmt.Sprintf("%.0f", orgID))
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
	resp, result = ts.jsonRequestWithCSRF(t, "POST", "/api/orgs", map[string]string{
		"name": "Inv Org", "slug": "inv-org", "description": "",
	}, ownerToken)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	orgID := uint(result["data"].(map[string]any)["id"].(float64))

	// Get the member role ID
	roles, err := ts.rbacSvc.GetOrgRoles(orgID)
	require.NoError(t, err)
	var memberRoleID uint
	for _, r := range roles {
		if r.Name == models.RoleMember {
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
	orgPath := fmt.Sprintf("/api/orgs/%d", orgID)
	resp, result = ts.jsonRequestWithCSRF(t, "POST", orgPath+"/invitations", inviteBody, ownerToken)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	invData := result["data"].(map[string]any)
	rawToken := invData["token"].(string)
	assert.NotEmpty(t, rawToken)

	// List pending invitations
	resp, result = ts.jsonRequestWithCSRF(t, "GET", orgPath+"/invitations", nil, ownerToken)
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
	acceptPath := fmt.Sprintf("/api/orgs/%d/invitations/%s/accept", orgID, rawToken)
	resp, _ = ts.jsonRequestWithCSRF(t, "POST", acceptPath, nil, memberToken)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Members list should now have 2 members
	resp, result = ts.jsonRequestWithCSRF(t, "GET", orgPath+"/members", nil, ownerToken)
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
	resp, result = ts.jsonRequestWithCSRF(t, "POST", "/api/orgs", map[string]string{
		"name": "Audit Org", "slug": "audit-org", "description": "",
	}, accessToken)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	orgID := result["data"].(map[string]any)["id"].(float64)

	// Perform an org-scoped action that creates an audit log (create a package)
	csrfToken, cookies := ts.getCSRFToken(t)
	pkgBody := mustJSON(t, map[string]string{"name": "audit-pkg", "registry": "pypi"})
	req, err := http.NewRequest("POST", ts.server.URL+"/api/packages", bytes.NewReader(pkgBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("X-Org-ID", fmt.Sprintf("%.0f", orgID))
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
	orgPath := fmt.Sprintf("/api/orgs/%.0f", orgID)
	resp, result = ts.jsonRequestWithCSRF(t, "GET", orgPath+"/audit-logs", nil, accessToken)
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

	resp, result = ts.jsonRequestWithCSRF(t, "POST", "/api/orgs", map[string]string{
		"name": "Settings Org", "slug": "settings-org", "description": "",
	}, accessToken)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	orgID := result["data"].(map[string]any)["id"].(float64)

	// Update settings via org-scoped endpoint
	csrfToken, cookies := ts.getCSRFToken(t)
	settingsBody := `{"pypi_poll_interval":"10m"}`
	req, err := http.NewRequest("PUT", ts.server.URL+"/api/settings", bytes.NewReader([]byte(settingsBody)))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("X-Org-ID", fmt.Sprintf("%.0f", orgID))
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
	assert.Equal(t, "10m", settingsData["pypi_poll_interval"])
}

// =====================================================================
// Test: CSRF enforcement on state-changing requests
// =====================================================================

func TestIntegration_CSRF_Required(t *testing.T) {
	ts := setupIntegrationServer(t)

	// POST without CSRF token should be rejected
	body := map[string]string{
		"email": "csrf@example.com", "password": "SecurePass123!", "firstName": "C", "lastName": "S",
	}
	data, _ := json.Marshal(body)
	req, err := http.NewRequest("POST", ts.server.URL+"/api/auth/register", bytes.NewReader(data))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://localhost:3000")
	// No CSRF token or cookie

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Should be forbidden (CSRF protection)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}
