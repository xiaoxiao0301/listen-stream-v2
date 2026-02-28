package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/listen-stream/server/shared/pkg/logger"
)

// Logging middleware logs HTTP requests with structured information
func Logging(log logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Start timer
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery
		
		// Get request ID
		requestID := GetRequestID(c)
		
		// Process request
		c.Next()
		
		// Calculate latency
		latency := time.Since(start)
		
		// Get status code
		statusCode := c.Writer.Status()
		
		// Get client IP
		clientIP := c.ClientIP()
		
		// Get method
		method := c.Request.Method
		
		// Prepare fields
		fields := []logger.Field{
			logger.String("request_id", requestID),
			logger.String("method", method),
			logger.String("path", path),
			logger.String("query", query),
			logger.Int("status", statusCode),
			logger.String("ip", clientIP),
			logger.String("user_agent", c.Request.UserAgent()),
			logger.Int64("latency_ms", latency.Milliseconds()),
		}
		
		// Add user ID if available
		if userID, exists := c.Get("user_id"); exists {
			if uid, ok := userID.(string); ok {
				fields = append(fields, logger.String("user_id", uid))
			}
		}
		
		// Add device ID if available
		if deviceID, exists := c.Get("device_id"); exists {
			if did, ok := deviceID.(string); ok {
				fields = append(fields, logger.String("device_id", did))
			}
		}
		
		// Log based on status code
		if statusCode >= 500 {
			// Get error if exists
			if len(c.Errors) > 0 {
				fields = append(fields, logger.String("error", c.Errors.String()))
			}
			log.Error("HTTP request failed with server error", fields...)
		} else if statusCode >= 400 {
			log.Warn("HTTP request failed with client error", fields...)
		} else {
			log.Info("HTTP request completed", fields...)
		}
	}
}
