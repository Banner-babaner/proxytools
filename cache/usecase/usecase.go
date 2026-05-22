package usecase

import (
	"crypto/md5"
	"fmt"
	"net/http"
	"time"

	"github.com/Banner-babaner/proxytools/cache/entity"
	"github.com/Banner-babaner/proxytools/cache/repository"
)

type CacheService struct {
	repo       repository.CacheRepository
	enabled    bool
	defaultTTL time.Duration
	maxSize    int64
	rules      []entity.CacheRule
}

func NewCacheService(cfg entity.CacheConfig, repoBuilder func() repository.CacheRepository) *CacheService {
	return &CacheService{
		repo:       repoBuilder(),
		enabled:    cfg.Enabled,
		defaultTTL: cfg.DefaultTTL,
		maxSize:    cfg.MaxSize,
		rules:      cfg.Rules,
	}
}

func (cs *CacheService) GetTTL(method, path, host string) time.Duration {
	if !cs.enabled {
		return 0
	}

	for _, rule := range cs.rules {
		if rule.Domain != "" && rule.Domain == host {
			return rule.TTL
		}
		if rule.Path != "" && matchPath(rule.Path, path) {
			return rule.TTL
		}
	}

	if method != http.MethodGet {
		return 0
	}

	return cs.defaultTTL
}

func (cs *CacheService) GenerateKey(method, url string) string {
	hash := md5.Sum([]byte(method + ":" + url))
	return fmt.Sprintf("%x", hash[:])
}

func (cs *CacheService) Get(key string) (*entity.CacheEntry, bool) {
	if !cs.enabled {
		return nil, false
	}

	entry, ok := cs.repo.Get(key)
	if !ok {
		return nil, false
	}

	if time.Since(entry.CreatedAt) > entry.TTL {
		cs.repo.Delete(key)
		return nil, false
	}

	return entry, true
}

func (cs *CacheService) Set(key string, statusCode int, headers http.Header, body []byte, ttl time.Duration, tags []string) {
	if !cs.enabled || ttl == 0 {
		return
	}

	entry := &entity.CacheEntry{
		StatusCode: statusCode,
		Headers:    headers.Clone(),
		Body:       copyBytes(body),
		Size:       int64(len(body)),
		CreatedAt:  time.Now(),
		TTL:        ttl,
		Tags:       tags,
		Key:        key,
	}

	cs.repo.Set(key, entry)
}

func (cs *CacheService) Invalidate(req entity.InvalidateRequest) (int, error) {
	if req.Clear {
		cs.repo.Clear()
		return 0, nil
	}
	if req.Key != "" {
		cs.repo.Delete(req.Key)
		return 1, nil
	}
	if req.Prefix != "" {
		return cs.repo.DeleteByPrefix(req.Prefix), nil
	}
	if req.Pattern != "" {
		return cs.repo.DeleteByPattern(req.Pattern)
	}
	if len(req.Tags) > 0 {
		count := 0
		for _, tag := range req.Tags {
			count += cs.repo.DeleteByTag(tag)
		}
		return count, nil
	}
	return 0, fmt.Errorf("no invalidation criteria specified")
}

func (cs *CacheService) Stats() entity.CacheStats {
	return cs.repo.Stats()
}

func (cs *CacheService) StartAutoInvalidation() {
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			cs.repo.Delete("") // триггерит очистку просроченных в реализации
		}
	}()
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

func copyBytes(src []byte) []byte {
	dst := make([]byte, len(src))
	copy(dst, src)
	return dst
}