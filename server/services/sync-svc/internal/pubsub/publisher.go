package pubsub

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync/atomic"
	"time"

	"sync-svc/internal/domain"

	"github.com/redis/go-redis/v9"
)

const (
	// 频道命名规范
	userChannelPrefix      = "sync:user:"      // 用户专用频道: sync:user:{userID}
	broadcastChannel       = "sync:broadcast"  // 全局广播频道
	instanceChannelPrefix  = "sync:instance:"  // 实例专用频道: sync:instance:{instanceID}
)

// Publisher Redis Pub/Sub发布器
type Publisher struct {
	redis      *redis.Client
	instanceID string // 实例ID，防止接收自己发布的消息
	stats      PublisherStats
}

// PublisherStats 发布器统计
type PublisherStats struct {
	TotalPublished    int64 `json:"total_published"`     // 总发布数
	UserPublished     int64 `json:"user_published"`      // 用户频道发布数
	BroadcastPublished int64 `json:"broadcast_published"` // 广播频道发布数
	FailedPublished   int64 `json:"failed_published"`    // 发布失败数
}

// NewPublisher 创建发布器
func NewPublisher(redisClient *redis.Client, instanceID string) *Publisher {
	return &Publisher{
		redis:      redisClient,
		instanceID: instanceID,
	}
}

// PublishToUser 发布消息到指定用户频道
func (p *Publisher) PublishToUser(ctx context.Context, userID string, message *domain.SyncMessage) error {
	// 添加实例ID，防止循环
	message.InstanceID = p.instanceID

	// 序列化消息
	msgBytes, err := json.Marshal(message)
	if err != nil {
		atomic.AddInt64(&p.stats.FailedPublished, 1)
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// 发布到用户频道
	channel := p.getUserChannel(userID)
	if err := p.redis.Publish(ctx, channel, msgBytes).Err(); err != nil {
		atomic.AddInt64(&p.stats.FailedPublished, 1)
		return fmt.Errorf("failed to publish to %s: %w", channel, err)
	}

	atomic.AddInt64(&p.stats.TotalPublished, 1)
	atomic.AddInt64(&p.stats.UserPublished, 1)

	log.Printf("Published to user channel: user=%s, type=%s, id=%s, instance=%s", 
		userID, message.Type, message.ID, p.instanceID)

	return nil
}

// PublishBroadcast 发布全局广播消息
func (p *Publisher) PublishBroadcast(ctx context.Context, message *domain.SyncMessage) error {
	// 添加实例ID
	message.InstanceID = p.instanceID

	// 序列化消息
	msgBytes, err := json.Marshal(message)
	if err != nil {
		atomic.AddInt64(&p.stats.FailedPublished, 1)
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// 发布到广播频道
	if err := p.redis.Publish(ctx, broadcastChannel, msgBytes).Err(); err != nil {
		atomic.AddInt64(&p.stats.FailedPublished, 1)
		return fmt.Errorf("failed to publish broadcast: %w", err)
	}

	atomic.AddInt64(&p.stats.TotalPublished, 1)
	atomic.AddInt64(&p.stats.BroadcastPublished, 1)

	log.Printf("Published broadcast: type=%s, id=%s, instance=%s", 
		message.Type, message.ID, p.instanceID)

	return nil
}

// PublishToInstance 发布消息到指定实例（用于实例间通信）
func (p *Publisher) PublishToInstance(ctx context.Context, instanceID string, message *domain.SyncMessage) error {
	// 添加实例ID
	message.InstanceID = p.instanceID

	// 序列化消息
	msgBytes, err := json.Marshal(message)
	if err != nil {
		atomic.AddInt64(&p.stats.FailedPublished, 1)
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// 发布到实例频道
	channel := p.getInstanceChannel(instanceID)
	if err := p.redis.Publish(ctx, channel, msgBytes).Err(); err != nil {
		atomic.AddInt64(&p.stats.FailedPublished, 1)
		return fmt.Errorf("failed to publish to instance %s: %w", instanceID, err)
	}

	atomic.AddInt64(&p.stats.TotalPublished, 1)

	log.Printf("Published to instance: target=%s, type=%s, id=%s, from=%s", 
		instanceID, message.Type, message.ID, p.instanceID)

	return nil
}

// GetStats 获取统计信息
func (p *Publisher) GetStats() PublisherStats {
	return PublisherStats{
		TotalPublished:     atomic.LoadInt64(&p.stats.TotalPublished),
		UserPublished:      atomic.LoadInt64(&p.stats.UserPublished),
		BroadcastPublished: atomic.LoadInt64(&p.stats.BroadcastPublished),
		FailedPublished:    atomic.LoadInt64(&p.stats.FailedPublished),
	}
}

// getUserChannel 获取用户频道名
func (p *Publisher) getUserChannel(userID string) string {
	return userChannelPrefix + userID
}

// getInstanceChannel 获取实例频道名
func (p *Publisher) getInstanceChannel(instanceID string) string {
	return instanceChannelPrefix + instanceID
}

// Ping 测试Redis连接
func (p *Publisher) Ping(ctx context.Context) error {
	return p.redis.Ping(ctx).Err()
}

// Close 关闭发布器（目前无需特殊清理）
func (p *Publisher) Close() error {
	log.Println("Publisher closed")
	return nil
}

// BatchPublishToUsers 批量发布到多个用户
func (p *Publisher) BatchPublishToUsers(ctx context.Context, userIDs []string, message *domain.SyncMessage) error {
	if len(userIDs) == 0 {
		return nil
	}

	// 添加实例ID
	message.InstanceID = p.instanceID

	// 序列化消息（只序列化一次）
	msgBytes, err := json.Marshal(message)
	if err != nil {
		atomic.AddInt64(&p.stats.FailedPublished, 1)
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// 使用Pipeline批量发布
	pipe := p.redis.Pipeline()
	for _, userID := range userIDs {
		channel := p.getUserChannel(userID)
		pipe.Publish(ctx, channel, msgBytes)
	}

	// 执行批量操作
	_, err = pipe.Exec(ctx)
	if err != nil {
		atomic.AddInt64(&p.stats.FailedPublished, int64(len(userIDs)))
		return fmt.Errorf("failed to batch publish: %w", err)
	}

	atomic.AddInt64(&p.stats.TotalPublished, int64(len(userIDs)))
	atomic.AddInt64(&p.stats.UserPublished, int64(len(userIDs)))

	log.Printf("Batch published to %d users: type=%s, id=%s", len(userIDs), message.Type, message.ID)

	return nil
}

// GetInstanceID 获取实例ID
func (p *Publisher) GetInstanceID() string {
	return p.instanceID
}

// SetInstanceID 设置实例ID（用于测试）
func (p *Publisher) SetInstanceID(instanceID string) {
	p.instanceID = instanceID
}

// ResetStats 重置统计（用于测试）
func (p *Publisher) ResetStats() {
	atomic.StoreInt64(&p.stats.TotalPublished, 0)
	atomic.StoreInt64(&p.stats.UserPublished, 0)
	atomic.StoreInt64(&p.stats.BroadcastPublished, 0)
	atomic.StoreInt64(&p.stats.FailedPublished, 0)
}

// MessageEnvelope 消息信封（包含元数据）
type MessageEnvelope struct {
	Message    *domain.SyncMessage `json:"message"`
	InstanceID string              `json:"instance_id"`
	PublishedAt time.Time          `json:"published_at"`
}

// PublishEnvelope 发布带元数据的消息信封
func (p *Publisher) PublishEnvelope(ctx context.Context, userID string, message *domain.SyncMessage) error {
	envelope := MessageEnvelope{
		Message:    message,
		InstanceID: p.instanceID,
		PublishedAt: time.Now(),
	}

	// 序列化信封
	msgBytes, err := json.Marshal(envelope)
	if err != nil {
		atomic.AddInt64(&p.stats.FailedPublished, 1)
		return fmt.Errorf("failed to marshal envelope: %w", err)
	}

	// 发布到用户频道
	channel := p.getUserChannel(userID)
	if err := p.redis.Publish(ctx, channel, msgBytes).Err(); err != nil {
		atomic.AddInt64(&p.stats.FailedPublished, 1)
		return fmt.Errorf("failed to publish envelope: %w", err)
	}

	atomic.AddInt64(&p.stats.TotalPublished, 1)
	atomic.AddInt64(&p.stats.UserPublished, 1)

	return nil
}
