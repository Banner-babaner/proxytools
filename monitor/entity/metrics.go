package entity

type Metrics struct {
    Uptime         float64 `json:"uptime"`
    TotalRequests  int64   `json:"total_requests"`
    Allowed        int64   `json:"allowed"`
    Denied         int64   `json:"denied"`
    CacheHits      int64   `json:"cache_hits"`
    CacheMisses    int64   `json:"cache_misses"`
    ActiveConns    int64   `json:"active_conns"`
    TotalUpMB      float64 `json:"total_up_mb"`
    TotalDownMB    float64 `json:"total_down_mb"`
    RateLimited    int64   `json:"rate_limited"`
    AvgLatencyMs   float64 `json:"avg_latency_ms"`
    CurrentRPS     int64   `json:"current_rps"`
}