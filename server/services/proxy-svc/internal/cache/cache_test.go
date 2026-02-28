package cache

import (
	"context"
	"testing"
	"time"
)

func TestMemoryCache(t *testing.T) {
	cache := NewMemoryCache(3, 1*time.Second)

	// Test Set and Get
	cache.Set("key1", []byte("value1"))
	cache.Set("key2", []byte("value2"))
	cache.Set("key3", []byte("value3"))

	if val, ok := cache.Get("key1"); !ok || string(val) != "value1" {
		t.Errorf("Expected value1, got %s", val)
	}

	// Test LRU eviction
	cache.Set("key4", []byte("value4")) // Should evict key2 (least recently used)

	if _, ok := cache.Get("key2"); ok {
		t.Error("key2 should have been evicted")
	}

	// Test expiration
	time.Sleep(1100 * time.Millisecond)
	if _, ok := cache.Get("key1"); ok {
		t.Error("key1 should have expired")
	}

	// Test stats
	stats := cache.Stats()
	if stats.Size > 3 {
		t.Errorf("Cache size should not exceed 3, got %d", stats.Size)
	}
}

func TestMemoryCacheLRU(t *testing.T) {
	cache := NewMemoryCache(3, 10*time.Second)

	cache.Set("a", []byte("1"))
	cache.Set("b", []byte("2"))
	cache.Set("c", []byte("3"))

	// Access "a" to make it most recently used
	cache.Get("a")

	// Add new item, should evict "b" (least recently used)
	cache.Set("d", []byte("4"))

	if _, ok := cache.Get("b"); ok {
		t.Error("b should have been evicted")
	}

	if _, ok := cache.Get("a"); !ok {
		t.Error("a should still exist")
	}
}

func TestMemoryCacheCleanExpired(t *testing.T) {
	cache := NewMemoryCache(10, 100*time.Millisecond)

	cache.Set("key1", []byte("value1"))
	cache.Set("key2", []byte("value2"))

	time.Sleep(150 * time.Millisecond)

	expired := cache.CleanExpired()
	if expired != 2 {
		t.Errorf("Expected 2 expired entries, got %d", expired)
	}

	stats := cache.Stats()
	if stats.Size != 0 {
		t.Errorf("Expected empty cache, got size %d", stats.Size)
	}
}

func TestSingleFlight(t *testing.T) {
	sf := NewSingleFlight()
	calls := 0

	// Concurrent calls with same key
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func() {
			_, err := sf.Do("test", func() (interface{}, error) {
				calls++
				time.Sleep(100 * time.Millisecond)
				return "result", nil
			})
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should only be called once despite 10 concurrent requests
	if calls != 1 {
		t.Errorf("Expected 1 call, got %d", calls)
	}
}

func BenchmarkMemoryCacheSet(b *testing.B) {
	cache := NewMemoryCache(1000, 5*time.Minute)
	data := []byte("test value")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := string(rune(i % 1000))
		cache.Set(key, data)
	}
}

func BenchmarkMemoryCacheGet(b *testing.B) {
	cache := NewMemoryCache(1000, 5*time.Minute)

	// Pre-populate
	for i := 0; i < 100; i++ {
		key := string(rune(i))
		cache.Set(key, []byte("value"))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := string(rune(i % 100))
		cache.Get(key)
	}
}

func ExampleCacheLayer() {
	ctx := context.Background()
	_ = ctx
	// Example usage of CacheLayer
	// In real usage, you would initialize with actual Redis client
}
