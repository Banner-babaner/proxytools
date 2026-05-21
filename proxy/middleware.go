// internal/proxy/middleware.go
package proxy

import (
    "time"

    "github.com/gin-gonic/gin"
    "github.com/Banner-babaner/proxytools/logger"
)

// LoggerMiddleware логирует все запросы
func LoggerMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        start := time.Now()
        
        c.Next()
        
        duration := time.Since(start)
        
        logger.Info().
            Str("method", c.Request.Method).
            Str("path", c.Request.URL.Path).
            Int("status", c.Writer.Status()).
            Str("ip", c.ClientIP()).
            Dur("latency", duration).
            Int("size", c.Writer.Size()).
            Msg("HTTP request")
    }
}

// RecoveryMiddleware восстановление после паники
func RecoveryMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        defer func() {
            if err := recover(); err != nil {
                logger.Error().
                    Interface("error", err).
                    Str("path", c.Request.URL.Path).
                    Msg("Panic recovered")
                
                c.AbortWithStatus(500)
            }
        }()
        
        c.Next()
    }
}