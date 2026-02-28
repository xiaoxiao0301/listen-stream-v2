package limiter

import (
	"context"
	"testing"
	"time"

	"github.com/xiaoxiao0301/listen-stream-v2/server/shared/pkg/redis"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestLimiter(t *testing.T) (*RateLimiter, *redis.Client, func()) {
	if testing.Short() {
		t.Skip("Skipping limiter integration test in short mode")
	}

	cfg := &redis.Config{
		Host:        "localhost",
		Port:        6379,
		DB:          15,
		DialTimeout: 5 * time.Second,
	}

	client, err := redis.NewClient(cfg)
	require.NoError(t, err)

	limiter := NewRateLimiter(client)

	cleanup := func() {
		ctx := context.Background()
		client.Universal().FlushDB(ctx)
		client.Close()
	}

	return limiter, client, cleanup
}

func TestRateLimiter_Allow_FirstRequest(t *testing.T) {
	limiter, _, cleanup := setupTestLimiter(t)
	defer cleanup()

	ctx := context.Background()
	key := "test:limiter:user1"

	allowed, err := limiter.Allow(ctx, key, 10, time.Minute)
	require.NoError(t, err)
	assert.True(t, allowed)
}

func TestRateLimiter_Allow_WithinLimit(t *testing.T) {
	limiter, _, cleanup := setupTestLimiter(t)
	defer cleanup()

	ctx := context.Background()
	key := "test:limiter:user2"
	limit := int64(5)

	// Make 5 requests (within limit)
	for i := 0; i < 5; i++ {
		allowed, err := limiter.Allow(ctx, key, limit, time.Minute)
		require.NoError(t, err)
		assert.True(t, allowed, "request %d should be allowed", i+1)
	}
}

func TestRateLimiter_Allow_ExceedLimit(t *testing.T) {
	limiter, _, cleanup := setupTestLimiter(t)
	defer cleanup()

	ctx := context.Background()
	key := "test:limiter:user3"
	limit := int64(3)

	// Make 3 requests (at limit)
	for i := 0; i < 3; i++ {
		allowed, err := limiter.Allow(ctx, key, limit, time.Minute)
		require.NoError(t, err)
		assert.True(t, allowed)
	}

	// 4th request should be denied
	allowed, err := limiter.Allow(ctx, key, limit, time.Minute)
	require.NoError(t, err)
	assert.False(t, allowed, "4th request should exceed limit")
}

func TestRateLimiter_Allow_WindowExpiration(t *testing.T) {
	limiter, _, cleanup := setupTestLimiter(t)
	defer cleanup()

	ctx := context.Background()
	key := "test:limiter:user4"
	limit := int64(2)
	window := 2 * time.Second

	// Use up limit
	limiter.Allow(ctx, key, limit, window)
	limiter.Allow(ctx, key, limit, window)

	// 3rd should be denied
	allowed, err := limiter.Allow(ctx, key, limit, window)
	require.NoError(t, err)
	assert.False(t, allowed)

	// Wait for window to expire
	time.Sleep(2*time.Second + 100*time.Millisecond)

	// Should be allowed again
	allowed, err = limiter.Allow(ctx, key, limit, window)
	require.NoError(t, err)
	assert.True(t, allowed, "request after window expiration should be allowed")
}

func TestRateLimiter_AllowN_SingleRequest(t *testing.T) {
	limiter, _, cleanup := setupTestLimiter(t)
	defer cleanup()

	ctx := context.Background()
	key := "test:limiter:bulk1"

	allowed, err := limiter.AllowN(ctx, key, 5, 10, time.Minute)
	require.NoError(t, err)
	assert.True(t, allowed)
}

func TestRateLimiter_AllowN_ExceedLimit(t *testing.T) {
	limiter, _, cleanup := setupTestLimiter(t)
	defer cleanup()

	ctx := context.Background()
	key := "test:limiter:bulk2"
	limit := int64(10)

	// Use 8 quota
	allowed, err := limiter.AllowN(ctx, key, 8, limit, time.Minute)
	require.NoError(t, err)
	assert.True(t, allowed)

	// Try to use 5 more (would exceed limit of 10)
	allowed, err = limiter.AllowN(ctx, key, 5, limit, time.Minute)
	require.NoError(t, err)
	assert.False(t, allowed, "should exceed limit")
}

func TestRateLimiter_AllowN_MultipleRequests(t *testing.T) {
	limiter, _, cleanup := setupTestLimiter(t)
	defer cleanup()

	ctx := context.Background()
	key := "test:limiter:bulk3"
	limit := int64(100)

	// Make multiple batch requests
	allowed, err := limiter.AllowN(ctx, key, 30, limit, time.Minute)
	require.NoError(t, err)
	assert.True(t, allowed)

	allowed, err = limiter.AllowN(ctx, key, 30, limit, time.Minute)
	require.NoError(t, err)
	assert.True(t, allowed)

	allowed, err = limiter.AllowN(ctx, key, 30, limit, time.Minute)
	require.NoError(t, err)
	assert.True(t, allowed)

	// Total is 90, next 20 should exceed
	allowed, err = limiter.AllowN(ctx, key, 20, limit, time.Minute)
	require.NoError(t, err)
	assert.False(t, allowed)
}

func TestRateLimiter_Remaining(t *testing.T) {
	limiter, _, cleanup := setupTestLimiter(t)
	defer cleanup()

	ctx := context.Background()
	key := "test:limiter:remaining"
	limit := int64(10)

	// Use 3
	limiter.Allow(ctx, key, limit, time.Minute)
	limiter.Allow(ctx, key, limit, time.Minute)
	limiter.Allow(ctx, key, limit, time.Minute)

	// Check remaining
	remaining, err := limiter.Remaining(ctx, key, limit)
	require.NoError(t, err)
	assert.Equal(t, int64(7), remaining)
}

func TestRateLimiter_Remaining_NoRequests(t *testing.T) {
	limiter, _, cleanup := setupTestLimiter(t)
	defer cleanup()

	ctx := context.Background()
	key := "test:limiter:no_requests"
	limit := int64(10)

	// No requests made yet
	remaining, err := limiter.Remaining(ctx, key, limit)
	require.NoError(t, err)
	assert.Equal(t, limit, remaining)
}

func TestRateLimiter_Reset(t *testing.T) {
	limiter, _, cleanup := setupTestLimiter(t)
	defer cleanup()

	ctx := context.Background()
	key := "test:limiter:reset"
	limit := int64(5)

	// Use all quota
	for i := 0; i < 5; i++ {
		limiter.Allow(ctx, key, limit, time.Minute)
	}

	// Verify limit exceeded
	allowed, _ := limiter.Allow(ctx, key, limit, time.Minute)
	assert.False(t, allowed)

	// Reset
	err := limiter.Reset(ctx, key)
	require.NoError(t, err)

	// Should be allowed again
	allowed, err = limiter.Allow(ctx, key, limit, time.Minute)
	require.NoError(t, err)
	assert.True(t, allowed)
}

func TestIPRateLimiter(t *testing.T) {
	_, client, cleanup := setupTestLimiter(t)
	defer cleanup()

	ctx := context.Background()

	ipLimiter := NewIPRateLimiter(client, 1000, time.Hour)

	// Test IP limit
	allowed, err := ipLimiter.Allow(ctx, "192.168.1.100")
	require.NoError(t, err)
	assert.True(t, allowed)
}

func TestUserRateLimiter(t *testing.T) {
	_, client, cleanup := setupTestLimiter(t)
	defer cleanup()

	ctx := context.Background()

	userLimiter := NewUserRateLimiter(client, 100, time.Minute)

	// Test user limit
	allowed, err := userLimiter.Allow(ctx, "user123")
	require.NoError(t, err)
	assert.True(t, allowed)
}
