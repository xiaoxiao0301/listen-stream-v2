package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"golang.org/x/sync/singleflight"
)

// SingleFlightCache provides cache with singleflight pattern to prevent cache stampede.
type SingleFlightCache struct {
	client *Client
	sf     singleflight.Group
}

// NewSingleFlightCache creates a new singleflight cache.
func NewSingleFlightCache(client *Client) *SingleFlightCache {
	return &SingleFlightCache{
		client: client,
	}
}

// Get retrieves a value from cache. If not found, calls the loader function.
// Multiple concurrent calls for the same key will result in only one loader call.
func (c *SingleFlightCache) Get(ctx context.Context, key string, loader func() (interface{}, error), ttl time.Duration) (interface{}, error) {
	// Try to get from cache first
	val, err := c.client.Get(ctx, key)
	if err == nil {
		// Cache hit - unmarshal and return
		var result interface{}
		if err := json.Unmarshal([]byte(val), &result); err != nil {
			return nil, fmt.Errorf("failed to unmarshal cached value: %w", err)
		}
		return result, nil
	}
	
	if err != ErrKeyNotFound {
		// Real error, not just cache miss
		return nil, fmt.Errorf("cache lookup error: %w", err)
	}
	
	// Cache miss - use singleflight to load
	result, err, _ := c.sf.Do(key, func() (interface{}, error) {
		// Load the value
		data, err := loader()
		if err != nil {
			return nil, fmt.Errorf("loader error: %w", err)
		}
		
		// Marshal and store in cache
		jsonData, err := json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal data: %w", err)
		}
		
		if err := c.client.Set(ctx, key, jsonData, ttl); err != nil {
			// Log error but return the data anyway
			fmt.Printf("Warning: failed to cache data for key %s: %v\n", key, err)
		}
		
		return data, nil
	})
	
	return result, err
}

// GetString is like Get but for string values.
func (c *SingleFlightCache) GetString(ctx context.Context, key string, loader func() (string, error), ttl time.Duration) (string, error) {
	// Try to get from cache first
	val, err := c.client.Get(ctx, key)
	if err == nil {
		return val, nil
	}
	
	if err != ErrKeyNotFound {
		return "", fmt.Errorf("cache lookup error: %w", err)
	}
	
	// Cache miss - use singleflight to load
	result, err, _ := c.sf.Do(key, func() (interface{}, error) {
		// Load the value
		data, err := loader()
		if err != nil {
			return "", fmt.Errorf("loader error: %w", err)
		}
		
		// Store in cache
		if err := c.client.Set(ctx, key, data, ttl); err != nil {
			fmt.Printf("Warning: failed to cache data for key %s: %v\n", key, err)
		}
		
		return data, nil
	})
	
	if err != nil {
		return "", err
	}
	
	return result.(string), nil
}

// GetBytes is like Get but for byte slice values.
func (c *SingleFlightCache) GetBytes(ctx context.Context, key string, loader func() ([]byte, error), ttl time.Duration) ([]byte, error) {
	// Try to get from cache first
	val, err := c.client.Get(ctx, key)
	if err == nil {
		return []byte(val), nil
	}
	
	if err != ErrKeyNotFound {
		return nil, fmt.Errorf("cache lookup error: %w", err)
	}
	
	// Cache miss - use singleflight to load
	result, err, _ := c.sf.Do(key, func() (interface{}, error) {
		// Load the value
		data, err := loader()
		if err != nil {
			return nil, fmt.Errorf("loader error: %w", err)
		}
		
		// Store in cache
		if err := c.client.Set(ctx, key, data, ttl); err != nil {
			fmt.Printf("Warning: failed to cache data for key %s: %v\n", key, err)
		}
		
		return data, nil
	})
	
	if err != nil {
		return nil, err
	}
	
	return result.([]byte), nil
}

// Delete deletes a key from cache.
func (c *SingleFlightCache) Delete(ctx context.Context, key string) error {
	return c.client.Delete(ctx, key)
}

// Invalidate invalidates cache for a key.
// This is an alias for Delete for semantic clarity.
func (c *SingleFlightCache) Invalidate(ctx context.Context, key string) error {
	return c.Delete(ctx, key)
}

// Forget tells singleflight to forget a key so that subsequent calls will execute the loader again.
func (c *SingleFlightCache) Forget(key string) {
	c.sf.Forget(key)
}

// CacheWithFallback provides a cache with fallback to stale data.
type CacheWithFallback struct {
	client *Client
	sf     singleflight.Group
}

// NewCacheWithFallback creates a new cache with fallback.
func NewCacheWithFallback(client *Client) *CacheWithFallback {
	return &CacheWithFallback{
		client: client,
	}
}

// Get retrieves a value from cache with fallback to stale data.
// 
// It implements the following strategy:
// 1. Try L2 cache (Redis)
// 2. If miss, use singleflight to load from source
// 3. If source fails, try L3 stale cache
// 4. Store fresh data in both L2 and L3
func (c *CacheWithFallback) Get(ctx context.Context, key string, loader func() (interface{}, error), ttl time.Duration, staleTTL time.Duration) (interface{}, error) {
	// Try L2 cache (Redis)
	val, err := c.client.Get(ctx, CacheKey("l2", key))
	if err == nil {
		var result interface{}
		if err := json.Unmarshal([]byte(val), &result); err == nil {
			return result, nil
		}
	}
	
	// L2 miss - use singleflight to load
	result, err, _ := c.sf.Do(key, func() (interface{}, error) {
		// Try to load from source
		data, loadErr := loader()
		if loadErr != nil {
			// Source failed - try L3 stale cache
			staleKey := StaleCacheKey("l3", key)
			staleVal, staleErr := c.client.Get(ctx, staleKey)
			if staleErr == nil {
				var staleResult interface{}
				if unmarshalErr := json.Unmarshal([]byte(staleVal), &staleResult); unmarshalErr == nil {
					// Return stale data
					fmt.Printf("Using stale cache for key %s\n", key)
					return staleResult, nil
				}
			}
			
			// No stale data available
			return nil, fmt.Errorf("loader failed and no stale data available: %w", loadErr)
		}
		
		// Success - marshal data
		jsonData, err := json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal data: %w", err)
		}
		
		// Store in L2 cache
		l2Key := CacheKey("l2", key)
		if err := c.client.Set(ctx, l2Key, jsonData, ttl); err != nil {
			fmt.Printf("Warning: failed to cache data in L2: %v\n", err)
		}
		
		// Store in L3 stale cache (longer TTL)
		l3Key := StaleCacheKey("l3", key)
		if err := c.client.Set(ctx, l3Key, jsonData, staleTTL); err != nil {
			fmt.Printf("Warning: failed to cache data in L3: %v\n", err)
		}
		
		return data, nil
	})
	
	return result, err
}

// Invalidate invalidates both L2 and L3 caches.
func (c *CacheWithFallback) Invalidate(ctx context.Context, key string) error {
	l2Key := CacheKey("l2", key)
	l3Key := StaleCacheKey("l3", key)
	
	if err := c.client.Delete(ctx, l2Key, l3Key); err != nil {
		return fmt.Errorf("failed to invalidate cache: %w", err)
	}
	
	return nil
}

// Forget tells singleflight to forget a key.
func (c *CacheWithFallback) Forget(key string) {
	c.sf.Forget(key)
}
