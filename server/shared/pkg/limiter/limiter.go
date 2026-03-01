// Package limiter provides rate limiting functionality.
package limiter

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	redispkg "github.com/xiaoxiao0301/listen-stream-v2/server/shared/pkg/redis"
)

// atomicIncrExpire atomically increments a counter and sets TTL on first increment.
// This prevents the TOCTOU race condition between separate INCR and EXPIRE calls.
var atomicIncrExpire = redis.NewScript(`
local current = redis.call('INCR', KEYS[1])
if current == 1 then
  redis.call('PEXPIRE', KEYS[1], ARGV[1])
end
return current
`)

// RateLimiter provides rate limiting using Redis.
type RateLimiter struct {
	client *redispkg.Client
}

// NewRateLimiter creates a new rate limiter.
func NewRateLimiter(client *redispkg.Client) *RateLimiter {
	return &RateLimiter{client: client}
}

// Allow checks if a request is allowed under the rate limit.
// Uses an atomic Lua script to prevent TOCTOU races between INCR and EXPIRE.
func (rl *RateLimiter) Allow(ctx context.Context, key string, limit int64, window time.Duration) (bool, error) {
	result, err := atomicIncrExpire.Run(ctx, rl.client.Universal(), []string{key}, window.Milliseconds()).Int64()
	if err != nil {
		return false, fmt.Errorf("rate limit check failed: %w", err)
	}
	return result <= limit, nil
}

// AllowN checks if n requests are allowed using an atomic Lua script.
func (rl *RateLimiter) AllowN(ctx context.Context, key string, n, limit int64, window time.Duration) (bool, error) {
	script := redis.NewScript(`
local current = redis.call('INCRBY', KEYS[1], ARGV[2])
if current == tonumber(ARGV[2]) then
  redis.call('PEXPIRE', KEYS[1], ARGV[1])
end
return current
`)
	result, err := script.Run(ctx, rl.client.Universal(), []string{key}, window.Milliseconds(), n).Int64()
	if err != nil {
		return false, fmt.Errorf("rate limit check failed: %w", err)
	}
	return result <= limit, nil
}

// Remaining returns the number of requests remaining in the current window.
func (rl *RateLimiter) Remaining(ctx context.Context, key string, limit int64) (int64, error) {
	count, err := rl.client.Get(ctx, key)
	if err == redispkg.ErrKeyNotFound {
		return limit, nil
	}
	if err != nil {
		return 0, fmt.Errorf("failed to get counter: %w", err)
	}
	current, err := strconv.ParseInt(count, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid counter value: %w", err)
	}
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
func NewIPRateLimiter(client *redispkg.Client, limit int64, window time.Duration) *IPRateLimiter {
	return &IPRateLimiter{limiter: NewRateLimiter(client), limit: limit, window: window}
}

// Allow checks if a request from an IP is allowed.
func (ipl *IPRateLimiter) Allow(ctx context.Context, ip string) (bool, error) {
	key := redispkg.RateLimitKey("ip", ip, "minute")
	return ipl.limiter.Allow(ctx, key, ipl.limit, ipl.window)
}

// UserRateLimiter provides user-based rate limiting.
type UserRateLimiter struct {
	limiter *RateLimiter
	limit   int64
	window  time.Duration
}

// NewUserRateLimiter creates a new user rate limiter.
func NewUserRateLimiter(client *redispkg.Client, limit int64, window time.Duration) *UserRateLimiter {
	return &UserRateLimiter{limiter: NewRateLimiter(client), limit: limit, window: window}
}

// Allow checks if a request from a user is allowed.
func (url *UserRateLimiter) Allow(ctx context.Context, userID string) (bool, error) {
	key := redispkg.RateLimitKey("user", userID, "minute")
	return url.limiter.Allow(ctx, key, url.limit, url.window)
}
