package pubsub

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"sync-svc/internal/domain"

	"github.com/redis/go-redis/v9"
)

// MessageHandler 消息处理函数
type MessageHandler func(message *domain.SyncMessage, channel string) error

// Subscriber Redis Pub/Sub订阅器
type Subscriber struct {
	redis        *redis.Client
	instanceID   string
	handlers     map[string]MessageHandler // channel pattern -> handler
	handlersMux  sync.RWMutex
	
	// 订阅管理
	pubsub       *redis.PubSub
	cancelFunc   context.CancelFunc
	wg           sync.WaitGroup
	
	// 统计
	stats        SubscriberStats
	
	// 配置
	reconnectInterval time.Duration
	maxReconnectAttempts int
}

// SubscriberStats 订阅器统计
type SubscriberStats struct {
	TotalReceived      int64 `json:"total_received"`       // 总接收数
	UserReceived       int64 `json:"user_received"`        // 用户消息接收数
	BroadcastReceived  int64 `json:"broadcast_received"`   // 广播消息接收数
	ProcessedMessages  int64 `json:"processed_messages"`   // 处理成功数
	FailedMessages     int64 `json:"failed_messages"`      // 处理失败数
	DroppedMessages    int64 `json:"dropped_messages"`     // 丢弃消息数（来自自己的实例）
	ReconnectCount     int64 `json:"reconnect_count"`      // 重连次数
}

// SubscriberConfig 订阅器配置
type SubscriberConfig struct {
	InstanceID           string
	ReconnectInterval    time.Duration
	MaxReconnectAttempts int
}

// DefaultSubscriberConfig 默认配置
func DefaultSubscriberConfig(instanceID string) *SubscriberConfig {
	return &SubscriberConfig{
		InstanceID:           instanceID,
		ReconnectInterval:    5 * time.Second,
		MaxReconnectAttempts: 10,
	}
}

// NewSubscriber 创建订阅器
func NewSubscriber(redisClient *redis.Client, config *SubscriberConfig) *Subscriber {
	if config == nil {
		config = DefaultSubscriberConfig("default")
	}
	
	return &Subscriber{
		redis:                redisClient,
		instanceID:           config.InstanceID,
		handlers:             make(map[string]MessageHandler),
		reconnectInterval:    config.ReconnectInterval,
		maxReconnectAttempts: config.MaxReconnectAttempts,
	}
}

// Subscribe 订阅频道（支持通配符模式）
func (s *Subscriber) Subscribe(pattern string, handler MessageHandler) {
	s.handlersMux.Lock()
	defer s.handlersMux.Unlock()
	
	s.handlers[pattern] = handler
	log.Printf("Registered handler for pattern: %s", pattern)
}

// Unsubscribe 取消订阅
func (s *Subscriber) Unsubscribe(pattern string) {
	s.handlersMux.Lock()
	defer s.handlersMux.Unlock()
	
	delete(s.handlers, pattern)
	log.Printf("Unregistered handler for pattern: %s", pattern)
}

// Start 启动订阅器
func (s *Subscriber) Start(ctx context.Context) error {
	s.handlersMux.RLock()
	if len(s.handlers) == 0 {
		s.handlersMux.RUnlock()
		return fmt.Errorf("no handlers registered")
	}
	
	// 获取所有订阅模式
	patterns := make([]string, 0, len(s.handlers))
	for pattern := range s.handlers {
		patterns = append(patterns, pattern)
	}
	s.handlersMux.RUnlock()
	
	// 创建取消上下文
	subCtx, cancel := context.WithCancel(ctx)
	s.cancelFunc = cancel
	
	// 启动订阅
	s.wg.Add(1)
	go s.subscribeLoop(subCtx, patterns)
	
	log.Printf("Subscriber started for instance %s with patterns: %v", s.instanceID, patterns)
	return nil
}

// subscribeLoop 订阅循环（自动重连）
func (s *Subscriber) subscribeLoop(ctx context.Context, patterns []string) {
	defer s.wg.Done()
	
	reconnectAttempts := 0
	
	for {
		select {
		case <-ctx.Done():
			log.Println("Subscriber shutting down")
			return
		default:
		}
		
		// 创建订阅
		pubsub := s.redis.PSubscribe(ctx, patterns...)
		s.pubsub = pubsub
		
		log.Printf("Subscribed to patterns: %v", patterns)
		reconnectAttempts = 0
		
		// 处理消息
		_ = s.processMessages(ctx, pubsub)
		
		// 关闭订阅
		if err := pubsub.Close(); err != nil {
			log.Printf("Error closing pubsub: %v", err)
		}
		s.pubsub = nil
		
		// 检查是否应该停止
		select {
		case <-ctx.Done():
			return
		default:
		}
		
		// 重连逻辑
		reconnectAttempts++
		if s.maxReconnectAttempts > 0 && reconnectAttempts > s.maxReconnectAttempts {
			log.Printf("Max reconnect attempts reached (%d), stopping subscriber", s.maxReconnectAttempts)
			return
		}
		
		atomic.AddInt64(&s.stats.ReconnectCount, 1)
		log.Printf("Connection lost, reconnecting in %v (attempt %d)...", s.reconnectInterval, reconnectAttempts)
		
		timer := time.NewTimer(s.reconnectInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}

// processMessages 处理接收到的消息
func (s *Subscriber) processMessages(ctx context.Context, pubsub *redis.PubSub) error {
	ch := pubsub.Channel()
	
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg, ok := <-ch:
			if !ok {
				return fmt.Errorf("channel closed")
			}
			
			if msg == nil {
				continue
			}
			
			// 处理消息
			s.handleMessage(msg)
		}
	}
}

// handleMessage 处理单条消息
func (s *Subscriber) handleMessage(msg *redis.Message) {
	atomic.AddInt64(&s.stats.TotalReceived, 1)
	
	// 解析消息
	var syncMsg domain.SyncMessage
	if err := json.Unmarshal([]byte(msg.Payload), &syncMsg); err != nil {
		log.Printf("Failed to unmarshal message: %v", err)
		atomic.AddInt64(&s.stats.FailedMessages, 1)
		return
	}
	
	// 过滤来自自己实例的消息（避免循环）
	if syncMsg.InstanceID == s.instanceID {
		atomic.AddInt64(&s.stats.DroppedMessages, 1)
		log.Printf("Dropped message from own instance: %s, type=%s", s.instanceID, syncMsg.Type)
		return
	}
	
	// 更新接收统计
	if msg.Channel == broadcastChannel {
		atomic.AddInt64(&s.stats.BroadcastReceived, 1)
	} else if len(msg.Channel) > len(userChannelPrefix) {
		atomic.AddInt64(&s.stats.UserReceived, 1)
	}
	
	// 查找匹配的处理器
	s.handlersMux.RLock()
	handler := s.findHandler(msg.Pattern)
	s.handlersMux.RUnlock()
	
	if handler == nil {
		log.Printf("No handler found for pattern: %s", msg.Pattern)
		atomic.AddInt64(&s.stats.FailedMessages, 1)
		return
	}
	
	// 调用处理器
	if err := handler(&syncMsg, msg.Channel); err != nil {
		log.Printf("Handler error for channel %s: %v", msg.Channel, err)
		atomic.AddInt64(&s.stats.FailedMessages, 1)
		return
	}
	
	atomic.AddInt64(&s.stats.ProcessedMessages, 1)
	log.Printf("Processed message: channel=%s, type=%s, id=%s, from_instance=%s", 
		msg.Channel, syncMsg.Type, syncMsg.ID, syncMsg.InstanceID)
}

// findHandler 查找匹配的处理器
func (s *Subscriber) findHandler(pattern string) MessageHandler {
	// 直接匹配
	if handler, ok := s.handlers[pattern]; ok {
		return handler
	}
	
	// 如果pattern是通配符，返回第一个匹配的处理器
	for handlerPattern, handler := range s.handlers {
		if handlerPattern == pattern {
			return handler
		}
	}
	
	return nil
}

// Stop 停止订阅器
func (s *Subscriber) Stop() error {
	if s.cancelFunc != nil {
		s.cancelFunc()
	}
	
	// 等待所有goroutine退出
	s.wg.Wait()
	
	// 关闭pubsub
	if s.pubsub != nil {
		if err := s.pubsub.Close(); err != nil {
			log.Printf("Error closing pubsub: %v", err)
		}
		s.pubsub = nil
	}
	
	log.Println("Subscriber stopped")
	return nil
}

// GetStats 获取统计信息
func (s *Subscriber) GetStats() SubscriberStats {
	return SubscriberStats{
		TotalReceived:     atomic.LoadInt64(&s.stats.TotalReceived),
		UserReceived:      atomic.LoadInt64(&s.stats.UserReceived),
		BroadcastReceived: atomic.LoadInt64(&s.stats.BroadcastReceived),
		ProcessedMessages: atomic.LoadInt64(&s.stats.ProcessedMessages),
		FailedMessages:    atomic.LoadInt64(&s.stats.FailedMessages),
		DroppedMessages:   atomic.LoadInt64(&s.stats.DroppedMessages),
		ReconnectCount:    atomic.LoadInt64(&s.stats.ReconnectCount),
	}
}

// ResetStats 重置统计（用于测试）
func (s *Subscriber) ResetStats() {
	atomic.StoreInt64(&s.stats.TotalReceived, 0)
	atomic.StoreInt64(&s.stats.UserReceived, 0)
	atomic.StoreInt64(&s.stats.BroadcastReceived, 0)
	atomic.StoreInt64(&s.stats.ProcessedMessages, 0)
	atomic.StoreInt64(&s.stats.FailedMessages, 0)
	atomic.StoreInt64(&s.stats.DroppedMessages, 0)
	atomic.StoreInt64(&s.stats.ReconnectCount, 0)
}

// GetInstanceID 获取实例ID
func (s *Subscriber) GetInstanceID() string {
	return s.instanceID
}

// IsRunning 检查是否正在运行
func (s *Subscriber) IsRunning() bool {
	return s.pubsub != nil
}

// GetSubscribedPatterns 获取已订阅的模式
func (s *Subscriber) GetSubscribedPatterns() []string {
	s.handlersMux.RLock()
	defer s.handlersMux.RUnlock()
	
	patterns := make([]string, 0, len(s.handlers))
	for pattern := range s.handlers {
		patterns = append(patterns, pattern)
	}
	return patterns
}
