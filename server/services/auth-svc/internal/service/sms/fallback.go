package sms

import (
	"context"
	"fmt"
	"time"
)

// FallbackChain SMS Fallback链
// 按优先级顺序尝试多个提供商，直到成功或全部失败
type FallbackChain struct {
	providers []Provider
	enabled   bool
}

// NewFallbackChain 创建Fallback链
func NewFallbackChain(providers []Provider, enabled bool) *FallbackChain {
	// 过滤出可用的提供商
	available := make([]Provider, 0, len(providers))
	for _, p := range providers {
		if p.IsAvailable() {
			available = append(available, p)
		}
	}

	return &FallbackChain{
		providers: available,
		enabled:   enabled,
	}
}

// SendResult 发送结果
type SendResult struct {
	Success      bool          // 是否成功
	Provider     string        // 最终使用的提供商
	Attempts     int           // 尝试次数
	TotalLatency time.Duration // 总延迟
	Errors       []string      // 所有错误信息
}

// Send 发送短信（带Fallback）
func (c *FallbackChain) Send(ctx context.Context, phone, code string) (*SendResult, error) {
	if len(c.providers) == 0 {
		return nil, fmt.Errorf("no available sms providers")
	}

	result := &SendResult{
		Success:  false,
		Attempts: 0,
		Errors:   make([]string, 0),
	}

	startTime := time.Now()

	// 如果未启用Fallback，只尝试第一个提供商
	providers := c.providers
	if !c.enabled {
		providers = c.providers[:1]
	}

	// 按顺序尝试每个提供商
	for i, provider := range providers {
		result.Attempts++

		// 检查context是否已取消
		select {
		case <-ctx.Done():
			result.TotalLatency = time.Since(startTime)
			return result, ctx.Err()
		default:
		}

		// 尝试发送
		providerStart := time.Now()
		err := provider.Send(ctx, phone, code)
		providerLatency := time.Since(providerStart)

		if err == nil {
			// 发送成功
			result.Success = true
			result.Provider = provider.Name()
			result.TotalLatency = time.Since(startTime)
			return result, nil
		}

		// 检查是否是context错误
		if err == context.Canceled || err == context.DeadlineExceeded {
			result.TotalLatency = time.Since(startTime)
			return result, err
		}

		// 发送失败，记录错误
		errMsg := fmt.Sprintf("[%s] failed in %v: %v", provider.Name(), providerLatency, err)
		result.Errors = append(result.Errors, errMsg)

		// 如果是最后一个提供商，不需要等待
		if i < len(providers)-1 {
			// 短暂延迟后尝试下一个提供商
			select {
			case <-ctx.Done():
				result.TotalLatency = time.Since(startTime)
				return result, ctx.Err()
			case <-time.After(100 * time.Millisecond):
				// 继续下一个提供商
			}
		}
	}

	// 所有提供商都失败
	result.TotalLatency = time.Since(startTime)
	return result, fmt.Errorf("all sms providers failed: tried %d providers", result.Attempts)
}

// GetAvailableProviders 获取可用的提供商列表
func (c *FallbackChain) GetAvailableProviders() []string {
	names := make([]string, 0, len(c.providers))
	for _, p := range c.providers {
		names = append(names, p.Name())
	}
	return names
}

// IsEnabled 是否启用Fallback
func (c *FallbackChain) IsEnabled() bool {
	return c.enabled
}

// Count 返回可用提供商数量
func (c *FallbackChain) Count() int {
	return len(c.providers)
}
