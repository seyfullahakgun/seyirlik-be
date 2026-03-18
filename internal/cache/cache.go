package cache

import (
	"sync"
	"sync/atomic"
	"time"
)

// CacheItem tek bir cache kaydını temsil eder
type CacheItem struct {
	Value      any
	Expiration int64
}

// Stats cache istatistiklerini tutar
type Stats struct {
	Hits   uint64 `json:"hits"`
	Misses uint64 `json:"misses"`
	Items  int    `json:"items"`
}

// Cache thread-safe in-memory cache
type Cache struct {
	items  map[string]CacheItem
	mu     sync.RWMutex
	ttl    time.Duration
	hits   uint64
	misses uint64
}

// New yeni bir cache oluşturur
func New(ttl time.Duration) *Cache {
	c := &Cache{
		items: make(map[string]CacheItem),
		ttl:   ttl,
	}

	// Arka planda expired item'ları temizle
	go c.cleanupLoop()

	return c
}

// Set cache'e değer ekler
func (c *Cache) Set(key string, value any) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[key] = CacheItem{
		Value:      value,
		Expiration: time.Now().Add(c.ttl).UnixNano(),
	}
}

// Get cache'den değer okur
func (c *Cache) Get(key string) (any, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, found := c.items[key]
	if !found {
		atomic.AddUint64(&c.misses, 1)
		return nil, false
	}

	// Expire olmuş mu kontrol et
	if time.Now().UnixNano() > item.Expiration {
		atomic.AddUint64(&c.misses, 1)
		return nil, false
	}

	atomic.AddUint64(&c.hits, 1)
	return item.Value, true
}

// Delete cache'den değer siler
func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, key)
}

// cleanupLoop expired item'ları periyodik olarak temizler
func (c *Cache) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.cleanup()
	}
}

// cleanup expired item'ları temizler
func (c *Cache) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now().UnixNano()
	for key, item := range c.items {
		if now > item.Expiration {
			delete(c.items, key)
		}
	}
}

// GetStats cache istatistiklerini döner
func (c *Cache) GetStats() Stats {
	c.mu.RLock()
	itemCount := len(c.items)
	c.mu.RUnlock()

	return Stats{
		Hits:   atomic.LoadUint64(&c.hits),
		Misses: atomic.LoadUint64(&c.misses),
		Items:  itemCount,
	}
}
