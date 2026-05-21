// internal/ratelimit/handler.go
package ratelimit

import (
    "net/http"
    
    "github.com/gin-gonic/gin"
)

var limiterService *LimiterService

func SetLimiterService(ls *LimiterService) {
    limiterService = ls
}

// GetRateLimitStats godoc
// @Summary Получить статистику rate limit для IP
// @Tags rate_limit
// @Param ip query string true "IP адрес"
// @Success 200 {object} map[string]interface{}
// @Router /rate_limit/stats [get]
func GetRateLimitStats(c *gin.Context) {
    ip := c.Query("ip")
    if ip == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "ip parameter is required"})
        return
    }
    
    stats := limiterService.GetStats(ip)
    c.JSON(http.StatusOK, stats)
}