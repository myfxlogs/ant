package marketplace

import (
	"fmt"
	"sync"
	"time"
)

// ── Published listing cache ─────────────────────────────────────────────────

const publishedCacheTTL = 60 * time.Second

type publishedCacheEntry struct {
	data      []PublishedStrategy
	total     int
	expiresAt time.Time
}

type publishedCache struct {
	mu sync.RWMutex
	m  map[string]publishedCacheEntry
}

func newPublishedCache() *publishedCache {
	return &publishedCache{m: make(map[string]publishedCacheEntry)}
}

func (c *publishedCache) key(userID, assetClass, keyword, sortBy, priceFilter string, limit, offset int) string {
	return fmt.Sprintf("%s|%s|%s|%s|%s|%d|%d", userID, assetClass, keyword, sortBy, priceFilter, limit, offset)
}

func (c *publishedCache) get(key string) ([]PublishedStrategy, int, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	e, ok := c.m[key]
	if !ok || time.Now().After(e.expiresAt) {
		return nil, 0, false
	}
	return e.data, e.total, true
}

func (c *publishedCache) clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	for k := range c.m {
		delete(c.m, k)
	}
}

func (c *publishedCache) set(key string, data []PublishedStrategy, total int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.m) > 256 {
		for k, v := range c.m {
			if time.Now().After(v.expiresAt) {
				delete(c.m, k)
			}
		}
	}
	c.m[key] = publishedCacheEntry{data: data, total: total, expiresAt: time.Now().Add(publishedCacheTTL)}
}
