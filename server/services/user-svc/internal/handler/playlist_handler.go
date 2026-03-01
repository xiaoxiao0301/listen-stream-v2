package handler

import (
	"net/http"
	"strconv"

	"user-svc/internal/service"

	"github.com/gin-gonic/gin"
)

// PlaylistHandler 歌单处理器
type PlaylistHandler struct {
	service *service.PlaylistService
}

// NewPlaylistHandler 创建歌单处理器
func NewPlaylistHandler(service *service.PlaylistService) *PlaylistHandler {
	return &PlaylistHandler{
		service: service,
	}
}

// CreatePlaylist 创建歌单
func (h *PlaylistHandler) CreatePlaylist(c *gin.Context) {
	userID := c.GetString("user_id")

	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
		CoverURL    string `json:"cover_url"`
		IsPublic    bool   `json:"is_public"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	playlist, err := h.service.CreatePlaylist(c.Request.Context(), userID, req.Name, req.Description, req.CoverURL, req.IsPublic)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, playlist)
}

// GetPlaylist 获取歌单详情
func (h *PlaylistHandler) GetPlaylist(c *gin.Context) {
	playlistID := c.Param("id")

	playlist, err := h.service.GetPlaylist(c.Request.Context(), playlistID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, playlist)
}

// GetUserPlaylists 获取用户的歌单列表
func (h *PlaylistHandler) GetUserPlaylists(c *gin.Context) {
	userID := c.GetString("user_id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	playlists, total, err := h.service.GetUserPlaylists(c.Request.Context(), userID, page, pageSize)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  playlists,
		"total": total,
		"page":  page,
	})
}

// UpdatePlaylist 更新歌单
func (h *PlaylistHandler) UpdatePlaylist(c *gin.Context) {
	userID := c.GetString("user_id")
	playlistID := c.Param("id")

	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		CoverURL    string `json:"cover_url"`
		IsPublic    bool   `json:"is_public"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	playlist, err := h.service.UpdatePlaylist(c.Request.Context(), playlistID, userID, req.Name, req.Description, req.CoverURL, req.IsPublic)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, playlist)
}

// DeletePlaylist 删除歌单
func (h *PlaylistHandler) DeletePlaylist(c *gin.Context) {
	userID := c.GetString("user_id")
	playlistID := c.Param("id")

	if err := h.service.DeletePlaylist(c.Request.Context(), playlistID, userID); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "deleted successfully"})
}

// AddSongToPlaylist 添加歌曲到歌单
func (h *PlaylistHandler) AddSongToPlaylist(c *gin.Context) {
	userID := c.GetString("user_id")
	playlistID := c.Param("id")

	var req struct {
		SongID     string `json:"song_id" binding:"required"`
		SongName   string `json:"song_name" binding:"required"`
		SingerName string `json:"singer_name" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.AddSongToPlaylist(c.Request.Context(), playlistID, userID, req.SongID, req.SongName, req.SingerName); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "added successfully"})
}

// RemoveSongFromPlaylist 从歌单移除歌曲
func (h *PlaylistHandler) RemoveSongFromPlaylist(c *gin.Context) {
	userID := c.GetString("user_id")
	playlistID := c.Param("playlist_id")
	songID := c.Param("song_id")

	if err := h.service.RemoveSongFromPlaylist(c.Request.Context(), playlistID, userID, songID); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "removed successfully"})
}

// GetPlaylistSongs 获取歌单的歌曲列表
func (h *PlaylistHandler) GetPlaylistSongs(c *gin.Context) {
	playlistID := c.Param("id")

	songs, err := h.service.GetPlaylistSongs(c.Request.Context(), playlistID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": songs,
	})
}

// ListUserPlaylists 获取用户的歌单列表（别名）
func (h *PlaylistHandler) ListUserPlaylists(c *gin.Context) {
	h.GetUserPlaylists(c)
}

// ListPlaylistSongs 获取歌单的歌曲列表（别名）
func (h *PlaylistHandler) ListPlaylistSongs(c *gin.Context) {
	h.GetPlaylistSongs(c)
}
