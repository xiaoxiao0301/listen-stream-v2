package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/xiaoxiao0301/listen-stream-v2/server/shared/pkg/logger"
)

// GetSongDetail 获取歌曲详细信息
// GET /api/song/detail?song_mid=xxx
func (h *Handler) GetSongDetail(c *gin.Context) {
	ctx := c.Request.Context()
	songMid := c.Query("song_mid")

	if songMid == "" {
		BadRequest(c, "Missing song_mid parameter")
		return
	}

	detail, err := h.upstreamClient.GetSongDetail(ctx, songMid)
	if err != nil {
		h.log.WithFields(
			logger.String("request_id", getRequestID(c)),
			logger.String("error", err.Error()),
			logger.String("song_mid", songMid),
		).Error("Failed to get song detail")

		InternalError(c, "Failed to get song detail")
		return
	}

	Success(c, detail)
}

// GetSongURL 获取歌曲播放地址（带Fallback）
// GET /api/song/url?song_mid=xxx&song_name=xxx
// 智能Fallback: QQ Music → Joox → NetEase → Kugou
func (h *Handler) GetSongURL(c *gin.Context) {
	ctx := c.Request.Context()
	songMid := c.Query("song_mid")
	songName := c.Query("song_name")

	if songMid == "" {
		BadRequest(c, "Missing song_mid parameter")
		return
	}

	// 如果没有提供歌曲名，尝试从songMid获取
	if songName == "" {
		// 可以先尝试用songMid获取，如果失败再提示需要song_name
		h.log.WithFields(
			logger.String("request_id", getRequestID(c)),
			logger.String("song_mid", songMid),
		).Warn("song_name not provided, fallback may not work optimally")
	}

	// 获取播放URL（内部会自动fallback）
	songURL, err := h.upstreamClient.GetSongURL(ctx, songMid, songName)
	if err != nil {
		h.log.WithFields(
			logger.String("request_id", getRequestID(c)),
			logger.String("error", err.Error()),
			logger.String("song_mid", songMid),
			logger.String("song_name", songName),
		).Error("Failed to get song URL with fallback")

		// 根据错误类型返回不同的响应
		if err.Error() == "song not found in any source" {
			NotFound(c, "Song URL not available from any source")
		} else {
			InternalError(c, "Failed to get song URL")
		}
		return
	}

	Success(c, songURL)
}

// GetLyric 获取歌词
// GET /api/song/lyric?song_mid=xxx
func (h *Handler) GetLyric(c *gin.Context) {
	ctx := c.Request.Context()
	songMid := c.Query("song_mid")

	if songMid == "" {
		BadRequest(c, "Missing song_mid parameter")
		return
	}

	lyric, err := h.upstreamClient.GetLyric(ctx, songMid)
	if err != nil {
		h.log.WithFields(
			logger.String("request_id", getRequestID(c)),
			logger.String("error", err.Error()),
			logger.String("song_mid", songMid),
		).Error("Failed to get lyric")

		InternalError(c, "Failed to get lyric")
		return
	}

	Success(c, lyric)
}
