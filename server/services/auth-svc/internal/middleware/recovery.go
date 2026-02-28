package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"github.com/xiaoxiao0301/listen-stream-v2/server/shared/pkg/logger"
)

// Recovery middleware recovers from panics and returns 500 error
func Recovery(log logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Get stack trace
				stack := debug.Stack()
				
				// Get request ID
				requestID := GetRequestID(c)
				
				// Log panic with full details
				log.Error("Panic recovered",
					logger.String("request_id", requestID),
					logger.String("method", c.Request.Method),
					logger.String("path", c.Request.URL.Path),
					logger.String("ip", c.ClientIP()),
					logger.String("panic", fmt.Sprintf("%v", err)),
					logger.String("stack", string(stack)),
				)
				
				// Return error response
				c.JSON(http.StatusInternalServerError, gin.H{
					"success": false,
					"error": gin.H{
						"code":    "INTERNAL_ERROR",
						"message": "Internal server error",
					},
					"request_id": requestID,
				})
				
				// Abort further handlers
				c.Abort()
			}
		}()
		
		c.Next()
	}
}
