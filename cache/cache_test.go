package cache

import (
	"net/http"
	"testing"
	"time"

	"github.com/Banner-babaner/proxytools/config"
	"github.com/stretchr/testify/assert"
)

func TestNewCacheService(t *testing.T) {
	cfg := config.CacheConfig{
		Enabled:    true,
		DefaultTTL: 60,
		MaxSize:    10,
	}

	cs := NewCacheService(cfg)
	assert.NotNil(t, cs)
	assert.True(t, cs.enabled)
}

func TestGetTTLForPath_GET(t *testing.T) {
	cs := NewCacheService(config.CacheConfig{DefaultTTL: 60, Enabled: true})

	ttl := cs.GetTTLForPath("GET", "/test", "")
	assert.Equal(t, 60*time.Second, ttl)
}

func TestGetTTLForPath_POST(t *testing.T) {
	cs := NewCacheService(config.CacheConfig{DefaultTTL: 60, Enabled: true})

	assert.Equal(t, time.Duration(0), cs.GetTTLForPath("POST", "/test", ""))
	assert.Equal(t, time.Duration(0), cs.GetTTLForPath("PUT", "/test", ""))
	assert.Equal(t, time.Duration(0), cs.GetTTLForPath("DELETE", "/test", ""))
}

func TestGetTTLForPath_CustomRules(t *testing.T) {
	cfg := config.CacheConfig{
		DefaultTTL: 60,
		Rules: []config.CacheRule{
			{Path: "/api/*", TTL: 0},
			{Path: "/static/*", TTL: 3600},
			{Domain: "cdn.example.com", TTL: 7200},
		},
	}
	cs := NewCacheService(cfg)

	assert.Equal(t, time.Duration(0), cs.GetTTLForPath("GET", "/api/users", ""))
	assert.Equal(t, 3600*time.Second, cs.GetTTLForPath("GET", "/static/app.js", ""))
	assert.Equal(t, 7200*time.Second, cs.GetTTLForPath("GET", "/any", "cdn.example.com"))
	assert.Equal(t, 60*time.Second, cs.GetTTLForPath("GET", "/other", "other.com"))
}

func TestMatchPath(t *testing.T) {
	assert.True(t, matchPath("/api/*", "/api/users"))
	assert.True(t, matchPath("/api/*", "/api/"))
	assert.True(t, matchPath("/exact", "/exact"))
	assert.False(t, matchPath("/api/*", "/other"))
}

func TestSetAndGet(t *testing.T) {
	cs := NewCacheService(config.CacheConfig{Enabled: true, DefaultTTL: 60, MaxSize: 0})

	body := []byte(`{"test": "data"}`)
	headers := http.Header{"Content-Type": []string{"application/json"}}

	cs.Set("key1", 200, headers, body, 10*time.Second, []string{"tag1"})

	entry, ok := cs.Get("key1")
	assert.True(t, ok, "entry should exist in cache")
	if ok {
		assert.Equal(t, 200, entry.StatusCode)
		assert.Equal(t, body, entry.Body)
		assert.Equal(t, "application/json", entry.Headers.Get("Content-Type"))
		assert.Contains(t, entry.Tags, "tag1")
	}
}

func TestGet_Expired(t *testing.T) {
	cs := NewCacheService(config.CacheConfig{Enabled: true, MaxSize: 0})
	cs.Set("key1", 200, nil, []byte("test-data"), 1*time.Millisecond, nil)

	time.Sleep(10 * time.Millisecond)

	_, ok := cs.Get("key1")
	assert.False(t, ok)
}

func TestGet_NotFound(t *testing.T) {
	cs := NewCacheService(config.CacheConfig{Enabled: true})

	_, ok := cs.Get("nonexistent")
	assert.False(t, ok)
}

func TestGet_Disabled(t *testing.T) {
	cs := NewCacheService(config.CacheConfig{Enabled: false})
	cs.Set("key1", 200, nil, []byte("test-data"), 60*time.Second, nil)

	_, ok := cs.Get("key1")
	assert.False(t, ok)
}

func TestDelete(t *testing.T) {
	cs := NewCacheService(config.CacheConfig{Enabled: true, MaxSize: 0})
	cs.Set("key1", 200, nil, []byte("test-data"), 60*time.Second, nil)

	cs.Delete("key1")
	_, ok := cs.Get("key1")
	assert.False(t, ok)
}

func TestDeleteByPrefix(t *testing.T) {
	cs := NewCacheService(config.CacheConfig{Enabled: true, MaxSize: 0})

	cs.Set("pref:a", 200, nil, []byte("data-a"), 60*time.Second, nil)
	cs.Set("pref:b", 200, nil, []byte("data-b"), 60*time.Second, nil)
	cs.Set("other", 200, nil, []byte("data-c"), 60*time.Second, nil)

	// Проверяем что добавилось
	_, ok1 := cs.Get("pref:a")
	_, ok2 := cs.Get("pref:b")
	_, ok3 := cs.Get("other")
	assert.True(t, ok1)
	assert.True(t, ok2)
	assert.True(t, ok3)

	count := cs.DeleteByPrefix("pref:")
	assert.Equal(t, 2, count)

	_, ok := cs.Get("other")
	assert.True(t, ok, "other should still exist")
}

func TestDeleteByTag(t *testing.T) {
	cs := NewCacheService(config.CacheConfig{Enabled: true, MaxSize: 0})

	cs.Set("k1", 200, nil, []byte("data-1"), 60*time.Second, []string{"tagA"})
	cs.Set("k2", 200, nil, []byte("data-2"), 60*time.Second, []string{"tagB"})
	cs.Set("k3", 200, nil, []byte("data-3"), 60*time.Second, []string{"tagA", "tagB"})

	// Проверяем что добавилось
	_, ok1 := cs.Get("k1")
	_, ok2 := cs.Get("k2")
	_, ok3 := cs.Get("k3")
	assert.True(t, ok1)
	assert.True(t, ok2)
	assert.True(t, ok3)

	count := cs.DeleteByTag("tagA")
	assert.Equal(t, 2, count)

	_, ok := cs.Get("k2")
	assert.True(t, ok, "k2 should still exist")
}

func TestClear(t *testing.T) {
	cs := NewCacheService(config.CacheConfig{Enabled: true, MaxSize: 0})
	cs.Set("k1", 200, nil, []byte("data-1"), 60*time.Second, nil)
	cs.Set("k2", 200, nil, []byte("data-2"), 60*time.Second, nil)

	cs.Clear()

	_, ok := cs.Get("k1")
	assert.False(t, ok)
}

func TestGenerateKey(t *testing.T) {
	cs := NewCacheService(config.CacheConfig{})

	k1 := cs.GenerateKey("GET", "/api/test")
	k2 := cs.GenerateKey("GET", "/api/test")
	k3 := cs.GenerateKey("POST", "/api/test")

	assert.Equal(t, k1, k2)
	assert.NotEqual(t, k1, k3)
	assert.Len(t, k1, 32)
}

func TestSizeFilter_Large(t *testing.T) {
	cs := NewCacheService(config.CacheConfig{Enabled: true, MaxSize: 1})

	large := make([]byte, 200*1024) // 200KB > 10% от 1MB
	cs.Set("large", 200, nil, large, 60*time.Second, nil)
	_, ok := cs.Get("large")
	assert.False(t, ok)
}

func TestCacheResponseWriter(t *testing.T) {
	w := &mockResponseWriter{header: make(http.Header)}
	crw := NewCacheResponseWriter(w)

	crw.Header().Set("X-Test", "value")
	crw.WriteHeader(201)
	crw.Write([]byte("response body"))

	assert.Equal(t, 201, crw.StatusCode)
	assert.Equal(t, "response body", string(crw.BodyBytes()))
	assert.Equal(t, "value", crw.Header().Get("X-Test"))
}

type mockResponseWriter struct {
	header http.Header
	body   []byte
	status int
}

func (m *mockResponseWriter) Header() http.Header         { return m.header }
func (m *mockResponseWriter) Write(d []byte) (int, error) { m.body = append(m.body, d...); return len(d), nil }
func (m *mockResponseWriter) WriteHeader(s int)           { m.status = s }

func TestInvalidateByPattern(t *testing.T) {
	cs := NewCacheService(config.CacheConfig{Enabled: true, MaxSize: 0})

	cs.Set("GET:/api/users", 200, nil, []byte("users-data"), 60*time.Second, nil)
	cs.Set("GET:/api/products", 200, nil, []byte("products-data"), 60*time.Second, nil)
	cs.Set("GET:/static/app.js", 200, nil, []byte("app-data"), 60*time.Second, nil)

	// Проверяем что добавилось
	_, ok := cs.Get("GET:/api/users")
	assert.True(t, ok)

	count, err := cs.InvalidateByPattern("^GET:/api/.*")
	assert.NoError(t, err)
	assert.Equal(t, 2, count)

	_, ok = cs.Get("GET:/static/app.js")
	assert.True(t, ok, "static should still exist")
}

func TestInvalidateExpired(t *testing.T) {
	cs := NewCacheService(config.CacheConfig{Enabled: true, MaxSize: 0})

	cs.Set("fresh", 200, nil, []byte("fresh-data"), 60*time.Second, nil)
	cs.Set("stale", 200, nil, []byte("stale-data"), 1*time.Millisecond, nil)

	// Проверяем что добавилось
	_, ok := cs.Get("fresh")
	assert.True(t, ok)

	time.Sleep(10 * time.Millisecond)

	count := cs.InvalidateExpired()
	assert.Equal(t, 1, count)

	_, ok = cs.Get("fresh")
	assert.True(t, ok)
	_, ok = cs.Get("stale")
	assert.False(t, ok)
}