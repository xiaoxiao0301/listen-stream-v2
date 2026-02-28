// Package limiter provides rate limiting functionality.
package limiter

import (
	"context"
	"fmt"
	"time"

	"github.com/xiaoxiao0301/listen-stream-v2/server/shared/pkg/redis"
)

// RateLimiter provides rate limiting using Redis.
type RateLimiter struct {
	client *redis.Client
}

// NewRateLimiter creates a new rate limiter.
func NewRateLimiter(client *redis.Client) *RateLimiter {
	return &RateLimiter{
		client: client,
	}
}

// Allow checks if a request is allowed under the rate limit.
// Returns true if allowed, false if rate limit exceeded.
func (rl *RateLimiter) Allow(ctx context.Context, key string, limit int64, window time.Duration) (bool, error) {
	// Increment the counter
	count, err := rl.client.Incr(ctx, key)
	if err != nil {
		return false, fmt.Errorf("failed to increment counter: %w", err)
	}
	
	// Set expiration on first request
	if count == 1 {
		if err := rl.client.Expire(ctx, key, window); err != nil {
			return false, fmt.Errorf("failed to set expiration: %w", err)
		}
	}
	
	// Check if limit exceeded
	return count <= limit, nil
}

// AllowN checks if n requests are allowed.
func (rl *RateLimiter) AllowN(ctx context.Context, key string, n, limit int64, window time.Duration) (bool, error) {
	// Increment the counter by n
	count, err := rl.client.IncrBy(ctx, key, n)
	if err != nil {
		return false, fmt.Errorf("failed to increment counter: %w", err)
	}
	
	// Set expiration on first request
	if count == n {
		if err := rl.client.Expire(ctx, key, window); err != nil {
			return false, fmt.Errorf("failed to set expiration: %w", err)
		}
	}
	
	return count <= limit, nil
}

// Remaining returns the number of requests remaining in the current window.
func (rl *RateLimiter) Remaining(ctx context.Context, key string, limit int64) (int64, error) {
	count, err := rl.client.Get(ctx, key)
	if err == redis.ErrKeyNotFound {
		return limit, nil
	}
	if err != nil {
		return 0, fmt.Errorf("failed to get counter: %w", err)
	}
	
	var current int64
	fmt.Sscanf(count, "%d", &current)
	remaining := limit - current
	
	if remaining < 0 {
		remaining = 0
	}
	return remaining, nil
}

// Reset resets the rate limit for a key.
func (rl *RateLimiter) Reset(ctx context.Context, key string) error {
	if err := rl.client.Delete(ctx, key); err != nil {
		return fmt.Errorf("failed to reset counter: %w", err)
	}
	return nil
}

// TTL returns the time until the rate limit resets.
func (rl *RateLimiter) TTL(ctx context.Context, key string) (time.Duration, error) {
	ttl, err := rl.client.TTL(ctx, key)
	if err != nil {
		return 0, fmt.Errorf("failed to get TTL: %w", err)
	}
	return ttl, nil
}

// IPRateLimiter provides IP-based rate limiting.
type IPRateLimiter struct {
	limiter *RateLimiter
	limit   int64
	window  time.Duration
}

// NewIPRateLimiter creates a new IP rate limiter.
func NewIPRateLimiter(client *redis.Client, limit int64, window time.Duration) *IPRateLimiter {
	return &IPRateLimiter{
		limiter: NewRateLimiter(client),
		limit:   limit,
		window:  window,
	}
}

// Allow checks if a request from an IP is allowed.
func (ipl *IPRateLimiter) Allow(ctx context.Context, ip string) (bool, error) {
	key := redis.RateLimitKey("ip", ip, "minute")
	return ipl.limiter.Allow(ctx, key, ipl.limit, ipl.window)
}

// UserRateLimiter provides user-based rate limiting.
type UserRateLimiter struct {
	limiter *RateLimiter
	limit   int64
	window  time.Duration
}

// NewUserRateLimiter creates a new user rate limiter.
func NewUserRateLimiter(client *redis.Client, limit int64, window time.Duration) *UserRateLimiter {
	return &UserRateLimiter{
		limiter: NewRateLimiter(client),
		limit:   limit,
		window:  window,
	}
}

// Allow checks if a request from a user is allowed.
func (url *UserRateLimiter) Allow(ctx context.Context, userID string) (bool, error) {
	key := redis.RateLimitKey("user", userID, "minute")
	return url.limiter.Allow(ctx, key, url.limit, url.window)
}