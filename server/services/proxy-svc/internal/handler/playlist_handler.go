package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/xiaoxiao0301/listen-stream-v2/server/services/proxy-svc/internal/upstream"
	"github.com/xiaoxiao0301/listen-stream-v2/server/shared/pkg/logger"
)

// GetPlaylistCategories 获取歌单分类
// GET /api/playlist/categories
func (h *Handler) GetPlaylistCategories(c *gin.Context) {
	ctx := c.Request.Context()

	categories, err := h.upstreamClient.GetPlaylistCategories(ctx)
	if err != nil {
		h.log.WithFields(
			logger.String("request_id", getRequestID(c)),
			logger.String("error", err.Error()),
		).Error("Failed to get playlist categories")

		InternalError(c, "Failed to get playlist categories")
		return
	}

	Success(c, categories)
}

// GetPlaylistsByCategory 按分类获取歌单列表
// GET /api/playlist/list?number=1&size=20&sort=2&id=10000000
// number: 页码, size: 每页数量, sort: 排序(2-最新, 5-推荐), id: 分类ID
func (h *Handler) GetPlaylistsByCategory(c *gin.Context) {
	ctx := c.Request.Context()

	req := upstream.GetPlaylistsByCategoryRequest{
		Number: getIntParam(c, "number", 1),
		Size:   getIntParam(c, "size", 20),
		Sort:   getIntParam(c, "sort", 5),
		ID:     getIntParam(c, "id", 10000000),
	}

	result, err := h.upstreamClient.GetPlaylistsByCategory(ctx, req)
	if err != nil {
		h.log.WithFields(
			logger.String("request_id", getRequestID(c)),
			logger.String("error", err.Error()),
			logger.Int("number", req.Number),
			logger.Int("size", req.Size),
		).Error("Failed to get playlists by category")

		InternalError(c, "Failed to get playlists by category")
		return
	}

	Success(c, result)
}

// GetPlaylistDetail 获取歌单详细信息
// GET /api/playlist/detail?diss_id=xxx
func (h *Handler) GetPlaylistDetail(c *gin.Context) {
	ctx := c.Request.Context()
	dissID := c.Query("diss_id")

	if dissID == "" {
		BadRequest(c, "Missing diss_id parameter")
		return
	}

	detail, err := h.upstreamClient.GetPlaylistDetail(ctx, dissID)
	if err != nil {
		h.log.WithFields(
			logger.String("request_id", getRequestID(c)),
			logger.String("error", err.Error()),
			logger.String("diss_id", dissID),
		).Error("Failed to get playlist detail")

		InternalError(c, "Failed to get playlist detail")
		return
	}

	Success(c, detail)
}
