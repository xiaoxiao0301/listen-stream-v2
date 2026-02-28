package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/xiaoxiao0301/listen-stream-v2/server/shared/pkg/logger"
)

// GetRankingList 获取排行榜列表
// GET /api/ranking/list
func (h *Handler) GetRankingList(c *gin.Context) {
	ctx := c.Request.Context()

	rankings, err := h.upstreamClient.GetRankingList(ctx)
	if err != nil {
		h.log.WithFields(
			logger.String("request_id", getRequestID(c)),
			logger.String("error", err.Error()),
		).Error("Failed to get ranking list")

		InternalError(c, "Failed to get ranking list")
		return
	}

	Success(c, rankings)
}

// GetRankingDetail 获取排行榜详细信息
// GET /api/ranking/detail?top_id=4&page=1&size=100&period=2024_1
func (h *Handler) GetRankingDetail(c *gin.Context) {
	ctx := c.Request.Context()

	topID := getIntParam(c, "top_id", 4)
	page := getIntParam(c, "page", 1)
	size := getIntParam(c, "size", 100)
	period := getStringParam(c, "period", "")

	result, err := h.upstreamClient.GetRankingDetail(ctx, topID, page, size, period)
	if err != nil {
		h.log.WithFields(
			logger.String("request_id", getRequestID(c)),
			logger.String("error", err.Error()),
			logger.Int("top_id", topID),
		).Error("Failed to get ranking detail")

		InternalError(c, "Failed to get ranking detail")
		return
	}

	Success(c, result)
}
