package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/xiaoxiao0301/listen-stream-v2/server/shared/pkg/logger"
)

// GetRadioCategories 获取电台分类
// GET /api/radio/categories
func (h *Handler) GetRadioCategories(c *gin.Context) {
	ctx := c.Request.Context()

	radios, err := h.upstreamClient.GetRadioCategories(ctx)
	if err != nil {
		h.log.WithFields(
			logger.String("request_id", getRequestID(c)),
			logger.String("error", err.Error()),
		).Error("Failed to get radio categories")

		InternalError(c, "Failed to get radio categories")
		return
	}

	Success(c, radios)
}

// GetRadioSongs 获取电台歌曲列表
// GET /api/radio/songs?radio_id=xxx
func (h *Handler) GetRadioSongs(c *gin.Context) {
	ctx := c.Request.Context()
	radioID := getIntParam(c, "radio_id", 0)

	if radioID == 0 {
		BadRequest(c, "Missing or invalid radio_id parameter")
		return
	}

	songs, err := h.upstreamClient.GetRadioSongs(ctx, radioID)
	if err != nil {
		h.log.WithFields(
			logger.String("request_id", getRequestID(c)),
			logger.String("error", err.Error()),
			logger.Int("radio_id", radioID),
		).Error("Failed to get radio songs")

		InternalError(c, "Failed to get radio songs")
		return
	}

	Success(c, songs)
}
