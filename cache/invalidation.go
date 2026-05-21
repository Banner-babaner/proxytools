// internal/cache/invalidation.go
package cache

import (
    "regexp"
    "strings"
    "time"
    "github.com/Banner-babaner/proxytools/logger"
)

// InvalidateByPattern инвалидация по регулярному выражению
func (cs *CacheService) InvalidateByPattern(pattern string) (int, error) {
    re, err := regexp.Compile(pattern)
    if err != nil {
        return 0, err
    }

    cs.mu.Lock()
    defer cs.mu.Unlock()

    count := 0
    for key, entry := range cs.entries {
        if re.MatchString(key) {
            cs.curSize -= entry.Size
            delete(cs.entries, key)
            count++
        }
    }

    logger.Info().
        Str("pattern", pattern).
        Int("invalidated", count).
        Msg("Cache invalidated by pattern")

    return count, nil
}

// InvalidateExpired удаляет просроченные записи
func (cs *CacheService) InvalidateExpired() int {
    cs.mu.Lock()
    defer cs.mu.Unlock()

    count := 0
    now := time.Now()
    for key, entry := range cs.entries {
        if now.Sub(entry.CreatedAt) > entry.TTL {
            cs.curSize -= entry.Size
            delete(cs.entries, key)
            count++
        }
    }

    if count > 0 {
        logger.Debug().
            Int("count", count).
            Msg("Expired cache entries removed")
    }

    return count
}

// StartAutoInvalidation автоматическая инвалидация по TTL
func (cs *CacheService) StartAutoInvalidation() {
    ticker := time.NewTicker(30 * time.Second)
    go func() {
        for range ticker.C {
            cs.InvalidateExpired()
        }
    }()
}

// CascadeInvalidate каскадная инвалидация связанных ресурсов
func (cs *CacheService) CascadeInvalidate(key string) {
    cs.Delete(key)
    
    // Находим связанные ключи (префикс до первого /)
    parts := strings.Split(key, ":")
    if len(parts) > 0 {
        prefix := parts[0]
        cs.DeleteByPrefix(prefix)
        logger.Debug().
            Str("key", key).
            Str("prefix", prefix).
            Msg("Cascade invalidated")
    }
}