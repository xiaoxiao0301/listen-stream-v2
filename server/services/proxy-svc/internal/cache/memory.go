package cache

import (
	"container/list"
	"sync"
	"time"
)

// MemoryCache L1内存缓存（LRU策略）
type MemoryCache struct {
	maxSize int
	ttl     time.Duration
	mu      sync.RWMutex
	cache   map[string]*list.Element
	lru     *list.List

	// 统计
	hits   uint64
	misses uint64
}

// cacheEntry LRU缓存条目
type cacheEntry struct {
	key       string
	value     []byte
	expiresAt time.Time
}

// NewMemoryCache 创建内存缓存
func NewMemoryCache(maxSize int, ttl time.Duration) *MemoryCache {
	return &MemoryCache{
		maxSize: maxSize,
		ttl:     ttl,
		cache:   make(map[string]*list.Element),
		lru:     list.New(),
	}
}

// Get 获取缓存
func (m *MemoryCache) Get(key string) ([]byte, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	elem, exists := m.cache[key]
	if !exists {
		m.misses++
		return nil, false
	}

	entry := elem.Value.(*cacheEntry)

	// 检查过期
	if time.Now().After(entry.expiresAt) {
		m.lru.Remove(elem)
		delete(m.cache, key)
		m.misses++
		return nil, false
	}

	// LRU：移动到链表头部
	m.lru.MoveToFront(elem)
	m.hits++

	return entry.value, true
}

// Set 设置缓存
func (m *MemoryCache) Set(key string, value []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 如果已存在，更新并移到前面
	if elem, exists := m.cache[key]; exists {
		entry := elem.Value.(*cacheEntry)
		entry.value = value
		entry.expiresAt = time.Now().Add(m.ttl)
		m.lru.MoveToFront(elem)
		return
	}

	// 新增条目
	entry := &cacheEntry{
		key:       key,
		value:     value,
		expiresAt: time.Now().Add(m.ttl),
	}

	elem := m.lru.PushFront(entry)
	m.cache[key] = elem

	// 超出容量，删除最久未使用的
	if m.lru.Len() > m.maxSize {
		oldest := m.lru.Back()
		if oldest != nil {
			m.lru.Remove(oldest)
			oldEntry := oldest.Value.(*cacheEntry)
			delete(m.cache, oldEntry.key)
		}
	}
}

// Delete 删除缓存
func (m *MemoryCache) Delete(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if elem, exists := m.cache[key]; exists {
		m.lru.Remove(elem)
		delete(m.cache, key)
	}
}

// Clear 清空缓存
func (m *MemoryCache) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.cache = make(map[string]*list.Element)
	m.lru = list.New()
	m.hits = 0
	m.misses = 0
}

// Stats 获取统计信息
func (m *MemoryCache) Stats() MemoryCacheStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	total := m.hits + m.misses
	hitRate := 0.0
	if total > 0 {
		hitRate = float64(m.hits) / float64(total)
	}

	return MemoryCacheStats{
		Size:    m.lru.Len(),
		MaxSize: m.maxSize,
		Hits:    m.hits,
		Misses:  m.misses,
		HitRate: hitRate,
	}
}

// MemoryCacheStats 内存缓存统计
type MemoryCacheStats struct {
	Size    int
	MaxSize int
	Hits    uint64
	Misses  uint64
	HitRate float64
}

// CleanExpired 清理过期条目（定期调用）
func (m *MemoryCache) CleanExpired() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	expired := 0

	// 遍历删除过期条目
	for elem := m.lru.Back(); elem != nil; {
		entry := elem.Value.(*cacheEntry)
		if now.After(entry.expiresAt) {
			prev := elem.Prev()
			m.lru.Remove(elem)
			delete(m.cache, entry.key)
			expired++
			elem = prev
		} else {
			break // LRU链表，后面的都未过期
		}
	}

	return expired
}
