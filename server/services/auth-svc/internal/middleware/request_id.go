package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RequestID middleware injects request ID into context
// Uses X-Request-ID header if provided, otherwise generates a new UUID
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Try to get request ID from header
		requestID := c.GetHeader("X-Request-ID")
		
		// Generate new UUID if not provided
		if requestID == "" {
			requestID = uuid.New().String()
		}
		
		// Store in context for handlers to use
		c.Set("request_id", requestID)
		
		// Echo back in response header
		c.Header("X-Request-ID", requestID)
		
		c.Next()
	}
}

// GetRequestID retrieves the request ID from context
func GetRequestID(c *gin.Context) string {
	if requestID, exists := c.Get("request_id"); exists {
		if id, ok := requestID.(string); ok {
			return id
		}
	}
	return ""
}
