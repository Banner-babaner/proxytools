// internal/monitor/metrics_test.go
package monitor

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewMetricsService(t *testing.T) {
	ms := NewMetricsService()
	assert.NotNil(t, ms)
	assert.False(t, ms.StartTime.IsZero())
}

func TestRecordRequest_Allowed(t *testing.T) {
	ms := NewMetricsService()

	ms.RecordRequest(true, 0.05, 1024, 2048)

	assert.Equal(t, int64(1), ms.TotalRequests.Load())
	assert.Equal(t, int64(1), ms.AllowedRequests.Load())
	assert.Equal(t, int64(0), ms.DeniedRequests.Load())
	assert.Equal(t, int64(1024), ms.TotalBytesUp.Load())
	assert.Equal(t, int64(2048), ms.TotalBytesDown.Load())
}

func TestRecordRequest_Denied(t *testing.T) {
	ms := NewMetricsService()

	ms.RecordRequest(false, 0.01, 512, 0)

	assert.Equal(t, int64(1), ms.TotalRequests.Load())
	assert.Equal(t, int64(0), ms.AllowedRequests.Load())
	assert.Equal(t, int64(1), ms.DeniedRequests.Load())
}

func TestRecordRequest_Multiple(t *testing.T) {
	ms := NewMetricsService()

	for i := 0; i < 100; i++ {
		ms.RecordRequest(true, 0.01, 100, 200)
	}

	assert.Equal(t, int64(100), ms.TotalRequests.Load())
	assert.Equal(t, int64(10000), ms.TotalBytesUp.Load())
}

func TestRecordCache(t *testing.T) {
	ms := NewMetricsService()

	ms.RecordCacheHit()
	ms.RecordCacheHit()
	ms.RecordCacheMiss()

	assert.Equal(t, int64(2), ms.CacheHits.Load())
	assert.Equal(t, int64(1), ms.CacheMisses.Load())
}

func TestGetStats(t *testing.T) {
	ms := NewMetricsService()
	ms.RecordRequest(true, 0.05, 100, 200)
	ms.RecordCacheHit()

	stats := ms.GetStats()

	assert.Contains(t, stats, "uptime")
	assert.Contains(t, stats, "total_requests")
	assert.Contains(t, stats, "allowed")
	assert.Contains(t, stats, "denied")
	assert.Contains(t, stats, "cache_hits")
	assert.Contains(t, stats, "cache_misses")
	assert.Contains(t, stats, "active_conns")
	assert.Contains(t, stats, "total_up_mb")
	assert.Contains(t, stats, "total_down_mb")
	assert.Contains(t, stats, "rate_limited")
	assert.Contains(t, stats, "current_rps")

	assert.Equal(t, int64(1), stats["total_requests"])
	assert.Equal(t, int64(1), stats["cache_hits"])
}

func TestUptime(t *testing.T) {
	ms := NewMetricsService()

	time.Sleep(100 * time.Millisecond)

	stats := ms.GetStats()
	uptime := stats["uptime"].(float64)

	assert.GreaterOrEqual(t, uptime, 0.1)
	assert.Less(t, uptime, 2.0)
}

func TestActiveConnections(t *testing.T) {
	ms := NewMetricsService()

	ms.ActiveConns.Add(1)
	ms.ActiveConns.Add(1)

	stats := ms.GetStats()
	assert.Equal(t, int64(2), stats["active_conns"])

	ms.ActiveConns.Add(-1)
	stats = ms.GetStats()
	assert.Equal(t, int64(1), stats["active_conns"])
}

func TestRateLimitedCount(t *testing.T) {
	ms := NewMetricsService()

	ms.RateLimitedCount.Add(5)

	stats := ms.GetStats()
	assert.Equal(t, int64(5), stats["rate_limited"])
}