package auth_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/veilence/veilence-mx/backend/internal/usecase/auth"
	"github.com/veilence/veilence-mx/backend/internal/entity"
)

// dummyHandler is a simple handler that returns 200 OK for testing middleware.
func dummyHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := auth.UserIDFromContext(r.Context())
		email := auth.EmailFromContext(r.Context())
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"userId": userID,
			"email":  email,
		})
	}
}

func TestMiddleware_ValidJWT(t *testing.T) {
	db := setupAuthTestDB(t)
	svc := newAuthService(db)

	_, err := svc.Register("mw-jwt@example.com", "Password123", "MW", "User")
	require.NoError(t, err)

	_, tokens, err := svc.Login("mw-jwt@example.com", "Password123", "127.0.0.1", "TestBrowser/1.0")
	require.NoError(t, err)

	handler := auth.Middleware(svc)(dummyHandler())

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, "mw-jwt@example.com", resp["email"])
	assert.NotZero(t, resp["userId"])
}

func TestMiddleware_ExpiredToken(t *testing.T) {
	db := setupAuthTestDB(t)
	svc := newAuthService(db)

	handler := auth.Middleware(svc)(dummyHandler())

	// Use a made-up expired-looking token
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoxLCJlbWFpbCI6InRlc3RAZXhhbXBsZS5jb20iLCJ0b2tlbl90eXBlIjoiYWNjZXNzIiwiZXhwIjoxMDAwMDAwMDAwfQ.invalid")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.NotNil(t, resp["error"])
}

func TestMiddleware_NoToken(t *testing.T) {
	db := setupAuthTestDB(t)
	svc := newAuthService(db)

	handler := auth.Middleware(svc)(dummyHandler())

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	errVal, ok := resp["error"]
	assert.True(t, ok)
	assert.NotNil(t, errVal)
}

func TestMiddleware_ValidAPIKey(t *testing.T) {
	db := setupAuthTestDB(t)
	svc := newAuthService(db)

	user, err := svc.Register("mw-apikey@example.com", "Password123", "MW", "APIKey")
	require.NoError(t, err)

	_, rawKey, err := svc.CreateAPIKey(user.ID, "test-key", entity.APIKeyScopeRead, nil)
	require.NoError(t, err)

	handler := auth.Middleware(svc)(dummyHandler())

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-API-Key", rawKey)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, "mw-apikey@example.com", resp["email"])
}

func TestMiddleware_InvalidAPIKey(t *testing.T) {
	db := setupAuthTestDB(t)
	svc := newAuthService(db)

	handler := auth.Middleware(svc)(dummyHandler())

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-API-Key", "vmx_invalidkey_does_not_exist_1234567890")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestMiddleware_InvalidBearerFormat(t *testing.T) {
	db := setupAuthTestDB(t)
	svc := newAuthService(db)

	handler := auth.Middleware(svc)(dummyHandler())

	// Authorization header without "Bearer " prefix should fall through to no auth
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
