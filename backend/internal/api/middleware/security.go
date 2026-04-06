package middleware

import (
	"fmt"
	"net/http"
)

// SecurityHeadersConfig holds configuration for the security headers middleware.
type SecurityHeadersConfig struct {
	// FrontendURL is the origin URL of the frontend application.
	// Used to set a restrictive Content-Security-Policy.
	// If empty, defaults to 'self'.
	FrontendURL string
}

// SecurityHeaders returns a middleware that sets security-related HTTP response
// headers on every request. These headers protect against common web
// vulnerabilities including clickjacking, MIME sniffing, and XSS attacks.
func SecurityHeaders(next http.Handler) http.Handler {
	return SecurityHeadersWithConfig(SecurityHeadersConfig{})(next)
}

// SecurityHeadersWithConfig returns a middleware that sets security-related
// HTTP response headers, using the provided configuration for dynamic values
// like Content-Security-Policy.
func SecurityHeadersWithConfig(config SecurityHeadersConfig) func(http.Handler) http.Handler {
	// Build CSP directive once at init time
	csp := "default-src 'self'"
	if config.FrontendURL != "" {
		csp = fmt.Sprintf("default-src 'self' %s; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self'; connect-src 'self' %s; frame-ancestors 'none'",
			config.FrontendURL, config.FrontendURL)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("X-XSS-Protection", "1; mode=block")
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
			w.Header().Set("Content-Security-Policy", csp)
			w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")

			next.ServeHTTP(w, r)
		})
	}
}
