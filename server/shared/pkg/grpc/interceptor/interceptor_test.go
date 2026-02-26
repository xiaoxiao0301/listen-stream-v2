package interceptor

import (
	"context"
	"fmt"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// Mock handler for testing
func mockUnaryHandler(ctx context.Context, req interface{}) (interface{}, error) {
	return "response", nil
}

func mockUnaryHandlerWithError(ctx context.Context, req interface{}) (interface{}, error) {
	return nil, status.Error(codes.Internal, "test error")
}

func mockUnaryHandlerWithPanic(ctx context.Context, req interface{}) (interface{}, error) {
	panic("test panic")
}

// TestLoggingInterceptor tests the logging interceptor
func TestLoggingInterceptor(t *testing.T) {
	logger := &testLogger{t: t}
	interceptor := NewLoggingInterceptor(logger)

	unary := interceptor.UnaryServerInterceptor()

	ctx := context.Background()
	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/Method",
	}

	// Test successful call
	resp, err := unary(ctx, "request", info, mockUnaryHandler)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if resp != "response" {
		t.Errorf("expected response, got %v", resp)
	}

	// Test error call
	_, err = unary(ctx, "request", info, mockUnaryHandlerWithError)
	if err == nil {
		t.Error("expected error")
	}
}

// TestRecoveryInterceptor tests panic recovery
func TestRecoveryInterceptor(t *testing.T) {
	var recovered interface{}
	interceptor := &RecoveryInterceptor{
		OnPanic: func(p interface{}, stack []byte) {
			recovered = p
		},
	}

	unary := interceptor.UnaryServerInterceptor()

	ctx := context.Background()
	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/Method",
	}

	// Test panic recovery
	_, err := unary(ctx, "request", info, mockUnaryHandlerWithPanic)
	if err == nil {
		t.Error("expected error from panic")
	}

	if recovered == nil {
		t.Error("expected panic to be recovered")
	}

	if recovered != "test panic" {
		t.Errorf("expected panic value 'test panic', got %v", recovered)
	}

	// Verify error is gRPC Internal error
	st, ok := status.FromError(err)
	if !ok {
		t.Error("expected gRPC status error")
	}
	if st.Code() != codes.Internal {
		t.Errorf("expected Internal code, got %v", st.Code())
	}
}

// TestTracingInterceptor tests OpenTelemetry tracing
func TestTracingInterceptor(t *testing.T) {
	interceptor := NewTracingInterceptor("test")

	unary := interceptor.UnaryServerInterceptor()

	ctx := context.Background()
	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/Method",
	}

	// Test successful call
	resp, err := unary(ctx, "request", info, mockUnaryHandler)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if resp != "response" {
		t.Errorf("expected response, got %v", resp)
	}

	// Test error call
	_, err = unary(ctx, "request", info, mockUnaryHandlerWithError)
	if err == nil {
		t.Error("expected error")
	}
}

// TestAuthInterceptor tests JWT authentication
func TestAuthInterceptor(t *testing.T) {
	// Mock token validator
	validator := func(token string) (string, error) {
		if token == "valid-token" {
			return "user123", nil
		}
		return "", fmt.Errorf("invalid token")
	}

	interceptor := NewAuthInterceptor(validator)
	unary := interceptor.UnaryServerInterceptor()

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/Method",
	}

	// Test with valid token
	md := metadata.Pairs("authorization", "Bearer valid-token")
	ctx := metadata.NewIncomingContext(context.Background(), md)

	resp, err := unary(ctx, "request", info, func(ctx context.Context, req interface{}) (interface{}, error) {
		// Check user ID was added to context
		userID := GetUserID(ctx)
		if userID != "user123" {
			return nil, fmt.Errorf("expected user ID user123, got %s", userID)
		}
		return "response", nil
	})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if resp != "response" {
		t.Errorf("expected response, got %v", resp)
	}

	// Test with invalid token
	md = metadata.Pairs("authorization", "Bearer invalid-token")
	ctx = metadata.NewIncomingContext(context.Background(), md)

	_, err = unary(ctx, "request", info, mockUnaryHandler)
	if err == nil {
		t.Error("expected authentication error")
	}

	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.Unauthenticated {
		t.Errorf("expected Unauthenticated error, got %v", err)
	}

	// Test without token
	ctx = context.Background()
	_, err = unary(ctx, "request", info, mockUnaryHandler)
	if err == nil {
		t.Error("expected authentication error")
	}
}

// TestAuthInterceptorSkipMethods tests method skipping
func TestAuthInterceptorSkipMethods(t *testing.T) {
	validator := func(token string) (string, error) {
		return "", fmt.Errorf("should not be called")
	}

	interceptor := NewAuthInterceptor(validator).
		WithSkipMethods("/test.Service/Public")

	unary := interceptor.UnaryServerInterceptor()

	ctx := context.Background()
	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/Public",
	}

	// Should not require auth
	resp, err := unary(ctx, "request", info, mockUnaryHandler)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if resp != "response" {
		t.Errorf("expected response, got %v", resp)
	}
}

// TestRateLimitInterceptor tests rate limiting
func TestRateLimitInterceptor(t *testing.T) {
	// Allow 2 requests per second
	interceptor := NewRateLimitInterceptor(2, 2, GlobalKeyExtractor)

	unary := interceptor.UnaryServerInterceptor()

	ctx := context.Background()
	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/Method",
	}

	// First two requests should succeed
	for i := 0; i < 2; i++ {
		_, err := unary(ctx, "request", info, mockUnaryHandler)
		if err != nil {
			t.Errorf("request %d: expected no error, got %v", i+1, err)
		}
	}

	// Third request should be rate limited
	_, err := unary(ctx, "request", info, mockUnaryHandler)
	if err == nil {
		t.Error("expected rate limit error")
	}

	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.ResourceExhausted {
		t.Errorf("expected ResourceExhausted error, got %v", err)
	}

	// Wait for token refill
	time.Sleep(600 * time.Millisecond)

	// Should succeed again
	_, err = unary(ctx, "request", info, mockUnaryHandler)
	if err != nil {
		t.Errorf("expected no error after refill, got %v", err)
	}
}

// TestPerUserKeyExtractor tests user-based rate limiting
func TestPerUserKeyExtractor(t *testing.T) {
	ctx1 := context.WithValue(context.Background(), userIDKey, "user1")
	ctx2 := context.WithValue(context.Background(), userIDKey, "user2")
	ctx3 := context.Background()

	key1 := PerUserKeyExtractor(ctx1, "/test")
	key2 := PerUserKeyExtractor(ctx2, "/test")
	key3 := PerUserKeyExtractor(ctx3, "/test")

	if key1 != "user:user1" {
		t.Errorf("expected 'user:user1', got %s", key1)
	}

	if key2 != "user:user2" {
		t.Errorf("expected 'user:user2', got %s", key2)
	}

	if key3 != "anonymous" {
		t.Errorf("expected 'anonymous', got %s", key3)
	}

	// Different users should have different keys
	if key1 == key2 {
		t.Error("different users should have different keys")
	}
}

// TestExtractRequestID tests request ID extraction
func TestExtractRequestID(t *testing.T) {
	// Test with request ID in metadata
	md := metadata.Pairs("x-request-id", "test-request-id")
	ctx := metadata.NewIncomingContext(context.Background(), md)

	requestID := extractRequestID(ctx)
	if requestID != "test-request-id" {
		t.Errorf("expected 'test-request-id', got %s", requestID)
	}

	// Test without request ID
	ctx = context.Background()
	requestID = extractRequestID(ctx)
	if requestID == "" {
		t.Error("expected generated request ID")
	}
}

// Benchmarks

func BenchmarkLoggingInterceptor(b *testing.B) {
	interceptor := NewLoggingInterceptor(nil)
	unary := interceptor.UnaryServerInterceptor()

	ctx := context.Background()
	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/Method",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = unary(ctx, "request", info, mockUnaryHandler)
	}
}

func BenchmarkAuthInterceptor(b *testing.B) {
	validator := func(token string) (string, error) {
		return "user123", nil
	}

	interceptor := NewAuthInterceptor(validator)
	unary := interceptor.UnaryServerInterceptor()

	md := metadata.Pairs("authorization", "Bearer valid-token")
	ctx := metadata.NewIncomingContext(context.Background(), md)

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/Method",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = unary(ctx, "request", info, mockUnaryHandler)
	}
}

// testLogger is a simple logger for testing
type testLogger struct {
	t *testing.T
}

func (l *testLogger) Info(msg string, fields ...any) {
	l.t.Logf("[INFO] %s %v", msg, fields)
}

func (l *testLogger) Error(msg string, fields ...any) {
	l.t.Logf("[ERROR] %s %v", msg, fields)
}

func (l *testLogger) Warn(msg string, fields ...any) {
	l.t.Logf("[WARN] %s %v", msg, fields)
}
