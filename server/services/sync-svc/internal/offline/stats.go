package offline

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// Stats 离线消息统计
type Stats struct {
	TotalPushed    int64 `json:"total_pushed"`     // 总推送数
	TotalPulled    int64 `json:"total_pulled"`     // 总拉取数
	TotalAcked     int64 `json:"total_acked"`      // 总确认数
	TotalExpired   int64 `json:"total_expired"`    // 总过期数
	CurrentPending int64 `json:"current_pending"`  // 当前待处理数
	LastCleanupAt  int64 `json:"last_cleanup_at"`  // 上次清理时间（Unix时间戳）
}

// StatsCollector 统计收集器
type StatsCollector struct {
	queue *Queue
	stats *Stats
	mu    sync.RWMutex
}

// NewStatsCollector 创建统计收集器
func NewStatsCollector(queue *Queue) *StatsCollector {
	return &StatsCollector{
		queue: queue,
		stats: &Stats{},
	}
}

// IncrementPushed 增加推送计数
func (sc *StatsCollector) IncrementPushed() {
	atomic.AddInt64(&sc.stats.TotalPushed, 1)
}

// IncrementPulled 增加拉取计数
func (sc *StatsCollector) IncrementPulled(count int) {
	atomic.AddInt64(&sc.stats.TotalPulled, int64(count))
}

// IncrementAcked 增加确认计数
func (sc *StatsCollector) IncrementAcked(count int) {
	atomic.AddInt64(&sc.stats.TotalAcked, int64(count))
}

// IncrementExpired 增加过期计数
func (sc *StatsCollector) IncrementExpired(count int) {
	atomic.AddInt64(&sc.stats.TotalExpired, int64(count))
}

// UpdateLastCleanup 更新上次清理时间
func (sc *StatsCollector) UpdateLastCleanup() {
	atomic.StoreInt64(&sc.stats.LastCleanupAt, time.Now().Unix())
}

// UpdatePendingCount 更新待处理消息数（需要扫描Redis）
func (sc *StatsCollector) UpdatePendingCount(ctx context.Context) error {
	// 扫描所有离线消息队列
	var cursor uint64
	var totalPending int64
	
	for {
		keys, nextCursor, err := sc.queue.redis.Scan(ctx, cursor, offlineQueueKeyPrefix+"*", 100).Result()
		if err != nil {
			return fmt.Errorf("failed to scan keys: %w", err)
		}
		
		for _, key := range keys {
			count, err := sc.queue.redis.LLen(ctx, key).Result()
			if err != nil {
				continue
			}
			totalPending += count
		}
		
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
	
	atomic.StoreInt64(&sc.stats.CurrentPending, totalPending)
	return nil
}

// GetStats 获取统计信息
func (sc *StatsCollector) GetStats() *Stats {
	return &Stats{
		TotalPushed:    atomic.LoadInt64(&sc.stats.TotalPushed),
		TotalPulled:    atomic.LoadInt64(&sc.stats.TotalPulled),
		TotalAcked:     atomic.LoadInt64(&sc.stats.TotalAcked),
		TotalExpired:   atomic.LoadInt64(&sc.stats.TotalExpired),
		CurrentPending: atomic.LoadInt64(&sc.stats.CurrentPending),
		LastCleanupAt:  atomic.LoadInt64(&sc.stats.LastCleanupAt),
	}
}

// Reset 重置统计（慎用）
func (sc *StatsCollector) Reset() {
	atomic.StoreInt64(&sc.stats.TotalPushed, 0)
	atomic.StoreInt64(&sc.stats.TotalPulled, 0)
	atomic.StoreInt64(&sc.stats.TotalAcked, 0)
	atomic.StoreInt64(&sc.stats.TotalExpired, 0)
	atomic.StoreInt64(&sc.stats.CurrentPending, 0)
	atomic.StoreInt64(&sc.stats.LastCleanupAt, 0)
}

// UserQueueInfo 用户队列信息
type UserQueueInfo struct {
	UserID       string    `json:"user_id"`
	MessageCount int64     `json:"message_count"`
	OldestMsg    time.Time `json:"oldest_msg"`
	NewestMsg    time.Time `json:"newest_msg"`
}

// GetUserQueueInfo 获取用户队列详细信息
func (sc *StatsCollector) GetUserQueueInfo(ctx context.Context, userID string) (*UserQueueInfo, error) {
	messages, err := sc.queue.Pull(ctx, userID, maxOfflineMessages)
	if err != nil {
		return nil, err
	}
	
	info := &UserQueueInfo{
		UserID:       userID,
		MessageCount: int64(len(messages)),
	}
	
	if len(messages) > 0 {
		// 找到最早和最新的消息
		info.NewestMsg = messages[0].CreatedAt
		info.OldestMsg = messages[len(messages)-1].CreatedAt
	}
	
	return info, nil
}

// Service 离线消息服务（集成队列和统计）
type Service struct {
	queue     *Queue
	collector *StatsCollector
}

// NewService 创建离线消息服务
func NewService(queue *Queue) *Service {
	return &Service{
		queue:     queue,
		collector: NewStatsCollector(queue),
	}
}

// Push 推送消息并统计
func (s *Service) Push(ctx context.Context, userID string, msgType string, data map[string]interface{}) (*OfflineMessage, error) {
	msg, err := s.queue.Push(ctx, userID, msgType, data)
	if err != nil {
		return nil, err
	}
	
	s.collector.IncrementPushed()
	return msg, nil
}

// Pull 拉取消息并统计
func (s *Service) Pull(ctx context.Context, userID string, limit int) ([]*OfflineMessage, error) {
	messages, err := s.queue.Pull(ctx, userID, limit)
	if err != nil {
		return nil, err
	}
	
	s.collector.IncrementPulled(len(messages))
	return messages, nil
}

// Ack 确认消息并统计
func (s *Service) Ack(ctx context.Context, userID string, msgID string, ackToken string) error {
	if err := s.queue.Ack(ctx, userID, msgID, ackToken); err != nil {
		return err
	}
	
	s.collector.IncrementAcked(1)
	return nil
}

// BatchAck 批量确认消息并统计
func (s *Service) BatchAck(ctx context.Context, userID string, acks []AckRequest) error {
	if err := s.queue.BatchAck(ctx, userID, acks); err != nil {
		return err
	}
	
	s.collector.IncrementAcked(len(acks))
	return nil
}

// Count 获取消息数量
func (s *Service) Count(ctx context.Context, userID string) (int64, error) {
	return s.queue.Count(ctx, userID)
}

// Clear 清空消息
func (s *Service) Clear(ctx context.Context, userID string) error {
	return s.queue.Clear(ctx, userID)
}

// CleanupExpired 清理过期消息
func (s *Service) CleanupExpired(ctx context.Context) (int, error) {
	cleaned, err := s.queue.CleanupExpired(ctx)
	if err != nil {
		return 0, err
	}
	
	s.collector.IncrementExpired(cleaned)
	s.collector.UpdateLastCleanup()
	return cleaned, nil
}

// GetStats 获取统计信息
func (s *Service) GetStats(ctx context.Context) (*Stats, error) {
	// 更新待处理消息数
	if err := s.collector.UpdatePendingCount(ctx); err != nil {
		return nil, err
	}
	
	return s.collector.GetStats(), nil
}

// GetUserQueueInfo 获取用户队列信息
func (s *Service) GetUserQueueInfo(ctx context.Context, userID string) (*UserQueueInfo, error) {
	return s.collector.GetUserQueueInfo(ctx, userID)
}
