package middleware

import (
"github.com/gin-gonic/gin"
"github.com/google/uuid"
)

const (
// RequestIDHeader X-Request-ID请求头
RequestIDHeader = "X-Request-ID"
// RequestIDKey 上下文中的Key
RequestIDKey = "request_id"
)

// RequestID 中间件：注入请求ID
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 优先使用客户端传入的Request ID
		requestID := c.GetHeader(RequestIDHeader)
		if requestID == "" {
			// 未提供则生成新的UUID
			requestID = uuid.NewString()
		}

		// 设置到响应头
		c.Writer.Header().Set(RequestIDHeader, requestID)

		// 存储到上下文
		c.Set(RequestIDKey, requestID)

		c.Next()
	}
}

// GetRequestID 从上下文获取请求ID
func GetRequestID(c *gin.Context) string {
	if id, exists := c.Get(RequestIDKey); exists {
		if requestID, ok := id.(string); ok {
			return requestID
		}
	}
	return ""
}
