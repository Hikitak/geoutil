package geoutil

import (
    "sync"
    "time"
)

// Cache implements a thread-safe TTL cache
type Cache struct {
    items map[string]cacheItem
    mu    sync.RWMutex
    ttl   time.Duration
}

type cacheItem struct {
    value  interface{}
    expiry time.Time
}

// NewCache creates a new TTL-based cache
// ttl: Time-to-live duration for cached items
func NewCache(ttl time.Duration) *Cache {
    c := &Cache{
        items: make(map[string]cacheItem),
        ttl:   ttl,
    }
    go c.cleanup()
    return c
}

// Set adds an item to the cache
// key: Cache key identifier
// value: Value to cache
func (c *Cache) Set(key string, value interface{}) {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.items[key] = cacheItem{
        value:  value,
        expiry: time.Now().Add(c.ttl),
    }
}

// Get retrieves an item from cache
// key: Cache key identifier
// Returns: (value, exists) tuple
func (c *Cache) Get(key string) (interface{}, bool) {
    c.mu.RLock()
    defer c.mu.RUnlock()
    item, found := c.items[key]
    if !found || time.Now().After(item.expiry) {
        return nil, false
    }
    return item.value, true
}

// cleanup removes expired items periodically
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
}