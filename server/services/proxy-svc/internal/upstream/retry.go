package upstream

import (
	"math"
	"time"
)

// RetryConfig 重试配置
type RetryConfig struct {
	MaxRetries  int           // 最大重试次数
	InitialWait time.Duration // 初始等待时间
	MaxWait     time.Duration // 最大等待时间
	Multiplier  float64       // 退避倍数
}

// DefaultRetryConfig 默认重试配置
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxRetries:  3,
		InitialWait: 100 * time.Millisecond,
		MaxWait:     2 * time.Second,
		Multiplier:  2.0,
	}
}

// CalculateBackoff 计算指数退避时间
func (r *RetryConfig) CalculateBackoff(attempt int) time.Duration {
	if attempt <= 0 {
		return 0
	}

	// 指数退避: initialWait * multiplier^(attempt-1)
	backoff := float64(r.InitialWait) * math.Pow(r.Multiplier, float64(attempt-1))

	// 限制最大等待时间
	if backoff > float64(r.MaxWait) {
		backoff = float64(r.MaxWait)
	}

	return time.Duration(backoff)
}

// ShouldRetry 判断是否应该重试
func (r *RetryConfig) ShouldRetry(attempt int, err error) bool {
	if attempt >= r.MaxRetries {
		return false
	}

	// 某些错误不应该重试
	if err == ErrSongNotFound || err == ErrRateLimitExceeded {
		return false
	}

	return true
}
