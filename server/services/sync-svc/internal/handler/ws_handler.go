package handler

import (
	"log"
	"net/http"
	"strconv"

	"sync-svc/internal/offline"
	"sync-svc/internal/ws"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// 生产环境应该检查Origin
		return true
	},
}

// WSHandler WebSocket处理器
type WSHandler struct {
	manager *ws.Manager
}

// NewWSHandler 创建WebSocket处理器
func NewWSHandler(manager *ws.Manager) *WSHandler {
	return &WSHandler{
		manager: manager,
	}
}

// HandleWebSocket 处理WebSocket连接
func (h *WSHandler) HandleWebSocket(c *gin.Context) {
	// 从JWT中间件获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	userIDStr, ok := userID.(string)
	if !ok || userIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	// 检查连接限制
	if err := h.manager.GetLimiter().Acquire(); err != nil {
		log.Printf("Connection limit exceeded: user=%s", userIDStr)
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "connection limit exceeded",
			"available": h.manager.GetLimiter().Available(),
		})
		return
	}

	// 升级到WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		h.manager.GetLimiter().Release()
		return
	}

	// 创建连接对象
	connID := uuid.New().String()
	wsConn := ws.NewConnection(connID, userIDStr, conn, h.manager)

	// 注册连接
	h.manager.Register(wsConn)

	// 启动读写协程
	ctx := c.Request.Context()
	go wsConn.ReadPump(ctx)
	go wsConn.WritePump(ctx)

	log.Printf("WebSocket connection established: id=%s, user=%s", connID, userIDStr)
}

// GetStats 获取统计信息
func (h *WSHandler) GetStats(c *gin.Context) {
	stats := h.manager.GetStats()
	c.JSON(http.StatusOK, stats)
}

// GetOnlineUsers 获取在线用户列表
func (h *WSHandler) GetOnlineUsers(c *gin.Context) {
	users := h.manager.GetOnlineUsers()
	c.JSON(http.StatusOK, gin.H{
		"online_users": users,
		"count":        len(users),
	})
}

// CheckUserOnline 检查用户是否在线
func (h *WSHandler) CheckUserOnline(c *gin.Context) {
	userID := c.Param("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id required"})
		return
	}

	isOnline := h.manager.IsUserOnline(userID)
	connectionCount := h.manager.GetRoom().GetUserConnectionCount(userID)

	c.JSON(http.StatusOK, gin.H{
		"user_id":          userID,
		"online":           isOnline,
		"connection_count": connectionCount,
	})
}

// GetOfflineMessages 获取离线消息
func (h *WSHandler) GetOfflineMessages(c *gin.Context) {
	// 从JWT中间件获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	userIDStr, ok := userID.(string)
	if !ok || userIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	// 获取limit参数
	limit := 50 // 默认50条
	if limitParam := c.Query("limit"); limitParam != "" {
		if parsedLimit, err := strconv.Atoi(limitParam); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	// 拉取离线消息
	messages, err := h.manager.GetOfflineService().Pull(c.Request.Context(), userIDStr, limit)
	if err != nil {
		log.Printf("Failed to get offline messages: user=%s, error=%v", userIDStr, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get offline messages"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"messages": messages,
		"count":    len(messages),
	})
}

// AckOfflineMessage 确认单条离线消息
func (h *WSHandler) AckOfflineMessage(c *gin.Context) {
	// 从JWT中间件获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	userIDStr, ok := userID.(string)
	if !ok || userIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	// 解析请求体
	var req struct {
		MessageID string `json:"message_id" binding:"required"`
		AckToken  string `json:"ack_token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 确认消息
	if err := h.manager.AckOfflineMessage(c.Request.Context(), userIDStr, req.MessageID, req.AckToken); err != nil {
		log.Printf("Failed to ack offline message: user=%s, id=%s, error=%v", userIDStr, req.MessageID, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "acknowledged"})
}

// BatchAckOfflineMessages 批量确认离线消息
func (h *WSHandler) BatchAckOfflineMessages(c *gin.Context) {
	// 从JWT中间件获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	userIDStr, ok := userID.(string)
	if !ok || userIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	// 解析请求体
	var req struct {
		Acks []struct {
			MessageID string `json:"message_id" binding:"required"`
			AckToken  string `json:"ack_token" binding:"required"`
		} `json:"acks" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 转换为offline.AckRequest
	acks := make([]offline.AckRequest, len(req.Acks))
	for i, ack := range req.Acks {
		acks[i] = offline.AckRequest{
			MessageID: ack.MessageID,
			AckToken:  ack.AckToken,
		}
	}

	// 批量确认
	if err := h.manager.BatchAckOfflineMessages(c.Request.Context(), userIDStr, acks); err != nil {
		log.Printf("Failed to batch ack offline messages: user=%s, count=%d, error=%v", userIDStr, len(acks), err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "acknowledged",
		"count":   len(acks),
	})
}

// GetOfflineMessageCount 获取离线消息数量
func (h *WSHandler) GetOfflineMessageCount(c *gin.Context) {
	// 从JWT中间件获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	userIDStr, ok := userID.(string)
	if !ok || userIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	// 获取数量
	count, err := h.manager.GetOfflineMessageCount(c.Request.Context(), userIDStr)
	if err != nil {
		log.Printf("Failed to get offline message count: user=%s, error=%v", userIDStr, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get count"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user_id": userIDStr,
		"count":   count,
	})
}

// GetOfflineStats 获取离线消息统计
func (h *WSHandler) GetOfflineStats(c *gin.Context) {
	stats, err := h.manager.GetOfflineService().GetStats(c.Request.Context())
	if err != nil {
		log.Printf("Failed to get offline stats: error=%v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get stats"})
		return
	}

	c.JSON(http.StatusOK, stats)
}

