package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/xiaoxiao0301/listen-stream-v2/server/services/proxy-svc/internal/client"
	"github.com/xiaoxiao0301/listen-stream-v2/server/shared/pkg/logger"
	userv1 "github.com/xiaoxiao0301/listen-stream-v2/server/shared/proto/user/v1"
)

// UserHandler 用户相关接口处理器
type UserHandler struct {
	userClient *client.UserClient
	log        logger.Logger
}

// NewUserHandler 创建用户处理器
func NewUserHandler(userClient *client.UserClient, log logger.Logger) *UserHandler {
	return &UserHandler{
		userClient: userClient,
		log:        log,
	}
}

// ===== 收藏管理 =====

// AddFavorite 添加收藏
// POST /api/user/favorites
// Body: {"song_id": "xxx", "song_name": "xxx", "artist_name": "xxx"}
func (h *UserHandler) AddFavorite(c *gin.Context) {
	ctx := c.Request.Context()
	userID := getUserID(c)

	var req struct {
		SongID     string `json:"song_id" binding:"required"`
		SongName   string `json:"song_name" binding:"required"`
		ArtistName string `json:"artist_name"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	metadata := &userv1.FavoriteMetadata{
		Name:   req.SongName,
		Artist: req.ArtistName,
	}

	_, err := h.userClient.AddFavorite(ctx, userID, req.SongID, userv1.FavoriteType_FAVORITE_TYPE_SONG, metadata)
	if err != nil {
		h.log.WithFields(
			logger.String("request_id", getRequestID(c)),
			logger.String("user_id", userID),
			logger.String("error", err.Error()),
		).Error("Failed to add favorite")

		InternalError(c, "Failed to add favorite")
		return
	}

	Success(c, gin.H{"message": "Favorite added successfully"})
}

// RemoveFavorite 取消收藏
// DELETE /api/user/favorites/:song_id
func (h *UserHandler) RemoveFavorite(c *gin.Context) {
	ctx := c.Request.Context()
	userID := getUserID(c)
	songID := c.Param("song_id")

	if songID == "" {
		BadRequest(c, "Missing song_id parameter")
		return
	}

	if err := h.userClient.RemoveFavorite(ctx, userID, songID); err != nil {
		h.log.WithFields(
			logger.String("request_id", getRequestID(c)),
			logger.String("user_id", userID),
			logger.String("error", err.Error()),
		).Error("Failed to remove favorite")

		InternalError(c, "Failed to remove favorite")
		return
	}

	Success(c, gin.H{"message": "Favorite removed successfully"})
}

// ListFavorites 获取收藏列表
// GET /api/user/favorites?page=1&page_size=20
func (h *UserHandler) ListFavorites(c *gin.Context) {
	ctx := c.Request.Context()
	userID := getUserID(c)

	page := getIntParam(c, "page", 1)
	pageSize := getIntParam(c, "page_size", 20)

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	resp, err := h.userClient.ListFavorites(ctx, userID, userv1.FavoriteType_FAVORITE_TYPE_SONG, int32(page), int32(pageSize))
	if err != nil {
		h.log.WithFields(
			logger.String("request_id", getRequestID(c)),
			logger.String("user_id", userID),
			logger.String("error", err.Error()),
		).Error("Failed to list favorites")

		InternalError(c, "Failed to list favorites")
		return
	}

	Success(c, gin.H{
		"items":     resp.Favorites,
		"total":     resp.Total,
		"page":      page,
		"page_size": pageSize,
	})
}

// ===== 播放历史 =====

// AddPlayHistory 添加播放历史
// POST /api/user/history
// Body: {"song_id": "xxx", "song_name": "xxx", "artist_name": "xxx", "album_name": "xxx", "duration": 240}
func (h *UserHandler) AddPlayHistory(c *gin.Context) {
	ctx := c.Request.Context()
	userID := getUserID(c)

	var req struct {
		SongID     string `json:"song_id" binding:"required"`
		SongName   string `json:"song_name" binding:"required"`
		ArtistName string `json:"artist_name"`
		AlbumName  string `json:"album_name"`
		Duration   int32  `json:"duration"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	if err := h.userClient.AddPlayHistory(ctx, userID, req.SongID, req.SongName, req.ArtistName, req.AlbumName, req.Duration); err != nil {
		h.log.WithFields(
			logger.String("request_id", getRequestID(c)),
			logger.String("user_id", userID),
			logger.String("error", err.Error()),
		).Error("Failed to add play history")

		InternalError(c, "Failed to add play history")
		return
	}

	Success(c, gin.H{"message": "Play history added successfully"})
}

// ListPlayHistory 获取播放历史
// GET /api/user/history?page=1&page_size=20
func (h *UserHandler) ListPlayHistory(c *gin.Context) {
	ctx := c.Request.Context()
	userID := getUserID(c)

	page := getIntParam(c, "page", 1)
	pageSize := getIntParam(c, "page_size", 20)

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	resp, err := h.userClient.ListPlayHistory(ctx, userID, int32(page), int32(pageSize))
	if err != nil {
		h.log.WithFields(
			logger.String("request_id", getRequestID(c)),
			logger.String("user_id", userID),
			logger.String("error", err.Error()),
		).Error("Failed to list play history")

		InternalError(c, "Failed to list play history")
		return
	}

	Success(c, gin.H{
		"items":     resp.History,
		"total":     resp.Total,
		"page":      page,
		"page_size": pageSize,
	})
}

// ===== 歌单管理 =====

// CreatePlaylist 创建歌单
// POST /api/user/playlists
// Body: {"name": "xxx", "cover_url": "xxx", "is_public": true}
func (h *UserHandler) CreatePlaylist(c *gin.Context) {
	ctx := c.Request.Context()
	userID := getUserID(c)

	var req struct {
		Name     string `json:"name" binding:"required"`
		CoverURL string `json:"cover_url"`
		IsPublic bool   `json:"is_public"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	resp, err := h.userClient.CreatePlaylist(ctx, userID, req.Name, req.CoverURL, req.IsPublic)
	if err != nil {
		h.log.WithFields(
			logger.String("request_id", getRequestID(c)),
			logger.String("user_id", userID),
			logger.String("error", err.Error()),
		).Error("Failed to create playlist")

		InternalError(c, "Failed to create playlist")
		return
	}

	Success(c, gin.H{
		"playlist_id": resp.Playlist.Id,
		"message":     "Playlist created successfully",
	})
}

// UpdatePlaylist 更新歌单
// PUT /api/user/playlists/:playlist_id
// Body: {"name": "xxx", "cover_url": "xxx", "is_public": true}
func (h *UserHandler) UpdatePlaylist(c *gin.Context) {
	ctx := c.Request.Context()
	userID := getUserID(c)
	playlistID := c.Param("playlist_id")

	if playlistID == "" {
		BadRequest(c, "Missing playlist_id parameter")
		return
	}

	var req struct {
		Name     string `json:"name"`
		CoverURL string `json:"cover_url"`
		IsPublic *bool  `json:"is_public"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	if err := h.userClient.UpdatePlaylist(ctx, userID, playlistID, req.Name, req.CoverURL, req.IsPublic); err != nil {
		h.log.WithFields(
			logger.String("request_id", getRequestID(c)),
			logger.String("user_id", userID),
			logger.String("error", err.Error()),
		).Error("Failed to update playlist")

		InternalError(c, "Failed to update playlist")
		return
	}

	Success(c, gin.H{"message": "Playlist updated successfully"})
}

// DeletePlaylist 删除歌单
// DELETE /api/user/playlists/:playlist_id
func (h *UserHandler) DeletePlaylist(c *gin.Context) {
	ctx := c.Request.Context()
	userID := getUserID(c)
	playlistID := c.Param("playlist_id")

	if playlistID == "" {
		BadRequest(c, "Missing playlist_id parameter")
		return
	}

	if err := h.userClient.DeletePlaylist(ctx, userID, playlistID); err != nil {
		h.log.WithFields(
			logger.String("request_id", getRequestID(c)),
			logger.String("user_id", userID),
			logger.String("error", err.Error()),
		).Error("Failed to delete playlist")

		InternalError(c, "Failed to delete playlist")
		return
	}

	Success(c, gin.H{"message": "Playlist deleted successfully"})
}

// ListPlaylists 获取用户歌单列表
// GET /api/user/playlists
func (h *UserHandler) ListPlaylists(c *gin.Context) {
	ctx := c.Request.Context()
	userID := getUserID(c)

	resp, err := h.userClient.ListPlaylists(ctx, userID)
	if err != nil {
		h.log.WithFields(
			logger.String("request_id", getRequestID(c)),
			logger.String("user_id", userID),
			logger.String("error", err.Error()),
		).Error("Failed to list playlists")

		InternalError(c, "Failed to list playlists")
		return
	}

	Success(c, gin.H{"items": resp.Playlists})
}

// AddSongToPlaylist 添加歌曲到歌单
// POST /api/user/playlists/:playlist_id/songs
// Body: {"song_id": "xxx", "position": 0}
func (h *UserHandler) AddSongToPlaylist(c *gin.Context) {
	ctx := c.Request.Context()
	userID := getUserID(c)
	playlistID := c.Param("playlist_id")

	if playlistID == "" {
		BadRequest(c, "Missing playlist_id parameter")
		return
	}

	var req struct {
		SongID   string `json:"song_id" binding:"required"`
		Position int32  `json:"position"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	if err := h.userClient.AddSongToPlaylist(ctx, userID, playlistID, req.SongID, req.Position); err != nil {
		h.log.WithFields(
			logger.String("request_id", getRequestID(c)),
			logger.String("user_id", userID),
			logger.String("error", err.Error()),
		).Error("Failed to add song to playlist")

		InternalError(c, "Failed to add song to playlist")
		return
	}

	Success(c, gin.H{"message": "Song added to playlist successfully"})
}

// RemoveSongFromPlaylist 从歌单移除歌曲
// DELETE /api/user/playlists/:playlist_id/songs/:song_id
func (h *UserHandler) RemoveSongFromPlaylist(c *gin.Context) {
	ctx := c.Request.Context()
	userID := getUserID(c)
	playlistID := c.Param("playlist_id")
	songID := c.Param("song_id")

	if playlistID == "" || songID == "" {
		BadRequest(c, "Missing playlist_id or song_id parameter")
		return
	}

	if err := h.userClient.RemoveSongFromPlaylist(ctx, userID, playlistID, songID); err != nil {
		h.log.WithFields(
			logger.String("request_id", getRequestID(c)),
			logger.String("user_id", userID),
			logger.String("error", err.Error()),
		).Error("Failed to remove song from playlist")

		InternalError(c, "Failed to remove song from playlist")
		return
	}

	Success(c, gin.H{"message": "Song removed from playlist successfully"})
}

// GetPlaylistSongs 获取歌单歌曲列表
// GET /api/user/playlists/:playlist_id/songs
func (h *UserHandler) GetPlaylistSongs(c *gin.Context) {
	ctx := c.Request.Context()
	playlistID := c.Param("playlist_id")

	if playlistID == "" {
		BadRequest(c, "Missing playlist_id parameter")
		return
	}

	resp, err := h.userClient.GetPlaylistSongs(ctx, playlistID)
	if err != nil {
		h.log.WithFields(
			logger.String("request_id", getRequestID(c)),
			logger.String("error", err.Error()),
		).Error("Failed to get playlist songs")

		InternalError(c, "Failed to get playlist songs")
		return
	}

	Success(c, gin.H{"items": resp.Songs})
}

// getUserID 从JWT中获取用户ID
func getUserID(c *gin.Context) string {
	if userID, exists := c.Get("user_id"); exists {
		return userID.(string)
	}
	return ""
}
