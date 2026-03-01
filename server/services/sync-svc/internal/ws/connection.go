package ws

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"sync-svc/internal/domain"

	"github.com/gorilla/websocket"
)

const (
	// WriteWait 写入超时时间
	WriteWait = 10 * time.Second
	// PongWait Pong响应等待时间
	PongWait = 60 * time.Second
	// PingPeriod Ping发送周期（必须小于PongWait）
	PingPeriod = 30 * time.Second
	// MaxMessageSize 最大消息大小（1MB）
	MaxMessageSize = 1024 * 1024
)

// Connection WebSocket连接
type Connection struct {
	// 连接标识
	ID     string
	UserID string

	// WebSocket连接
	conn *websocket.Conn

	// 消息发送通道
	send chan []byte

	// 心跳相关
	lastPingTime time.Time
	lastPongTime time.Time
	pingMu       sync.RWMutex

	// 连接状态
	isActive    int32 // 原子操作，1表示活跃
	createdAt   time.Time
	closedAt    time.Time
	closeReason string

	// 控制通道
	closeChan chan struct{}
	closeOnce sync.Once

	// 管理器引用
	manager *Manager
}

// NewConnection 创建新连接
func NewConnection(id, userID string, conn *websocket.Conn, manager *Manager) *Connection {
	now := time.Now()
	return &Connection{
		ID:           id,
		UserID:       userID,
		conn:         conn,
		send:         make(chan []byte, 256),
		lastPingTime: now,
		lastPongTime: now,
		isActive:     1,
		createdAt:    now,
		closeChan:    make(chan struct{}),
		manager:      manager,
	}
}

// IsActive 检查连接是否活跃
func (c *Connection) IsActive() bool {
	return atomic.LoadInt32(&c.isActive) == 1
}

// UpdatePingTime 更新Ping时间
func (c *Connection) UpdatePingTime() {
	c.pingMu.Lock()
	c.lastPingTime = time.Now()
	c.pingMu.Unlock()
}

// UpdatePongTime 更新Pong时间
func (c *Connection) UpdatePongTime() {
	c.pingMu.Lock()
	c.lastPongTime = time.Now()
	c.pingMu.Unlock()
}

// GetLastPongTime 获取最后Pong时间
func (c *Connection) GetLastPongTime() time.Time {
	c.pingMu.RLock()
	defer c.pingMu.RUnlock()
	return c.lastPongTime
}

// Close 关闭连接
func (c *Connection) Close(reason string) {
	c.closeOnce.Do(func() {
		atomic.StoreInt32(&c.isActive, 0)
		c.closedAt = time.Now()
		c.closeReason = reason
		close(c.closeChan)
		close(c.send)
		
		// 设置关闭消息
		c.conn.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, reason),
			time.Now().Add(WriteWait),
		)
		c.conn.Close()
		
		log.Printf("Connection closed: id=%s, user=%s, reason=%s, duration=%v",
			c.ID, c.UserID, reason, c.closedAt.Sub(c.createdAt))
	})
}

// ReadPump 读取消息泵
func (c *Connection) ReadPump(ctx context.Context) {
	defer func() {
		c.manager.Unregister(c)
	}()

	c.conn.SetReadLimit(MaxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(PongWait))
	
	// 设置Pong处理器
	c.conn.SetPongHandler(func(string) error {
		c.UpdatePongTime()
		c.conn.SetReadDeadline(time.Now().Add(PongWait))
		return nil
	})

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.closeChan:
			return
		default:
			_, message, err := c.conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("WebSocket error: %v", err)
				}
				return
			}

			// 处理接收到的消息
			c.handleMessage(message)
		}
	}
}

// WritePump 写入消息泵
func (c *Connection) WritePump(ctx context.Context) {
	ticker := time.NewTicker(PingPeriod)
	defer func() {
		ticker.Stop()
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.closeChan:
			return
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(WriteWait))
			if !ok {
				// 通道已关闭
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("Write error: %v", err)
				return
			}

		case <-ticker.C:
			c.sendPing()
		}
	}
}

// sendPing 发送Ping消息
func (c *Connection) sendPing() {
	c.conn.SetWriteDeadline(time.Now().Add(WriteWait))
	if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
		log.Printf("Ping error: %v", err)
		c.Close("ping failed")
		return
	}
	c.UpdatePingTime()
}

// Send 发送消息
func (c *Connection) Send(message []byte) bool {
	if !c.IsActive() {
		return false
	}

	select {
	case c.send <- message:
		return true
	default:
		// 发送通道已满，关闭连接
		log.Printf("Send buffer full for connection %s, closing", c.ID)
		c.Close("send buffer full")
		return false
	}
}

// SendJSON 发送JSON消息
func (c *Connection) SendJSON(v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	
	if !c.Send(data) {
		return websocket.ErrCloseSent
	}
	return nil
}

// handleMessage 处理接收到的消息
func (c *Connection) handleMessage(message []byte) {
	var msg domain.SyncMessage
	if err := json.Unmarshal(message, &msg); err != nil {
		log.Printf("Invalid message format from user %s: %v", c.UserID, err)
		c.sendError("invalid_message_format", "Message must be valid JSON")
		return
	}

	// 验证消息类型
	if msg.Type == "" {
		log.Printf("Missing message type from user %s", c.UserID)
		c.sendError("missing_type", "Message type is required")
		return
	}

	// 处理不同类型的消息
	switch msg.Type {
	case domain.MessageTypePong:
		// 客户端Pong响应
		c.UpdatePongTime()
		log.Printf("Received Pong from user %s", c.UserID)

	case domain.MessageTypePing:
		// 客户端主动Ping，响应Pong
		c.sendPong()
		log.Printf("Received Ping from user %s, sent Pong", c.UserID)

	default:
		// 其他消息类型：记录但不处理
		// 主要用于调试和监控
		log.Printf("Received message from user %s: type=%s, id=%s", 
			c.UserID, msg.Type, msg.ID)
		
		// 可以在这里添加自定义消息处理逻辑
		// 例如：客户端状态上报、ACK确认等
	}
}

// sendError 发送错误消息给客户端
func (c *Connection) sendError(code, message string) {
	errMsg := map[string]interface{}{
		"type":    "error",
		"code":    code,
		"message": message,
		"timestamp": time.Now(),
	}
	
	if err := c.SendJSON(errMsg); err != nil {
		log.Printf("Failed to send error to user %s: %v", c.UserID, err)
	}
}

// sendPong 发送Pong消息给客户端
func (c *Connection) sendPong() {
	pongMsg := domain.PongMessage{
		Type:      domain.MessageTypePong,
		Timestamp: time.Now(),
	}
	
	if err := c.SendJSON(pongMsg); err != nil {
		log.Printf("Failed to send Pong to user %s: %v", c.UserID, err)
	}
}
