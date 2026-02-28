package upstream

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/xiaoxiao0301/listen-stream-v2/server/shared/pkg/logger"
)

func TestCircuitBreaker(t *testing.T) {
	// 创建mock logger
	log := &mockLogger{}
	breaker := NewCircuitBreaker(DefaultBreakerSettings(), log)

	// 测试初始状态
	if breaker.State() != "closed" {
		t.Errorf("Expected initial state 'closed', got'%s'", breaker.State())
	}

	// 模拟5次失败，触发熔断
	for i := 0; i < 5; i++ {
		_, err := breaker.Execute(func() (interface{}, error) {
			return nil, ErrUpstreamUnavailable
		})
		if err == nil {
			t.Error("Expected error, got nil")
		}
	}

	// 熔断器应该打开
	if breaker.State() != "open" {
		t.Errorf("Expected state 'open' after failures, got '%s'", breaker.State())
	}

	// 打开状态应该拒绝请求
	_, err := breaker.Execute(func() (interface{}, error) {
		return "success", nil
	})
	if err != ErrCircuitOpen {
		t.Errorf("Expected ErrCircuitOpen, got %v", err)
	}
}

// mockLogger 用于测试的mock logger
type mockLogger struct{}

func (m *mockLogger) Debug(msg string, fields ...logger.Field)       {}
func (m *mockLogger) Info(msg string, fields ...logger.Field)        {}
func (m *mockLogger) Warn(msg string, fields ...logger.Field)        {}
func (m *mockLogger) Error(msg string, fields ...logger.Field)       {}
func (m *mockLogger) Fatal(msg string, fields ...logger.Field)       {}
func (m *mockLogger) SetLevel(level logger.Level)                    {}
func (m *mockLogger) GetLevel() logger.Level                         { return logger.DebugLevel }
func (m *mockLogger) WithContext(ctx context.Context) logger.Logger  { return m }
func (m *mockLogger) WithFields(fields ...logger.Field) logger.Logger { return m }
func (m *mockLogger) Writer() io.Writer                              { return io.Discard }

func TestRetryConfig(t *testing.T) {
	config := DefaultRetryConfig()

	// 测试退避时间计算
	tests := []struct {
		attempt  int
		expected time.Duration
	}{
		{1, 100 * time.Millisecond},
		{2, 200 * time.Millisecond},
		{3, 400 * time.Millisecond},
		{4, 800 * time.Millisecond},
		{5, 1600 * time.Millisecond},
		{6, 2000 * time.Millisecond}, // 达到最大值
	}

	for _, tt := range tests {
		backoff := config.CalculateBackoff(tt.attempt)
		if backoff != tt.expected {
			t.Errorf("Attempt %d: expected %v, got %v", tt.attempt, tt.expected, backoff)
		}
	}
}

func TestRateLimiter(t *testing.T) {
	limiter := NewRateLimiter(10) // 10 req/s

	// 前10个请求应该成功
	for i := 0; i < 10; i++ {
		if !limiter.Allow() {
			t.Errorf("Request %d should be allowed", i+1)
		}
	}

	// 第11个请求应该被限制
	if limiter.Allow() {
		t.Error("Request 11 should be rate limited")
	}

	// 等待1秒后应该可以再次请求
	time.Sleep(1 * time.Second)
	if !limiter.Allow() {
		t.Error("Request should be allowed after waiting")
	}
}

func TestClient(t *testing.T) {
	// 这是一个mock测试，实际需要mock HTTP服务器
	t.Skip("Requires mock HTTP server")
}

func BenchmarkCircuitBreaker(b *testing.B) {
	breaker := NewCircuitBreaker(DefaultBreakerSettings(), nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		breaker.Execute(func() (interface{}, error) {
			return "success", nil
		})
	}
}

func BenchmarkRateLimiter(b *testing.B) {
	limiter := NewRateLimiter(1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		limiter.Allow()
	}
}

func ExampleCircuitBreaker() {
	ctx := context.Background()
	_ = ctx

	// 创建熔断器
	// breaker := NewCircuitBreaker(DefaultBreakerSettings(), logger.Default())

	// 使用熔断器执行请求
	// result, err := breaker.Execute(func() (interface{}, error) {
	//     return fetchDataFromUpstream()
	// })
}
