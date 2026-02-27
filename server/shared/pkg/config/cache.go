package config

import (
	"sync"
	"time"
)

// Cache provides a simple in-memory cache with TTL.
type Cache struct {
	mu      sync.RWMutex
	items   map[string]*cacheItem
	ttl     time.Duration
	cleaner *time.Ticker
	done    chan struct{}
}

type cacheItem struct {
	value      interface{}
	expiration time.Time
}

// NewCache creates a new cache with the specified TTL.
func NewCache(ttl time.Duration) *Cache {
	c := &Cache{
		items: make(map[string]*cacheItem),
		ttl:   ttl,
		done:  make(chan struct{}),
	}
	
	// Start cleanup goroutine
	c.cleaner = time.NewTicker(ttl)
	go c.cleanup()
	
	return c
}

// Get retrieves a value from the cache.
func (c *Cache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	item, found := c.items[key]
	if !found {
		return nil, false
	}
	
	// Check if expired
	if time.Now().After(item.expiration) {
		return nil, false
	}
	
	return item.value, true
}

// Set stores a value in the cache.
func (c *Cache) Set(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.items[key] = &cacheItem{
		value:      value,
		expiration: time.Now().Add(c.ttl),
	}
}

// Delete removes a value from the cache.
func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	delete(c.items, key)
}

// Clear removes all items from the cache.
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.items = make(map[string]*cacheItem)
}

// Size returns the number of items in the cache.
func (c *Cache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	return len(c.items)
}

// cleanup removes expired items periodically.
func (c *Cache) cleanup() {
	for {
		select {
		case <-c.cleaner.C:
			c.removeExpired()
		case <-c.done:
			c.cleaner.Stop()
			return
		}
	}
}

// removeExpired removes all expired items from the cache.
func (c *Cache) removeExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	now := time.Now()
	for key, item := range c.items {
		if now.After(item.expiration) {
			delete(c.items, key)
		}
	}
}

// Close stops the cleanup goroutine.
func (c *Cache) Close() {
	close(c.done)
}
