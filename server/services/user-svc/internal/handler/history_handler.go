package handler

import (
	"net/http"
	"strconv"

	"user-svc/internal/service"

	"github.com/gin-gonic/gin"
)

// HistoryHandler 播放历史处理器
type HistoryHandler struct {
	service *service.PlayHistoryService
}

// NewHistoryHandler 创建播放历史处理器
func NewHistoryHandler(service *service.PlayHistoryService) *HistoryHandler {
	return &HistoryHandler{
		service: service,
	}
}

// NewPlayHistoryHandler 创建播放历史处理器（别名）
func NewPlayHistoryHandler(service *service.PlayHistoryService) *HistoryHandler {
	return NewHistoryHandler(service)
}

// AddHistory 添加播放历史
func (h *HistoryHandler) AddHistory(c *gin.Context) {
	userID := c.GetString("user_id")

	var req struct {
		SongID     string `json:"song_id" binding:"required"`
		SongName   string `json:"song_name" binding:"required"`
		SingerName string `json:"singer_name" binding:"required"`
		AlbumCover string `json:"album_cover"`
		Duration   int    `json:"duration" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	history, err := h.service.AddHistory(c.Request.Context(), userID, req.SongID, req.SongName, req.SingerName, req.AlbumCover, req.Duration)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, history)
}

// GetHistory 获取播放历史
func (h *HistoryHandler) GetHistory(c *gin.Context) {
	userID := c.GetString("user_id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	histories, total, err := h.service.GetHistory(c.Request.Context(), userID, page, pageSize)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  histories,
		"total": total,
		"page":  page,
	})
}

// DeleteHistory 删除播放历史
func (h *HistoryHandler) DeleteHistory(c *gin.Context) {
	userID := c.GetString("user_id")
	historyID := c.Param("id")

	if err := h.service.DeleteHistory(c.Request.Context(), historyID, userID); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "deleted successfully"})
}

// AddPlayHistory 添加播放历史（别名）
func (h *HistoryHandler) AddPlayHistory(c *gin.Context) {
	h.AddHistory(c)
}

// ListPlayHistories 获取播放历史（别名）
func (h *HistoryHandler) ListPlayHistories(c *gin.Context) {
	h.GetHistory(c)
}

// DeletePlayHistory 删除播放历史（别名）
func (h *HistoryHandler) DeletePlayHistory(c *gin.Context) {
	h.DeleteHistory(c)
}
