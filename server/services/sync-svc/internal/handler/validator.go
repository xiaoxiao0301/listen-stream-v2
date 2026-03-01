package handler

import (
	"log"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimiter 简单的速率限制器
type RateLimiter struct {
	mu       sync.Mutex
	clients  map[string]*clientLimit
	maxReq   int           // 最大请求数
	window   time.Duration // 时间窗口
	cleanupInterval time.Duration
}

type clientLimit struct {
	requests []time.Time
	mu       sync.Mutex
}

// NewRateLimiter 创建速率限制器
func NewRateLimiter(maxReq int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		clients:         make(map[string]*clientLimit),
		maxReq:          maxReq,
		window:          window,
		cleanupInterval: window * 2,
	}
	
	// 启动清理协程
	go rl.cleanup()
	
	return rl
}

// Allow 检查是否允许请求
func (rl *RateLimiter) Allow(clientID string) bool {
	rl.mu.Lock()
	client, exists := rl.clients[clientID]
	if !exists {
		client = &clientLimit{
			requests: make([]time.Time, 0, rl.maxReq),
		}
		rl.clients[clientID] = client
	}
	rl.mu.Unlock()

	client.mu.Lock()
	defer client.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.window)

	// 移除过期的请求记录
	validRequests := make([]time.Time, 0, len(client.requests))
	for _, reqTime := range client.requests {
		if reqTime.After(cutoff) {
			validRequests = append(validRequests, reqTime)
		}
	}
	client.requests = validRequests

	// 检查是否超过限制
	if len(client.requests) >= rl.maxReq {
		return false
	}

	// 添加新请求
	client.requests = append(client.requests, now)
	return true
}

// cleanup 定期清理过期的客户端记录
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(rl.cleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		cutoff := time.Now().Add(-rl.window * 2)
		
		for clientID, client := range rl.clients {
			client.mu.Lock()
			if len(client.requests) == 0 || client.requests[len(client.requests)-1].Before(cutoff) {
				delete(rl.clients, clientID)
			}
			client.mu.Unlock()
		}
		rl.mu.Unlock()
	}
}

// RateLimitMiddleware 速率限制中间件
func RateLimitMiddleware(limiter *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 使用用户ID作为限制键（如果有）
		clientID := c.GetString("user_id")
		if clientID == "" {
			// 如果没有用户ID，使用IP地址
			clientID = c.ClientIP()
		}

		if !limiter.Allow(clientID) {
			log.Printf("Rate limit exceeded for client: %s", clientID)
			c.JSON(429, gin.H{
				"error":   "rate_limit_exceeded",
				"message": "Too many requests, please try again later",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// ValidateEventRequest 验证事件请求
func ValidateEventRequest(c *gin.Context) {
	// 检查请求大小（防止过大payload）
	const maxBodySize = 1 << 20 // 1MB

	if c.Request.ContentLength > maxBodySize {
		log.Printf("Request body too large: %d bytes from %s", 
			c.Request.ContentLength, c.ClientIP())
		c.JSON(413, gin.H{
			"error":   "payload_too_large",
			"message": "Request body must be less than 1MB",
		})
		c.Abort()
		return
	}

	c.Next()
}

// RequestLogger 请求日志中间件（增强版）
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		// 处理请求
		c.Next()

		// 记录请求信息
		latency := time.Since(start)
		statusCode := c.Writer.Status()
		clientIP := c.ClientIP()
		method := c.Request.Method
		userID := c.GetString("user_id")

		// 构建日志消息
		logMsg := map[string]interface{}{
			"method":     method,
			"path":       path,
			"query":      query,
			"status":     statusCode,
			"latency_ms": latency.Milliseconds(),
			"client_ip":  clientIP,
		}

		if userID != "" {
			logMsg["user_id"] = userID
		}

		// 记录错误
		if len(c.Errors) > 0 {
			logMsg["errors"] = c.Errors.Errors()
		}

		// 根据状态码决定日志级别
		if statusCode >= 500 {
			log.Printf("[ERROR] %v", logMsg)
		} else if statusCode >= 400 {
			log.Printf("[WARN] %v", logMsg)
		} else {
			log.Printf("[INFO] %v", logMsg)
		}
	}
}
