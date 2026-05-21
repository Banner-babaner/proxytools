// internal/cache/cache.go
package cache

import (
    "bytes"
    "crypto/md5"
    "fmt"
    "net/http"
    "sync"
    "time"
    "github.com/Banner-babaner/proxytools/logger"
    "github.com/Banner-babaner/proxytools/config"
)

// CacheEntry запись в кэше
type CacheEntry struct {
	StatusCode int
	Headers    http.Header
	Body       []byte
	Size       int64
	CreatedAt  time.Time
	TTL        time.Duration
	Tags       []string
	Key        string
}

type CacheService struct {
	mu      sync.RWMutex
	entries map[string]*CacheEntry
	config  config.CacheConfig
	enabled bool
	maxSize int64
	curSize int64
}

func NewCacheService(cfg config.CacheConfig) *CacheService {
	return &CacheService{
		entries: make(map[string]*CacheEntry),
		config:  cfg,
		enabled: cfg.Enabled,
		maxSize: int64(cfg.MaxSize) * 1024 * 1024,
	}
}

func (cs *CacheService) GetTTLForPath(method, path, host string) time.Duration {
	for _, rule := range cs.config.Rules {
		if rule.Domain != "" && rule.Domain == host {
			if rule.TTL == 0 {
				return 0
			}
			return time.Duration(rule.TTL) * time.Second
		}
		if rule.Path != "" && matchPath(rule.Path, path) {
			if rule.TTL == 0 {
				return 0
			}
			return time.Duration(rule.TTL) * time.Second
		}
	}

	if method != http.MethodGet {
		return 0
	}

	return time.Duration(cs.config.DefaultTTL) * time.Second
}

func matchPath(pattern, path string) bool {
	if pattern == path {
		return true
	}
	if len(pattern) > 0 && pattern[len(pattern)-1] == '*' {
		prefix := pattern[:len(pattern)-1]
		return len(path) >= len(prefix) && path[:len(prefix)] == prefix
	}
	return false
}

func (cs *CacheService) GenerateKey(method, url string) string {
	hash := md5.Sum([]byte(method + ":" + url))
	return fmt.Sprintf("%x", hash[:])
}

func (cs *CacheService) Get(key string) (*CacheEntry, bool) {
	if !cs.enabled {
		return nil, false
	}

	cs.mu.RLock()
	entry, exists := cs.entries[key]
	cs.mu.RUnlock()

	if !exists {
		return nil, false
	}

	if time.Since(entry.CreatedAt) > entry.TTL {
		cs.Delete(key)
		return nil, false
	}

	logger.Debug().
		Str("key", key).
		Int("status", entry.StatusCode).
		Msg("Cache hit")

	return entry, true
}

func (cs *CacheService) Set(key string, statusCode int, headers http.Header, body []byte, ttl time.Duration, tags []string) {
	if !cs.enabled || ttl == 0 {
		return
	}

	bodySize := int64(len(body))
	
	// Фильтр размера только если maxSize задан
	if cs.maxSize > 0 {
		if bodySize > cs.maxSize/10 {
			return
		}
	}

	var clonedHeaders http.Header
	if headers != nil {
		clonedHeaders = headers.Clone()
	} else {
		clonedHeaders = make(http.Header)
	}

	entry := &CacheEntry{
		StatusCode: statusCode,
		Headers:    clonedHeaders,
		Body:       make([]byte, len(body)),
		Size:       bodySize,
		CreatedAt:  time.Now(),
		TTL:        ttl,
		Tags:       tags,
		Key:        key,
	}
	copy(entry.Body, body)

	cs.mu.Lock()
	defer cs.mu.Unlock()

	if cs.maxSize > 0 && cs.curSize+bodySize > cs.maxSize {
		cs.evict(bodySize)
	}

	cs.entries[key] = entry
	cs.curSize += bodySize

	logger.Debug().
		Str("key", key).
		Int64("size", bodySize).
		Dur("ttl", ttl).
		Msg("Cached response")
}

func (cs *CacheService) Delete(key string) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if entry, exists := cs.entries[key]; exists {
		cs.curSize -= entry.Size
		delete(cs.entries, key)
	}
}

func (cs *CacheService) DeleteByPrefix(prefix string) int {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	count := 0
	for key, entry := range cs.entries {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			cs.curSize -= entry.Size
			delete(cs.entries, key)
			count++
		}
	}
	return count
}

func (cs *CacheService) DeleteByTag(tag string) int {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	count := 0
	for key, entry := range cs.entries {
		for _, t := range entry.Tags {
			if t == tag {
				cs.curSize -= entry.Size
				delete(cs.entries, key)
				count++
				break
			}
		}
	}
	return count
}

func (cs *CacheService) Clear() {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	cs.entries = make(map[string]*CacheEntry)
	cs.curSize = 0

	logger.Info().Msg("Cache cleared")
}

func (cs *CacheService) evict(needed int64) {
	var oldest *CacheEntry
	var oldestKey string

	for key, entry := range cs.entries {
		if oldest == nil || entry.CreatedAt.Before(oldest.CreatedAt) {
			oldest = entry
			oldestKey = key
		}
	}

	if oldest != nil {
		cs.curSize -= oldest.Size
		delete(cs.entries, oldestKey)
	}
}

// CacheResponseWriter обёртка для захвата ответа
type CacheResponseWriter struct {
	http.ResponseWriter
	Buffer     *bytes.Buffer
	StatusCode int
}

func NewCacheResponseWriter(w http.ResponseWriter) *CacheResponseWriter {
	return &CacheResponseWriter{
		ResponseWriter: w,
		Buffer:         new(bytes.Buffer),
		StatusCode:     http.StatusOK,
	}
}

func (cw *CacheResponseWriter) Write(data []byte) (int, error) {
	cw.Buffer.Write(data)
	return cw.ResponseWriter.Write(data)
}

func (cw *CacheResponseWriter) WriteHeader(statusCode int) {
	cw.StatusCode = statusCode
	cw.ResponseWriter.WriteHeader(statusCode)
}

func (cw *CacheResponseWriter) BodyBytes() []byte {
	return cw.Buffer.Bytes()
}