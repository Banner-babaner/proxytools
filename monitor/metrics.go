package monitor

import (
    "sync/atomic"
    "time"
	"sync"
    "github.com/gorilla/websocket"
    "github.com/Banner-babaner/proxytools/logger"
)

type MetricsService struct {
    TotalRequests    atomic.Int64
    AllowedRequests  atomic.Int64
    DeniedRequests   atomic.Int64
    CacheHits        atomic.Int64
    CacheMisses      atomic.Int64
    ActiveConns      atomic.Int64
    TotalBytesUp     atomic.Int64
    TotalBytesDown   atomic.Int64
    RateLimitedCount atomic.Int64
    StartTime        time.Time

    rpsHistory    []int64
    latencyHist   []float64
    mu            sync.RWMutex

    wsClients map[*websocket.Conn]bool
    wsMu      sync.Mutex
}

func NewMetricsService() *MetricsService {
    ms := &MetricsService{
        StartTime:  time.Now(),
        rpsHistory: make([]int64, 60),
        latencyHist: make([]float64, 60),
        wsClients:  make(map[*websocket.Conn]bool),
    }

    go ms.collectRPS()
    return ms
}

func (ms *MetricsService) collectRPS() {
    ticker := time.NewTicker(1 * time.Second)
    var lastTotal int64
    second := 0

    for range ticker.C {
        current := ms.TotalRequests.Load()
        rps := current - lastTotal
        lastTotal = current

        ms.mu.Lock()
        ms.rpsHistory[second%60] = rps
        ms.mu.Unlock()

        second++
    }
}


func (ms *MetricsService) RecordRequest(allowed bool, latency float64, bytesUp, bytesDown int64) {
    ms.TotalRequests.Add(1)
    if allowed {
        ms.AllowedRequests.Add(1)
    } else {
        ms.DeniedRequests.Add(1)
    }
    ms.TotalBytesUp.Add(bytesUp)
    ms.TotalBytesDown.Add(bytesDown)

    ms.mu.Lock()
    ms.latencyHist[time.Now().Second()] = latency
    ms.mu.Unlock()
}

func (ms *MetricsService) RecordCacheHit() {
    ms.CacheHits.Add(1)
}

func (ms *MetricsService) RecordCacheMiss() {
    ms.CacheMisses.Add(1)
}

func (ms *MetricsService) GetStats() map[string]interface{} {
    uptime := time.Since(ms.StartTime).Seconds()

    ms.mu.RLock()
    var avgLatency float64
    for _, l := range ms.latencyHist {
        avgLatency += l
    }
    avgLatency /= 60
    ms.mu.RUnlock()

    return map[string]interface{}{
        "uptime":           uptime,
        "total_requests":   ms.TotalRequests.Load(),
        "allowed":          ms.AllowedRequests.Load(),
        "denied":           ms.DeniedRequests.Load(),
        "cache_hits":       ms.CacheHits.Load(),
        "cache_misses":     ms.CacheMisses.Load(),
        "active_conns":     ms.ActiveConns.Load(),
        "total_up_mb":      float64(ms.TotalBytesUp.Load()) / 1024 / 1024,
        "total_down_mb":    float64(ms.TotalBytesDown.Load()) / 1024 / 1024,
        "rate_limited":     ms.RateLimitedCount.Load(),
        "avg_latency_ms":   avgLatency,
        "current_rps":      ms.rpsHistory[time.Now().Second()%60],
    }
}

func (ms *MetricsService) AddWSClient(conn *websocket.Conn) {
    ms.wsMu.Lock()
    ms.wsClients[conn] = true
    ms.wsMu.Unlock()
}


func (ms *MetricsService) RemoveWSClient(conn *websocket.Conn) {
    ms.wsMu.Lock()
    delete(ms.wsClients, conn)
    ms.wsMu.Unlock()
}

func (ms *MetricsService) BroadcastMetrics() {
    ticker := time.NewTicker(1 * time.Second)
    for range ticker.C {
        stats := ms.GetStats()

        ms.wsMu.Lock()
        for conn := range ms.wsClients {
            if err := conn.WriteJSON(stats); err != nil {
                logger.Debug().Err(err).Msg("WebSocket write error")
                conn.Close()
                delete(ms.wsClients, conn)
            }
        }
        ms.wsMu.Unlock()
    }
}