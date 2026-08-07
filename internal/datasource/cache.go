package datasource

import (
	"sync"
	"time"
)

// cachedEntry holds cached data with an expiration time.
type cachedEntry struct {
	data      interface{}
	expiresAt time.Time
}

// ResponseCache is an in-memory TTL cache for API responses.
// Each cache key can have its own TTL.
type ResponseCache struct {
	mu     sync.RWMutex
	entries map[string]cachedEntry
}

// NewResponseCache creates a new response cache.
func NewResponseCache() *ResponseCache {
	return &ResponseCache{
		entries: make(map[string]cachedEntry),
	}
}

// Get returns cached data if it exists and hasn't expired.
// Returns nil and false if not found or expired.
func (c *ResponseCache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.entries[key]
	if !ok {
		return nil, false
	}
	if time.Now().After(entry.expiresAt) {
		return nil, false
	}
	return entry.data, true
}

// Set stores data in the cache with the given TTL.
func (c *ResponseCache) Set(key string, data interface{}, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[key] = cachedEntry{
		data:      data,
		expiresAt: time.Now().Add(ttl),
	}
}

// Delete removes a specific key from the cache.
func (c *ResponseCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, key)
}

// Cleanup removes all expired entries.
func (c *ResponseCache) Cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for key, entry := range c.entries {
		if now.After(entry.expiresAt) {
			delete(c.entries, key)
		}
	}
}

// CacheTTLs defines TTLs for different types of API responses.
var CacheTTLs = struct {
	TrendingTokens time.Duration
	NewPools       time.Duration
	TokenPrice     time.Duration
	MarketCap      time.Duration
	TopGainers     time.Duration
}{
	TrendingTokens: 5 * time.Minute,
	NewPools:       5 * time.Minute,
	TokenPrice:     2 * time.Minute,
	MarketCap:      15 * time.Minute,
	TopGainers:     5 * time.Minute,
}
