package middleware_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/veilence/veilence-mx/backend/internal/controller/restapi/middleware"
)

func okHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}
}

func TestRateLimiter_AllowsWithinLimit(t *testing.T) {
	rl := middleware.NewRateLimiter(100)
	handler := rl.Limit(okHandler())

	// Should allow requests within the limit
	for i := range 10 {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code, "request %d should be allowed", i+1)
	}
}

func TestRateLimiter_BlocksOverLimit(t *testing.T) {
	// Very low limit for testing
	rl := middleware.NewRateLimiter(5)
	handler := rl.Limit(okHandler())

	// Exhaust the limit
	for range 5 {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "10.0.0.1:12345"
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	}

	// Next request should be blocked
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusTooManyRequests, w.Code)

	// Check response body
	var resp map[string]any
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.NotNil(t, resp["error"])

	// Check Retry-After header
	retryAfter := w.Header().Get("Retry-After")
	assert.NotEmpty(t, retryAfter)
}

func TestRateLimiter_DifferentIPsIndependent(t *testing.T) {
	rl := middleware.NewRateLimiter(3)
	handler := rl.Limit(okHandler())

	// IP 1 uses up limit
	for range 3 {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "1.1.1.1:12345"
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	}

	// IP 1 should be blocked
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "1.1.1.1:12345"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTooManyRequests, w.Code)

	// IP 2 should still be allowed
	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "2.2.2.2:12345"
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRateLimiter_ResponseFormat(t *testing.T) {
	rl := middleware.NewRateLimiter(1)
	handler := rl.Limit(okHandler())

	// First request passes
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "3.3.3.3:12345"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Second request should be rate limited with correct format
	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "3.3.3.3:12345"
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusTooManyRequests, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var resp map[string]any
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)

	// Should match the APIResponse error format
	errMsg, ok := resp["error"]
	assert.True(t, ok, "response should have 'error' field")
	assert.Equal(t, "rate limit exceeded", errMsg)
}

// --- X-RateLimit Header Tests ---

func TestRateLimiter_XRateLimitHeaders_OnSuccess(t *testing.T) {
	rl := middleware.NewRateLimiter(100)
	handler := rl.Limit(okHandler())

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "4.4.4.4:12345"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Check X-RateLimit-Limit header
	limitHeader := w.Header().Get("X-RateLimit-Limit")
	assert.Equal(t, "100", limitHeader)

	// Check X-RateLimit-Remaining header
	remainingHeader := w.Header().Get("X-RateLimit-Remaining")
	remaining, err := strconv.Atoi(remainingHeader)
	require.NoError(t, err)
	assert.True(t, remaining >= 0 && remaining <= 100, "remaining should be between 0 and 100, got %d", remaining)

	// Check X-RateLimit-Reset header (should be a Unix timestamp)
	resetHeader := w.Header().Get("X-RateLimit-Reset")
	assert.NotEmpty(t, resetHeader)
	_, err = strconv.ParseInt(resetHeader, 10, 64)
	require.NoError(t, err, "X-RateLimit-Reset should be a Unix timestamp")
}

func TestRateLimiter_XRateLimitHeaders_OnRejection(t *testing.T) {
	rl := middleware.NewRateLimiter(1)
	handler := rl.Limit(okHandler())

	// Use up the limit
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "5.5.5.5:12345"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Second request exceeds limit
	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "5.5.5.5:12345"
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusTooManyRequests, w.Code)

	// Headers should still be present on 429 responses
	assert.Equal(t, "1", w.Header().Get("X-RateLimit-Limit"))
	assert.Equal(t, "0", w.Header().Get("X-RateLimit-Remaining"))
	assert.NotEmpty(t, w.Header().Get("X-RateLimit-Reset"))
	assert.NotEmpty(t, w.Header().Get("Retry-After"))
}

func TestRateLimiter_RemainingDecreases(t *testing.T) {
	rl := middleware.NewRateLimiter(10)
	handler := rl.Limit(okHandler())

	var lastRemaining int = 999
	for i := range 5 {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "6.6.6.6:12345"
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		remaining, _ := strconv.Atoi(w.Header().Get("X-RateLimit-Remaining"))
		if i > 0 {
			assert.Less(t, remaining, lastRemaining, "remaining should decrease with each request")
		}
		lastRemaining = remaining
	}
}

// --- Rate Limiter Group Tests ---

func TestRateLimiterGroup_DefaultLimits(t *testing.T) {
	group := middleware.NewRateLimiterGroup(nil)

	// Auth category should have its own limits
	authMiddleware := group.ForCategory(middleware.CategoryAuth)
	handler := authMiddleware(okHandler())

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "7.7.7.7:12345"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "10", w.Header().Get("X-RateLimit-Limit"))
}

func TestRateLimiterGroup_CustomLimits(t *testing.T) {
	group := middleware.NewRateLimiterGroup(map[middleware.EndpointCategory]int{
		middleware.CategoryAuth: 20,
		middleware.CategoryAPI:  200,
		middleware.CategorySync: 2,
	})

	// Sync category should have stricter limits
	syncMiddleware := group.ForCategory(middleware.CategorySync)
	handler := syncMiddleware(okHandler())

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "8.8.8.8:12345"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "2", w.Header().Get("X-RateLimit-Limit"))
}

func TestRateLimiterGroup_CategoriesIndependent(t *testing.T) {
	group := middleware.NewRateLimiterGroup(map[middleware.EndpointCategory]int{
		middleware.CategoryAuth: 2,
		middleware.CategoryAPI:  100,
		middleware.CategorySync: 2,
	})

	authHandler := group.ForCategory(middleware.CategoryAuth)(okHandler())
	apiHandler := group.ForCategory(middleware.CategoryAPI)(okHandler())

	// Exhaust auth limit
	for range 2 {
		req := httptest.NewRequest(http.MethodGet, "/auth", nil)
		req.RemoteAddr = "9.9.9.9:12345"
		w := httptest.NewRecorder()
		authHandler.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	}

	// Auth should be blocked
	req := httptest.NewRequest(http.MethodGet, "/auth", nil)
	req.RemoteAddr = "9.9.9.9:12345"
	w := httptest.NewRecorder()
	authHandler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTooManyRequests, w.Code)

	// API should still be allowed (different limiter)
	req = httptest.NewRequest(http.MethodGet, "/api", nil)
	req.RemoteAddr = "9.9.9.9:12345"
	w = httptest.NewRecorder()
	apiHandler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

// --- Security Headers Tests ---

func TestSecurityHeaders(t *testing.T) {
	handler := middleware.SecurityHeaders(okHandler())

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", w.Header().Get("X-Frame-Options"))
	assert.Equal(t, "1; mode=block", w.Header().Get("X-XSS-Protection"))
	assert.Equal(t, "strict-origin-when-cross-origin", w.Header().Get("Referrer-Policy"))
	assert.Equal(t, "default-src 'self'", w.Header().Get("Content-Security-Policy"))
	assert.Equal(t, "camera=(), microphone=(), geolocation=()", w.Header().Get("Permissions-Policy"))
	assert.Equal(t, "max-age=31536000; includeSubDomains", w.Header().Get("Strict-Transport-Security"))
}

// --- Body Size Limit Tests ---

func TestBodySizeLimit(t *testing.T) {
	handler := middleware.BodySizeLimit(10)(okHandler())

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
