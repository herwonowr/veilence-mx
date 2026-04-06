package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/veilence/veilence-mx/backend/internal/metrics"
)

// responseRecorder wraps http.ResponseWriter to capture the status code.
type responseRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (rr *responseRecorder) WriteHeader(code int) {
	rr.statusCode = code
	rr.ResponseWriter.WriteHeader(code)
}

// Metrics returns middleware that records Prometheus metrics for each request.
func Metrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		rec := &responseRecorder{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(rec, r)

		duration := time.Since(start).Seconds()

		// Use the route pattern (e.g. "/api/packages/{id}") instead of the
		// actual URL to avoid high-cardinality label values.
		routePattern := chi.RouteContext(r.Context()).RoutePattern()
		if routePattern == "" {
			routePattern = "unknown"
		}

		statusStr := strconv.Itoa(rec.statusCode)

		metrics.HTTPRequestsTotal.WithLabelValues(r.Method, routePattern, statusStr).Inc()
		metrics.HTTPRequestDuration.WithLabelValues(r.Method, routePattern).Observe(duration)
	})
}
