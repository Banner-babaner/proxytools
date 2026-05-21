package ratelimit

import (
	"sync"
	"time"
	"github.com/Banner-babaner/proxytools/logger"
    "github.com/Banner-babaner/proxytools/config"
)


type LimiterService struct {
    mu       sync.RWMutex
    clients  map[string]*clientBucket
    config   config.RateLimitConfig
    enabled  bool
}

type clientBucket struct {
    tokens      float64
    lastUpdated time.Time
    rps         float64
    rpm         int
    connections int
    activeConns int
}

func NewLimiterService(cfg config.RateLimitConfig) *LimiterService {
    return &LimiterService{
        clients: make(map[string]*clientBucket),
        config:  cfg,
        enabled: cfg.Enabled,
    }
}


func (ls *LimiterService) Allow(ip string) bool {
    if !ls.enabled {
        return true
    }
    
    ls.mu.Lock()
    defer ls.mu.Unlock()
    
    bucket, exists := ls.clients[ip]
    if !exists {
        bucket = &clientBucket{
            tokens:      float64(ls.config.Default.RPS),
            lastUpdated: time.Now(),
            rps:         float64(ls.config.Default.RPS),
            rpm:         ls.config.Default.RPM,
            connections: ls.config.Default.Connections,
        }
        ls.clients[ip] = bucket
    }
    

    now := time.Now()
    elapsed := now.Sub(bucket.lastUpdated).Seconds()
    bucket.tokens += elapsed * bucket.rps
    if bucket.tokens > bucket.rps {
        bucket.tokens = bucket.rps
    }
    bucket.lastUpdated = now
    
    if bucket.tokens >= 1 {
        bucket.tokens--
        return true
    }
    
    logger.Warn().
        Str("ip", ip).
        Float64("tokens", bucket.tokens).
        Msg("Rate limit exceeded")
    
    return false
}


func (ls *LimiterService) IncrementConnections(ip string) bool {
    if !ls.enabled {
        return true
    }
    
    ls.mu.Lock()
    defer ls.mu.Unlock()
    
    bucket, exists := ls.clients[ip]
    if !exists {
        bucket = &clientBucket{
            tokens:      float64(ls.config.Default.RPS),
            lastUpdated: time.Now(),
            rps:         float64(ls.config.Default.RPS),
            connections: ls.config.Default.Connections,
        }
        ls.clients[ip] = bucket
    }
    
    if bucket.activeConns >= bucket.connections {
        return false
    }
    
    bucket.activeConns++
    return true
}


func (ls *LimiterService) DecrementConnections(ip string) {
    if !ls.enabled {
        return
    }
    
    ls.mu.Lock()
    defer ls.mu.Unlock()
    
    if bucket, exists := ls.clients[ip]; exists {
        if bucket.activeConns > 0 {
            bucket.activeConns--
        }
    }
}


func (ls *LimiterService) GetStats(ip string) map[string]interface{} {
    ls.mu.RLock()
    defer ls.mu.RUnlock()
    
    if bucket, exists := ls.clients[ip]; exists {
        return map[string]interface{}{
            "tokens":       bucket.tokens,
            "rps":          bucket.rps,
            "connections":  bucket.activeConns,
            "max_conns":    bucket.connections,
        }
    }
    
    return nil
}


func (ls *LimiterService) Cleanup() {
    ticker := time.NewTicker(5 * time.Minute)
    go func() {
        for range ticker.C {
            ls.mu.Lock()
            for ip, bucket := range ls.clients {
                if time.Since(bucket.lastUpdated) > 10*time.Minute {
                    delete(ls.clients, ip)
                }
            }
            ls.mu.Unlock()
        }
    }()
}