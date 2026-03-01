package handler

import (
	"net/http"
	"strconv"

	"user-svc/internal/service"

	"github.com/gin-gonic/gin"
)

// FavoriteHandler 收藏处理器
type FavoriteHandler struct {
	service *service.FavoriteService
}

// NewFavoriteHandler 创建收藏处理器
func NewFavoriteHandler(service *service.FavoriteService) *FavoriteHandler {
	return &FavoriteHandler{
		service: service,
	}
}

// AddFavorite 添加收藏
func (h *FavoriteHandler) AddFavorite(c *gin.Context) {
	userID := c.GetString("user_id")

	var req struct {
		SongID     string `json:"song_id" binding:"required"`
		SongName   string `json:"song_name" binding:"required"`
		SingerName string `json:"singer_name" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	favorite, err := h.service.AddFavorite(c.Request.Context(), userID, req.SongID, req.SongName, req.SingerName)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, favorite)
}

// RemoveFavorite 移除收藏
func (h *FavoriteHandler) RemoveFavorite(c *gin.Context) {
	userID := c.GetString("user_id")
	favoriteID := c.Param("id")

	if err := h.service.RemoveFavorite(c.Request.Context(), userID, favoriteID); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "removed successfully"})
}

// GetFavorites 获取收藏列表
func (h *FavoriteHandler) GetFavorites(c *gin.Context) {
	userID := c.GetString("user_id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	favorites, total, err := h.service.GetFavorites(c.Request.Context(), userID, page, pageSize)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  favorites,
		"total": total,
		"page":  page,
	})
}

// IsFavorite 检查是否已收藏
func (h *FavoriteHandler) IsFavorite(c *gin.Context) {
	userID := c.GetString("user_id")
	songID := c.Query("song_id")

	if songID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "song_id is required"})
		return
	}

	exists, err := h.service.IsFavorite(c.Request.Context(), userID, songID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"is_favorite": exists})
}

// ListFavorites 获取收藏列表（别名方法）
func (h *FavoriteHandler) ListFavorites(c *gin.Context) {
	h.GetFavorites(c)
}

// CheckFavorite 检查是否已收藏（别名方法）
func (h *FavoriteHandler) CheckFavorite(c *gin.Context) {
	h.IsFavorite(c)
}
