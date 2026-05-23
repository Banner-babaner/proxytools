package infrastructure

import (
	"regexp"
	"sync"


	"cache/entity"
	"cache/repository"
)

type LRUCache struct {
	mu      sync.RWMutex
	entries map[string]*entity.CacheEntry
	maxSize int64
	curSize int64
}

var _ repository.CacheRepository = (*LRUCache)(nil)

func NewLRUCache(maxSizeMB int64) *LRUCache {
	return &LRUCache{
		entries: make(map[string]*entity.CacheEntry),
		maxSize: maxSizeMB * 1024 * 1024,
	}
}

func (c *LRUCache) Get(key string) (*entity.CacheEntry, bool) {
	c.mu.RLock()
	entry, ok := c.entries[key]
	c.mu.RUnlock()
	return entry, ok
}

func (c *LRUCache) Set(key string, entry *entity.CacheEntry) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.curSize+entry.Size > c.maxSize {
		c.evict(entry.Size)
	}

	c.entries[key] = entry
	c.curSize += entry.Size
}

func (c *LRUCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if entry, ok := c.entries[key]; ok {
		c.curSize -= entry.Size
		delete(c.entries, key)
	}
}

func (c *LRUCache) DeleteByPrefix(prefix string) int {
	c.mu.Lock()
	defer c.mu.Unlock()

	count := 0
	for key, entry := range c.entries {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			c.curSize -= entry.Size
			delete(c.entries, key)
			count++
		}
	}
	return count
}

func (c *LRUCache) DeleteByTag(tag string) int {
	c.mu.Lock()
	defer c.mu.Unlock()

	count := 0
	for key, entry := range c.entries {
		for _, t := range entry.Tags {
			if t == tag {
				c.curSize -= entry.Size
				delete(c.entries, key)
				count++
				break
			}
		}
	}
	return count
}

func (c *LRUCache) DeleteByPattern(pattern string) (int, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return 0, err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	count := 0
	for key, entry := range c.entries {
		if re.MatchString(key) {
			c.curSize -= entry.Size
			delete(c.entries, key)
			count++
		}
	}
	return count, nil
}

func (c *LRUCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = make(map[string]*entity.CacheEntry)
	c.curSize = 0
}

func (c *LRUCache) Stats() entity.CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return entity.CacheStats{
		Entries: len(c.entries),
		SizeMB:  float64(c.curSize) / 1024 / 1024,
		MaxSize: c.maxSize,
		Enabled: true,
	}
}

func (c *LRUCache) evict(needed int64) {
	var oldest *entity.CacheEntry
	var oldestKey string

	for key, entry := range c.entries {
		if oldest == nil || entry.CreatedAt.Before(oldest.CreatedAt) {
			oldest = entry
			oldestKey = key
		}
	}

	if oldest != nil {
		c.curSize -= oldest.Size
		delete(c.entries, oldestKey)
	}
}