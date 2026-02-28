package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/xiaoxiao0301/listen-stream-v2/server/services/proxy-svc/internal/upstream"
	"github.com/xiaoxiao0301/listen-stream-v2/server/shared/pkg/logger"
)

// Handler 处理器基类
type Handler struct {
	upstreamClient upstream.ClientInterface
	log            logger.Logger
}

// NewHandler 创建处理器
func NewHandler(upstreamClient upstream.ClientInterface, log logger.Logger) *Handler {
	return &Handler{
		upstreamClient: upstreamClient,
		log:            log,
	}
}

// getIntParam 获取整数参数（带默认值）
func getIntParam(c *gin.Context, key string, defaultValue int) int {
	valueStr := c.Query(key)
	if valueStr == "" {
		return defaultValue
	}

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}

	return value
}

// getStringParam 获取字符串参数（带默认值）
func getStringParam(c *gin.Context, key string, defaultValue string) string {
	value := c.Query(key)
	if value == "" {
		return defaultValue
	}
	return value
}
