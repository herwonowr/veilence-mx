package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/veilence/veilence-mx/backend/internal/controller/restapi/middleware"
)

func TestSecurityHeadersWithConfig_AllHeadersPresent(t *testing.T) {
	handler := middleware.SecurityHeadersWithConfig(middleware.SecurityHeadersConfig{
		FrontendURL: "https://app.veilence.com",
	})(okHandler())

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", w.Header().Get("X-Frame-Options"))
	assert.Equal(t, "1; mode=block", w.Header().Get("X-XSS-Protection"))
	assert.Equal(t, "strict-origin-when-cross-origin", w.Header().Get("Referrer-Policy"))
	assert.Equal(t, "camera=(), microphone=(), geolocation=()", w.Header().Get("Permissions-Policy"))
	assert.Equal(t, "max-age=31536000; includeSubDomains", w.Header().Get("Strict-Transport-Security"))
}

func TestSecurityHeadersWithConfig_CSPIncludesFrontend(t *testing.T) {
	handler := middleware.SecurityHeadersWithConfig(middleware.SecurityHeadersConfig{
		FrontendURL: "https://app.veilence.com",
	})(okHandler())

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	csp := w.Header().Get("Content-Security-Policy")
	assert.Contains(t, csp, "https://app.veilence.com")
	assert.Contains(t, csp, "default-src 'self'")
	assert.Contains(t, csp, "script-src 'self'")
	assert.Contains(t, csp, "frame-ancestors 'none'")
}

func TestSecurityHeadersWithConfig_DefaultCSPWithoutFrontend(t *testing.T) {
	handler := middleware.SecurityHeadersWithConfig(middleware.SecurityHeadersConfig{})(okHandler())

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, "default-src 'self'", w.Header().Get("Content-Security-Policy"))
}

func TestSecurityHeaders_BackwardCompatible(t *testing.T) {
	// The no-arg SecurityHeaders should still work with default CSP
	handler := middleware.SecurityHeaders(okHandler())

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "default-src 'self'", w.Header().Get("Content-Security-Policy"))
	assert.Equal(t, "DENY", w.Header().Get("X-Frame-Options"))
	assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "max-age=31536000; includeSubDomains", w.Header().Get("Strict-Transport-Security"))
}

func TestSecurityHeadersWithConfig_HSTS(t *testing.T) {
	handler := middleware.SecurityHeadersWithConfig(middleware.SecurityHeadersConfig{})(okHandler())

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	hsts := w.Header().Get("Strict-Transport-Security")
	assert.Contains(t, hsts, "max-age=31536000")
	assert.Contains(t, hsts, "includeSubDomains")
}

func TestSecurityHeadersWithConfig_PermissionsPolicy(t *testing.T) {
	handler := middleware.SecurityHeadersWithConfig(middleware.SecurityHeadersConfig{})(okHandler())

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	pp := w.Header().Get("Permissions-Policy")
	assert.Contains(t, pp, "camera=()")
	assert.Contains(t, pp, "microphone=()")
	assert.Contains(t, pp, "geolocation=()")
}
