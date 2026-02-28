package upstream

import (
	"sync"
	"time"
)

// RateLimiter 速率限制器（令牌桶算法）
type RateLimiter struct {
	rate       int
	tokens     int
	maxTokens  int
	lastUpdate time.Time
	mu         sync.Mutex
}

// NewRateLimiter 创建速率限制器
// rate: 每秒允许的请求数
func NewRateLimiter(rate int) *RateLimiter {
	if rate <= 0 {
		rate = 20 // 默认20 req/s
	}

	return &RateLimiter{
		rate:       rate,
		tokens:     rate,
		maxTokens:  rate,
		lastUpdate: time.Now(),
	}
}

// Allow 检查是否允许请求
func (rl *RateLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(rl.lastUpdate)

	// 补充令牌
	newTokens := int(elapsed.Seconds() * float64(rl.rate))
	if newTokens > 0 {
		rl.tokens += newTokens
		if rl.tokens > rl.maxTokens {
			rl.tokens = rl.maxTokens
		}
		rl.lastUpdate = now
	}

	// 消耗令牌
	if rl.tokens > 0 {
		rl.tokens--
		return true
	}

	return false
}

// Wait 等待直到可以执行请求
func (rl *RateLimiter) Wait() time.Duration {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if rl.tokens > 0 {
		return 0
	}

	// 计算需要等待的时间
	waitTime := time.Second / time.Duration(rl.rate)
	return waitTime
}

// Used 获取已使用的令牌数
func (rl *RateLimiter) Used() int {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	return rl.maxTokens - rl.tokens
}

// Available 获取可用令牌数
func (rl *RateLimiter) Available() int {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	return rl.tokens
}

// Reset 重置速率限制器
func (rl *RateLimiter) Reset() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.tokens = rl.maxTokens
	rl.lastUpdate = time.Now()
}
