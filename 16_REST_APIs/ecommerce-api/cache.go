package main

import (
	"sync"
	"time"
)

// QueryCache stores pre-computed responses keyed by the full query string
// (e.g. "category=electronics&sort=price&order=desc&page=1") — the guide's
// Caching section, applied to the one endpoint most worth caching: the
// filtered/sorted/paginated product list, which re-does real work
// (filtering, sorting) on every request without it.
type QueryCache struct {
	mu      sync.Mutex
	entries map[string]cacheEntry
	ttl     time.Duration
}

type cacheEntry struct {
	result    PagedResult
	expiresAt time.Time
}

func NewQueryCache(ttl time.Duration) *QueryCache {
	return &QueryCache{entries: make(map[string]cacheEntry), ttl: ttl}
}

func (c *QueryCache) Get(key string) (PagedResult, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, ok := c.entries[key]
	if !ok || time.Now().After(entry.expiresAt) {
		return PagedResult{}, false
	}
	return entry.result, true
}

func (c *QueryCache) Set(key string, result PagedResult) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[key] = cacheEntry{result: result, expiresAt: time.Now().Add(c.ttl)}
}

// Invalidate clears the ENTIRE cache — used whenever a product is created,
// updated, or deleted. A short TTL (see main.go) is this project's
// deliberate, pragmatic answer to the guide's "hardest part of caching is
// invalidation" warning: rather than tracking exactly which cached queries
// a given write could affect, everything simply expires quickly, and
// writes clear it eagerly too, so staleness is bounded to a few seconds at
// most even under heavy write traffic.
func (c *QueryCache) Invalidate() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = make(map[string]cacheEntry)
}
