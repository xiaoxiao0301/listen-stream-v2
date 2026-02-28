package middleware

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// RateLimiter 速率限制器
type RateLimiter struct {
	// IP维度限流器
	ipLimiters map[string]*rate.Limiter
	ipMu       sync.RWMutex
	ipRate     rate.Limit
	ipBurst    int

	// User维度限流器
	userLimiters map[string]*rate.Limiter
	userMu       sync.RWMutex
	userRate     rate.Limit
	userBurst    int

	// 清理器
	cleanupInterval time.Duration
	lastCleanup     time.Time
	cleanupMu       sync.Mutex
}

// NewRateLimiter 创建速率限制器
func NewRateLimiter(ipPerSecond, ipBurst, userPerSecond, userBurst int) *RateLimiter {
	rl := &RateLimiter{
		ipLimiters:      make(map[string]*rate.Limiter),
		ipRate:          rate.Limit(ipPerSecond),
		ipBurst:         ipBurst,
		userLimiters:    make(map[string]*rate.Limiter),
		userRate:        rate.Limit(userPerSecond),
		userBurst:       userBurst,
		cleanupInterval: 10 * time.Minute,
		lastCleanup:     time.Now(),
	}

	return rl
}

// getIPLimiter 获取IP限流器
func (rl *RateLimiter) getIPLimiter(ip string) *rate.Limiter {
	rl.ipMu.RLock()
	limiter, exists := rl.ipLimiters[ip]
	rl.ipMu.RUnlock()

	if !exists {
		rl.ipMu.Lock()
		// 双重检查
		limiter, exists = rl.ipLimiters[ip]
		if !exists {
			limiter = rate.NewLimiter(rl.ipRate, rl.ipBurst)
			rl.ipLimiters[ip] = limiter
		}
		rl.ipMu.Unlock()
	}

	return limiter
}

// getUserLimiter 获取用户限流器
func (rl *RateLimiter) getUserLimiter(userID string) *rate.Limiter {
	rl.userMu.RLock()
	limiter, exists := rl.userLimiters[userID]
	rl.userMu.RUnlock()

	if !exists {
		rl.userMu.Lock()
		// 双重检查
		limiter, exists = rl.userLimiters[userID]
		if !exists {
			limiter = rate.NewLimiter(rl.userRate, rl.userBurst)
			rl.userLimiters[userID] = limiter
		}
		rl.userMu.Unlock()
	}

	return limiter
}

// cleanup 清理过期的限流器
func (rl *RateLimiter) cleanup() {
	rl.cleanupMu.Lock()
	defer rl.cleanupMu.Unlock()

	now := time.Now()
	if now.Sub(rl.lastCleanup) < rl.cleanupInterval {
		return
	}

	// 清理IP限流器（简化：清空所有，实际应该清理不活跃的）
	rl.ipMu.Lock()
	if len(rl.ipLimiters) > 10000 {
		rl.ipLimiters = make(map[string]*rate.Limiter)
	}
	rl.ipMu.Unlock()

	// 清理用户限流器
	rl.userMu.Lock()
	if len(rl.userLimiters) > 10000 {
		rl.userLimiters = make(map[string]*rate.Limiter)
	}
	rl.userMu.Unlock()

	rl.lastCleanup = now
}

// Limit 限流中间件
func (rl *RateLimiter) Limit() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 定期清理
		rl.cleanup()

		// IP维度限流
		ip := c.ClientIP()
		ipLimiter := rl.getIPLimiter(ip)
		if !ipLimiter.Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"code":    429,
				"message": "Rate limit exceeded (IP)",
			})
			c.Abort()
			return
		}

		// User维度限流（如果已认证）
		if userID, exists := c.Get("user_id"); exists {
			if uid, ok := userID.(string); ok && uid != "" {
				userLimiter := rl.getUserLimiter(uid)
				if !userLimiter.Allow() {
					c.JSON(http.StatusTooManyRequests, gin.H{
						"code":    429,
						"message": fmt.Sprintf("Rate limit exceeded (User: %s)", uid),
					})
					c.Abort()
					return
				}
			}
		}

		c.Next()
	}
}
