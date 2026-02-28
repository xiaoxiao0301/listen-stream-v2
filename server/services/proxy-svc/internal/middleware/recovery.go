package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"github.com/xiaoxiao0301/listen-stream-v2/server/shared/pkg/logger"
)

// Recovery panic恢复中间件
func Recovery(log logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// 获取堆栈信息
				stack := string(debug.Stack())

				// 获取请求ID
				requestID := GetRequestID(c)

				// 记录panic日志
				log.WithFields(
					logger.String("request_id", requestID),
					logger.String("panic", fmt.Sprintf("%v", err)),
					logger.String("stack", stack),
				).Error("Panic recovered")

				// 返回500错误
				c.JSON(http.StatusInternalServerError, gin.H{
					"code":       500,
					"message":    "Internal Server Error",
					"request_id": requestID,
				})

				c.Abort()
			}
		}()

		c.Next()
	}
}
