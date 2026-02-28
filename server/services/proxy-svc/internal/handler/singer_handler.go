package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/xiaoxiao0301/listen-stream-v2/server/services/proxy-svc/internal/upstream"
	"github.com/xiaoxiao0301/listen-stream-v2/server/shared/pkg/logger"
)

// GetSingerCategories 获取歌手分类
// GET /api/singer/categories
func (h *Handler) GetSingerCategories(c *gin.Context) {
	ctx := c.Request.Context()

	categories, err := h.upstreamClient.GetSingerCategories(ctx)
	if err != nil {
		h.log.WithFields(
			logger.String("request_id", getRequestID(c)),
			logger.String("error", err.Error()),
		).Error("Failed to get singer categories")

		InternalError(c, "Failed to get singer categories")
		return
	}

	Success(c, categories)
}

// GetSingerList 获取歌手列表
// GET /api/singer/list?page=1&size=20&area=-100&genre=-100&sex=-100&index=-100
func (h *Handler) GetSingerList(c *gin.Context) {
	ctx := c.Request.Context()

	req := upstream.GetSingerListRequest{
		Page:  getIntParam(c, "page", 1),
		Size:  getIntParam(c, "size", 20),
		Area:  getIntParam(c, "area", -100),
		Genre: getIntParam(c, "genre", -100),
		Sex:   getIntParam(c, "sex", -100),
		Index: getIntParam(c, "index", -100),
	}

	result, err := h.upstreamClient.GetSingerList(ctx, req)
	if err != nil {
		h.log.WithFields(
			logger.String("request_id", getRequestID(c)),
			logger.String("error", err.Error()),
			logger.Int("page", req.Page),
		).Error("Failed to get singer list")

		InternalError(c, "Failed to get singer list")
		return
	}

	Success(c, result)
}

// GetSingerDetail 获取歌手详细信息
// GET /api/singer/detail?singer_mid=xxx&page=1
func (h *Handler) GetSingerDetail(c *gin.Context) {
	ctx := c.Request.Context()
	singerMid := c.Query("singer_mid")

	if singerMid == "" {
		BadRequest(c, "Missing singer_mid parameter")
		return
	}

	page := getIntParam(c, "page", 1)

	detail, err := h.upstreamClient.GetSingerDetail(ctx, singerMid, page)
	if err != nil {
		h.log.WithFields(
			logger.String("request_id", getRequestID(c)),
			logger.String("error", err.Error()),
			logger.String("singer_mid", singerMid),
		).Error("Failed to get singer detail")

		InternalError(c, "Failed to get singer detail")
		return
	}

	Success(c, detail)
}

// GetSingerAlbums 获取歌手专辑列表
// GET /api/singer/albums?singer_mid=xxx&page=1&size=20
func (h *Handler) GetSingerAlbums(c *gin.Context) {
	ctx := c.Request.Context()
	singerMid := c.Query("singer_mid")

	if singerMid == "" {
		BadRequest(c, "Missing singer_mid parameter")
		return
	}

	page := getIntParam(c, "page", 1)
	size := getIntParam(c, "size", 20)

	result, err := h.upstreamClient.GetSingerAlbums(ctx, singerMid, page, size)
	if err != nil {
		h.log.WithFields(
			logger.String("request_id", getRequestID(c)),
			logger.String("error", err.Error()),
			logger.String("singer_mid", singerMid),
		).Error("Failed to get singer albums")

		InternalError(c, "Failed to get singer albums")
		return
	}

	Success(c, result)
}

// GetSingerMVs 获取歌手MV列表
// GET /api/singer/mvs?singer_mid=xxx&page=1&size=20
func (h *Handler) GetSingerMVs(c *gin.Context) {
	ctx := c.Request.Context()
	singerMid := c.Query("singer_mid")

	if singerMid == "" {
		BadRequest(c, "Missing singer_mid parameter")
		return
	}

	page := getIntParam(c, "page", 1)
	size := getIntParam(c, "size", 20)

	result, err := h.upstreamClient.GetSingerMVs(ctx, singerMid, page, size)
	if err != nil {
		h.log.WithFields(
			logger.String("request_id", getRequestID(c)),
			logger.String("error", err.Error()),
			logger.String("singer_mid", singerMid),
		).Error("Failed to get singer MVs")

		InternalError(c, "Failed to get singer MVs")
		return
	}

	Success(c, result)
}

// GetSingerSongs 获取歌手歌曲列表
// GET /api/singer/songs?singer_mid=xxx&page=1&size=20
func (h *Handler) GetSingerSongs(c *gin.Context) {
	ctx := c.Request.Context()
	singerMid := c.Query("singer_mid")

	if singerMid == "" {
		BadRequest(c, "Missing singer_mid parameter")
		return
	}

	page := getIntParam(c, "page", 1)
	size := getIntParam(c, "size", 20)

	result, err := h.upstreamClient.GetSingerSongs(ctx, singerMid, page, size)
	if err != nil {
		h.log.WithFields(
			logger.String("request_id", getRequestID(c)),
			logger.String("error", err.Error()),
			logger.String("singer_mid", singerMid),
		).Error("Failed to get singer songs")

		InternalError(c, "Failed to get singer songs")
		return
	}

	Success(c, result)
}
