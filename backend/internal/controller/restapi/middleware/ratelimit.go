package middleware

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/veilence/veilence-mx/backend/internal/usecase/auth"
)

// RateLimiter implements a per-identity token bucket rate limiter using
// in-memory storage. It tracks request rates per identity (user ID when
// authenticated, IP address when not) and rejects requests that exceed
// the configured limit with a 429 Too Many Requests response.
type RateLimiter struct {
	buckets sync.Map // map[string]*bucket

	rate     float64 // tokens per second
	capacity float64 // max tokens (burst)
	limit    int     // original requests-per-minute for header reporting
}

// bucket represents a token bucket for a single identity.
type bucket struct {
	mu     sync.Mutex
	tokens float64
	last   time.Time
}

// NewRateLimiter creates a new rate limiter with the given requests-per-minute
// limit. Each identity (user ID or IP) gets its own token bucket that refills
// at the specified rate.
func NewRateLimiter(requestsPerMinute int) *RateLimiter {
	rl := &RateLimiter{
		rate:     float64(requestsPerMinute) / 60.0,
		capacity: float64(requestsPerMinute),
		limit:    requestsPerMinute,
	}

	// Start cleanup goroutine to evict stale buckets every 5 minutes
	go rl.cleanup()

	return rl
}

// cleanup periodically removes stale buckets to prevent memory leaks.
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rl.buckets.Range(func(key, value any) bool {
			b := value.(*bucket)
			b.mu.Lock()
			idle := time.Since(b.last)
			b.mu.Unlock()
			if idle > 10*time.Minute {
				rl.buckets.Delete(key)
			}
			return true
		})
	}
}

// allow checks whether a request from the given identity should be allowed.
// It returns:
//   - allowed: true if the request is allowed, false otherwise
//   - remaining: the number of remaining requests in the current window
//   - retryAfter: seconds until the next token is available (only meaningful if !allowed)
//   - resetAt: unix timestamp when the bucket will be fully refilled
func (rl *RateLimiter) allow(identity string) (allowed bool, remaining int, retryAfter float64, resetAt time.Time) {
	val, _ := rl.buckets.LoadOrStore(identity, &bucket{
		tokens: rl.capacity,
		last:   time.Now(),
	})
	b := val.(*bucket)

	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(b.last).Seconds()
	b.last = now

	// Refill tokens
	b.tokens += elapsed * rl.rate
	if b.tokens > rl.capacity {
		b.tokens = rl.capacity
	}

	// Calculate reset time (time until bucket is full)
	deficit := rl.capacity - b.tokens
	resetDuration := time.Duration(deficit/rl.rate*1000) * time.Millisecond
	resetAt = now.Add(resetDuration)

	if b.tokens >= 1.0 {
		b.tokens -= 1.0
		return true, int(b.tokens), 0, resetAt
	}

	// Calculate retry-after: time until 1 token is available
	tokenDeficit := 1.0 - b.tokens
	retryAfter = tokenDeficit / rl.rate

	return false, 0, retryAfter, resetAt
}

// rateLimitError is the JSON response body for rate limit exceeded errors.
type rateLimitError struct {
	Data  any     `json:"data"`
	Error *string `json:"error"`
}

// Limit returns a middleware that applies rate limiting using this limiter.
// Requests that exceed the limit receive a 429 status with a Retry-After header.
// It identifies clients by user ID (when authenticated) or by IP address.
// Standard rate limit headers (X-RateLimit-Limit, X-RateLimit-Remaining,
// X-RateLimit-Reset) are set on every response.
func (rl *RateLimiter) Limit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		identity := rateLimitIdentity(r)
		allowed, remaining, retryAfter, resetAt := rl.allow(identity)

		// Always set rate limit headers
		w.Header().Set("X-RateLimit-Limit", strconv.Itoa(rl.limit))
		w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
		w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(resetAt.Unix(), 10))

		if !allowed {
			retryAfterSec := int(math.Ceil(retryAfter))
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Retry-After", strconv.Itoa(retryAfterSec))
			w.WriteHeader(http.StatusTooManyRequests)
			msg := "rate limit exceeded"
			json.NewEncoder(w).Encode(rateLimitError{Error: &msg})
			return
		}

		next.ServeHTTP(w, r)
	})
}

// rateLimitIdentity determines the identity key for rate limiting.
// Uses "user:<id>" for authenticated users, "ip:<addr>" for anonymous requests.
func rateLimitIdentity(r *http.Request) string {
	if userID := auth.UserIDFromContext(r.Context()); userID != 0 {
		return fmt.Sprintf("user:%d", userID)
	}
	return "ip:" + extractIP(r)
}

// extractIP extracts the client IP address from the request. It uses
// RemoteAddr directly, stripping the port if present. This should be used
// after the RealIP middleware has set RemoteAddr from X-Forwarded-For.
func extractIP(r *http.Request) string {
	ip := r.RemoteAddr
	// Strip port from host:port format
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		// Check if it's an IPv6 address in brackets
		if strings.Contains(ip, "]") {
			// [::1]:port format
			if bracketIdx := strings.LastIndex(ip, "]"); bracketIdx < idx {
				ip = ip[:idx]
			}
		} else {
			ip = ip[:idx]
		}
	}
	return ip
}

// EndpointCategory defines rate limit categories for different endpoint types.
type EndpointCategory string

const (
	// CategoryAuth is for authentication endpoints (login, register, etc.).
	CategoryAuth EndpointCategory = "auth"
	// CategoryAPI is for general API endpoints.
	CategoryAPI EndpointCategory = "api"
	// CategorySync is for sync/trigger endpoints.
	CategorySync EndpointCategory = "sync"
)

// DefaultLimits are the default rate limits per minute for each endpoint category.
var DefaultLimits = map[EndpointCategory]int{
	CategoryAuth: 10,
	CategoryAPI:  100,
	CategorySync: 5,
}

// RateLimiterGroup manages multiple rate limiters for different endpoint categories.
type RateLimiterGroup struct {
	limiters map[EndpointCategory]*RateLimiter
}

// NewRateLimiterGroup creates a new rate limiter group with the given limits per category.
// If limits is nil, DefaultLimits are used.
func NewRateLimiterGroup(limits map[EndpointCategory]int) *RateLimiterGroup {
	if limits == nil {
		limits = DefaultLimits
	}
	group := &RateLimiterGroup{
		limiters: make(map[EndpointCategory]*RateLimiter, len(limits)),
	}
	for category, rpm := range limits {
		group.limiters[category] = NewRateLimiter(rpm)
	}
	return group
}

// ForCategory returns the rate limiter middleware for the given endpoint category.
// Falls back to CategoryAPI if the category is not found.
func (g *RateLimiterGroup) ForCategory(category EndpointCategory) func(http.Handler) http.Handler {
	rl, ok := g.limiters[category]
	if !ok {
		rl = g.limiters[CategoryAPI]
	}
	return rl.Limit
}
