package middleware

import (
	"net/http"
)

// DefaultMaxBodySize is the default maximum request body size (1 MB).
const DefaultMaxBodySize = 1 << 20 // 1 MB

// BodySizeLimit returns a middleware that limits the size of incoming request
// bodies. Requests with bodies exceeding the limit will receive an error when
// the handler attempts to read past the limit. The limit is specified in bytes.
func BodySizeLimit(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Body != nil {
				r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			}
			next.ServeHTTP(w, r)
		})
	}
}
