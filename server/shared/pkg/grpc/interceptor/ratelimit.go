package interceptor

import (
	"context"
	"fmt"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// RateLimitInterceptor implements token bucket rate limiting for gRPC.
type RateLimitInterceptor struct {
	// Limiter stores rate limiters per key (user ID, IP, method, etc.)
	limiters map[string]*tokenBucket
	mu       sync.RWMutex

	// Rate limit configuration
	rate     int           // tokens per second
	capacity int           // bucket capacity
	keyFunc  KeyExtractor  // function to extract rate limit key

	// Cleanup interval for unused limiters
	cleanupInterval time.Duration
}

// KeyExtractor extracts a rate limit key from the context.
//
// Common strategies:
// - Per user: extract user ID
// - Per IP: extract client IP
// - Per method: extract method name
// - Global: return constant key
type KeyExtractor func(ctx context.Context, method string) string

// tokenBucket implements the token bucket algorithm.
type tokenBucket struct {
	tokens   float64
	capacity int
	rate     int
	lastTime time.Time
	mu       sync.Mutex
}

// NewRateLimitInterceptor creates a new rate limit interceptor.
//
// rate: tokens per second
// capacity: maximum bucket capacity
// keyFunc: function to extract rate limit key
func NewRateLimitInterceptor(rate, capacity int, keyFunc KeyExtractor) *RateLimitInterceptor {
	if keyFunc == nil {
		keyFunc = PerUserKeyExtractor
	}

	r := &RateLimitInterceptor{
		limiters:        make(map[string]*tokenBucket),
		rate:            rate,
		capacity:        capacity,
		keyFunc:         keyFunc,
		cleanupInterval: 5 * time.Minute,
	}

	// Start cleanup goroutine
	go r.cleanup()

	return r
}

// UnaryServerInterceptor returns a unary server interceptor for rate limiting.
func (r *RateLimitInterceptor) UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// Extract rate limit key
		key := r.keyFunc(ctx, info.FullMethod)

		// Get or create limiter
		limiter := r.getLimiter(key)

		// Try to take a token
		if !limiter.allow() {
			return nil, status.Error(codes.ResourceExhausted, "rate limit exceeded")
		}

		return handler(ctx, req)
	}
}

// StreamServerInterceptor returns a stream server interceptor for rate limiting.
func (r *RateLimitInterceptor) StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		// Extract rate limit key
		key := r.keyFunc(ss.Context(), info.FullMethod)

		// Get or create limiter
		limiter := r.getLimiter(key)

		// Try to take a token
		if !limiter.allow() {
			return status.Error(codes.ResourceExhausted, "rate limit exceeded")
		}

		return handler(srv, ss)
	}
}

// getLimiter gets or creates a limiter for the given key.
func (r *RateLimitInterceptor) getLimiter(key string) *tokenBucket {
	r.mu.RLock()
	limiter, exists := r.limiters[key]
	r.mu.RUnlock()

	if exists {
		return limiter
	}

	// Create new limiter
	r.mu.Lock()
	defer r.mu.Unlock()

	// Double-check after acquiring write lock
	if limiter, exists := r.limiters[key]; exists {
		return limiter
	}

	limiter = &tokenBucket{
		tokens:   float64(r.capacity),
		capacity: r.capacity,
		rate:     r.rate,
		lastTime: time.Now(),
	}

	r.limiters[key] = limiter
	return limiter
}

// cleanup removes unused limiters periodically.
func (r *RateLimitInterceptor) cleanup() {
	ticker := time.NewTicker(r.cleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		r.mu.Lock()
		now := time.Now()

		for key, limiter := range r.limiters {
			limiter.mu.Lock()
			// Remove if unused for more than cleanup interval
			if now.Sub(limiter.lastTime) > r.cleanupInterval {
				delete(r.limiters, key)
			}
			limiter.mu.Unlock()
		}

		r.mu.Unlock()
	}
}

// allow tries to take a token from the bucket.
func (tb *tokenBucket) allow() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(tb.lastTime).Seconds()

	// Refill tokens based on elapsed time
	tb.tokens += elapsed * float64(tb.rate)
	if tb.tokens > float64(tb.capacity) {
		tb.tokens = float64(tb.capacity)
	}

	tb.lastTime = now

	// Try to take a token
	if tb.tokens >= 1.0 {
		tb.tokens -= 1.0
		return true
	}

	return false
}

// Common key extractors

// PerUserKeyExtractor extracts user ID from context.
func PerUserKeyExtractor(ctx context.Context, method string) string {
	userID := GetUserID(ctx)
	if userID != "" {
		return fmt.Sprintf("user:%s", userID)
	}
	return "anonymous"
}

// PerIPKeyExtractor extracts client IP from context.
func PerIPKeyExtractor(ctx context.Context, method string) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "unknown"
	}

	// Try x-forwarded-for first
	if xff := md.Get("x-forwarded-for"); len(xff) > 0 {
		return fmt.Sprintf("ip:%s", xff[0])
	}

	// Fall back to x-real-ip
	if xri := md.Get("x-real-ip"); len(xri) > 0 {
		return fmt.Sprintf("ip:%s", xri[0])
	}

	return "unknown"
}

// PerMethodKeyExtractor uses method name as key (global per-method limit).
func PerMethodKeyExtractor(ctx context.Context, method string) string {
	return fmt.Sprintf("method:%s", method)
}

// GlobalKeyExtractor uses a single global limit.
func GlobalKeyExtractor(ctx context.Context, method string) string {
	return "global"
}

// PerUserPerMethodKeyExtractor combines user ID and method.
func PerUserPerMethodKeyExtractor(ctx context.Context, method string) string {
	userID := GetUserID(ctx)
	if userID != "" {
		return fmt.Sprintf("user:%s:method:%s", userID, method)
	}
	return fmt.Sprintf("anonymous:method:%s", method)
}
