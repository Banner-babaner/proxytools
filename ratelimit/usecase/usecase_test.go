package usecase

import (
	"sync"
	"testing"
	"time"

	"github.com/Banner-babaner/proxytools/config"
	"github.com/stretchr/testify/assert"
)


func TestNewLimiterService(t *testing.T) {
	cfg := config.RateLimitConfig{
		Enabled: true,
		Default: config.RateLimitDefaults{
			RPS:         10,
			RPM:         100,
			RPH:         1000,
			Connections: 5,
		},
	}


	ls := NewLimiterService(cfg)
	assert.NotNil(t, ls)
	assert.True(t, ls.enabled)
	assert.Equal(t, 10, ls.config.Default.RPS)
	assert.Equal(t, 100, ls.config.Default.RPM)
	assert.Equal(t, 5, ls.config.Default.Connections)
}

func TestAllow_FirstRequest(t *testing.T) {
	ls := NewLimiterService(config.RateLimitConfig{
		Enabled: true,
		Default: config.RateLimitDefaults{RPS: 10},
	})

	assert.True(t, ls.Allow("192.168.1.1"))
}

func TestAllow_RateLimitExceeded(t *testing.T) {
	ls := NewLimiterService(config.RateLimitConfig{
		Enabled: true,
		Default: config.RateLimitDefaults{RPS: 2},
	})

	ip := "192.168.1.1"

	assert.True(t, ls.Allow(ip))
	assert.True(t, ls.Allow(ip))
	assert.False(t, ls.Allow(ip)) // превышен RPS=2
}

func TestAllow_TokenRefill(t *testing.T) {
	ls := NewLimiterService(config.RateLimitConfig{
		Enabled: true,
		Default: config.RateLimitDefaults{RPS: 5},
	})

	ip := "10.0.0.1"

	for i := 0; i < 5; i++ {
		assert.True(t, ls.Allow(ip))
	}
	assert.False(t, ls.Allow(ip))

	time.Sleep(300 * time.Millisecond)

	assert.True(t, ls.Allow(ip))
}

func TestAllow_TokenBurst(t *testing.T) {
	ls := NewLimiterService(config.RateLimitConfig{
		Enabled: true,
		Default: config.RateLimitDefaults{RPS: 3},
	})

	ip := "192.168.1.1"

	assert.True(t, ls.Allow(ip))
	assert.True(t, ls.Allow(ip))
	assert.True(t, ls.Allow(ip))
	assert.False(t, ls.Allow(ip))
}

func TestAllow_Disabled(t *testing.T) {
	ls := NewLimiterService(config.RateLimitConfig{
		Enabled: false,
	})

	for i := 0; i < 1000; i++ {
		assert.True(t, ls.Allow("192.168.1.1"))
	}
}

func TestAllow_MultipleClients(t *testing.T) {
	ls := NewLimiterService(config.RateLimitConfig{
		Enabled: true,
		Default: config.RateLimitDefaults{RPS: 3},
	})

	ls.Allow("1.1.1.1")
	ls.Allow("1.1.1.1")
	ls.Allow("1.1.1.1")
	assert.False(t, ls.Allow("1.1.1.1"))

	// Клиент 2 не затронут
	assert.True(t, ls.Allow("2.2.2.2"))
	assert.True(t, ls.Allow("2.2.2.2"))
}

func TestAllow_DifferentIPs(t *testing.T) {
	ls := NewLimiterService(config.RateLimitConfig{
		Enabled: true,
		Default: config.RateLimitDefaults{RPS: 1},
	})

	assert.True(t, ls.Allow("10.0.0.1"))
	assert.False(t, ls.Allow("10.0.0.1")) // этот IP исчерпан

	assert.True(t, ls.Allow("10.0.0.2")) // другой IP ок
}

func TestAllow_NewClientCreation(t *testing.T) {
	ls := NewLimiterService(config.RateLimitConfig{
		Enabled: true,
		Default: config.RateLimitDefaults{RPS: 5},
	})


	assert.True(t, ls.Allow("new-client-ip"))

	ls.mu.RLock()
	_, exists := ls.clients["new-client-ip"]
	ls.mu.RUnlock()
	assert.True(t, exists)
}

func TestIncrementConnections(t *testing.T) {
	ls := NewLimiterService(config.RateLimitConfig{
		Enabled: true,
		Default: config.RateLimitDefaults{Connections: 2},
	})

	ip := "192.168.1.1"

	assert.True(t, ls.IncrementConnections(ip))
	assert.True(t, ls.IncrementConnections(ip))
	assert.False(t, ls.IncrementConnections(ip))
}

func TestIncrementConnections_Disabled(t *testing.T) {
	ls := NewLimiterService(config.RateLimitConfig{
		Enabled: false,
	})

	assert.True(t, ls.IncrementConnections("192.168.1.1"))
	assert.True(t, ls.IncrementConnections("192.168.1.1"))
	assert.True(t, ls.IncrementConnections("192.168.1.1"))
}

func TestDecrementConnections(t *testing.T) {
	ls := NewLimiterService(config.RateLimitConfig{
		Enabled: true,
		Default: config.RateLimitDefaults{Connections: 2},
	})

	ip := "10.0.0.1"

	ls.IncrementConnections(ip)
	ls.IncrementConnections(ip)

	ls.DecrementConnections(ip)

	// Теперь можно открыть ещё одно
	assert.True(t, ls.IncrementConnections(ip))
}

func TestDecrementConnections_Zero(t *testing.T) {
	ls := NewLimiterService(config.RateLimitConfig{
		Enabled: true,
		Default: config.RateLimitDefaults{Connections: 2},
	})

	assert.NotPanics(t, func() {
		ls.DecrementConnections("unknown")
	})
}

func TestDecrementConnections_UnknownIP(t *testing.T) {
	ls := NewLimiterService(config.RateLimitConfig{
		Enabled: true,
		Default: config.RateLimitDefaults{Connections: 5},
	})

	// Не паникует
	ls.DecrementConnections("never-seen")
}

func TestDecrementConnections_Disabled(t *testing.T) {
	ls := NewLimiterService(config.RateLimitConfig{
		Enabled: false,
	})

	// Не паникует
	ls.DecrementConnections("any-ip")
}

func TestGetStats(t *testing.T) {
	ls := NewLimiterService(config.RateLimitConfig{
		Enabled: true,
		Default: config.RateLimitDefaults{
			RPS:         10,
			Connections: 5,
		},
	})

	ip := "192.168.1.1"
	ls.Allow(ip)
	ls.IncrementConnections(ip)

	stats := ls.GetStats(ip)
	assert.NotNil(t, stats)
	assert.Contains(t, stats, "tokens")
	assert.Contains(t, stats, "connections")
	assert.Contains(t, stats, "rps")
	assert.Contains(t, stats, "max_conns")

	// Проверяем значения
	assert.Equal(t, 1, stats["connections"])
	assert.Equal(t, 5, stats["max_conns"])
	assert.Less(t, stats["tokens"].(float64), 10.0) // один токен использован
}

func TestGetStats_UnknownIP(t *testing.T) {
	ls := NewLimiterService(config.RateLimitConfig{Enabled: true})

	stats := ls.GetStats("unknown")
	assert.Nil(t, stats)
}

func TestGetStats_MultipleIPs(t *testing.T) {
	ls := NewLimiterService(config.RateLimitConfig{
		Enabled: true,
		Default: config.RateLimitDefaults{RPS: 10},
	})

	ls.Allow("ip-1")
	ls.Allow("ip-2")
	ls.Allow("ip-3")

	assert.NotNil(t, ls.GetStats("ip-1"))
	assert.NotNil(t, ls.GetStats("ip-2"))
	assert.NotNil(t, ls.GetStats("ip-3"))
}

func TestCleanup_OldEntries(t *testing.T) {
	ls := NewLimiterService(config.RateLimitConfig{
		Enabled: true,
		Default: config.RateLimitDefaults{RPS: 10},
	})

	ls.Allow("ip-old")

	// Искусственно старим запись
	ls.mu.Lock()
	if bucket, exists := ls.clients["ip-old"]; exists {
		bucket.lastUpdated = time.Now().Add(-15 * time.Minute)
	}
	ls.mu.Unlock()

	ls.mu.Lock()
	for ip, bucket := range ls.clients {
		if time.Since(bucket.lastUpdated) > 10*time.Minute {
			delete(ls.clients, ip)
		}
	}
	ls.mu.Unlock()

	assert.Nil(t, ls.GetStats("ip-old"))
}

func TestCleanup_FreshEntries(t *testing.T) {
	ls := NewLimiterService(config.RateLimitConfig{
		Enabled: true,
		Default: config.RateLimitDefaults{RPS: 10},
	})

	ls.Allow("ip-fresh")

	ls.mu.Lock()
	for ip, bucket := range ls.clients {
		if time.Since(bucket.lastUpdated) > 10*time.Minute {
			delete(ls.clients, ip)
		}
	}
	ls.mu.Unlock()

	assert.NotNil(t, ls.GetStats("ip-fresh"))
}

func TestConcurrentAllow(t *testing.T) {
	ls := NewLimiterService(config.RateLimitConfig{
		Enabled: true,
		Default: config.RateLimitDefaults{RPS: 100},
	})

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				ls.Allow("concurrent-ip")
			}
		}()
	}

	wg.Wait()

	stats := ls.GetStats("concurrent-ip")
	assert.NotNil(t, stats)
}

func TestConcurrentIncrementDecrement(t *testing.T) {
	ls := NewLimiterService(config.RateLimitConfig{
		Enabled: true,
		Default: config.RateLimitDefaults{Connections: 50},
	})

	var wg sync.WaitGroup
	for i := 0; i < 25; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ls.IncrementConnections("concurrent-conn")
		}()
	}

	wg.Wait()

	stats := ls.GetStats("concurrent-conn")
	assert.NotNil(t, stats)
	assert.Equal(t, 25, stats["connections"])

	for i := 0; i < 25; i++ {
		ls.DecrementConnections("concurrent-conn")
	}

	stats = ls.GetStats("concurrent-conn")
	assert.Equal(t, 0, stats["connections"])
}

func TestConcurrentGetStats(t *testing.T) {
	ls := NewLimiterService(config.RateLimitConfig{
		Enabled: true,
		Default: config.RateLimitDefaults{RPS: 100},
	})

	ls.Allow("stats-ip")

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			stats := ls.GetStats("stats-ip")
			assert.NotNil(t, stats)
		}()
	}

	wg.Wait()
}

func TestAllow_MaxTokensReset(t *testing.T) {
	ls := NewLimiterService(config.RateLimitConfig{
		Enabled: true,
		Default: config.RateLimitDefaults{RPS: 3},
	})

	ip := "reset-ip"

	// Используем все токены
	ls.Allow(ip)
	ls.Allow(ip)
	ls.Allow(ip)

	time.Sleep(1100 * time.Millisecond)

	assert.True(t, ls.Allow(ip))
	assert.True(t, ls.Allow(ip))
	assert.True(t, ls.Allow(ip))
	assert.False(t, ls.Allow(ip))
}

func TestLimiterService_CleanupMethod(t *testing.T) {
	ls := NewLimiterService(config.RateLimitConfig{
		Enabled: true,
		Default: config.RateLimitDefaults{RPS: 10},
	})

	ls.Allow("cleanup-test")

	ls.Cleanup()

	stats := ls.GetStats("cleanup-test")
	assert.NotNil(t, stats)

}

func TestAllow_PartialTokenRefill(t *testing.T) {
	ls := NewLimiterService(config.RateLimitConfig{
		Enabled: true,
		Default: config.RateLimitDefaults{RPS: 10},
	})

	ip := "partial-ip"

	for i := 0; i < 5; i++ {
		ls.Allow(ip)
	}

	time.Sleep(500 * time.Millisecond)

	passed := 0
	for i := 0; i < 10; i++ {
		if ls.Allow(ip) {
			passed++
		}
	}

	assert.GreaterOrEqual(t, passed, 3)
}

func TestIncrementConnections_NewClient(t *testing.T) {
	ls := NewLimiterService(config.RateLimitConfig{
		Enabled: true,
		Default: config.RateLimitDefaults{Connections: 3},
	})

	ip := "new-conn-client"

	assert.True(t, ls.IncrementConnections(ip))

	ls.mu.RLock()
	bucket, exists := ls.clients[ip]
	ls.mu.RUnlock()

	assert.True(t, exists)
	assert.Equal(t, 1, bucket.activeConns)
}

func TestAllow_ZeroRPS(t *testing.T) {
	ls := NewLimiterService(config.RateLimitConfig{
		Enabled: true,
		Default: config.RateLimitDefaults{RPS: 0},
	})

	assert.False(t, ls.Allow("zero-rps"))
}