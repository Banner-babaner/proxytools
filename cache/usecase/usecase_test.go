package usecase

import (
	"net/http"
	"testing"
	"time"

	"github.com/Banner-babaner/proxytools/cache/entity"
	"github.com/Banner-babaner/proxytools/cache/mocks"
	"github.com/Banner-babaner/proxytools/cache/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newTestCacheService(enabled bool, defaultTTL time.Duration) (*CacheService, *mocks.CacheRepository) {
	mockRepo := new(mocks.CacheRepository)

	cfg := entity.CacheConfig{
		Enabled:    enabled,
		DefaultTTL: defaultTTL,
		MaxSize:    100,
	}

	cs := &CacheService{
		repo:       mockRepo,
		enabled:    cfg.Enabled,
		defaultTTL: cfg.DefaultTTL,
		maxSize:    cfg.MaxSize,
		rules:      cfg.Rules,
	}

	return cs, mockRepo
}

func TestNewCacheService(t *testing.T) {
	cfg := entity.CacheConfig{
		Enabled:    true,
		DefaultTTL: 60 * time.Second,
		MaxSize:    100,
	}
	mockRepo := new(mocks.CacheRepository)

	cs := NewCacheService(cfg, func() repository.CacheRepository { return mockRepo })
	assert.NotNil(t, cs)
	assert.True(t, cs.enabled)
}

func TestGetTTL_Enabled_GET(t *testing.T) {
	cs, _ := newTestCacheService(true, 60*time.Second)

	ttl := cs.GetTTL("GET", "/test", "")
	assert.Equal(t, 60*time.Second, ttl)
}

func TestGetTTL_Disabled(t *testing.T) {
	cs, _ := newTestCacheService(false, 60*time.Second)

	ttl := cs.GetTTL("GET", "/test", "")
	assert.Equal(t, time.Duration(0), ttl)
}

func TestGetTTL_NonGET(t *testing.T) {
	cs, _ := newTestCacheService(true, 60*time.Second)

	assert.Equal(t, time.Duration(0), cs.GetTTL("POST", "/test", ""))
	assert.Equal(t, time.Duration(0), cs.GetTTL("PUT", "/test", ""))
	assert.Equal(t, time.Duration(0), cs.GetTTL("DELETE", "/test", ""))
}

func TestGetTTL_CustomRule(t *testing.T) {
	cs, _ := newTestCacheService(true, 60*time.Second)
	cs.rules = []entity.CacheRule{
		{Path: "/api/*", TTL: 0},
		{Path: "/static/*", TTL: 3600 * time.Second},
		{Domain: "cdn.example.com", TTL: 7200 * time.Second},
	}

	assert.Equal(t, time.Duration(0), cs.GetTTL("GET", "/api/users", ""))
	assert.Equal(t, 3600*time.Second, cs.GetTTL("GET", "/static/app.js", ""))
	assert.Equal(t, 7200*time.Second, cs.GetTTL("GET", "/any", "cdn.example.com"))
	assert.Equal(t, 60*time.Second, cs.GetTTL("GET", "/other", "other.com"))
}

func TestGenerateKey(t *testing.T) {
	cs, _ := newTestCacheService(true, 60*time.Second)

	k1 := cs.GenerateKey("GET", "/api/test")
	k2 := cs.GenerateKey("GET", "/api/test")
	k3 := cs.GenerateKey("POST", "/api/test")

	assert.Equal(t, k1, k2)
	assert.NotEqual(t, k1, k3)
	assert.Len(t, k1, 32)
}

func TestGet_Success(t *testing.T) {
	cs, mockRepo := newTestCacheService(true, 60*time.Second)

	expected := &entity.CacheEntry{
		StatusCode: 200,
		Body:       []byte("data"),
		CreatedAt:  time.Now(),
		TTL:        60 * time.Second,
	}
	mockRepo.On("Get", "key1").Return(expected, true)

	entry, ok := cs.Get("key1")
	assert.True(t, ok)
	assert.Equal(t, 200, entry.StatusCode)
}

func TestGet_Expired(t *testing.T) {
	cs, mockRepo := newTestCacheService(true, 60*time.Second)

	expired := &entity.CacheEntry{
		StatusCode: 200,
		Body:       []byte("data"),
		CreatedAt:  time.Now().Add(-120 * time.Second),
		TTL:        60 * time.Second,
	}
	mockRepo.On("Get", "key1").Return(expired, true)
	mockRepo.On("Delete", "key1").Return()

	_, ok := cs.Get("key1")
	assert.False(t, ok)
	mockRepo.AssertCalled(t, "Delete", "key1")
}

func TestGet_NotFound(t *testing.T) {
	cs, mockRepo := newTestCacheService(true, 60*time.Second)

	mockRepo.On("Get", "key1").Return(nil, false)

	_, ok := cs.Get("key1")
	assert.False(t, ok)
}

func TestGet_Disabled(t *testing.T) {
	cs, _ := newTestCacheService(false, 60*time.Second)

	_, ok := cs.Get("key1")
	assert.False(t, ok)
}

func TestSet_Success(t *testing.T) {
	cs, mockRepo := newTestCacheService(true, 60*time.Second)

	headers := http.Header{"Content-Type": []string{"application/json"}}
	mockRepo.On("Set", "key1", mock.AnythingOfType("*entity.CacheEntry")).Return()

	cs.Set("key1", 200, headers, []byte("data"), 60*time.Second, []string{"tag1"})
	mockRepo.AssertCalled(t, "Set", "key1", mock.Anything)
}

func TestSet_Disabled(t *testing.T) {
	cs, mockRepo := newTestCacheService(false, 60*time.Second)

	cs.Set("key1", 200, nil, []byte("data"), 60*time.Second, nil)
	mockRepo.AssertNotCalled(t, "Set", mock.Anything, mock.Anything)
}

func TestSet_ZeroTTL(t *testing.T) {
	cs, mockRepo := newTestCacheService(true, 60*time.Second)

	cs.Set("key1", 200, nil, []byte("data"), 0, nil)
	mockRepo.AssertNotCalled(t, "Set", mock.Anything, mock.Anything)
}

func TestInvalidate_Clear(t *testing.T) {
	cs, mockRepo := newTestCacheService(true, 60*time.Second)

	mockRepo.On("Clear").Return()

	_, err := cs.Invalidate(entity.InvalidateRequest{Clear: true})
	assert.NoError(t, err)
	mockRepo.AssertCalled(t, "Clear")
}

func TestInvalidate_ByKey(t *testing.T) {
	cs, mockRepo := newTestCacheService(true, 60*time.Second)

	mockRepo.On("Delete", "key1").Return()

	count, err := cs.Invalidate(entity.InvalidateRequest{Key: "key1"})
	assert.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestInvalidate_ByPrefix(t *testing.T) {
	cs, mockRepo := newTestCacheService(true, 60*time.Second)

	mockRepo.On("DeleteByPrefix", "pref:").Return(5)

	count, err := cs.Invalidate(entity.InvalidateRequest{Prefix: "pref:"})
	assert.NoError(t, err)
	assert.Equal(t, 5, count)
}

func TestInvalidate_ByTag(t *testing.T) {
	cs, mockRepo := newTestCacheService(true, 60*time.Second)

	mockRepo.On("DeleteByTag", "tagA").Return(3)

	count, err := cs.Invalidate(entity.InvalidateRequest{Tags: []string{"tagA"}})
	assert.NoError(t, err)
	assert.Equal(t, 3, count)
}

func TestInvalidate_ByPattern(t *testing.T) {
	cs, mockRepo := newTestCacheService(true, 60*time.Second)

	mockRepo.On("DeleteByPattern", "^GET:/api/.*").Return(2, nil)

	count, err := cs.Invalidate(entity.InvalidateRequest{Pattern: "^GET:/api/.*"})
	assert.NoError(t, err)
	assert.Equal(t, 2, count)
}

func TestInvalidate_NoCriteria(t *testing.T) {
	cs, _ := newTestCacheService(true, 60*time.Second)

	_, err := cs.Invalidate(entity.InvalidateRequest{})
	assert.Error(t, err)
}

func TestStats(t *testing.T) {
	cs, mockRepo := newTestCacheService(true, 60*time.Second)

	expected := entity.CacheStats{Entries: 10, SizeMB: 5.5, MaxSize: 100, Enabled: true}
	mockRepo.On("Stats").Return(expected)

	stats := cs.Stats()
	assert.Equal(t, 10, stats.Entries)
	assert.Equal(t, 5.5, stats.SizeMB)
}

func TestMatchPath(t *testing.T) {
	assert.True(t, matchPath("/api/*", "/api/users"))
	assert.True(t, matchPath("/api/*", "/api/"))
	assert.True(t, matchPath("/exact", "/exact"))
	assert.False(t, matchPath("/api/*", "/other/path"))
	assert.False(t, matchPath("/exact", "/exact/extra"))
}