package middleware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

// BenchmarkRateLimiter_Allow measures raw token bucket throughput for a single identity.
// Baseline: should complete in <500ns/op (in-memory sync.Map + mutex).
func BenchmarkRateLimiter_Allow(b *testing.B) {
	rl := &RateLimiter{
		rate:     100.0 / 60.0, // 100 rpm
		capacity: 100,
		limit:    100,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rl.allow("bench-identity")
	}
}

// BenchmarkRateLimiter_Allow_MultipleIdentities measures rate limiting across many IPs.
func BenchmarkRateLimiter_Allow_MultipleIdentities(b *testing.B) {
	rl := &RateLimiter{
		rate:     100.0 / 60.0,
		capacity: 100,
		limit:    100,
	}

	identities := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		identities[i] = fmt.Sprintf("ip:10.0.%d.%d", i/256, i%256)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rl.allow(identities[i%len(identities)])
	}
}

// BenchmarkRateLimiter_Allow_Parallel measures concurrent rate limiter throughput.
func BenchmarkRateLimiter_Allow_Parallel(b *testing.B) {
	rl := &RateLimiter{
		rate:     100.0 / 60.0,
		capacity: 100,
		limit:    100,
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			rl.allow("parallel-identity")
		}
	})
}

// BenchmarkRateLimiter_Allow_Parallel_MultipleIdentities measures concurrent throughput
// with different identities (typical production pattern).
func BenchmarkRateLimiter_Allow_Parallel_MultipleIdentities(b *testing.B) {
	rl := &RateLimiter{
		rate:     100.0 / 60.0,
		capacity: 100,
		limit:    100,
	}

	var counter uint64
	var mu sync.Mutex

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		mu.Lock()
		counter++
		id := fmt.Sprintf("ip:10.0.%d.%d", counter/256, counter%256)
		mu.Unlock()

		for pb.Next() {
			rl.allow(id)
		}
	})
}

// BenchmarkRateLimiterMiddleware measures the full middleware path (HTTP handler overhead).
func BenchmarkRateLimiterMiddleware(b *testing.B) {
	rl := &RateLimiter{
		rate:     10000.0 / 60.0, // Very high limit so we don't hit 429s
		capacity: 10000,
		limit:    10000,
	}

	handler := rl.Limit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/test", nil)
	req.RemoteAddr = "127.0.0.1:12345"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}
}

// BenchmarkExtractIP measures IP extraction from various request formats.
func BenchmarkExtractIP(b *testing.B) {
	cases := []struct {
		name string
		addr string
	}{
		{"IPv4", "192.168.1.1:8080"},
		{"IPv6", "[::1]:8080"},
		{"NoPort", "192.168.1.1"},
	}

	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = tc.addr

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = extractIP(req)
			}
		})
	}
}
