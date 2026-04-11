package testutil

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/veilence/veilence-mx/backend/internal/auth"
	"github.com/veilence/veilence-mx/backend/internal/rbac"
	"github.com/veilence/veilence-mx/backend/internal/repository"
)

// TestJWTSecret is a shared JWT secret for test environments.
const TestJWTSecret = "test-secret-key-for-jwt-signing-1234567890"

// TestServer wraps an httptest.Server with helpers for integration testing.
type TestServer struct {
	// Server is the underlying httptest.Server.
	Server *httptest.Server
	// Router is the Chi router the server is using.
	Router chi.Router
	// DB is the test database backing the server.
	DB *gorm.DB
	// AuthService is the auth service for creating tokens.
	AuthService *auth.Service
}

// Close shuts down the test server.
func (ts *TestServer) Close() {
	ts.Server.Close()
}

// URL returns the base URL of the test server.
func (ts *TestServer) URL() string {
	return ts.Server.URL
}

// NewTestRouter creates a Chi router suitable for integration tests. The
// returned router does NOT include auth or RBAC middleware — the caller can
// add specific middleware as needed for each test.
func NewTestRouter() chi.Router {
	return chi.NewRouter()
}

// ---------------------------------------------------------------------------
// Auth service factory
// ---------------------------------------------------------------------------

// NewTestAuthService creates an auth.Service backed by the given database
// using the shared TestJWTSecret. The database must already have user-related
// tables migrated (use SetupTestDB).
func NewTestAuthService(db *gorm.DB) *auth.Service {
	return auth.NewService(
		repository.NewUserRepo(db),
		repository.NewRefreshTokenRepo(db),
		repository.NewAPIKeyRepo(db),
		repository.NewPasswordResetTokenRepo(db),
		repository.NewEmailVerificationTokenRepo(db),
		repository.NewSessionRepo(db),
		TestJWTSecret,
	)
}

// ---------------------------------------------------------------------------
// Context helpers
// ---------------------------------------------------------------------------

// WithOrgContext returns a context with the org ID set, matching what the
// RequireOrg middleware does.
func WithOrgContext(ctx context.Context, orgID uint) context.Context {
	return rbac.WithOrgID(ctx, orgID)
}

// ---------------------------------------------------------------------------
// Middleware factories for handler tests
// ---------------------------------------------------------------------------

// InjectOrg creates a middleware that injects the org context.
// This replaces the real RequireOrg middleware in handler tests.
func InjectOrg(orgID uint) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := rbac.WithOrgID(r.Context(), orgID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// InjectAuthAndOrg creates a middleware that runs a real auth middleware
// for the given service AND injects org context. This is useful for
// integration tests that need both auth + org scoping.
func InjectAuthAndOrg(svc *auth.Service, orgID uint) func(http.Handler) http.Handler {
	authMW := auth.Middleware(svc)
	return func(next http.Handler) http.Handler {
		return authMW(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := rbac.WithOrgID(r.Context(), orgID)
			next.ServeHTTP(w, r.WithContext(ctx))
		}))
	}
}

// ---------------------------------------------------------------------------
// Test user helpers
// ---------------------------------------------------------------------------

// LoginTestUser registers a user, logs them in, and returns the access token.
// This is the recommended way to get an auth token for integration tests.
func LoginTestUser(t *testing.T, svc *auth.Service, email, password, firstName, lastName string) (userID uint, accessToken string) {
	t.Helper()

	user, err := svc.Register(email, password, firstName, lastName)
	require.NoError(t, err, "failed to register test user")

	_, tokens, err := svc.Login(email, password, "127.0.0.1", "TestBrowser/1.0")
	require.NoError(t, err, "failed to login test user")

	return user.ID, tokens.AccessToken
}

// ---------------------------------------------------------------------------
// Request builders
// ---------------------------------------------------------------------------

// NewAuthenticatedRequest creates an HTTP request with the given access token
// set as a Bearer token in the Authorization header.
func NewAuthenticatedRequest(method, url, accessToken string, body string) *http.Request {
	var bodyReader io.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, url, bodyReader)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	return req
}

// NewRequestWithOrgHeader creates an HTTP request with X-Org-ID header set.
func NewRequestWithOrgHeader(method, url string, orgID uint, body string) *http.Request {
	var bodyReader io.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, url, bodyReader)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Org-ID", fmt.Sprintf("%d", orgID))
	return req
}

// IDStr converts a uint ID to a string, commonly needed for URL construction.
func IDStr(id uint) string {
	return fmt.Sprintf("%d", id)
}
