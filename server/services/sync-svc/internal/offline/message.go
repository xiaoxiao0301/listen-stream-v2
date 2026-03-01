package offline

import (
	"encoding/json"
	"time"
)

// OfflineMessage 离线消息
type OfflineMessage struct {
	ID        string                 `json:"id"`         // 消息ID
	UserID    string                 `json:"user_id"`    // 用户ID
	Type      string                 `json:"type"`       // 消息类型
	Data      map[string]interface{} `json:"data"`       // 消息数据
	AckToken  string                 `json:"ack_token"`  // ACK确认令牌
	CreatedAt time.Time              `json:"created_at"` // 创建时间
	ExpiresAt time.Time              `json:"expires_at"` // 过期时间
}

// ToJSON 转换为JSON字符串
func (m *OfflineMessage) ToJSON() (string, error) {
	bytes, err := json.Marshal(m)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// FromJSON 从JSON字符串解析
func FromJSON(data string) (*OfflineMessage, error) {
	var msg OfflineMessage
	if err := json.Unmarshal([]byte(data), &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

// IsExpired 检查消息是否过期
func (m *OfflineMessage) IsExpired() bool {
	return time.Now().After(m.ExpiresAt)
}

// MessageOptions 消息选项
type MessageOptions struct {
	TTL time.Duration // 消息生存时间，默认7天
}

// DefaultMessageOptions 默认消息选项
func DefaultMessageOptions() *MessageOptions {
	return &MessageOptions{
		TTL: 7 * 24 * time.Hour, // 7天
	}
}
