package middleware

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/veilence/veilence-mx/backend/internal/usecase/auth"
)

const (
	// CSRFTokenLength is the byte length of a CSRF token before base64 encoding.
	CSRFTokenLength = 32
	// CSRFCookieName is the name of the cookie that stores the CSRF token.
	CSRFCookieName = "_csrf_token"
	// CSRFHeaderName is the header the client must send to prove it read the cookie.
	CSRFHeaderName = "X-CSRF-Token"
	// CSRFCookieMaxAge is the maximum age of the CSRF cookie in seconds (24 hours).
	CSRFCookieMaxAge = 86400
)

// csrfError is the JSON response for CSRF validation failures.
type csrfError struct {
	Data  any     `json:"data"`
	Error *string `json:"error"`
}

// CSRFConfig holds configuration for the CSRF middleware.
type CSRFConfig struct {
	// Secure sets the Secure flag on the CSRF cookie (should be true in production).
	Secure bool
	// Domain sets the Domain attribute on the CSRF cookie.
	Domain string
}

// generateCSRFToken creates a cryptographically secure random CSRF token.
func generateCSRFToken() (string, error) {
	b := make([]byte, CSRFTokenLength)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// csrfExemptPaths lists POST routes that must be exempt from CSRF validation.
// These are pre-authentication endpoints where no session exists to hijack,
// and on a fresh browser the CSRF cookie has not yet been set (it is only
// issued on GET responses via ensureCSRFCookie). Blocking these routes causes
// login/register to fail with 403 on every fresh session.
var csrfExemptPaths = map[string]bool{
	"/api/auth/login":               true,
	"/api/auth/register":            true,
	"/api/auth/forgot-password":     true,
	"/api/auth/reset-password":      true,
	"/api/auth/refresh":             true,
	"/api/auth/verify-email":        true,
	"/api/auth/resend-verification": true,
}

// CSRF returns a middleware implementing the double-submit cookie pattern.
//
// For state-changing methods (POST, PUT, PATCH, DELETE), it validates that
// the X-CSRF-Token header matches the _csrf_token cookie value.
//
// API key authenticated requests are exempt from CSRF validation since they
// are not vulnerable to CSRF attacks (the API key is not automatically sent
// by the browser).
//
// Pre-authentication routes (login, register, forgot-password, reset-password)
// are exempt because no session exists to hijack, and the CSRF cookie may not
// yet exist on a fresh browser.
func CSRF(config CSRFConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Safe methods: ensure a CSRF cookie exists, then pass through
			if isSafeMethod(r.Method) {
				ensureCSRFCookie(w, r, config)
				next.ServeHTTP(w, r)
				return
			}

			// Pre-auth routes are exempt - no session to hijack
			if csrfExemptPaths[r.URL.Path] {
				ensureCSRFCookie(w, r, config)
				next.ServeHTTP(w, r)
				return
			}

			// API key authenticated requests are exempt from CSRF
			if auth.AuthMethodFromContext(r.Context()) == auth.AuthMethodAPIKey {
				next.ServeHTTP(w, r)
				return
			}

			// For state-changing methods from browser clients, validate the token
			cookie, err := r.Cookie(CSRFCookieName)
			if err != nil || cookie.Value == "" {
				respondCSRFError(w, "CSRF token missing")
				return
			}

			headerToken := r.Header.Get(CSRFHeaderName)
			if headerToken == "" {
				respondCSRFError(w, "CSRF token header missing")
				return
			}

			// Constant-time comparison to prevent timing attacks
			if subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(headerToken)) != 1 {
				respondCSRFError(w, "CSRF token mismatch")
				return
			}

			// Rotate the CSRF token after successful validation
			rotateCSRFCookie(w, config)

			next.ServeHTTP(w, r)
		})
	}
}

// isSafeMethod returns true for HTTP methods that do not change state.
func isSafeMethod(method string) bool {
	switch strings.ToUpper(method) {
	case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace:
		return true
	default:
		return false
	}
}

// ensureCSRFCookie sets a CSRF cookie if one is not already present.
func ensureCSRFCookie(w http.ResponseWriter, r *http.Request, config CSRFConfig) {
	if _, err := r.Cookie(CSRFCookieName); err == nil {
		return // Cookie already exists
	}
	setCSRFCookie(w, config)
}

// rotateCSRFCookie generates a new CSRF token and sets it as a cookie.
func rotateCSRFCookie(w http.ResponseWriter, config CSRFConfig) {
	setCSRFCookie(w, config)
}

// setCSRFCookie generates a new CSRF token and writes it as a cookie.
func setCSRFCookie(w http.ResponseWriter, config CSRFConfig) {
	token, err := generateCSRFToken()
	if err != nil {
		slog.Error("failed to generate CSRF token", "error", err)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     CSRFCookieName,
		Value:    token,
		Path:     "/",
		Domain:   config.Domain,
		MaxAge:   CSRFCookieMaxAge,
		Expires:  time.Now().Add(time.Duration(CSRFCookieMaxAge) * time.Second),
		HttpOnly: false, // Must be readable by JavaScript for double-submit
		Secure:   config.Secure,
		SameSite: http.SameSiteStrictMode,
	})
}

// respondCSRFError writes a 403 Forbidden JSON response for CSRF failures.
func respondCSRFError(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	json.NewEncoder(w).Encode(csrfError{Error: &msg})
}
