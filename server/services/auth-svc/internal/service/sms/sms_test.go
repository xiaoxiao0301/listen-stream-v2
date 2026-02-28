package sms

import (
	"context"
	"errors"
	"testing"
	"time"
)

// MockProvider Mock SMS提供商（用于测试）
type MockProvider struct {
	name       string
	available  bool
	shouldFail bool
	delay      time.Duration
}

func NewMockProvider(name string, available bool) *MockProvider {
	return &MockProvider{
		name:      name,
		available: available,
	}
}

func (m *MockProvider) Name() string {
	return m.name
}

func (m *MockProvider) IsAvailable() bool {
	return m.available
}

func (m *MockProvider) Send(ctx context.Context, phone, code string) error {
	// 模拟网络延迟
	if m.delay > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(m.delay):
		}
	}

	if m.shouldFail {
		return errors.New("mock provider send failed")
	}
	return nil
}

func (m *MockProvider) SetShouldFail(fail bool) {
	m.shouldFail = fail
}

func (m *MockProvider) SetDelay(delay time.Duration) {
	m.delay = delay
}

// ===== Provider Tests =====

func TestAliyunProvider_Name(t *testing.T) {
	provider := NewAliyunProvider(AliyunConfig{})
	if provider.Name() != "aliyun" {
		t.Errorf("expected name 'aliyun', got '%s'", provider.Name())
	}
}

func TestAliyunProvider_IsAvailable(t *testing.T) {
	tests := []struct {
		name     string
		config   AliyunConfig
		expected bool
	}{
		{
			name: "all fields set",
			config: AliyunConfig{
				Enabled:         true,
				AccessKeyID:     "test_id",
				AccessKeySecret: "test_secret",
				SignName:        "test_sign",
				TemplateCode:    "test_template",
			},
			expected: true,
		},
		{
			name: "disabled",
			config: AliyunConfig{
				Enabled:         false,
				AccessKeyID:     "test_id",
				AccessKeySecret: "test_secret",
				SignName:        "test_sign",
				TemplateCode:    "test_template",
			},
			expected: false,
		},
		{
			name: "missing access key",
			config: AliyunConfig{
				Enabled:         true,
				AccessKeySecret: "test_secret",
				SignName:        "test_sign",
				TemplateCode:    "test_template",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := NewAliyunProvider(tt.config)
			if got := provider.IsAvailable(); got != tt.expected {
				t.Errorf("IsAvailable() = %v, expected %v", got, tt.expected)
			}
		})
	}
}

func TestTencentProvider_Name(t *testing.T) {
	provider := NewTencentProvider(TencentConfig{})
	if provider.Name() != "tencent" {
		t.Errorf("expected name 'tencent', got '%s'", provider.Name())
	}
}

func TestTwilioProvider_Name(t *testing.T) {
	provider := NewTwilioProvider(TwilioConfig{})
	if provider.Name() != "twilio" {
		t.Errorf("expected name 'twilio', got '%s'", provider.Name())
	}
}

// ===== FallbackChain Tests =====

func TestFallbackChain_SingleProvider_Success(t *testing.T) {
	mock := NewMockProvider("mock1", true)
	chain := NewFallbackChain([]Provider{mock}, true)

	ctx := context.Background()
	result, err := chain.Send(ctx, "+8613800138000", "123456")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !result.Success {
		t.Error("expected success")
	}
	if result.Provider != "mock1" {
		t.Errorf("expected provider 'mock1', got '%s'", result.Provider)
	}
	if result.Attempts != 1 {
		t.Errorf("expected 1 attempt, got %d", result.Attempts)
	}
}

func TestFallbackChain_Fallback_Success(t *testing.T) {
	mock1 := NewMockProvider("mock1", true)
	mock1.SetShouldFail(true)

	mock2 := NewMockProvider("mock2", true)

	chain := NewFallbackChain([]Provider{mock1, mock2}, true)

	ctx := context.Background()
	result, err := chain.Send(ctx, "+8613800138000", "123456")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !result.Success {
		t.Error("expected success")
	}
	if result.Provider != "mock2" {
		t.Errorf("expected provider 'mock2', got '%s'", result.Provider)
	}
	if result.Attempts != 2 {
		t.Errorf("expected 2 attempts, got %d", result.Attempts)
	}
	if len(result.Errors) != 1 {
		t.Errorf("expected 1 error, got %d", len(result.Errors))
	}
}

func TestFallbackChain_AllFail(t *testing.T) {
	mock1 := NewMockProvider("mock1", true)
	mock1.SetShouldFail(true)

	mock2 := NewMockProvider("mock2", true)
	mock2.SetShouldFail(true)

	chain := NewFallbackChain([]Provider{mock1, mock2}, true)

	ctx := context.Background()
	result, err := chain.Send(ctx, "+8613800138000", "123456")

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if result.Success {
		t.Error("expected failure")
	}
	if result.Attempts != 2 {
		t.Errorf("expected 2 attempts, got %d", result.Attempts)
	}
	if len(result.Errors) != 2 {
		t.Errorf("expected 2 errors, got %d", len(result.Errors))
	}
}

func TestFallbackChain_DisabledFallback(t *testing.T) {
	mock1 := NewMockProvider("mock1", true)
	mock1.SetShouldFail(true)

	mock2 := NewMockProvider("mock2", true)

	// Fallback disabled
	chain := NewFallbackChain([]Provider{mock1, mock2}, false)

	ctx := context.Background()
	result, err := chain.Send(ctx, "+8613800138000", "123456")

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if result.Success {
		t.Error("expected failure")
	}
	// Should only try the first provider
	if result.Attempts != 1 {
		t.Errorf("expected 1 attempt, got %d", result.Attempts)
	}
}

func TestFallbackChain_ContextCancellation(t *testing.T) {
	mock := NewMockProvider("mock1", true)
	mock.SetDelay(2 * time.Second) // Long delay

	chain := NewFallbackChain([]Provider{mock}, true)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	result, err := chain.Send(ctx, "+8613800138000", "123456")

	if err != context.DeadlineExceeded {
		t.Errorf("expected DeadlineExceeded error, got %v", err)
	}
	if result.Success {
		t.Error("expected failure")
	}
}

func TestFallbackChain_NoAvailableProviders(t *testing.T) {
	mock := NewMockProvider("mock1", false) // Not available
	chain := NewFallbackChain([]Provider{mock}, true)

	ctx := context.Background()
	_, err := chain.Send(ctx, "+8613800138000", "123456")

	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestFallbackChain_GetAvailableProviders(t *testing.T) {
	mock1 := NewMockProvider("mock1", true)
	mock2 := NewMockProvider("mock2", true)
	mock3 := NewMockProvider("mock3", false) // Not available

	chain := NewFallbackChain([]Provider{mock1, mock2, mock3}, true)

	providers := chain.GetAvailableProviders()
	if len(providers) != 2 {
		t.Errorf("expected 2 available providers, got %d", len(providers))
	}
	if providers[0] != "mock1" || providers[1] != "mock2" {
		t.Errorf("unexpected providers: %v", providers)
	}
}

// ===== Config Tests =====

func TestNewConfig(t *testing.T) {
	config := NewConfig()

	if !config.FallbackEnabled {
		t.Error("expected FallbackEnabled to be true")
	}
	if config.Aliyun.Endpoint != "dysmsapi.aliyuncs.com" {
		t.Errorf("unexpected Aliyun endpoint: %s", config.Aliyun.Endpoint)
	}
	if config.Tencent.Region != "ap-guangzhou" {
		t.Errorf("unexpected Tencent region: %s", config.Tencent.Region)
	}
}
