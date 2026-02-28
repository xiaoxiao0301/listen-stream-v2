package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/xiaoxiao0301/listen-stream-v2/server/shared/pkg/logger"
)

// GetMVCategories 获取MV分类
// GET /api/mv/categories
func (h *Handler) GetMVCategories(c *gin.Context) {
	ctx := c.Request.Context()

	categories, err := h.upstreamClient.GetMVCategories(ctx)
	if err != nil {
		h.log.WithFields(
			logger.String("request_id", getRequestID(c)),
			logger.String("error", err.Error()),
		).Error("Failed to get MV categories")

		InternalError(c, "Failed to get MV categories")
		return
	}

	Success(c, categories)
}

// GetMVList 获取MV列表
// GET /api/mv/list?area=15&version=7&page=1&size=20
func (h *Handler) GetMVList(c *gin.Context) {
	ctx := c.Request.Context()

	area := getIntParam(c, "area", 15)
	version := getIntParam(c, "version", 7)
	page := getIntParam(c, "page", 1)
	size := getIntParam(c, "size", 20)

	result, err := h.upstreamClient.GetMVList(ctx, area, version, page, size)
	if err != nil {
		h.log.WithFields(
			logger.String("request_id", getRequestID(c)),
			logger.String("error", err.Error()),
			logger.Int("area", area),
		).Error("Failed to get MV list")

		InternalError(c, "Failed to get MV list")
		return
	}

	Success(c, result)
}

// GetMVDetail 获取MV详细信息
// GET /api/mv/detail?vid=xxx
func (h *Handler) GetMVDetail(c *gin.Context) {
	ctx := c.Request.Context()
	vid := c.Query("vid")

	if vid == "" {
		BadRequest(c, "Missing vid parameter")
		return
	}

	detail, err := h.upstreamClient.GetMVDetail(ctx, vid)
	if err != nil {
		h.log.WithFields(
			logger.String("request_id", getRequestID(c)),
			logger.String("error", err.Error()),
			logger.String("vid", vid),
		).Error("Failed to get MV detail")

		InternalError(c, "Failed to get MV detail")
		return
	}

	Success(c, detail)
}
