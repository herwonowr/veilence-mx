// Package cache provides Redis-backed implementations of usecase cache interfaces.
package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RateLimiter implements usecase.RateLimiter using Redis SET NX with TTL.
type RateLimiter struct {
	client *redis.Client
}

// NewRateLimiter creates a new Redis-backed rate limiter.
func NewRateLimiter(client *redis.Client) *RateLimiter {
	return &RateLimiter{client: client}
}

// Allow checks if the key is allowed (not in cooldown). If allowed, sets the key
// with the given TTL so subsequent calls within the cooldown period return false.
func (r *RateLimiter) Allow(ctx context.Context, key string, cooldown time.Duration) (bool, error) {
	// SET NX returns true only if the key did not exist - atomic check-and-set.
	ok, err := r.client.SetNX(ctx, key, "1", cooldown).Result()
	if err != nil {
		return false, fmt.Errorf("RateLimiter.Allow: %w", err)
	}
	return ok, nil
}
