package usecase

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/Banner-babaner/proxytools/ipfilter/entity"
	"github.com/Banner-babaner/proxytools/ipfilter/mocks"
	"github.com/Banner-babaner/proxytools/ipfilter/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newTestFilterService(cfg entity.IPFilterConfig) (*FilterService, *mocks.IPListRepository, *mocks.IPCache) {
	mockRepo := new(mocks.IPListRepository)
	mockCache := new(mocks.IPCache)

	mockRepo.On("Insert", mock.Anything, mock.Anything).Return(nil)

	fs := &FilterService{
		builder:       func() repository.IPListRepository { return mockRepo },
		repo:          mockRepo,
		cache:         mockCache,
		defaultPolicy: cfg.DefaultPolicy,
		lists:         cfg.Lists,
	}

	return fs, mockRepo, mockCache
}

func TestNewFilterService(t *testing.T) {
	cfg := entity.IPFilterConfig{
		DefaultPolicy: "deny",
		Lists: entity.ListsConfig{
			Whitelist: []string{"192.168.1.0/24"},
			Blacklist: []string{"10.0.0.0/8"},
			Graylist:  []string{"172.16.0.0/12"},
		},
		Cache: entity.CacheConfig{Enabled: false},
	}

	mockRepo := new(mocks.IPListRepository)
	mockRepo.On("Insert", mock.Anything, mock.Anything).Return(nil)

	fs, err := NewFilterService(cfg,
		func() repository.IPListRepository { return mockRepo },
		nil,
	)
	assert.NoError(t, err)
	assert.NotNil(t, fs)
	assert.Equal(t, "deny", fs.defaultPolicy)
}

func TestNewFilterService_WithCache(t *testing.T) {
	cfg := entity.IPFilterConfig{
		DefaultPolicy: "deny",
		Cache:         entity.CacheConfig{Enabled: true, TTL: 60, MaxSize: 1000},
	}

	mockRepo := new(mocks.IPListRepository)
	mockRepo.On("Insert", mock.Anything, mock.Anything).Return(nil)
	mockCache := new(mocks.IPCache)

	fs, err := NewFilterService(cfg,
		func() repository.IPListRepository { return mockRepo },
		func(maxSize int, ttlSeconds time.Duration) (repository.IPCache, error) {
			return mockCache, nil
		},
	)
	assert.NoError(t, err)
	assert.NotNil(t, fs.cache)
}

func TestNewFilterService_WithoutCache(t *testing.T) {
	cfg := entity.IPFilterConfig{
		DefaultPolicy: "allow",
		Cache:         entity.CacheConfig{Enabled: false},
	}

	mockRepo := new(mocks.IPListRepository)
	mockRepo.On("Insert", mock.Anything, mock.Anything).Return(nil)

	fs, err := NewFilterService(cfg,
		func() repository.IPListRepository { return mockRepo },
		nil,
	)
	assert.NoError(t, err)
	assert.Nil(t, fs.cache)
}


func TestCheckAccess_Whitelist(t *testing.T) {
	fs, mockRepo, mockCache := newTestFilterService(entity.IPFilterConfig{DefaultPolicy: "deny"})

	mockCache.On("Get", "192.168.1.100").Return(entity.ListType(0), false, false)
	mockRepo.On("Search", "192.168.1.100").Return(entity.Whitelist, true)
	mockCache.On("Set", "192.168.1.100", entity.Whitelist, true).Return()

	result := fs.CheckAccess("192.168.1.100")
	assert.Equal(t, entity.Allowed, result)
}

func TestCheckAccess_Blacklist(t *testing.T) {
	fs, mockRepo, mockCache := newTestFilterService(entity.IPFilterConfig{DefaultPolicy: "allow"})

	mockCache.On("Get", "10.0.0.5").Return(entity.ListType(0), false, false)
	mockRepo.On("Search", "10.0.0.5").Return(entity.Blacklist, true)
	mockCache.On("Set", "10.0.0.5", entity.Blacklist, true).Return()

	result := fs.CheckAccess("10.0.0.5")
	assert.Equal(t, entity.Denied, result)
}

func TestCheckAccess_Graylist(t *testing.T) {
	fs, mockRepo, mockCache := newTestFilterService(entity.IPFilterConfig{DefaultPolicy: "deny"})

	mockCache.On("Get", "172.16.0.1").Return(entity.ListType(0), false, false)
	mockRepo.On("Search", "172.16.0.1").Return(entity.Graylist, true)
	mockCache.On("Set", "172.16.0.1", entity.Graylist, true).Return()

	result := fs.CheckAccess("172.16.0.1")
	assert.Equal(t, entity.CaptchaRequired, result)
}

func TestCheckAccess_DefaultDeny(t *testing.T) {
	fs, mockRepo, mockCache := newTestFilterService(entity.IPFilterConfig{DefaultPolicy: "deny"})

	mockCache.On("Get", "1.1.1.1").Return(entity.ListType(0), false, false)
	mockRepo.On("Search", "1.1.1.1").Return(entity.ListType(0), false)
	mockCache.On("Set", "1.1.1.1", entity.ListType(0), false).Return()

	result := fs.CheckAccess("1.1.1.1")
	assert.Equal(t, entity.Denied, result)
}

func TestCheckAccess_DefaultAllow(t *testing.T) {
	fs, mockRepo, mockCache := newTestFilterService(entity.IPFilterConfig{DefaultPolicy: "allow"})

	mockCache.On("Get", "1.1.1.1").Return(entity.ListType(0), false, false)
	mockRepo.On("Search", "1.1.1.1").Return(entity.ListType(0), false)
	mockCache.On("Set", "1.1.1.1", entity.ListType(0), false).Return()

	result := fs.CheckAccess("1.1.1.1")
	assert.Equal(t, entity.Allowed, result)
}

func TestCheckAccess_BlacklistPriority(t *testing.T) {
	fs, mockRepo, mockCache := newTestFilterService(entity.IPFilterConfig{DefaultPolicy: "deny"})

	mockCache.On("Get", "10.0.0.1").Return(entity.ListType(0), false, false)
	mockRepo.On("Search", "10.0.0.1").Return(entity.Blacklist, true)
	mockCache.On("Set", "10.0.0.1", entity.Blacklist, true).Return()

	result := fs.CheckAccess("10.0.0.1")
	assert.Equal(t, entity.Denied, result)
}

func TestCheckAccess_CacheHit(t *testing.T) {
	fs, _, mockCache := newTestFilterService(entity.IPFilterConfig{DefaultPolicy: "deny"})

	mockCache.On("Get", "10.0.0.1").Return(entity.Blacklist, true, true)

	result := fs.CheckAccess("10.0.0.1")
	assert.Equal(t, entity.Denied, result)
}

func TestCheckAccess_CacheMissThenHit(t *testing.T) {
	cfg := entity.IPFilterConfig{DefaultPolicy: "deny"}
	mockRepo := new(mocks.IPListRepository)
	mockCache := new(mocks.IPCache)

	mockRepo.On("Insert", mock.Anything, mock.Anything).Return(nil)

	fs := &FilterService{
		builder:       func() repository.IPListRepository { return mockRepo },
		repo:          mockRepo,
		cache:         mockCache,
		defaultPolicy: cfg.DefaultPolicy,
	}

	mockCache.On("Get", "192.168.1.1").Return(entity.ListType(0), false, false).Once()
	mockRepo.On("Search", "192.168.1.1").Return(entity.Whitelist, true).Once()
	mockCache.On("Set", "192.168.1.1", entity.Whitelist, true).Return().Once()

	result := fs.CheckAccess("192.168.1.1")
	assert.Equal(t, entity.Allowed, result)

	mockCache.On("Get", "192.168.1.1").Return(entity.Whitelist, true, true).Once()

	result = fs.CheckAccess("192.168.1.1")
	assert.Equal(t, entity.Allowed, result)

	mockRepo.AssertNumberOfCalls(t, "Search", 1)
}


func TestAddIP_Blacklist(t *testing.T) {
	fs, mockRepo, mockCache := newTestFilterService(entity.IPFilterConfig{DefaultPolicy: "allow"})

	mockRepo.On("Insert", "5.5.5.5", entity.Blacklist).Return(nil)
	mockCache.On("Remove", "5.5.5.5").Return()

	err := fs.AddIP("5.5.5.5", "blacklist")
	assert.NoError(t, err)
	assert.Contains(t, fs.lists.Blacklist, "5.5.5.5")
}

func TestAddIP_Whitelist(t *testing.T) {
	fs, mockRepo, mockCache := newTestFilterService(entity.IPFilterConfig{DefaultPolicy: "deny"})

	mockRepo.On("Insert", "10.0.0.1", entity.Whitelist).Return(nil)
	mockCache.On("Remove", "10.0.0.1").Return()

	err := fs.AddIP("10.0.0.1", "whitelist")
	assert.NoError(t, err)
	assert.Contains(t, fs.lists.Whitelist, "10.0.0.1")
}

func TestAddIP_Graylist(t *testing.T) {
	fs, mockRepo, mockCache := newTestFilterService(entity.IPFilterConfig{DefaultPolicy: "deny"})

	mockRepo.On("Insert", "172.16.0.1", entity.Graylist).Return(nil)
	mockCache.On("Remove", "172.16.0.1").Return()

	err := fs.AddIP("172.16.0.1", "graylist")
	assert.NoError(t, err)
	assert.Contains(t, fs.lists.Graylist, "172.16.0.1")
}

func TestAddIP_InvalidType(t *testing.T) {
	fs, _, _ := newTestFilterService(entity.IPFilterConfig{})

	err := fs.AddIP("5.5.5.5", "purplelist")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown list type")
}


func TestRemoveIP_Blacklist(t *testing.T) {
	cfg := entity.IPFilterConfig{
		DefaultPolicy: "allow",
		Lists: entity.ListsConfig{
			Blacklist: []string{"1.2.3.4", "5.6.7.8"},
		},
	}
	mockRepo := new(mocks.IPListRepository)
	mockCache := new(mocks.IPCache)

	mockRepo.On("Insert", mock.Anything, mock.Anything).Return(nil)

	fs := &FilterService{
		builder:       func() repository.IPListRepository { return mockRepo },
		repo:          mockRepo,
		cache:         mockCache,
		defaultPolicy: cfg.DefaultPolicy,
		lists:         cfg.Lists,
	}

	mockRepo.On("Insert", mock.Anything, mock.Anything).Return(nil)
	mockCache.On("Remove", "1.2.3.4").Return()

	fs.RemoveIP("1.2.3.4", "blacklist")

	assert.NotContains(t, fs.lists.Blacklist, "1.2.3.4")
	assert.Contains(t, fs.lists.Blacklist, "5.6.7.8")
}

func TestRemoveIP_Whitelist(t *testing.T) {
	cfg := entity.IPFilterConfig{
		DefaultPolicy: "deny",
		Lists:         entity.ListsConfig{Whitelist: []string{"192.168.1.1"}},
	}
	mockRepo := new(mocks.IPListRepository)
	mockCache := new(mocks.IPCache)

	mockRepo.On("Insert", mock.Anything, mock.Anything).Return(nil)
	mockCache.On("Remove", "192.168.1.1").Return()

	fs := &FilterService{
		builder:       func() repository.IPListRepository { return mockRepo },
		repo:          mockRepo,
		cache:         mockCache,
		defaultPolicy: cfg.DefaultPolicy,
		lists:         cfg.Lists,
	}

	fs.RemoveIP("192.168.1.1", "whitelist")
	assert.Empty(t, fs.lists.Whitelist)
}

func TestGetLists(t *testing.T) {
	cfg := entity.IPFilterConfig{
		Lists: entity.ListsConfig{
			Whitelist: []string{"192.168.1.1"},
			Blacklist: []string{"1.2.3.4"},
			Graylist:  []string{"172.16.0.1"},
		},
	}
	mockRepo := new(mocks.IPListRepository)
	mockRepo.On("Insert", mock.Anything, mock.Anything).Return(nil)

	fs := &FilterService{
		builder: func() repository.IPListRepository { return mockRepo },
		repo:    mockRepo,
		lists:   cfg.Lists,
	}

	lists := fs.GetLists()
	assert.Len(t, lists.Whitelist, 1)
	assert.Len(t, lists.Blacklist, 1)
	assert.Len(t, lists.Graylist, 1)
}


func TestConcurrentCheckAccess(t *testing.T) {
	fs, mockRepo, mockCache := newTestFilterService(entity.IPFilterConfig{DefaultPolicy: "deny"})

	mockCache.On("Get", mock.Anything).Return(entity.ListType(0), false, false)
	mockRepo.On("Search", mock.Anything).Return(entity.ListType(0), false)
	mockCache.On("Set", mock.Anything, mock.Anything, mock.Anything).Return()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				fs.CheckAccess("10.1.1.1")
				fs.CheckAccess("192.168.1.1")
				fs.CheckAccess("1.1.1.1")
			}
		}()
	}
	wg.Wait()
}

func TestConcurrentAddAndCheck(t *testing.T) {
	fs, mockRepo, mockCache := newTestFilterService(entity.IPFilterConfig{DefaultPolicy: "deny"})

	mockRepo.On("Insert", mock.Anything, entity.Whitelist).Return(nil)
	mockCache.On("Remove", mock.Anything).Return()
	mockCache.On("Get", mock.Anything).Return(entity.ListType(0), false, false)
	mockRepo.On("Search", mock.Anything).Return(entity.ListType(0), false)
	mockCache.On("Set", mock.Anything, mock.Anything, mock.Anything).Return()

	var wg sync.WaitGroup

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			ip := fmt.Sprintf("10.0.0.%d", n)
			fs.AddIP(ip, "whitelist")
		}(i)
	}

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				fs.CheckAccess("10.0.0.1")
				fs.CheckAccess("1.1.1.1")
			}
		}()
	}
	wg.Wait()
}


func TestDetermineAccess(t *testing.T) {
	fs := &FilterService{defaultPolicy: "deny"}

	assert.Equal(t, entity.Denied, fs.determineAccess(entity.Blacklist, true))
	assert.Equal(t, entity.Allowed, fs.determineAccess(entity.Whitelist, true))
	assert.Equal(t, entity.CaptchaRequired, fs.determineAccess(entity.Graylist, true))
	assert.Equal(t, entity.Denied, fs.determineAccess(0, false))
}

func TestDetermineAccess_DefaultAllow(t *testing.T) {
	fs := &FilterService{defaultPolicy: "allow"}
	assert.Equal(t, entity.Allowed, fs.determineAccess(0, false))
}


func TestRemoveFromSlice(t *testing.T) {
	slice := []string{"a", "b", "c"}
	assert.Equal(t, []string{"a", "c"}, removeFromSlice(slice, "b"))
	assert.Equal(t, slice, removeFromSlice(slice, "x"))
	assert.Empty(t, removeFromSlice([]string{"only"}, "only"))
	assert.Empty(t, removeFromSlice([]string{}, "anything"))
}


func TestAccessResult_Constants(t *testing.T) {
	assert.Equal(t, entity.AccessResult(0), entity.Allowed)
	assert.Equal(t, entity.AccessResult(1), entity.Denied)
	assert.Equal(t, entity.AccessResult(2), entity.CaptchaRequired)
}

func TestListType_Constants(t *testing.T) {
	assert.Equal(t, entity.ListType(0), entity.Whitelist)
	assert.Equal(t, entity.ListType(1), entity.Blacklist)
	assert.Equal(t, entity.ListType(2), entity.Graylist)
}