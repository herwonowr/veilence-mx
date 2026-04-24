package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
)

// sanitizeError is the JSON response for sanitization failures.
type sanitizeError struct {
	Data  any     `json:"data"`
	Error *string `json:"error"`
}

// pathTraversalPatterns detects path traversal attempts in decoded form.
var pathTraversalPatterns = []string{
	"../",
	"..\\",
	"%2e%2e",
	"%2e%2e%2f",
	"%2e%2e/",
	"..%2f",
	"%2e%2e%5c",
	"..%5c",
}

// sqlInjectionPattern detects common SQL injection patterns beyond what
// parameterized queries handle (e.g., in free-text search fields).
var sqlInjectionPattern = regexp.MustCompile(
	`(?i)` +
		`(\b(union)\s+(all\s+)?select\b)` +
		`|(\b(select|insert|update|delete|drop|alter|create|exec|execute)\b\s+.*\b(from|into|table|database|index|set|values|where|having|group)\b)` +
		`|(\b(or|and)\b\s+['"]?\d+['"]?\s*=\s*['"]?\d+)` +
		`|(;\s*\b(drop|delete|update|insert|alter)\b)` +
		`|(--\s)` +
		`|(/\*.*?\*/)`,
)

// Sanitize returns a middleware that sanitizes request bodies and query parameters.
// It performs path traversal rejection and SQL injection detection.
func Sanitize(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check query parameters for path traversal and SQL injection
		for key, values := range r.URL.Query() {
			for _, v := range values {
				if containsPathTraversal(v) {
					respondSanitizeError(w, "path traversal detected in query parameter: "+key)
					return
				}
				if sqlInjectionPattern.MatchString(v) {
					slog.Warn("SQL injection pattern detected in query parameter",
						"param", key,
						"remote", r.RemoteAddr,
					)
					respondSanitizeError(w, "potentially malicious input detected in query parameter: "+key)
					return
				}
			}
		}

		// Check URL path for path traversal
		if containsPathTraversal(r.URL.Path) || containsPathTraversal(r.URL.RawPath) {
			respondSanitizeError(w, "path traversal detected in URL")
			return
		}

		// For requests with JSON bodies, sanitize string values
		if r.Body != nil && r.ContentLength != 0 && isJSONContent(r) {
			body, err := io.ReadAll(r.Body)
			r.Body.Close()
			if err != nil {
				respondSanitizeError(w, "failed to read request body")
				return
			}

			// Check raw body for path traversal
			bodyStr := string(body)
			if containsPathTraversal(bodyStr) {
				respondSanitizeError(w, "path traversal detected in request body")
				return
			}

			// Check for SQL injection patterns in the raw body
			if sqlInjectionPattern.MatchString(bodyStr) {
				slog.Warn("SQL injection pattern detected in request body",
					"remote", r.RemoteAddr,
					"path", r.URL.Path,
				)
				respondSanitizeError(w, "potentially malicious input detected in request body")
				return
			}

			// Restore body for downstream handlers
			r.Body = io.NopCloser(bytes.NewReader(body))
		}

		next.ServeHTTP(w, r)
	})
}

// containsPathTraversal checks if a string contains path traversal patterns.
func containsPathTraversal(s string) bool {
	lower := strings.ToLower(s)
	for _, pattern := range pathTraversalPatterns {
		if strings.Contains(lower, pattern) {
			return true
		}
	}
	return false
}

// isJSONContent checks if the request has a JSON content type.
func isJSONContent(r *http.Request) bool {
	ct := r.Header.Get("Content-Type")
	return strings.HasPrefix(ct, "application/json")
}

// respondSanitizeError writes a 400 Bad Request JSON response for sanitization failures.
func respondSanitizeError(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(sanitizeError{Error: &msg})
}
