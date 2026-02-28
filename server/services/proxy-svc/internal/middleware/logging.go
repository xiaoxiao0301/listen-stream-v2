package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xiaoxiao0301/listen-stream-v2/server/shared/pkg/logger"
)

// Logging 日志中间件
func Logging(log logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		// 处理请求
		c.Next()

		// 计算耗时
		latency := time.Since(start)

		// 获取请求ID
		requestID := GetRequestID(c)

		// 记录日志
		fields := []logger.Field{
			logger.String("request_id", requestID),
			logger.String("method", c.Request.Method),
			logger.String("path", path),
			logger.String("query", query),
			logger.Int("status", c.Writer.Status()),
			logger.Duration("latency", latency),
			logger.String("client_ip", c.ClientIP()),
			logger.String("user_agent", c.Request.UserAgent()),
		}

		// 如果有错误，添加错误信息
		if len(c.Errors) > 0 {
			fields = append(fields, logger.String("errors", c.Errors.String()))
		}

		// 根据状态码决定日志级别
		switch {
		case c.Writer.Status() >= 500:
			log.WithFields(fields...).Error("HTTP request error")
		case c.Writer.Status() >= 400:
			log.WithFields(fields...).Warn("HTTP request warning")
		default:
			log.WithFields(fields...).Info("HTTP request")
		}
	}
}
