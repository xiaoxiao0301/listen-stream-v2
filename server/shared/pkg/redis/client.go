// Package redis provides Redis client utilities with support for single instance and cluster modes.
package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Config holds Redis client configuration.
type Config struct {
	// Single instance or sentinel mode
	Host     string
	Port     int
	Password string
	DB       int
	
	// Cluster mode
	Cluster      bool
	ClusterAddrs []string
	
	// Connection pool
	PoolSize     int
	MinIdleConns int
	
	// Timeouts
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	PoolTimeout  time.Duration
	
	// Retry
	MaxRetries int
}

// Client wraps a Redis client (single instance or cluster).
type Client struct {
	universal redis.UniversalClient
	config    *Config
}

// ErrKeyNotFound is returned when a key doesn't exist.
var ErrKeyNotFound = fmt.Errorf("key not found")

// NewClient creates a new Redis client.
func NewClient(cfg *Config) (*Client, error) {
	if cfg.Cluster {
		return newClusterClient(cfg)
	}
	return newSingleClient(cfg)
}

// newSingleClient creates a single instance Redis client.
func newSingleClient(cfg *Config) (*Client, error) {
	opts := &redis.Options{
		Addr:         fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
		DialTimeout:  cfg.DialTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		PoolTimeout:  cfg.PoolTimeout,
		MaxRetries:   cfg.MaxRetries,
	}
	
	rdb := redis.NewClient(opts)
	
	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to ping redis: %w", err)
	}
	
	return &Client{
		universal: rdb,
		config:    cfg,
	}, nil
}

// newClusterClient creates a Redis cluster client.
func newClusterClient(cfg *Config) (*Client, error) {
	opts := &redis.ClusterOptions{
		Addrs:        cfg.ClusterAddrs,
		Password:     cfg.Password,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
		DialTimeout:  cfg.DialTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		PoolTimeout:  cfg.PoolTimeout,
		MaxRetries:   cfg.MaxRetries,
	}
	
	rdb := redis.NewClusterClient(opts)
	
	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to ping redis cluster: %w", err)
	}
	
	return &Client{
		universal: rdb,
		config:    cfg,
	}, nil
}

// Universal returns the underlying UniversalClient.
// This can be used for operations not wrapped by this package.
func (c *Client) Universal() redis.UniversalClient {
	return c.universal
}

// Get retrieves a value from Redis.
func (c *Client) Get(ctx context.Context, key string) (string, error) {
	val, err := c.universal.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", ErrKeyNotFound
	}
	if err != nil {
		return "", fmt.Errorf("failed to get key: %w", err)
	}
	return val, nil
}

// Set stores a value in Redis with an optional expiration.
func (c *Client) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	if err := c.universal.Set(ctx, key, value, expiration).Err(); err != nil {
		return fmt.Errorf("failed to set key: %w", err)
	}
	return nil
}

// SetNX sets a value only if the key does not exist.
func (c *Client) SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error) {
	ok, err := c.universal.SetNX(ctx, key, value, expiration).Result()
	if err != nil {
		return false, fmt.Errorf("failed to setnx: %w", err)
	}
	return ok, nil
}

// Delete deletes one or more keys.
func (c *Client) Delete(ctx context.Context, keys ...string) error {
	if err := c.universal.Del(ctx, keys...).Err(); err != nil {
		return fmt.Errorf("failed to delete keys: %w", err)
	}
	return nil
}

// Exists checks if a key exists.
func (c *Client) Exists(ctx context.Context, key string) (bool, error) {
	n, err := c.universal.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check existence: %w", err)
	}
	return n > 0, nil
}

// Expire sets an expiration on a key.
func (c *Client) Expire(ctx context.Context, key string, expiration time.Duration) error {
	if err := c.universal.Expire(ctx, key, expiration).Err(); err != nil {
		return fmt.Errorf("failed to set expiration: %w", err)
	}
	return nil
}

// TTL returns the remaining time to live of a key.
func (c *Client) TTL(ctx context.Context, key string) (time.Duration, error) {
	ttl, err := c.universal.TTL(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to get ttl: %w", err)
	}
	return ttl, nil
}

// Incr increments the integer value of a key by one.
func (c *Client) Incr(ctx context.Context, key string) (int64, error) {
	val, err := c.universal.Incr(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to increment: %w", err)
	}
	return val, nil
}

// IncrBy increments the integer value of a key by the given amount.
func (c *Client) IncrBy(ctx context.Context, key string, value int64) (int64, error) {
	val, err := c.universal.IncrBy(ctx, key, value).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to increment by: %w", err)
	}
	return val, nil
}

// Decr decrements the integer value of a key by one.
func (c *Client) Decr(ctx context.Context, key string) (int64, error) {
	val, err := c.universal.Decr(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to decrement: %w", err)
	}
	return val, nil
}

// HSet sets a field in a hash.
func (c *Client) HSet(ctx context.Context, key string, values ...interface{}) error {
	if err := c.universal.HSet(ctx, key, values...).Err(); err != nil {
		return fmt.Errorf("failed to hset: %w", err)
	}
	return nil
}

// HGet retrieves a field from a hash.
func (c *Client) HGet(ctx context.Context, key, field string) (string, error) {
	val, err := c.universal.HGet(ctx, key, field).Result()
	if err == redis.Nil {
		return "", ErrKeyNotFound
	}
	if err != nil {
		return "", fmt.Errorf("failed to hget: %w", err)
	}
	return val, nil
}

// HGetAll retrieves all fields from a hash.
func (c *Client) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	val, err := c.universal.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to hgetall: %w", err)
	}
	return val, nil
}

// HDel deletes one or more fields from a hash.
func (c *Client) HDel(ctx context.Context, key string, fields ...string) error {
	if err := c.universal.HDel(ctx, key, fields...).Err(); err != nil {
		return fmt.Errorf("failed to hdel: %w", err)
	}
	return nil
}

// LPush prepends one or more values to a list.
func (c *Client) LPush(ctx context.Context, key string, values ...interface{}) error {
	if err := c.universal.LPush(ctx, key, values...).Err(); err != nil {
		return fmt.Errorf("failed to lpush: %w", err)
	}
	return nil
}

// RPush appends one or more values to a list.
func (c *Client) RPush(ctx context.Context, key string, values ...interface{}) error {
	if err := c.universal.RPush(ctx, key, values...).Err(); err != nil {
		return fmt.Errorf("failed to rpush: %w", err)
	}
	return nil
}

// LPop removes and returns the first element of a list.
func (c *Client) LPop(ctx context.Context, key string) (string, error) {
	val, err := c.universal.LPop(ctx, key).Result()
	if err == redis.Nil {
		return "", ErrKeyNotFound
	}
	if err != nil {
		return "", fmt.Errorf("failed to lpop: %w", err)
	}
	return val, nil
}

// RPop removes and returns the last element of a list.
func (c *Client) RPop(ctx context.Context, key string) (string, error) {
	val, err := c.universal.RPop(ctx, key).Result()
	if err == redis.Nil {
		return "", ErrKeyNotFound
	}
	if err != nil {
		return "", fmt.Errorf("failed to rpop: %w", err)
	}
	return val, nil
}

// LRange retrieves a range of elements from a list.
func (c *Client) LRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	val, err := c.universal.LRange(ctx, key, start, stop).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to lrange: %w", err)
	}
	return val, nil
}

// LLen returns the length of a list.
func (c *Client) LLen(ctx context.Context, key string) (int64, error) {
	val, err := c.universal.LLen(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to llen: %w", err)
	}
	return val, nil
}

// SAdd adds one or more members to a set.
func (c *Client) SAdd(ctx context.Context, key string, members ...interface{}) error {
	if err := c.universal.SAdd(ctx, key, members...).Err(); err != nil {
		return fmt.Errorf("failed to sadd: %w", err)
	}
	return nil
}

// SRem removes one or more members from a set.
func (c *Client) SRem(ctx context.Context, key string, members ...interface{}) error {
	if err := c.universal.SRem(ctx, key, members...).Err(); err != nil {
		return fmt.Errorf("failed to srem: %w", err)
	}
	return nil
}

// SMembers retrieves all members of a set.
func (c *Client) SMembers(ctx context.Context, key string) ([]string, error) {
	val, err := c.universal.SMembers(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to smembers: %w", err)
	}
	return val, nil
}

// SIsMember checks if a value is a member of a set.
func (c *Client) SIsMember(ctx context.Context, key string, member interface{}) (bool, error) {
	ok, err := c.universal.SIsMember(ctx, key, member).Result()
	if err != nil {
		return false, fmt.Errorf("failed to sismember: %w", err)
	}
	return ok, nil
}

// ZAdd adds one or more members to a sorted set.
func (c *Client) ZAdd(ctx context.Context, key string, members ...redis.Z) error {
	if err := c.universal.ZAdd(ctx, key, members...).Err(); err != nil {
		return fmt.Errorf("failed to zadd: %w", err)
	}
	return nil
}

// ZRange retrieves a range of members from a sorted set by index.
func (c *Client) ZRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	val, err := c.universal.ZRange(ctx, key, start, stop).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to zrange: %w", err)
	}
	return val, nil
}

// ZRangeWithScores retrieves a range of members with scores from a sorted set.
func (c *Client) ZRangeWithScores(ctx context.Context, key string, start, stop int64) ([]redis.Z, error) {
	val, err := c.universal.ZRangeWithScores(ctx, key, start, stop).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to zrange with scores: %w", err)
	}
	return val, nil
}

// ZRem removes one or more members from a sorted set.
func (c *Client) ZRem(ctx context.Context, key string, members ...interface{}) error {
	if err := c.universal.ZRem(ctx, key, members...).Err(); err != nil {
		return fmt.Errorf("failed to zrem: %w", err)
	}
	return nil
}

// Pipeline creates a new pipeline for batching commands.
func (c *Client) Pipeline() redis.Pipeliner {
	return c.universal.Pipeline()
}

// TxPipeline creates a new transaction pipeline.
func (c *Client) TxPipeline() redis.Pipeliner {
	return c.universal.TxPipeline()
}

// Ping checks if the Redis server is reachable.
func (c *Client) Ping(ctx context.Context) error {
	if err := c.universal.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("ping failed: %w", err)
	}
	return nil
}

// Close closes the Redis client.
func (c *Client) Close() error {
	if err := c.universal.Close(); err != nil {
		return fmt.Errorf("failed to close client: %w", err)
	}
	return nil
}

// PoolStats returns connection pool statistics.
func (c *Client) PoolStats() *redis.PoolStats {
	return c.universal.PoolStats()
}