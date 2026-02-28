package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/xiaoxiao0301/listen-stream-v2/server/shared/pkg/logger"
)

// GetBanners 获取轮播图
// GET /api/home/banners
func (h *Handler) GetBanners(c *gin.Context) {
	ctx := c.Request.Context()

	banners, err := h.upstreamClient.GetBanners(ctx)
	if err != nil {
		h.log.WithFields(
			logger.String("request_id", getRequestID(c)),
			logger.String("error", err.Error()),
		).Error("Failed to get banners")

		InternalError(c, "Failed to get banners")
		return
	}

	Success(c, banners)
}

// GetDailyRecommendPlaylists 获取每日推荐歌单（需要Cookie）
// GET /api/home/daily-recommend
func (h *Handler) GetDailyRecommendPlaylists(c *gin.Context) {
	ctx := c.Request.Context()

	playlists, err := h.upstreamClient.GetDailyRecommendPlaylists(ctx)
	if err != nil {
		h.log.WithFields(
			logger.String("request_id", getRequestID(c)),
			logger.String("error", err.Error()),
		).Error("Failed to get daily recommend playlists")

		InternalError(c, "Failed to get daily recommend playlists")
		return
	}

	Success(c, playlists)
}

// GetRecommendPlaylists 获取推荐歌单
// GET /api/home/recommend-playlists
func (h *Handler) GetRecommendPlaylists(c *gin.Context) {
	ctx := c.Request.Context()

	playlists, err := h.upstreamClient.GetRecommendPlaylists(ctx)
	if err != nil {
		h.log.WithFields(
			logger.String("request_id", getRequestID(c)),
			logger.String("error", err.Error()),
		).Error("Failed to get recommend playlists")

		InternalError(c, "Failed to get recommend playlists")
		return
	}

	Success(c, playlists)
}

// GetNewSongs 获取新歌榜
// GET /api/home/new-songs?type=1
// type: 1-内地, 2-港台, 3-欧美, 4-韩国, 5-日本
func (h *Handler) GetNewSongs(c *gin.Context) {
	ctx := c.Request.Context()
	typ := getIntParam(c, "type", 1)

	songs, err := h.upstreamClient.GetNewSongs(ctx, typ)
	if err != nil {
		h.log.WithFields(
			logger.String("request_id", getRequestID(c)),
			logger.String("error", err.Error()),
			logger.Int("type", typ),
		).Error("Failed to get new songs")

		InternalError(c, "Failed to get new songs")
		return
	}

	Success(c, songs)
}

// GetNewAlbums 获取新碟榜
// GET /api/home/new-albums?type=1
// type: 1-内地, 2-港台, 3-欧美, 4-韩国, 5-日本
func (h *Handler) GetNewAlbums(c *gin.Context) {
	ctx := c.Request.Context()
	typ := getIntParam(c, "type", 1)

	albums, err := h.upstreamClient.GetNewAlbums(ctx, typ)
	if err != nil {
		h.log.WithFields(
			logger.String("request_id", getRequestID(c)),
			logger.String("error", err.Error()),
			logger.Int("type", typ),
		).Error("Failed to get new albums")

		InternalError(c, "Failed to get new albums")
		return
	}

	Success(c, albums)
}
