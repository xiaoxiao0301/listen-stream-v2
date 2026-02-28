package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/xiaoxiao0301/listen-stream-v2/server/shared/pkg/logger"
)

// GetAlbumDetail 获取专辑详细信息
// GET /api/album/detail?album_mid=xxx
func (h *Handler) GetAlbumDetail(c *gin.Context) {
	ctx := c.Request.Context()
	albumMid := c.Query("album_mid")

	if albumMid == "" {
		BadRequest(c, "Missing album_mid parameter")
		return
	}

	detail, err := h.upstreamClient.GetAlbumDetail(ctx, albumMid)
	if err != nil {
		h.log.WithFields(
			logger.String("request_id", getRequestID(c)),
			logger.String("error", err.Error()),
			logger.String("album_mid", albumMid),
		).Error("Failed to get album detail")

		InternalError(c, "Failed to get album detail")
		return
	}

	Success(c, detail)
}

// GetAlbumSongs 获取专辑歌曲列表
// GET /api/album/songs?album_mid=xxx
func (h *Handler) GetAlbumSongs(c *gin.Context) {
	ctx := c.Request.Context()
	albumMid := c.Query("album_mid")

	if albumMid == "" {
		BadRequest(c, "Missing album_mid parameter")
		return
	}

	songs, err := h.upstreamClient.GetAlbumSongs(ctx, albumMid)
	if err != nil {
		h.log.WithFields(
			logger.String("request_id", getRequestID(c)),
			logger.String("error", err.Error()),
			logger.String("album_mid", albumMid),
		).Error("Failed to get album songs")

		InternalError(c, "Failed to get album songs")
		return
	}

	Success(c, songs)
}
