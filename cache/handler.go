// internal/cache/handler.go
package cache

import (
    "net/http"

    "github.com/gin-gonic/gin"
    "github.com/Banner-babaner/proxytools/logger"
)

var cacheService *CacheService

func SetCacheService(cs *CacheService) {
    cacheService = cs
}

// InvalidateRequest тело запроса на инвалидацию
type InvalidateRequest struct {
    Key     string   `json:"key"`
    Prefix  string   `json:"prefix"`
    Pattern string   `json:"pattern"`
    Tags    []string `json:"tags"`
    Clear   bool     `json:"clear_all"`
}

// InvalidateCache godoc
// @Summary Инвалидировать кэш
// @Tags cache
// @Accept json
// @Produce json
// @Param request body InvalidateRequest true "Параметры инвалидации"
// @Success 200 {object} map[string]interface{}
// @Router /cache/invalidate [post]
func InvalidateCache(c *gin.Context) {
    var req InvalidateRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    var count int

    if req.Clear {
        cacheService.Clear()
        c.JSON(http.StatusOK, gin.H{"message": "Cache cleared"})
        return
    }

    if req.Key != "" {
        cacheService.Delete(req.Key)
        count = 1
    } else if req.Prefix != "" {
        count = cacheService.DeleteByPrefix(req.Prefix)
    } else if req.Pattern != "" {
        var err error
        count, err = cacheService.InvalidateByPattern(req.Pattern)
        if err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
            return
        }
    } else if len(req.Tags) > 0 {
        for _, tag := range req.Tags {
            count += cacheService.DeleteByTag(tag)
        }
    } else {
        c.JSON(http.StatusBadRequest, gin.H{"error": "No invalidation criteria specified"})
        return
    }

    logger.Info().
        Int("invalidated", count).
        Msg("Cache invalidated via API")

    c.JSON(http.StatusOK, gin.H{"invalidated": count})
}

// GetCacheStats godoc
// @Summary Получить статистику кэша
// @Tags cache
// @Success 200 {object} map[string]interface{}
// @Router /cache/stats [get]
func GetCacheStats(c *gin.Context) {
    cacheService.mu.RLock()
    defer cacheService.mu.RUnlock()

    c.JSON(http.StatusOK, gin.H{
        "entries":   len(cacheService.entries),
        "size_mb":   float64(cacheService.curSize) / 1024 / 1024,
        "max_size":  cacheService.maxSize,
        "enabled":   cacheService.enabled,
    })
}