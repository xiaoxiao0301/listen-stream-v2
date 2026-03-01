package ws

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"sync/atomic"

	"sync-svc/internal/domain"
	"sync-svc/internal/offline"
	"sync-svc/internal/pubsub"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// Manager WebSocket连接管理器
type Manager struct {
	// 连接池
	connections map[string]*Connection
	mu          sync.RWMutex

	// 房间管理
	room *Room

	// 连接限制器
	limiter *ConnectionLimiter

	// 心跳检查器
	heartbeat *HeartbeatChecker

	// 离线消息服务
	offline *offline.Service

	// Redis Pub/Sub
	publisher  *pubsub.Publisher
	subscriber *pubsub.Subscriber
	instanceID string

	// 注册/注销通道
	register   chan *Connection
	unregister chan *Connection

	// 广播通道
	broadcast chan *domain.SyncMessage

	// 统计
	stats ManagerStats
}

// ManagerStats 管理器统计信息
type ManagerStats struct {
	TotalRegistered   int64
	TotalUnregistered int64
	CurrentConnections int64
}

// NewManager 创建连接管理器
func NewManager(maxConnections int, redisClient *redis.Client, instanceID string) *Manager {
	// 创建离线消息服务
	offlineQueue := offline.NewQueue(redisClient, nil)
	offlineService := offline.NewService(offlineQueue)

	// 创建Publisher和Subscriber
	publisher := pubsub.NewPublisher(redisClient, instanceID)
	subscriberConfig := pubsub.DefaultSubscriberConfig(instanceID)
	subscriber := pubsub.NewSubscriber(redisClient, subscriberConfig)

	m := &Manager{
		connections: make(map[string]*Connection),
		room:        NewRoom(),
		limiter:     NewConnectionLimiter(maxConnections),
		offline:     offlineService,
		publisher:   publisher,
		subscriber:  subscriber,
		instanceID:  instanceID,
		register:    make(chan *Connection, 256),
		unregister:  make(chan *Connection, 256),
		broadcast:   make(chan *domain.SyncMessage, 1024),
	}

	m.heartbeat = NewHeartbeatChecker(m)

	// 订阅用户消息频道（pattern: sync:user:*）
	subscriber.Subscribe("sync:user:*", m.handlePubSubMessage)
	
	// 订阅广播频道
	subscriber.Subscribe("sync:broadcast", m.handlePubSubBroadcast)

	return m
}

// Start 启动管理器
func (m *Manager) Start(ctx context.Context) {
	log.Println("WebSocket manager started")

	// 启动心跳检查器
	go m.heartbeat.Start(ctx)

	// 启动Subscriber
	if err := m.subscriber.Start(ctx); err != nil {
		log.Printf("Failed to start subscriber: %v", err)
	} else {
		log.Printf("Subscriber started for instance: %s", m.instanceID)
	}

	for {
		select {
		case <-ctx.Done():
			m.shutdown()
			return

		case conn := <-m.register:
			m.handleRegister(conn)

		case conn := <-m.unregister:
			m.handleUnregister(conn)

		case message := <-m.broadcast:
			m.handleBroadcast(message)
		}
	}
}

// Register 注册新连接
func (m *Manager) Register(conn *Connection) {
	m.register <- conn
}

// Unregister 注销连接
func (m *Manager) Unregister(conn *Connection) {
	m.unregister <- conn
}

// Broadcast 广播消息到指定用户
func (m *Manager) Broadcast(message *domain.SyncMessage) {
	select {
	case m.broadcast <- message:
	default:
		log.Printf("Broadcast channel full, message dropped: type=%s", message.Type)
	}
}

// handleRegister 处理连接注册
func (m *Manager) handleRegister(conn *Connection) {
	m.mu.Lock()
	m.connections[conn.ID] = conn
	m.mu.Unlock()

	m.room.Join(conn.UserID, conn)

	atomic.AddInt64(&m.stats.TotalRegistered, 1)
	atomic.AddInt64(&m.stats.CurrentConnections, 1)

	log.Printf("Connection registered: id=%s, user=%s, total=%d",
		conn.ID, conn.UserID, atomic.LoadInt64(&m.stats.CurrentConnections))

	// 用户上线：拉取并推送离线消息
	go m.pushOfflineMessages(conn)
}

// pushOfflineMessages 推送离线消息给上线用户
func (m *Manager) pushOfflineMessages(conn *Connection) {
	ctx := context.Background()

	// 拉取离线消息
	messages, err := m.offline.Pull(ctx, conn.UserID, 50) // 最多50条
	if err != nil {
		log.Printf("Failed to pull offline messages: user=%s, error=%v", conn.UserID, err)
		return
	}

	if len(messages) == 0 {
		log.Printf("No offline messages for user: %s", conn.UserID)
		return
	}

	log.Printf("Pushing %d offline messages to user: %s", len(messages), conn.UserID)

	// 推送每条离线消息
	for _, msg := range messages {
		syncMsg := &domain.SyncMessage{
			ID:        msg.ID,
			Type:      domain.MessageType(msg.Type),
			UserID:    msg.UserID,
			Data:      msg.Data,
			Timestamp: msg.CreatedAt,
			AckToken:  msg.AckToken,
		}

		// 序列化消息
		msgBytes, err := json.Marshal(syncMsg)
		if err != nil {
			log.Printf("Failed to marshal offline message: user=%s, id=%s, error=%v", conn.UserID, msg.ID, err)
			continue
		}

		// 发送到连接
		select {
		case conn.send <- msgBytes:
			log.Printf("Sent offline message: user=%s, id=%s, type=%s", conn.UserID, msg.ID, msg.Type)
		case <-conn.closeChan:
			log.Printf("Connection closed while pushing offline messages: user=%s", conn.UserID)
			return
		default:
			log.Printf("Send channel full, skipping offline message: user=%s, id=%s", conn.UserID, msg.ID)
		}
	}
}

// handleUnregister 处理连接注销
func (m *Manager) handleUnregister(conn *Connection) {
	m.mu.Lock()
	if _, exists := m.connections[conn.ID]; exists {
		delete(m.connections, conn.ID)
	}
	m.mu.Unlock()

	m.room.Leave(conn.UserID, conn.ID)
	m.limiter.Release()

	if conn.IsActive() {
		conn.Close("unregistered")
	}

	atomic.AddInt64(&m.stats.TotalUnregistered, 1)
	atomic.AddInt64(&m.stats.CurrentConnections, -1)

	log.Printf("Connection unregistered: id=%s, user=%s, total=%d",
		conn.ID, conn.UserID, atomic.LoadInt64(&m.stats.CurrentConnections))
}

// handleBroadcast 处理广播消息
func (m *Manager) handleBroadcast(message *domain.SyncMessage) {
	if message.UserID == "" {
		log.Printf("Broadcast message missing user_id: type=%s", message.Type)
		return
	}

	// 确保消息有ID和时间戳
	if message.ID == "" {
		message.ID = uuid.New().String()
	}

	// 检查用户是否在线
	if m.room.IsUserOnline(message.UserID) {
		// 用户在线：直接广播到本地连接
		m.room.BroadcastJSON(message.UserID, message)
		log.Printf("Broadcasted message to online user (local): user=%s, type=%s, id=%s", 
			message.UserID, message.Type, message.ID)
	} else {
		// 用户离线：保存到离线队列
		ctx := context.Background()
		offlineMsg, err := m.offline.Push(ctx, message.UserID, string(message.Type), message.Data)
		if err != nil {
			log.Printf("Failed to save offline message: user=%s, type=%s, error=%v", 
				message.UserID, message.Type, err)
		} else {
			log.Printf("Saved offline message: user=%s, type=%s, id=%s", 
				message.UserID, message.Type, offlineMsg.ID)
		}
	}

	// 发布到Redis Pub/Sub（跨实例广播）
	ctx := context.Background()
	if err := m.publisher.PublishToUser(ctx, message.UserID, message); err != nil {
		log.Printf("Failed to publish to Redis: user=%s, type=%s, error=%v", 
			message.UserID, message.Type, err)
	} else {
		log.Printf("Published to Redis Pub/Sub: user=%s, type=%s, id=%s", 
			message.UserID, message.Type, message.ID)
	}
}

// GetConnection 获取指定连接
func (m *Manager) GetConnection(connID string) (*Connection, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	conn, exists := m.connections[connID]
	return conn, exists
}

// GetUserConnections 获取用户的所有连接
func (m *Manager) GetUserConnections(userID string) []*Connection {
	return m.room.GetUserConnections(userID)
}

// IsUserOnline 检查用户是否在线
func (m *Manager) IsUserOnline(userID string) bool {
	return m.room.IsUserOnline(userID)
}

// GetOnlineUsers 获取所有在线用户
func (m *Manager) GetOnlineUsers() []string {
	return m.room.GetOnlineUsers()
}

// GetStats 获取统计信息
func (m *Manager) GetStats() map[string]interface{} {
	roomStats := m.room.GetStats()
	heartbeatStats := m.heartbeat.GetStats()
	pubStats := m.publisher.GetStats()
	subStats := m.subscriber.GetStats()

	return map[string]interface{}{
		"total_registered":   atomic.LoadInt64(&m.stats.TotalRegistered),
		"total_unregistered": atomic.LoadInt64(&m.stats.TotalUnregistered),
		"current_connections": atomic.LoadInt64(&m.stats.CurrentConnections),
		"max_connections":    m.limiter.MaxConnections(),
		"available_connections": m.limiter.Available(),
		"instance_id": m.instanceID,
		"rooms": map[string]interface{}{
			"total":                   roomStats.TotalRooms,
			"total_connections":       roomStats.TotalConnections,
			"max_connections_per_user": roomStats.MaxConnectionsPerUser,
			"avg_connections_per_user": roomStats.AvgConnectionsPerUser,
		},
		"heartbeat": map[string]interface{}{
			"total":     heartbeatStats.TotalConnections,
			"healthy":   heartbeatStats.HealthyConnections,
			"warning":   heartbeatStats.WarningConnections,
			"unhealthy": heartbeatStats.UnhealthyConnections,
			"max_delay": heartbeatStats.MaxPongDelay.Seconds(),
			"min_delay": heartbeatStats.MinPongDelay.Seconds(),
			"avg_delay": heartbeatStats.AvgPongDelay.Seconds(),
		},
		"pubsub": map[string]interface{}{
			"publisher": map[string]interface{}{
				"total_published":     pubStats.TotalPublished,
				"user_published":      pubStats.UserPublished,
				"broadcast_published": pubStats.BroadcastPublished,
				"failed_published":    pubStats.FailedPublished,
			},
			"subscriber": map[string]interface{}{
				"total_received":     subStats.TotalReceived,
				"user_received":      subStats.UserReceived,
				"broadcast_received": subStats.BroadcastReceived,
				"processed_messages": subStats.ProcessedMessages,
				"failed_messages":    subStats.FailedMessages,
				"dropped_messages":   subStats.DroppedMessages,
				"reconnect_count":    subStats.ReconnectCount,
			},
		},
	}
}

// GetLimiter 返回连接限制器
func (m *Manager) GetLimiter() *ConnectionLimiter {
	return m.limiter
}

// GetRoom 返回房间管理器
func (m *Manager) GetRoom() *Room {
	return m.room
}

// GetOfflineService 返回离线消息服务
func (m *Manager) GetOfflineService() *offline.Service {
	return m.offline
}

// handlePubSubMessage 处理Pub/Sub用户消息
func (m *Manager) handlePubSubMessage(message *domain.SyncMessage, channel string) error {
	// 提取userID从channel名称 (sync:user:{userID})
	userID := message.UserID
	
	log.Printf("Received Pub/Sub message: channel=%s, user=%s, type=%s, id=%s, from_instance=%s", 
		channel, userID, message.Type, message.ID, message.InstanceID)

	// 检查用户是否在本实例在线
	if m.room.IsUserOnline(userID) {
		// 用户在本实例在线：广播到本地连接
		m.room.BroadcastJSON(userID, message)
		log.Printf("Broadcasted Pub/Sub message to local connections: user=%s, type=%s", userID, message.Type)
	} else {
		log.Printf("User not online on this instance: user=%s, instance=%s", userID, m.instanceID)
	}

	return nil
}

// handlePubSubBroadcast 处理全局广播消息
func (m *Manager) handlePubSubBroadcast(message *domain.SyncMessage, channel string) error {
	log.Printf("Received broadcast message: type=%s, id=%s, from_instance=%s", 
		message.Type, message.ID, message.InstanceID)

	// 广播给所有在线用户
	onlineUsers := m.room.GetOnlineUsers()
	for _, userID := range onlineUsers {
		m.room.BroadcastJSON(userID, message)
	}

	log.Printf("Broadcasted to %d online users", len(onlineUsers))
	return nil
}

// BroadcastToAll 广播消息给所有在线用户
func (m *Manager) BroadcastToAll(message *domain.SyncMessage) {
	if message.ID == "" {
		message.ID = uuid.New().String()
	}

	// 本地广播
	onlineUsers := m.room.GetOnlineUsers()
	for _, userID := range onlineUsers {
		m.room.BroadcastJSON(userID, message)
	}

	// 发布到Redis广播频道
	ctx := context.Background()
	if err := m.publisher.PublishBroadcast(ctx, message); err != nil {
		log.Printf("Failed to publish broadcast: error=%v", err)
	}

	log.Printf("Broadcasted to all: type=%s, id=%s, local_users=%d", message.Type, message.ID, len(onlineUsers))
}

// GetPublisher 返回Publisher
func (m *Manager) GetPublisher() *pubsub.Publisher {
	return m.publisher
}

// GetSubscriber 返回Subscriber
func (m *Manager) GetSubscriber() *pubsub.Subscriber {
	return m.subscriber
}

// GetInstanceID 返回实例ID
func (m *Manager) GetInstanceID() string {
	return m.instanceID
}

// AckOfflineMessage 确认离线消息
func (m *Manager) AckOfflineMessage(ctx context.Context, userID string, msgID string, ackToken string) error {
	return m.offline.Ack(ctx, userID, msgID, ackToken)
}

// BatchAckOfflineMessages 批量确认离线消息
func (m *Manager) BatchAckOfflineMessages(ctx context.Context, userID string, acks []offline.AckRequest) error {
	return m.offline.BatchAck(ctx, userID, acks)
}

// GetOfflineMessageCount 获取用户离线消息数量
func (m *Manager) GetOfflineMessageCount(ctx context.Context, userID string) (int64, error) {
	return m.offline.Count(ctx, userID)
}

// shutdown 关闭所有连接
func (m *Manager) shutdown() {
	log.Println("Shutting down WebSocket manager...")

	m.heartbeat.Stop()

	// 停止Subscriber
	if err := m.subscriber.Stop(); err != nil {
		log.Printf("Error stopping subscriber: %v", err)
	}

	// 关闭Publisher
	if err := m.publisher.Close(); err != nil {
		log.Printf("Error closing publisher: %v", err)
	}

	m.mu.Lock()
	connections := make([]*Connection, 0, len(m.connections))
	for _, conn := range m.connections {
		connections = append(connections, conn)
	}
	m.mu.Unlock()

	for _, conn := range connections {
		conn.Close("server shutdown")
	}

	log.Printf("WebSocket manager shutdown complete, closed %d connections", len(connections))
}
