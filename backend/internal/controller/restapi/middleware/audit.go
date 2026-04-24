package middleware

import (
	"crypto/rand"
	"fmt"
	"net/http"

	"github.com/veilence/veilence-mx/backend/internal/usecase/audit"
)

// CorrelationMiddleware is a Chi middleware that generates or propagates
// a correlation ID for each request. It reads from the X-Correlation-ID
// header or generates a new UUID v4, stores it in the context, and adds
// it to the response header.
func CorrelationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		correlationID := r.Header.Get("X-Correlation-ID")
		if correlationID == "" {
			correlationID = generateUUID()
		}

		ctx := audit.WithCorrelationID(r.Context(), correlationID)

		w.Header().Set("X-Correlation-ID", correlationID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequestCaptureMiddleware is a Chi middleware that extracts IP address and
// User-Agent from the HTTP request and stores them in the context so the
// audit service can access them without importing net/http.
func RequestCaptureMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := audit.WithIPAddress(r.Context(), r.RemoteAddr)
		ctx = audit.WithUserAgent(ctx, r.Header.Get("User-Agent"))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// generateUUID creates a UUID v4 string without external dependencies.
func generateUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 10
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
