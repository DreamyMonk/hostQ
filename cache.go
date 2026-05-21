package main

import (
	"sync"
	"time"
)

// memCache is a tiny in-memory TTL cache used to coalesce expensive shell-out
// listings (sites, services, PHP, certs). It is intentionally minimal — no
// background eviction; entries are checked lazily on read.
type memCache struct {
	mu   sync.RWMutex
	data map[string]cacheEntry
}

type cacheEntry struct {
	value   interface{}
	expires time.Time
}

func newMemCache() *memCache { return &memCache{data: map[string]cacheEntry{}} }

func (c *memCache) get(key string) (interface{}, bool) {
	c.mu.RLock()
	e, ok := c.data[key]
	c.mu.RUnlock()
	if !ok || time.Now().After(e.expires) {
		return nil, false
	}
	return e.value, true
}

func (c *memCache) set(key string, value interface{}, ttl time.Duration) {
	c.mu.Lock()
	c.data[key] = cacheEntry{value: value, expires: time.Now().Add(ttl)}
	c.mu.Unlock()
}

func (c *memCache) invalidate(keys ...string) {
	c.mu.Lock()
	if len(keys) == 0 {
		c.data = map[string]cacheEntry{}
	} else {
		for _, k := range keys {
			delete(c.data, k)
		}
	}
	c.mu.Unlock()
}
