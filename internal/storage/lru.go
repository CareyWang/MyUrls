package storage

import (
	"container/list"
	"sync"
	"time"
)

// entry 是lru缓存中的条目.
type entry struct {
	key       string
	value     string
	expiresAt time.Time
}

// isExpired 检查条目是否过期.
func (e *entry) isExpired() bool {
	if e.expiresAt.IsZero() {
		return false
	}
	return time.Now().After(e.expiresAt)
}

// LRUCache 是一个线程安全的、带TTL的LRU缓存.
type LRUCache struct {
	capacity int
	ttl      time.Duration
	ll       *list.List
	cache    map[string]*list.Element
	mu       sync.RWMutex
}

// NewLRUCache 创建一个新的LRUCache.
// capacity 是缓存的最大容量.
// ttl 是缓存项的默认存活时间.
func NewLRUCache(capacity int, ttl time.Duration) *LRUCache {
	if capacity <= 0 {
		capacity = 1 // 至少为1
	}
	lru := &LRUCache{
		capacity: capacity,
		ttl:      ttl,
		ll:       list.New(),
		cache:    make(map[string]*list.Element),
	}
	// 启动后台goroutine定期清理过期的缓存项
	go lru.cleanupLoop(time.Minute)
	return lru
}

// Get 从缓存中获取一个值.
func (c *LRUCache) Get(key string) (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, hit := c.cache[key]; hit {
		ent := elem.Value.(*entry)

		// 检查缓存是否过期
		if ent.isExpired() {
			c.removeElement(elem)
			return "", false
		}

		c.ll.MoveToFront(elem)
		return ent.value, true
	}
	return "", false
}

// Set 向缓存中添加一个值.
// expiration 是此特定键的过期时间.
func (c *LRUCache) Set(key string, value string, expiration time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	expiresAt := time.Now().Add(expiration)
	// 如果传入的过期时间小于等于0，则使用缓存的默认TTL
	if expiration <= 0 {
		expiresAt = time.Now().Add(c.ttl)
	}

	if elem, hit := c.cache[key]; hit {
		c.ll.MoveToFront(elem)
		ent := elem.Value.(*entry)
		ent.value = value
		ent.expiresAt = expiresAt
	} else {
		ent := &entry{key: key, value: value, expiresAt: expiresAt}
		elem := c.ll.PushFront(ent)
		c.cache[key] = elem

		if c.ll.Len() > c.capacity {
			c.removeOldest()
		}
	}
}

// Delete 从缓存中删除一个键.
func (c *LRUCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, hit := c.cache[key]; hit {
		c.removeElement(elem)
	}
}

// Clear 清空所有缓存.
func (c *LRUCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.ll = list.New()
	c.cache = make(map[string]*list.Element)
}

// removeOldest 删除最旧的条目.
func (c *LRUCache) removeOldest() {
	elem := c.ll.Back()
	if elem != nil {
		c.removeElement(elem)
	}
}

// removeElement 从缓存中移除一个元素.
func (c *LRUCache) removeElement(e *list.Element) {
	c.ll.Remove(e)
	ent := e.Value.(*entry)
	delete(c.cache, ent.key)
}

// cleanupLoop 是一个定期清理过期缓存项的循环.
func (c *LRUCache) cleanupLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()
		for key, elem := range c.cache {
			if elem.Value.(*entry).isExpired() {
				c.removeElement(elem)
				// 重新检查，因为 removeElement 修改了映射
				delete(c.cache, key)
			}
		}
		c.mu.Unlock()
	}
}
