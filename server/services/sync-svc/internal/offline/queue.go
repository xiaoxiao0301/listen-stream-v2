package offline

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	// 离线消息队列键前缀
	offlineQueueKeyPrefix = "offline:queue:"
	
	// 消息最大保留数量（每用户）
	maxOfflineMessages = 100
	
	// 消息默认过期时间
	defaultMessageTTL = 7 * 24 * time.Hour // 7天
)

// Queue 离线消息队列
type Queue struct {
	redis *redis.Client
	opts  *MessageOptions
}

// NewQueue 创建离线消息队列
func NewQueue(redisClient *redis.Client, opts *MessageOptions) *Queue {
	if opts == nil {
		opts = DefaultMessageOptions()
	}
	
	return &Queue{
		redis: redisClient,
		opts:  opts,
	}
}

// Push 推送离线消息到队列
func (q *Queue) Push(ctx context.Context, userID string, msgType string, data map[string]interface{}) (*OfflineMessage, error) {
	// 生成消息ID和ACK令牌
	msgID := uuid.New().String()
	ackToken, err := generateAckToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate ack token: %w", err)
	}
	
	// 构建离线消息
	now := time.Now()
	msg := &OfflineMessage{
		ID:        msgID,
		UserID:    userID,
		Type:      msgType,
		Data:      data,
		AckToken:  ackToken,
		CreatedAt: now,
		ExpiresAt: now.Add(q.opts.TTL),
	}
	
	// 序列化消息
	msgJSON, err := msg.ToJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to serialize message: %w", err)
	}
	
	// 获取队列键
	queueKey := q.getQueueKey(userID)
	
	// 使用事务确保原子性
	pipe := q.redis.Pipeline()
	
	// 推送到队列头部（LPUSH）
	pipe.LPush(ctx, queueKey, msgJSON)
	
	// 限制队列长度（LTRIM保留最新的maxOfflineMessages条）
	pipe.LTrim(ctx, queueKey, 0, maxOfflineMessages-1)
	
	// 设置队列过期时间（防止僵尸键）
	pipe.Expire(ctx, queueKey, q.opts.TTL+24*time.Hour)
	
	// 执行事务
	if _, err := pipe.Exec(ctx); err != nil {
		return nil, fmt.Errorf("failed to push message: %w", err)
	}
	
	log.Printf("Pushed offline message: user=%s, type=%s, id=%s", userID, msgType, msgID)
	
	return msg, nil
}

// Pull 拉取用户的所有离线消息
func (q *Queue) Pull(ctx context.Context, userID string, limit int) ([]*OfflineMessage, error) {
	if limit <= 0 {
		limit = maxOfflineMessages
	}
	
	queueKey := q.getQueueKey(userID)
	
	// 获取所有消息（倒序，最新的在前面）
	results, err := q.redis.LRange(ctx, queueKey, 0, int64(limit-1)).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to pull messages: %w", err)
	}
	
	if len(results) == 0 {
		return []*OfflineMessage{}, nil
	}
	
	// 解析消息
	messages := make([]*OfflineMessage, 0, len(results))
	expiredIDs := make([]string, 0)
	
	for _, msgJSON := range results {
		msg, err := FromJSON(msgJSON)
		if err != nil {
			log.Printf("Failed to parse offline message: %v", err)
			continue
		}
		
		// 检查是否过期
		if msg.IsExpired() {
			expiredIDs = append(expiredIDs, msg.ID)
			continue
		}
		
		messages = append(messages, msg)
	}
	
	// 删除过期消息
	if len(expiredIDs) > 0 {
		go q.removeExpiredMessages(context.Background(), userID, expiredIDs)
	}
	
	log.Printf("Pulled offline messages: user=%s, count=%d, expired=%d", userID, len(messages), len(expiredIDs))
	
	return messages, nil
}

// Ack 确认消息（删除已确认的消息）
func (q *Queue) Ack(ctx context.Context, userID string, msgID string, ackToken string) error {
	queueKey := q.getQueueKey(userID)
	
	// 获取队列中的所有消息
	results, err := q.redis.LRange(ctx, queueKey, 0, -1).Result()
	if err != nil {
		return fmt.Errorf("failed to get messages: %w", err)
	}
	
	// 查找目标消息
	var targetJSON string
	for _, msgJSON := range results {
		msg, err := FromJSON(msgJSON)
		if err != nil {
			continue
		}
		
		if msg.ID == msgID {
			// 验证AckToken
			if msg.AckToken != ackToken {
				return fmt.Errorf("invalid ack token")
			}
			targetJSON = msgJSON
			break
		}
	}
	
	if targetJSON == "" {
		return fmt.Errorf("message not found: %s", msgID)
	}
	
	// 删除消息（LREM删除所有匹配的值）
	removed, err := q.redis.LRem(ctx, queueKey, 0, targetJSON).Result()
	if err != nil {
		return fmt.Errorf("failed to remove message: %w", err)
	}
	
	if removed == 0 {
		return fmt.Errorf("message not removed: %s", msgID)
	}
	
	log.Printf("Acked offline message: user=%s, id=%s", userID, msgID)
	
	return nil
}

// BatchAck 批量确认消息
func (q *Queue) BatchAck(ctx context.Context, userID string, acks []AckRequest) error {
	queueKey := q.getQueueKey(userID)
	
	// 获取队列中的所有消息
	results, err := q.redis.LRange(ctx, queueKey, 0, -1).Result()
	if err != nil {
		return fmt.Errorf("failed to get messages: %w", err)
	}
	
	// 构建消息映射
	msgMap := make(map[string]string) // msgID -> msgJSON
	tokenMap := make(map[string]string) // msgID -> ackToken
	
	for _, msgJSON := range results {
		msg, err := FromJSON(msgJSON)
		if err != nil {
			continue
		}
		msgMap[msg.ID] = msgJSON
		tokenMap[msg.ID] = msg.AckToken
	}
	
	// 验证并删除消息
	pipe := q.redis.Pipeline()
	validCount := 0
	
	for _, ack := range acks {
		msgJSON, exists := msgMap[ack.MessageID]
		if !exists {
			log.Printf("Message not found for ack: user=%s, id=%s", userID, ack.MessageID)
			continue
		}
		
		expectedToken, _ := tokenMap[ack.MessageID]
		if expectedToken != ack.AckToken {
			log.Printf("Invalid ack token: user=%s, id=%s", userID, ack.MessageID)
			continue
		}
		
		pipe.LRem(ctx, queueKey, 0, msgJSON)
		validCount++
	}
	
	if validCount == 0 {
		return fmt.Errorf("no valid acks")
	}
	
	// 执行批量删除
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("failed to batch ack: %w", err)
	}
	
	log.Printf("Batch acked offline messages: user=%s, count=%d", userID, validCount)
	
	return nil
}

// Count 获取用户离线消息数量
func (q *Queue) Count(ctx context.Context, userID string) (int64, error) {
	queueKey := q.getQueueKey(userID)
	return q.redis.LLen(ctx, queueKey).Result()
}

// Clear 清空用户的所有离线消息
func (q *Queue) Clear(ctx context.Context, userID string) error {
	queueKey := q.getQueueKey(userID)
	return q.redis.Del(ctx, queueKey).Err()
}

// CleanupExpired 清理过期消息（定时任务）
func (q *Queue) CleanupExpired(ctx context.Context) (int, error) {
	// 扫描所有离线消息队列键
	var cursor uint64
	var totalCleaned int
	
	for {
		keys, nextCursor, err := q.redis.Scan(ctx, cursor, offlineQueueKeyPrefix+"*", 100).Result()
		if err != nil {
			return totalCleaned, fmt.Errorf("failed to scan keys: %w", err)
		}
		
		// 清理每个队列中的过期消息
		for _, key := range keys {
			cleaned, err := q.cleanupQueueExpired(ctx, key)
			if err != nil {
				log.Printf("Failed to cleanup queue %s: %v", key, err)
				continue
			}
			totalCleaned += cleaned
		}
		
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
	
	log.Printf("Cleaned up expired offline messages: total=%d", totalCleaned)
	
	return totalCleaned, nil
}

// getQueueKey 获取队列键
func (q *Queue) getQueueKey(userID string) string {
	return offlineQueueKeyPrefix + userID
}

// cleanupQueueExpired 清理单个队列中的过期消息
func (q *Queue) cleanupQueueExpired(ctx context.Context, queueKey string) (int, error) {
	results, err := q.redis.LRange(ctx, queueKey, 0, -1).Result()
	if err != nil {
		return 0, err
	}
	
	expiredCount := 0
	pipe := q.redis.Pipeline()
	
	for _, msgJSON := range results {
		msg, err := FromJSON(msgJSON)
		if err != nil {
			continue
		}
		
		if msg.IsExpired() {
			pipe.LRem(ctx, queueKey, 0, msgJSON)
			expiredCount++
		}
	}
	
	if expiredCount > 0 {
		if _, err := pipe.Exec(ctx); err != nil {
			return 0, err
		}
	}
	
	return expiredCount, nil
}

// removeExpiredMessages 异步删除过期消息
func (q *Queue) removeExpiredMessages(ctx context.Context, userID string, msgIDs []string) {
	queueKey := q.getQueueKey(userID)
	
	results, err := q.redis.LRange(ctx, queueKey, 0, -1).Result()
	if err != nil {
		log.Printf("Failed to get messages for cleanup: %v", err)
		return
	}
	
	pipe := q.redis.Pipeline()
	removed := 0
	
	for _, msgJSON := range results {
		msg, err := FromJSON(msgJSON)
		if err != nil {
			continue
		}
		
		for _, expiredID := range msgIDs {
			if msg.ID == expiredID {
				pipe.LRem(ctx, queueKey, 0, msgJSON)
				removed++
				break
			}
		}
	}
	
	if removed > 0 {
		if _, err := pipe.Exec(ctx); err != nil {
			log.Printf("Failed to remove expired messages: %v", err)
		} else {
			log.Printf("Removed expired messages: user=%s, count=%d", userID, removed)
		}
	}
}

// generateAckToken 生成ACK令牌
func generateAckToken() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// AckRequest ACK请求
type AckRequest struct {
	MessageID string `json:"message_id"`
	AckToken  string `json:"ack_token"`
}
