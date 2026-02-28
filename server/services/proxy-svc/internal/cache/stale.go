package cache

import (
	"context"
	"fmt"
	"time"
)

// StaleCache L3降级缓存（允许返回过期数据）
type StaleCache struct {
	redis    *RedisCache
	staleTTL time.Duration // stale数据保留时间（例如24小时）
}

// NewStaleCache 创建降级缓存
func NewStaleCache(redis *RedisCache, staleTTL time.Duration) *StaleCache {
	return &StaleCache{
		redis:    redis,
		staleTTL: staleTTL,
	}
}

// GetStale 获取stale数据（即使已过期）
func (s *StaleCache) GetStale(ctx context.Context, key string) ([]byte, error) {
	staleKey := s.makeStaleKey(key)
	return s.redis.Get(ctx, staleKey)
}

// SetStale 设置stale数据（长时间保留）
func (s *StaleCache) SetStale(ctx context.Context, key string, data []byte) error {
	staleKey := s.makeStaleKey(key)
	return s.redis.Set(ctx, staleKey, data, s.staleTTL)
}

// DeleteStale 删除stale数据
func (s *StaleCache) DeleteStale(ctx context.Context, key string) error {
	staleKey := s.makeStaleKey(key)
	return s.redis.Delete(ctx, staleKey)
}

// makeStaleKey 生成stale key（添加特殊前缀）
func (s *StaleCache) makeStaleKey(key string) string {
	return "stale:" + key
}

// RefreshStale 刷新stale数据（当成功从上游获取新数据时调用）
func (s *StaleCache) RefreshStale(ctx context.Context, key string, data []byte) error {
	return s.SetStale(ctx, key, data)
}

// GetWithStale 优先获取新鲜数据，失败时返回stale
func (s *StaleCache) GetWithStale(
	ctx context.Context,
	key string,
	loader func(ctx context.Context) ([]byte, error),
) ([]byte, bool, error) {
	// 尝试加载新数据
	data, err := loader(ctx)
	if err == nil {
		// 成功加载，更新stale
		s.RefreshStale(ctx, key, data)
		return data, false, nil
	}

	// 加载失败，尝试返回stale
	staleData, staleErr := s.GetStale(ctx, key)
	if staleErr != nil {
		return nil, false, fmt.Errorf("loader failed and no stale data: %w", err)
	}

	return staleData, true, nil
}

// GetStaleAge 获取stale数据的年龄（距离上次更新的时间）
func (s *StaleCache) GetStaleAge(ctx context.Context, key string) (time.Duration, error) {
	staleKey := s.makeStaleKey(key)
	ttl, err := s.redis.TTL(ctx, staleKey)
	if err != nil {
		return 0, err
	}

	// TTL返回剩余时间，我们需要计算已过时间
	age := s.staleTTL - ttl
	return age, nil
}
