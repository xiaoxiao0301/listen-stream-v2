package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/consul/api"
	"github.com/redis/go-redis/v9"
)

// ConfigService 配置管理服务（Consul KV + PostgreSQL双存储）
type ConfigService struct {
	consulClient *api.Client
	redisClient  *redis.Client
	configPrefix string // Consul KV前缀（如: "listen-stream/"）
	cachePrefix  string // Redis缓存前缀
	cacheTTL     time.Duration
}

// NewConfigService 创建配置服务
func NewConfigService(
	consulClient *api.Client,
	redisClient *redis.Client,
	configPrefix string,
) *ConfigService {
	return &ConfigService{
		consulClient: consulClient,
		redisClient:  redisClient,
		configPrefix: configPrefix,
		cachePrefix:  "config:cache:",
		cacheTTL:     30 * time.Second, // 30秒缓存
	}
}

// ConfigItem 配置项
type ConfigItem struct {
	Key         string    `json:"key"`
	Value       string    `json:"value"`
	Version     uint64    `json:"version"` // Consul ModifyIndex
	Description string    `json:"description,omitempty"`
	UpdatedAt   time.Time `json:"updated_at"`
	UpdatedBy   string    `json:"updated_by"`
}

// ConfigHistory 配置变更历史（存储在PostgreSQL）
type ConfigHistory struct {
	ID           string    `json:"id" db:"id"`
	ConfigKey    string    `json:"config_key" db:"config_key"`
	OldValue     string    `json:"old_value" db:"old_value"`
	NewValue     string    `json:"new_value" db:"new_value"`
	Version      int64     `json:"version" db:"version"`
	AdminID      string    `json:"admin_id" db:"admin_id"`
	AdminName    string    `json:"admin_name" db:"admin_name"`
	Reason       string    `json:"reason" db:"reason"` // 变更原因
	Rollbackable bool      `json:"rollbackable" db:"rollbackable"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

// Get 获取配置（带缓存）
func (s *ConfigService) Get(ctx context.Context, key string) (string, error) {
	// 1. 尝试从Redis缓存读取
	cacheKey := s.cachePrefix + key
	cached, err := s.redisClient.Get(ctx, cacheKey).Result()
	if err == nil {
		return cached, nil
	}

	// 2. 从Consul读取
	fullKey := s.configPrefix + key
	pair, _, err := s.consulClient.KV().Get(fullKey, nil)
	if err != nil {
		return "", fmt.Errorf("consul get %s: %w", fullKey, err)
	}
	if pair == nil {
		return "", fmt.Errorf("config key not found: %s", key)
	}

	value := string(pair.Value)

	// 3. 写入Redis缓存
	_ = s.redisClient.Set(ctx, cacheKey, value, s.cacheTTL).Err()

	return value, nil
}

// Set 设置配置（写入Consul + 清除缓存）
func (s *ConfigService) Set(ctx context.Context, key, value string) error {
	fullKey := s.configPrefix + key

	// 1. 写入Consul
	p := &api.KVPair{
		Key:   fullKey,
		Value: []byte(value),
	}
	if _, err := s.consulClient.KV().Put(p, nil); err != nil {
		return fmt.Errorf("consul put %s: %w", fullKey, err)
	}

	// 2. 清除Redis缓存
	cacheKey := s.cachePrefix + key
	_ = s.redisClient.Del(ctx, cacheKey).Err()

	// 3. 发布变更通知（Redis Pub/Sub）
	notification := map[string]interface{}{
		"key":       key,
		"timestamp": time.Now().Unix(),
	}
	data, _ := json.Marshal(notification)
	_ = s.redisClient.Publish(ctx, "config:change", data).Err()

	return nil
}

// GetWithVersion 获取配置及版本号
func (s *ConfigService) GetWithVersion(ctx context.Context, key string) (*ConfigItem, error) {
	fullKey := s.configPrefix + key
	pair, _, err := s.consulClient.KV().Get(fullKey, nil)
	if err != nil {
		return nil, fmt.Errorf("consul get %s: %w", fullKey, err)
	}
	if pair == nil {
		return nil, fmt.Errorf("config key not found: %s", key)
	}

	return &ConfigItem{
		Key:       key,
		Value:     string(pair.Value),
		Version:   pair.ModifyIndex,
		UpdatedAt: time.Now(), // Consul不存储更新时间，这里使用当前时间
	}, nil
}

// List 列出所有配置
func (s *ConfigService) List(ctx context.Context, prefix string) ([]*ConfigItem, error) {
	fullPrefix := s.configPrefix + prefix
	pairs, _, err := s.consulClient.KV().List(fullPrefix, nil)
	if err != nil {
		return nil, fmt.Errorf("consul list %s: %w", fullPrefix, err)
	}

	items := make([]*ConfigItem, 0, len(pairs))
	for _, pair := range pairs {
		// 移除前缀
		key := pair.Key[len(s.configPrefix):]
		items = append(items, &ConfigItem{
			Key:       key,
			Value:     string(pair.Value),
			Version:   pair.ModifyIndex,
			UpdatedAt: time.Now(),
		})
	}

	return items, nil
}

// Delete 删除配置
func (s *ConfigService) Delete(ctx context.Context, key string) error {
	fullKey := s.configPrefix + key

	// 1. 从Consul删除
	if _, err := s.consulClient.KV().Delete(fullKey, nil); err != nil {
		return fmt.Errorf("consul delete %s: %w", fullKey, err)
	}

	// 2. 清除Redis缓存
	cacheKey := s.cachePrefix + key
	_ = s.redisClient.Del(ctx, cacheKey).Err()

	// 3. 发布变更通知
	notification := map[string]interface{}{
		"key":       key,
		"deleted":   true,
		"timestamp": time.Now().Unix(),
	}
	data, _ := json.Marshal(notification)
	_ = s.redisClient.Publish(ctx, "config:change", data).Err()

	return nil
}

// SaveHistory 保存配置变更历史（由Repository层实际写入PostgreSQL）
func (s *ConfigService) CreateHistory(
	configKey, oldValue, newValue string,
	adminID, adminName, reason string,
) *ConfigHistory {
	return &ConfigHistory{
		ID:           uuid.New().String(),
		ConfigKey:    configKey,
		OldValue:     oldValue,
		NewValue:     newValue,
		Version:      time.Now().Unix(),
		AdminID:      adminID,
		AdminName:    adminName,
		Reason:       reason,
		Rollbackable: true,
		CreatedAt:    time.Now(),
	}
}

// WatchChanges 监听配置变更（阻塞）
func (s *ConfigService) WatchChanges(ctx context.Context, callback func(key, value string)) error {
	pubsub := s.redisClient.Subscribe(ctx, "config:change")
	defer pubsub.Close()

	ch := pubsub.Channel()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg := <-ch:
			var notification map[string]interface{}
			if err := json.Unmarshal([]byte(msg.Payload), &notification); err != nil {
				continue
			}

			key, _ := notification["key"].(string)
			if key == "" {
				continue
			}

			// 获取新值
			value, err := s.Get(ctx, key)
			if err != nil {
				continue
			}

			callback(key, value)
		}
	}
}

// ClearCache 清除所有缓存
func (s *ConfigService) ClearCache(ctx context.Context) error {
	iter := s.redisClient.Scan(ctx, 0, s.cachePrefix+"*", 0).Iterator()
	var keys []string
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}
	if err := iter.Err(); err != nil {
		return err
	}

	if len(keys) > 0 {
		return s.redisClient.Del(ctx, keys...).Err()
	}
	return nil
}
