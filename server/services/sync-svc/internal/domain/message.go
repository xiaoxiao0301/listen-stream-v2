package domain

import "time"

// MessageType 消息类型
type MessageType string

const (
	MessageTypeFavoriteAdded     MessageType = "favorite.added"
	MessageTypeFavoriteRemoved   MessageType = "favorite.removed"
	MessageTypePlaylistCreated   MessageType = "playlist.created"
	MessageTypePlaylistUpdated   MessageType = "playlist.updated"
	MessageTypePlaylistDeleted   MessageType = "playlist.deleted"
	MessageTypePlaylistSongAdded MessageType = "playlist.song.added"
	MessageTypePlaylistSongRemoved MessageType = "playlist.song.removed"
	MessageTypeHistoryAdded      MessageType = "history.added"
	MessageTypePing              MessageType = "ping"
	MessageTypePong              MessageType = "pong"
)

// SyncMessage WebSocket同步消息
type SyncMessage struct {
	ID         string                 `json:"id"`         // 消息ID
	Type       MessageType            `json:"type"`       // 消息类型
	UserID     string                 `json:"user_id"`    // 目标用户ID
	Data       map[string]interface{} `json:"data"`       // 消息数据
	Timestamp  time.Time              `json:"timestamp"`  // 时间戳
	AckToken   string                 `json:"ack_token,omitempty"` // ACK令牌
	InstanceID string                 `json:"instance_id,omitempty"` // 发送实例ID（用于Pub/Sub）
}

// PingMessage 心跳Ping消息
type PingMessage struct {
	Type      MessageType `json:"type"` // "ping"
	Timestamp time.Time   `json:"timestamp"`
}

// PongMessage 心跳Pong响应
type PongMessage struct {
	Type      MessageType `json:"type"` // "pong"
	Timestamp time.Time   `json:"timestamp"`
}
