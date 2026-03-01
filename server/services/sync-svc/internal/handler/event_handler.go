package handler

import (
	"net/http"
	"time"

	"sync-svc/internal/domain"
	"sync-svc/internal/ws"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ErrorResponse 错误响应
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// EventHandler 事件处理器
type EventHandler struct {
	manager *ws.Manager
}

// NewEventHandler 创建事件处理器
func NewEventHandler(manager *ws.Manager) *EventHandler {
	return &EventHandler{
		manager: manager,
	}
}

// PublishEventRequest 发布事件请求
type PublishEventRequest struct {
	UserID string                 `json:"user_id" binding:"required"` // 目标用户ID
	Type   string                 `json:"type" binding:"required"`    // 消息类型
	Data   map[string]interface{} `json:"data"`                       // 消息数据
}

// PublishEvent 发布同步事件
// @Summary 发布同步事件
// @Description 接收来自其他服务的同步事件，广播给目标用户
// @Tags events
// @Accept json
// @Produce json
// @Param request body PublishEventRequest true "事件请求"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/events [post]
func (h *EventHandler) PublishEvent(c *gin.Context) {
	var req PublishEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	// 验证消息类型
	msgType := domain.MessageType(req.Type)
	if !isValidMessageType(msgType) {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_message_type",
			Message: "unsupported message type",
		})
		return
	}

	// 创建同步消息
	syncMsg := &domain.SyncMessage{
		ID:        uuid.New().String(),
		Type:      msgType,
		UserID:    req.UserID,
		Data:      req.Data,
		Timestamp: time.Now(),
	}

	// 广播消息
	h.manager.Broadcast(syncMsg)

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"message_id": syncMsg.ID,
		"user_id":    syncMsg.UserID,
		"type":       syncMsg.Type,
		"timestamp":  syncMsg.Timestamp,
	})
}

// BatchPublishEventRequest 批量发布事件请求
type BatchPublishEventRequest struct {
	UserIDs []string               `json:"user_ids" binding:"required,min=1"` // 目标用户ID列表
	Type    string                 `json:"type" binding:"required"`           // 消息类型
	Data    map[string]interface{} `json:"data"`                              // 消息数据
}

// BatchPublishEvent 批量发布同步事件
// @Summary 批量发布同步事件
// @Description 向多个用户发送相同的同步事件
// @Tags events
// @Accept json
// @Produce json
// @Param request body BatchPublishEventRequest true "批量事件请求"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/events/batch [post]
func (h *EventHandler) BatchPublishEvent(c *gin.Context) {
	var req BatchPublishEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	// 验证消息类型
	msgType := domain.MessageType(req.Type)
	if !isValidMessageType(msgType) {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_message_type",
			Message: "unsupported message type",
		})
		return
	}

	messageID := uuid.New().String()
	timestamp := time.Now()

	// 批量广播给每个用户
	for _, userID := range req.UserIDs {
		syncMsg := &domain.SyncMessage{
			ID:        messageID,
			Type:      msgType,
			UserID:    userID,
			Data:      req.Data,
			Timestamp: timestamp,
		}

		h.manager.Broadcast(syncMsg)
	}

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"message_id": messageID,
		"user_count": len(req.UserIDs),
		"type":       msgType,
		"timestamp":  timestamp,
	})
}

// BroadcastEventRequest 全局广播事件请求
type BroadcastEventRequest struct {
	Type string                 `json:"type" binding:"required"` // 消息类型
	Data map[string]interface{} `json:"data"`                    // 消息数据
}

// BroadcastEvent 全局广播事件
// @Summary 全局广播事件
// @Description 向所有在线用户广播事件
// @Tags events
// @Accept json
// @Produce json
// @Param request body BroadcastEventRequest true "广播事件请求"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/events/broadcast [post]
func (h *EventHandler) BroadcastEvent(c *gin.Context) {
	var req BroadcastEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	// 验证消息类型
	msgType := domain.MessageType(req.Type)
	if !isValidMessageType(msgType) {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_message_type",
			Message: "unsupported message type",
		})
		return
	}

	// 创建广播消息
	syncMsg := &domain.SyncMessage{
		ID:        uuid.New().String(),
		Type:      msgType,
		Data:      req.Data,
		Timestamp: time.Now(),
	}

	// 全局广播
	h.manager.BroadcastToAll(syncMsg)

	// 获取在线用户数
	onlineUsers := h.manager.GetOnlineUsers()

	c.JSON(http.StatusOK, gin.H{
		"success":         true,
		"message_id":      syncMsg.ID,
		"type":            syncMsg.Type,
		"timestamp":       syncMsg.Timestamp,
		"online_users":    len(onlineUsers),
	})
}

// GetPubSubStats 获取Pub/Sub统计
// @Summary 获取Pub/Sub统计
// @Description 获取发布/订阅统计信息
// @Tags stats
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/stats/pubsub [get]
func (h *EventHandler) GetPubSubStats(c *gin.Context) {
	pubStats := h.manager.GetPublisher().GetStats()
	subStats := h.manager.GetSubscriber().GetStats()

	c.JSON(http.StatusOK, gin.H{
		"instance_id": h.manager.GetInstanceID(),
		"publisher": gin.H{
			"total_published":     pubStats.TotalPublished,
			"user_published":      pubStats.UserPublished,
			"broadcast_published": pubStats.BroadcastPublished,
			"failed_published":    pubStats.FailedPublished,
		},
		"subscriber": gin.H{
			"total_received":     subStats.TotalReceived,
			"user_received":      subStats.UserReceived,
			"broadcast_received": subStats.BroadcastReceived,
			"processed_messages": subStats.ProcessedMessages,
			"failed_messages":    subStats.FailedMessages,
			"dropped_messages":   subStats.DroppedMessages,
			"reconnect_count":    subStats.ReconnectCount,
		},
	})
}

// isValidMessageType 验证消息类型
func isValidMessageType(msgType domain.MessageType) bool {
	validTypes := []domain.MessageType{
		domain.MessageTypeFavoriteAdded,
		domain.MessageTypeFavoriteRemoved,
		domain.MessageTypePlaylistCreated,
		domain.MessageTypePlaylistUpdated,
		domain.MessageTypePlaylistDeleted,
		domain.MessageTypePlaylistSongAdded,
		domain.MessageTypePlaylistSongRemoved,
		domain.MessageTypeHistoryAdded,
	}

	for _, valid := range validTypes {
		if msgType == valid {
			return true
		}
	}

	return false
}
