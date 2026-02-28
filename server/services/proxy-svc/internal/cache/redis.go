package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisCache L2 Redis缓存
type RedisCache struct {
	client *redis.Client
	prefix string // key前缀
}

// NewRedisCache 创建Redis缓存
func NewRedisCache(client *redis.Client, prefix string) *RedisCache {
	return &RedisCache{
		client: client,
		prefix: prefix,
	}
}

// Get 获取缓存
func (r *RedisCache) Get(ctx context.Context, key string) ([]byte, error) {
	fullKey := r.makeKey(key)
	data, err := r.client.Get(ctx, fullKey).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, ErrCacheMiss
		}
		return nil, fmt.Errorf("redis get failed: %w", err)
	}
	return data, nil
}

// Set 设置缓存
func (r *RedisCache) Set(ctx context.Context, key string, data []byte, ttl time.Duration) error {
	fullKey := r.makeKey(key)
	err := r.client.Set(ctx, fullKey, data, ttl).Err()
	if err != nil {
		return fmt.Errorf("redis set failed: %w", err)
	}
	return nil
}

// Delete 删除缓存
func (r *RedisCache) Delete(ctx context.Context, key string) error {
	fullKey := r.makeKey(key)
	err := r.client.Del(ctx, fullKey).Err()
	if err != nil {
		return fmt.Errorf("redis delete failed: %w", err)
	}
	return nil
}

// MGet 批量获取
func (r *RedisCache) MGet(ctx context.Context, keys []string) (map[string][]byte, error) {
	if len(keys) == 0 {
		return make(map[string][]byte), nil
	}

	// 构建完整key列表
	fullKeys := make([]string, len(keys))
	for i, key := range keys {
		fullKeys[i] = r.makeKey(key)
	}

	// 批量获取
	vals, err := r.client.MGet(ctx, fullKeys...).Result()
	if err != nil {
		return nil, fmt.Errorf("redis mget failed: %w", err)
	}

	// 构建结果map
	result := make(map[string][]byte)
	for i, val := range vals {
		if val != nil {
			if str, ok := val.(string); ok {
				result[keys[i]] = []byte(str)
			}
		}
	}

	return result, nil
}

// MSet 批量设置
func (r *RedisCache) MSet(ctx context.Context, entries map[string][]byte, ttl time.Duration) error {
	if len(entries) == 0 {
		return nil
	}

	pipe := r.client.Pipeline()
	for key, data := range entries {
		fullKey := r.makeKey(key)
		pipe.Set(ctx, fullKey, data, ttl)
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("redis mset failed: %w", err)
	}

	return nil
}

// Exists 检查key是否存在
func (r *RedisCache) Exists(ctx context.Context, key string) (bool, error) {
	fullKey := r.makeKey(key)
	n, err := r.client.Exists(ctx, fullKey).Result()
	if err != nil {
		return false, fmt.Errorf("redis exists failed: %w", err)
	}
	return n > 0, nil
}

// TTL 获取key的剩余过期时间
func (r *RedisCache) TTL(ctx context.Context, key string) (time.Duration, error) {
	fullKey := r.makeKey(key)
	ttl, err := r.client.TTL(ctx, fullKey).Result()
	if err != nil {
		return 0, fmt.Errorf("redis ttl failed: %w", err)
	}
	return ttl, nil
}

// makeKey 生成完整的key（带前缀）
func (r *RedisCache) makeKey(key string) string {
	if r.prefix == "" {
		return key
	}
	return r.prefix + ":" + key
}

// Ping 健康检查
func (r *RedisCache) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}
