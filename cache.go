package main

import (
	"container/list"
	"sync"
	"time"
)

type resolveCacheEntry struct {
	shortKey  string
	longURL   string
	expiresAt time.Time
}

// resolveCache is an LRU cache of resolved mappings. Entries expire with their
// mapping or after ttl, whichever comes first, and leave only on eviction or on
// the next read, so it needs no background cleanup.
type resolveCache struct {
	capacity int
	ttl      time.Duration
	now      func() time.Time

	mu      sync.Mutex
	order   *list.List
	entries map[string]*list.Element
}

func newResolveCache(capacity int, ttl time.Duration) *resolveCache {
	return &resolveCache{
		capacity: capacity,
		ttl:      ttl,
		now:      time.Now,
		order:    list.New(),
		entries:  make(map[string]*list.Element),
	}
}

func (c *resolveCache) get(shortKey string) (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	element, ok := c.entries[shortKey]
	if !ok {
		return "", false
	}
	entry := element.Value.(*resolveCacheEntry)
	if !c.now().Before(entry.expiresAt) {
		c.remove(element)
		return "", false
	}
	c.order.MoveToFront(element)
	return entry.longURL, true
}

// set caches a mapping for no longer than its lifetime.
func (c *resolveCache) set(shortKey, longURL string, life lifetime) {
	ttl := life.cappedAt(c.ttl)

	c.mu.Lock()
	defer c.mu.Unlock()
	expiresAt := c.now().Add(ttl)
	if element, ok := c.entries[shortKey]; ok {
		entry := element.Value.(*resolveCacheEntry)
		entry.longURL, entry.expiresAt = longURL, expiresAt
		c.order.MoveToFront(element)
		return
	}
	c.entries[shortKey] = c.order.PushFront(&resolveCacheEntry{shortKey, longURL, expiresAt})
	if c.order.Len() > c.capacity {
		c.remove(c.order.Back())
	}
}

func (c *resolveCache) remove(element *list.Element) {
	c.order.Remove(element)
	delete(c.entries, element.Value.(*resolveCacheEntry).shortKey)
}
