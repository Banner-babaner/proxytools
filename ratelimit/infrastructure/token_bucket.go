package infrastructure

import (
	"sync"
	"time"

	"github.com/Banner-babaner/proxytools/ratelimit/entity"
)

type clientBucket struct {
	tokens      float64
	lastUpdated time.Time
	rps         float64
	connections int
	maxConns    int
	activeConns int
}

type TokenBucket struct {
	mu      sync.Mutex
	clients map[string]*clientBucket
	rps     float64
	maxConns int
}


func NewTokenBucket(rps int, maxConns int) *TokenBucket {
	return &TokenBucket{
		clients:  make(map[string]*clientBucket),
		rps:      float64(rps),
		maxConns: maxConns,
	}
}

func (tb *TokenBucket) Allow(ip string) bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	bucket, exists := tb.clients[ip]
	if !exists {
		bucket = &clientBucket{
			tokens:      tb.rps,
			lastUpdated: time.Now(),
			rps:         tb.rps,
			maxConns:    tb.maxConns,
		}
		tb.clients[ip] = bucket
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

	return false
}

func (tb *TokenBucket) IncrementConnections(ip string) bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	bucket, exists := tb.clients[ip]
	if !exists {
		bucket = &clientBucket{
			tokens:      tb.rps,
			lastUpdated: time.Now(),
			rps:         tb.rps,
			maxConns:    tb.maxConns,
		}
		tb.clients[ip] = bucket
	}

	if bucket.activeConns >= bucket.maxConns {
		return false
	}

	bucket.activeConns++
	return true
}

func (tb *TokenBucket) DecrementConnections(ip string) {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	if bucket, exists := tb.clients[ip]; exists && bucket.activeConns > 0 {
		bucket.activeConns--
	}
}

func (tb *TokenBucket) GetStats(ip string) *entity.RateLimitStats {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	bucket, exists := tb.clients[ip]
	if !exists {
		return nil
	}

	return &entity.RateLimitStats{
		Tokens:      bucket.tokens,
		RPS:         bucket.rps,
		Connections: bucket.activeConns,
		MaxConns:    bucket.maxConns,
	}
}