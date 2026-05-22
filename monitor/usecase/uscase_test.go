package usecase

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewMetricsService(t *testing.T) {
	ms := NewMetricsService()
	assert.NotNil(t, ms)
	assert.False(t, ms.startTime.IsZero())
}

func TestRecordRequest_Allowed(t *testing.T) {
	ms := NewMetricsService()

	ms.RecordRequest(true, 0.05, 1024, 2048)

	assert.Equal(t, int64(1), ms.totalRequests.Load())
	assert.Equal(t, int64(1), ms.allowedRequests.Load())
	assert.Equal(t, int64(0), ms.deniedRequests.Load())
	assert.Equal(t, int64(1024), ms.totalBytesUp.Load())
	assert.Equal(t, int64(2048), ms.totalBytesDown.Load())
}

func TestRecordRequest_Denied(t *testing.T) {
	ms := NewMetricsService()

	ms.RecordRequest(false, 0.01, 512, 0)

	assert.Equal(t, int64(1), ms.totalRequests.Load())
	assert.Equal(t, int64(0), ms.allowedRequests.Load())
	assert.Equal(t, int64(1), ms.deniedRequests.Load())
}

func TestRecordRequest_Multiple(t *testing.T) {
	ms := NewMetricsService()

	for i := 0; i < 100; i++ {
		ms.RecordRequest(true, 0.01, 100, 200)
	}

	assert.Equal(t, int64(100), ms.totalRequests.Load())
	assert.Equal(t, int64(100), ms.allowedRequests.Load())
	assert.Equal(t, int64(10000), ms.totalBytesUp.Load())
	assert.Equal(t, int64(20000), ms.totalBytesDown.Load())
}

func TestRecordCacheHit(t *testing.T) {
	ms := NewMetricsService()

	ms.RecordCacheHit()
	ms.RecordCacheHit()

	assert.Equal(t, int64(2), ms.cacheHits.Load())
}

func TestRecordCacheMiss(t *testing.T) {
	ms := NewMetricsService()

	ms.RecordCacheMiss()
	ms.RecordCacheMiss()
	ms.RecordCacheMiss()

	assert.Equal(t, int64(3), ms.cacheMisses.Load())
}

func TestRecordRateLimit(t *testing.T) {
	ms := NewMetricsService()

	ms.RecordRateLimit()
	ms.RecordRateLimit()

	assert.Equal(t, int64(2), ms.rateLimitedCount.Load())
}

func TestConnections(t *testing.T) {
	ms := NewMetricsService()

	ms.IncrementConnections()
	ms.IncrementConnections()
	assert.Equal(t, int64(2), ms.activeConns.Load())

	ms.DecrementConnections()
	assert.Equal(t, int64(1), ms.activeConns.Load())

	ms.DecrementConnections()
	assert.Equal(t, int64(0), ms.activeConns.Load())
}

func TestGetStats_ContainsAllFields(t *testing.T) {
	ms := NewMetricsService()

	time.Sleep(10*time.Microsecond)

	ms.RecordRequest(true, 0.05, 100, 200)
	ms.RecordCacheHit()

	stats := ms.GetStats()

	assert.Greater(t, stats.Uptime, 0.0)
	assert.Equal(t, int64(1), stats.TotalRequests)
	assert.Equal(t, int64(1), stats.Allowed)
	assert.Equal(t, int64(0), stats.Denied)
	assert.Equal(t, int64(1), stats.CacheHits)
	assert.Equal(t, int64(0), stats.CacheMisses)
	assert.Equal(t, int64(0), stats.ActiveConns)
	assert.Greater(t, stats.TotalUpMB, 0.0)
	assert.Greater(t, stats.TotalDownMB, 0.0)
	assert.Equal(t, int64(0), stats.RateLimited)
}

func TestGetStats_Uptime(t *testing.T) {
	ms := NewMetricsService()

	time.Sleep(100 * time.Millisecond)

	stats := ms.GetStats()
	assert.GreaterOrEqual(t, stats.Uptime, 0.1)
}

func TestGetStats_Latency(t *testing.T) {
	ms := NewMetricsService()

	ms.RecordRequest(true, 0.05, 100, 200)
	ms.RecordRequest(true, 0.15, 100, 200)

	stats := ms.GetStats()
	assert.Greater(t, stats.AvgLatencyMs, 0.0)
}

func TestConcurrentRecording(t *testing.T) {
	ms := NewMetricsService()

	done := make(chan bool, 100)
	for i := 0; i < 100; i++ {
		go func() {
			for j := 0; j < 50; j++ {
				ms.RecordRequest(true, 0.01, 100, 200)
				ms.RecordCacheHit()
				ms.IncrementConnections()
				ms.DecrementConnections()
			}
			done <- true
		}()
	}

	for i := 0; i < 100; i++ {
		<-done
	}

	stats := ms.GetStats()
	assert.Equal(t, int64(5000), stats.TotalRequests)
	assert.Equal(t, int64(5000), stats.CacheHits)
	assert.Equal(t, int64(0), stats.ActiveConns)
}

func TestGetStats_Concurrent(t *testing.T) {
	ms := NewMetricsService()
	ms.RecordRequest(true, 0.01, 100, 200)

	done := make(chan bool, 50)
	for i := 0; i < 50; i++ {
		go func() {
			ms.GetStats()
			done <- true
		}()
	}

	for i := 0; i < 50; i++ {
		<-done
	}
}