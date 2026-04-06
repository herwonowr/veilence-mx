package audit

import (
	"context"
	"crypto/rand"
	"fmt"
	"net/http"
)

type correlationKey string

const ctxCorrelationID correlationKey = "correlation_id"

// generateUUID creates a UUID v4 string without external dependencies.
func generateUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 10
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

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

		ctx := context.WithValue(r.Context(), ctxCorrelationID, correlationID)

		w.Header().Set("X-Correlation-ID", correlationID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// CorrelationIDFromContext extracts the correlation ID from the request context.
// Returns an empty string if not set.
func CorrelationIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(ctxCorrelationID).(string); ok {
		return v
	}
	return ""
}
