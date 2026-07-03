package agent

import (
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"
)

// LLCache caches LLM outputs by source hash + prompt hash to avoid
// redundant LLM calls during Agent iteration loops.
// ADR-0024 §5.3: "按 source hash + prompt hash 缓存，避免重复调用"
type LLCache struct {
	mu      sync.RWMutex
	entries map[string]cacheEntry
	ttl     time.Duration
}

type cacheEntry struct {
	value     string
	expiresAt time.Time
}

// NewLLCache creates an LLM output cache with the given TTL.
func NewLLCache(ttl time.Duration) *LLCache {
	return &LLCache{
		entries: make(map[string]cacheEntry),
		ttl:     ttl,
	}
}

// Get returns the cached LLM response for the given source + prompt, if present and not expired.
func (c *LLCache) Get(sourceCode, prompt string) (string, bool) {
	key := cacheKey(sourceCode, prompt)
	c.mu.RLock()
	entry, ok := c.entries[key]
	c.mu.RUnlock()
	if !ok || time.Now().After(entry.expiresAt) {
		return "", false
	}
	return entry.value, true
}

// Set stores an LLM response for the given source + prompt.
func (c *LLCache) Set(sourceCode, prompt, value string) {
	key := cacheKey(sourceCode, prompt)
	c.mu.Lock()
	c.entries[key] = cacheEntry{
		value:     value,
		expiresAt: time.Now().Add(c.ttl),
	}
	c.mu.Unlock()
}

// cacheKey computes a SHA-256 hash of sourceCode + prompt as the cache key.
func cacheKey(sourceCode, prompt string) string {
	h := sha256.Sum256([]byte(sourceCode + "\x00" + prompt))
	return hex.EncodeToString(h[:])
}
