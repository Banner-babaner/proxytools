package usecase

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/Banner-babaner/proxytools/monitor/entity"
)

type MetricsService struct {
	mu sync.RWMutex

	totalRequests    atomic.Int64
	allowedRequests  atomic.Int64
	deniedRequests   atomic.Int64
	cacheHits        atomic.Int64
	cacheMisses      atomic.Int64
	activeConns      atomic.Int64
	totalBytesUp     atomic.Int64
	totalBytesDown   atomic.Int64
	rateLimitedCount atomic.Int64

	startTime      time.Time
	rpsHistory     [60]int64
	rpsIndex       int
	lastTotal      int64
	latencyHistory [60]float64
	latencyIndex   int
}

func NewMetricsService() *MetricsService {
	ms := &MetricsService{
		startTime: time.Now(),
	}
	go ms.collectRPS()
	return ms
}

func (ms *MetricsService) collectRPS() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		current := ms.totalRequests.Load()
		rps := current - ms.lastTotal
		ms.lastTotal = current

		ms.mu.Lock()
		ms.rpsHistory[ms.rpsIndex%60] = rps
		ms.rpsIndex++
		ms.mu.Unlock()
	}
}

func (ms *MetricsService) RecordRequest(allowed bool, latencySeconds float64, bytesUp, bytesDown int64) {
	ms.totalRequests.Add(1)
	if allowed {
		ms.allowedRequests.Add(1)
	} else {
		ms.deniedRequests.Add(1)
	}
	ms.totalBytesUp.Add(bytesUp)
	ms.totalBytesDown.Add(bytesDown)

	ms.mu.Lock()
	ms.latencyHistory[ms.latencyIndex%60] = latencySeconds
	ms.latencyIndex++
	ms.mu.Unlock()
}

func (ms *MetricsService) RecordCacheHit()  { ms.cacheHits.Add(1) }
func (ms *MetricsService) RecordCacheMiss() { ms.cacheMisses.Add(1) }
func (ms *MetricsService) RecordRateLimit() { ms.rateLimitedCount.Add(1) }

func (ms *MetricsService) IncrementConnections() { ms.activeConns.Add(1) }
func (ms *MetricsService) DecrementConnections() { ms.activeConns.Add(-1) }

func (ms *MetricsService) GetStats() entity.Metrics {
	uptime := time.Since(ms.startTime).Seconds()

	ms.mu.RLock()
	var avgLatency float64
	count := 0
	for _, l := range ms.latencyHistory {
		if l > 0 {
			avgLatency += l
			count++
		}
	}
	if count > 0 {
		avgLatency = (avgLatency / float64(count)) * 1000
	}

	currentRPS := int64(0)
	if ms.rpsIndex > 0 {
		currentRPS = ms.rpsHistory[(ms.rpsIndex-1)%60]
	}
	ms.mu.RUnlock()

	return entity.Metrics{
		Uptime:        uptime,
		TotalRequests: ms.totalRequests.Load(),
		Allowed:       ms.allowedRequests.Load(),
		Denied:        ms.deniedRequests.Load(),
		CacheHits:     ms.cacheHits.Load(),
		CacheMisses:   ms.cacheMisses.Load(),
		ActiveConns:   ms.activeConns.Load(),
		TotalUpMB:     float64(ms.totalBytesUp.Load()) / 1024 / 1024,
		TotalDownMB:   float64(ms.totalBytesDown.Load()) / 1024 / 1024,
		RateLimited:   ms.rateLimitedCount.Load(),
		AvgLatencyMs:  avgLatency,
		CurrentRPS:    currentRPS,
	}
}