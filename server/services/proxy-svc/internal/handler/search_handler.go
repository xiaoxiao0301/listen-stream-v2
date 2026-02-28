package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/xiaoxiao0301/listen-stream-v2/server/shared/pkg/logger"
)

// GetHotKeys 获取热门搜索关键词
// GET /api/search/hotkeys
func (h *Handler) GetHotKeys(c *gin.Context) {
	ctx := c.Request.Context()

	hotKeys, err := h.upstreamClient.GetHotKeys(ctx)
	if err != nil {
		h.log.WithFields(
			logger.String("request_id", getRequestID(c)),
			logger.String("error", err.Error()),
		).Error("Failed to get hot keys")

		InternalError(c, "Failed to get hot keys")
		return
	}

	Success(c, hotKeys)
}

// SearchSongs 搜索歌曲
// GET /api/search/songs?keyword=xxx&page=1&size=20
func (h *Handler) SearchSongs(c *gin.Context) {
	ctx := c.Request.Context()
	keyword := c.Query("keyword")

	if keyword == "" {
		BadRequest(c, "Missing keyword parameter")
		return
	}

	page := getIntParam(c, "page", 1)
	size := getIntParam(c, "size", 20)

	result, err := h.upstreamClient.SearchSongs(ctx, keyword, page, size)
	if err != nil {
		h.log.WithFields(
			logger.String("request_id", getRequestID(c)),
			logger.String("error", err.Error()),
			logger.String("keyword", keyword),
		).Error("Failed to search songs")

		InternalError(c, "Failed to search songs")
		return
	}

	Success(c, result)
}

// SearchSingers 搜索歌手
// GET /api/search/singers?keyword=xxx&page=1&size=20
func (h *Handler) SearchSingers(c *gin.Context) {
	ctx := c.Request.Context()
	keyword := c.Query("keyword")

	if keyword == "" {
		BadRequest(c, "Missing keyword parameter")
		return
	}

	page := getIntParam(c, "page", 1)
	size := getIntParam(c, "size", 20)

	result, err := h.upstreamClient.SearchSingers(ctx, keyword, page, size)
	if err != nil {
		h.log.WithFields(
			logger.String("request_id", getRequestID(c)),
			logger.String("error", err.Error()),
			logger.String("keyword", keyword),
		).Error("Failed to search singers")

		InternalError(c, "Failed to search singers")
		return
	}

	Success(c, result)
}

// SearchAlbums 搜索专辑
// GET /api/search/albums?keyword=xxx&page=1&size=20
func (h *Handler) SearchAlbums(c *gin.Context) {
	ctx := c.Request.Context()
	keyword := c.Query("keyword")

	if keyword == "" {
		BadRequest(c, "Missing keyword parameter")
		return
	}

	page := getIntParam(c, "page", 1)
	size := getIntParam(c, "size", 20)

	result, err := h.upstreamClient.SearchAlbums(ctx, keyword, page, size)
	if err != nil {
		h.log.WithFields(
			logger.String("request_id", getRequestID(c)),
			logger.String("error", err.Error()),
			logger.String("keyword", keyword),
		).Error("Failed to search albums")

		InternalError(c, "Failed to search albums")
		return
	}

	Success(c, result)
}

// SearchMVs 搜索MV
// GET /api/search/mvs?keyword=xxx&page=1&size=20
func (h *Handler) SearchMVs(c *gin.Context) {
	ctx := c.Request.Context()
	keyword := c.Query("keyword")

	if keyword == "" {
		BadRequest(c, "Missing keyword parameter")
		return
	}

	page := getIntParam(c, "page", 1)
	size := getIntParam(c, "size", 20)

	result, err := h.upstreamClient.SearchMVs(ctx, keyword, page, size)
	if err != nil {
		h.log.WithFields(
			logger.String("request_id", getRequestID(c)),
			logger.String("error", err.Error()),
			logger.String("keyword", keyword),
		).Error("Failed to search MVs")

		InternalError(c, "Failed to search MVs")
		return
	}

	Success(c, result)
}
