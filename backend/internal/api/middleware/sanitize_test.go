package middleware_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/veilence/veilence-mx/backend/internal/api/middleware"
)

func sanitizeHandler() http.Handler {
	return middleware.Sanitize(okHandler())
}

func TestSanitize_CleanRequest_PassesThrough(t *testing.T) {
	handler := sanitizeHandler()

	body := `{"name": "test package", "version": "1.0.0"}`
	req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSanitize_HTMLInBody_Escaped(t *testing.T) {
	handler := middleware.Sanitize(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		// The HTML should be escaped
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(body)
	}))

	body := `{"name": "<script>alert('xss')</script>"}`
	req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]string
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.NotContains(t, resp["name"], "<script>")
	assert.Contains(t, resp["name"], "&lt;script&gt;")
}

func TestSanitize_PathTraversal_QueryParam_Rejected(t *testing.T) {
	handler := sanitizeHandler()

	tests := []struct {
		name  string
		key   string
		value string
	}{
		{"dot-dot-slash", "path", "../../etc/passwd"},
		{"encoded-dots", "path", "%2e%2e/etc/passwd"},
		{"encoded-slash", "path", "..%2fetc/passwd"},
		{"backslash", "path", "..\\windows\\system32"},
		{"encoded-backslash", "path", "..%5cwindows\\system32"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			q := req.URL.Query()
			q.Set(tt.key, tt.value)
			req.URL.RawQuery = q.Encode()
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			assert.Equal(t, http.StatusBadRequest, w.Code, "should reject path traversal: %s", tt.name)

			var resp map[string]any
			json.NewDecoder(w.Body).Decode(&resp)
			assert.Contains(t, resp["error"], "path traversal")
		})
	}
}

func TestSanitize_PathTraversal_Body_Rejected(t *testing.T) {
	handler := sanitizeHandler()

	body := `{"file": "../../etc/passwd"}`
	req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSanitize_PathTraversal_URLPath_Rejected(t *testing.T) {
	handler := sanitizeHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/../../../etc/passwd", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSanitize_SQLInjection_QueryParam_Rejected(t *testing.T) {
	handler := sanitizeHandler()

	tests := []struct {
		name  string
		key   string
		value string
	}{
		{"union-select", "search", "1 UNION SELECT * FROM users"},
		{"or-1-equals-1", "id", "1 OR 1=1"},
		{"drop-table", "name", "test; DROP TABLE users"},
		{"comment-injection", "search", "admin'-- "},
		{"delete-from", "q", "x; DELETE FROM users"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			q := req.URL.Query()
			q.Set(tt.key, tt.value)
			req.URL.RawQuery = q.Encode()
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			assert.Equal(t, http.StatusBadRequest, w.Code, "should reject SQL injection: %s", tt.name)
		})
	}
}

func TestSanitize_SQLInjection_Body_Rejected(t *testing.T) {
	handler := sanitizeHandler()

	body := `{"name": "test; DROP TABLE packages"}`
	req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSanitize_LegitimateInput_NotFlagged(t *testing.T) {
	handler := sanitizeHandler()

	tests := []struct {
		name  string
		key   string
		value string
	}{
		{"normal-search", "search", "react library"},
		{"version-query", "version", "1.0.0"},
		{"package-name", "name", "@scope/package"},
		{"normal-text", "q", "select the best option"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			q := req.URL.Query()
			q.Set(tt.key, tt.value)
			req.URL.RawQuery = q.Encode()
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code, "legitimate input should not be flagged: %s", tt.name)
		})
	}
}

func TestSanitize_NonJSONBody_PassesThrough(t *testing.T) {
	handler := sanitizeHandler()

	body := "plain text body with <script>alert('xss')</script>"
	req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSanitize_NestedJSON_Sanitized(t *testing.T) {
	handler := middleware.Sanitize(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(body)
	}))

	body := `{"outer": {"inner": "<b>bold</b>"}, "list": ["<i>italic</i>", "clean"]}`
	req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)

	inner := resp["outer"].(map[string]any)["inner"].(string)
	assert.NotContains(t, inner, "<b>")
	assert.Contains(t, inner, "&lt;b&gt;")

	list := resp["list"].([]any)
	assert.NotContains(t, list[0], "<i>")
	assert.Equal(t, "clean", list[1])
}

func TestSanitize_EmptyBody_PassesThrough(t *testing.T) {
	handler := sanitizeHandler()

	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSanitize_GetRequest_NoBody(t *testing.T) {
	handler := sanitizeHandler()

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
