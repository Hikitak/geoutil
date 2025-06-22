package geoutil

import (
	"sync"
)

// cacheItem - элемент кеша
type cacheItem struct {
	value  interface{}
	expiry time.Time
}

// Cache - потокобезопасный кеш с TTL
type Cache struct {
	items map[string]cacheItem
	mu    sync.RWMutex
	ttl   time.Duration
}

// NewCache создает новый кеш
func NewCache(ttl time.Duration) *Cache {
	c := &Cache{
		items: make(map[string]cacheItem),
		ttl:   ttl,
	}
	go c.cleanup()
	return c
}

// Set добавляет значение в кеш
func (c *Cache) Set(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[key] = cacheItem{
		value:  value,
		expiry: time.Now().Add(c.ttl),
	}
}

// Get получает значение из кеша
func (c *Cache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	item, found := c.items[key]
	if !found || time.Now().After(item.expiry) {
		return nil, false
	}
	return item.value, true
}

// Удаляет просроченные элементы
func (c *Cache) cleanup() {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	
	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for k, v := range c.items {
			if now.After(v.expiry) {
				delete(c.items, k)
			}
		}
		c.mu.Unlock()
	}
}b