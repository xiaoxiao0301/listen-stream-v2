package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/xiaoxiao0301/listen-stream-v2/server/shared/pkg/logger"
)

// CacheLayer 三级缓存分层
// L1: 内存缓存（热点数据，5分钟，最多1000条）
// L2: Redis缓存（常规缓存）
// L3: Stale缓存（降级缓存，允许返回过期数据）
type CacheLayer struct {
	l1     *MemoryCache
	l2     *RedisCache
	l3     *StaleCache
	sf     *SingleFlight
	logger logger.Logger
}

// CacheConfig 缓存配置
type CacheConfig struct {
	// L1配置
	L1MaxSize int           // L1最大条目数
	L1TTL     time.Duration // L1过期时间

	// L2配置
	L2TTL time.Duration // L2过期时间

	// L3配置
	L3TTL       time.Duration // L3 stale数据保留时间
	L3Threshold time.Duration // L3降级阈值（上游响应超过此时间则返回stale）
}

// DefaultCacheConfig 默认缓存配置
func DefaultCacheConfig() *CacheConfig {
	return &CacheConfig{
		L1MaxSize:   1000,
		L1TTL:       5 * time.Minute,
		L2TTL:       30 * time.Minute,
		L3TTL:       24 * time.Hour,
		L3Threshold: 2 * time.Second,
	}
}

// NewCacheLayer 创建三级缓存
func NewCacheLayer(redisCache *RedisCache, config *CacheConfig, log logger.Logger) *CacheLayer {
	if config == nil {
		config = DefaultCacheConfig()
	}

	l1 := NewMemoryCache(config.L1MaxSize, config.L1TTL)
	l3 := NewStaleCache(redisCache, config.L3TTL)

	return &CacheLayer{
		l1:     l1,
		l2:     redisCache,
		l3:     l3,
		sf:     NewSingleFlight(),
		logger: log,
	}
}

// Get 从缓存中获取数据，依次查询 L1 -> L2 -> L3
func (c *CacheLayer) Get(ctx context.Context, key string) ([]byte, error) {
	// L1: 内存缓存
	if data, ok := c.l1.Get(key); ok {
		c.logger.Debug("Cache hit L1", logger.String("key", key))
		return data, nil
	}

	// L2: Redis缓存
	data, err := c.l2.Get(ctx, key)
	if err == nil && data != nil {
		c.logger.Debug("Cache hit L2", logger.String("key", key))
		// 回填L1
		c.l1.Set(key, data)
		return data, nil
	}

	// L3: Stale降级缓存（仅在上游不可用时）
	// 这里不直接返回stale，而是让调用方在上游失败时主动获取
	return nil, ErrCacheMiss
}

// GetWithFallback 带降级的获取：缓存未命中时调用loader，失败时返回stale
func (c *CacheLayer) GetWithFallback(
	ctx context.Context,
	key string,
	loader func(ctx context.Context) ([]byte, error),
	ttl time.Duration,
) ([]byte, error) {
	// 先尝试从L1/L2获取
	data, err := c.Get(ctx, key)
	if err == nil {
		return data, nil
	}

	// 使用SingleFlight防止缓存击穿
	result, err := c.sf.Do(key, func() (interface{}, error) {
		// 再次检查缓存（可能其他协程已填充）
		if data, err := c.Get(ctx, key); err == nil {
			return data, nil
		}

		// 调用loader加载数据
		data, err := loader(ctx)
		if err != nil {
			c.logger.Warn("Loader failed, trying stale cache", logger.String("key", key), logger.Error(err))
			// 上游失败，尝试返回L3 stale数据
			staleData, staleErr := c.l3.GetStale(ctx, key)
			if staleErr == nil && staleData != nil {
				c.logger.Info("Cache hit L3 (stale)", logger.String("key", key))
				return staleData, nil
			}
			return nil, fmt.Errorf("loader failed and no stale data: %w", err)
		}

		// 成功加载，写入所有层级
		if err := c.Set(ctx, key, data, ttl); err != nil {
			c.logger.Error("Failed to set cache", logger.String("key", key), logger.Error(err))
		}

		return data, nil
	})

	if err != nil {
		return nil, err
	}

	return result.([]byte), nil
}

// Set 写入所有缓存层级
func (c *CacheLayer) Set(ctx context.Context, key string, data []byte, ttl time.Duration) error {
	// L1: 内存缓存
	c.l1.Set(key, data)

	// L2: Redis缓存（带TTL）
	if err := c.l2.Set(ctx, key, data, ttl); err != nil {
		c.logger.Error("Failed to set L2 cache", logger.String("key", key), logger.Error(err))
		return err
	}

	// L3: Stale缓存（长时间保留，用于降级）
	if err := c.l3.SetStale(ctx, key, data); err != nil {
		c.logger.Warn("Failed to set L3 stale cache", logger.String("key", key), logger.Error(err))
		// L3失败不影响主流程
	}

	return nil
}

// Delete 删除所有层级的缓存
func (c *CacheLayer) Delete(ctx context.Context, key string) error {
	// L1
	c.l1.Delete(key)

	// L2
	if err := c.l2.Delete(ctx, key); err != nil {
		c.logger.Error("Failed to delete L2 cache", logger.String("key", key), logger.Error(err))
	}

	// L3
	if err := c.l3.DeleteStale(ctx, key); err != nil {
		c.logger.Warn("Failed to delete L3 stale cache", logger.String("key", key), logger.Error(err))
	}

	return nil
}

// WarmUp 预热热点数据
func (c *CacheLayer) WarmUp(ctx context.Context, entries map[string][]byte, ttl time.Duration) error {
	for key, data := range entries {
		if err := c.Set(ctx, key, data, ttl); err != nil {
			c.logger.Error("Failed to warm up cache", logger.String("key", key), logger.Error(err))
			return err
		}
	}
	c.logger.Info("Cache warm up completed", logger.Int("count", len(entries)))
	return nil
}

// Stats 获取缓存统计
func (c *CacheLayer) Stats() CacheStats {
	return CacheStats{
		L1Stats: c.l1.Stats(),
	}
}

// CacheStats 缓存统计
type CacheStats struct {
	L1Stats MemoryCacheStats
}

// CleanExpired 清理L1过期条目
func (c *CacheLayer) CleanExpired() int {
	return c.l1.CleanExpired()
}
