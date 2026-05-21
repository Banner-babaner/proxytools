package ratelimit

import (
    "net/http"
    
    "github.com/gin-gonic/gin"
)

var limiterService *LimiterService

func SetLimiterService(ls *LimiterService) {
    limiterService = ls
}

func GetRateLimitStats(c *gin.Context) {
    ip := c.Query("ip")
    if ip == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "ip parameter is required"})
        return
    }
    
    stats := limiterService.GetStats(ip)
    c.JSON(http.StatusOK, stats)
}