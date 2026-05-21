// internal/ipfilter/filter_test.go
package ipfilter

import (
	"testing"
	"sync"
	"fmt"
	"time"
	"github.com/Banner-babaner/proxytools/config"
	"github.com/stretchr/testify/assert"
)

func TestNewFilterService(t *testing.T) {
	cfg := config.IPFilterConfig{
		DefaultPolicy: "deny",
		Lists: config.ListsConfig{
			Whitelist: []string{"192.168.1.0/24"},
			Blacklist: []string{"10.0.0.0/8"},
			Graylist:  []string{"172.16.0.0/12"},
		},
	}

	fs, err := NewFilterService(cfg)
	assert.NoError(t, err)
	assert.NotNil(t, fs)
	assert.Equal(t, "deny", fs.defaultPolicy)
}

func TestNewFilterService_WithCache(t *testing.T) {
	cfg := config.IPFilterConfig{
		DefaultPolicy: "deny",
		Cache: struct {
			Enabled bool `mapstructure:"enabled"`
			TTL     int  `mapstructure:"ttl"`
			MaxSize int  `mapstructure:"max_size"`
		}{
			Enabled: true,
			TTL:     60,
			MaxSize: 1000,
		},
	}

	fs, err := NewFilterService(cfg)
	assert.NoError(t, err)
	assert.NotNil(t, fs.cache)
}

func TestNewFilterService_WithoutCache(t *testing.T) {
	cfg := config.IPFilterConfig{
		DefaultPolicy: "allow",
	}

	fs, err := NewFilterService(cfg)
	assert.NoError(t, err)
	assert.Nil(t, fs.cache)
}

func TestFilterService_CheckAccess_Whitelist(t *testing.T) {
	fs := &FilterService{
		trie:          NewIPTrie(),
		defaultPolicy: "deny",
	}
	fs.trie.Insert("192.168.1.100", Whitelist)

	result := fs.CheckAccess("192.168.1.100")
	assert.Equal(t, Allowed, result)
}

func TestFilterService_CheckAccess_Blacklist(t *testing.T) {
	fs := &FilterService{
		trie:          NewIPTrie(),
		defaultPolicy: "allow",
	}
	fs.trie.Insert("10.0.0.5", Blacklist)

	result := fs.CheckAccess("10.0.0.5")
	assert.Equal(t, Denied, result)
}

func TestFilterService_CheckAccess_Graylist(t *testing.T) {
	fs := &FilterService{
		trie:          NewIPTrie(),
		defaultPolicy: "deny",
	}
	fs.trie.Insert("172.16.0.1", Graylist)

	result := fs.CheckAccess("172.16.0.1")
	assert.Equal(t, CaptchaRequired, result)
}

func TestFilterService_CheckAccess_DefaultDeny(t *testing.T) {
	fs := &FilterService{
		trie:          NewIPTrie(),
		defaultPolicy: "deny",
	}

	result := fs.CheckAccess("1.1.1.1")
	assert.Equal(t, Denied, result)
}

func TestFilterService_CheckAccess_DefaultAllow(t *testing.T) {
	fs := &FilterService{
		trie:          NewIPTrie(),
		defaultPolicy: "allow",
	}

	result := fs.CheckAccess("1.1.1.1")
	assert.Equal(t, Allowed, result)
}

func TestFilterService_CheckAccess_BlacklistPriority(t *testing.T) {
	fs := &FilterService{
		trie:          NewIPTrie(),
		defaultPolicy: "deny",
	}
	// IP в обоих списках — blacklist приоритетнее
	fs.trie.Insert("10.0.0.1", Whitelist)
	fs.trie.Insert("10.0.0.1", Blacklist)

	result := fs.CheckAccess("10.0.0.1")
	assert.Equal(t, Denied, result)
}

func TestFilterService_CheckAccess_WithCache(t *testing.T) {
	fs := &FilterService{
		trie:          NewIPTrie(),
		defaultPolicy: "deny",
	}
	fs.trie.Insert("192.168.1.1", Whitelist)

	cache, _ := NewIPCache(100, 60*time.Second)
	fs.cache = cache

	// Первый запрос — из trie
	result := fs.CheckAccess("192.168.1.1")
	assert.Equal(t, Allowed, result)

	// Второй запрос — из кэша
	result = fs.CheckAccess("192.168.1.1")
	assert.Equal(t, Allowed, result)
}

func TestFilterService_CheckAccess_ExpiredCache(t *testing.T) {
	fs := &FilterService{
		trie:          NewIPTrie(),
		defaultPolicy: "deny",
	}
	fs.trie.Insert("10.0.0.1", Blacklist)

	cache, _ := NewIPCache(100, 1*time.Millisecond)
	fs.cache = cache

	// Кэшируем
	result := fs.CheckAccess("10.0.0.1")
	assert.Equal(t, Denied, result)

	// Ждём истечения
	time.Sleep(10 * time.Millisecond)

	// Должен снова пойти в trie
	result = fs.CheckAccess("10.0.0.1")
	assert.Equal(t, Denied, result)
}

func TestFilterService_AddIP_Blacklist(t *testing.T) {
	fs := &FilterService{
		trie:          NewIPTrie(),
		defaultPolicy: "allow",
		lists:         config.ListsConfig{},
	}

	err := fs.AddIP("5.5.5.5", "blacklist")
	assert.NoError(t, err)

	result := fs.CheckAccess("5.5.5.5")
	assert.Equal(t, Denied, result)
	assert.Contains(t, fs.lists.Blacklist, "5.5.5.5")
}

func TestFilterService_AddIP_Whitelist(t *testing.T) {
	fs := &FilterService{
		trie:          NewIPTrie(),
		defaultPolicy: "deny",
		lists:         config.ListsConfig{},
	}

	err := fs.AddIP("10.0.0.1", "whitelist")
	assert.NoError(t, err)

	result := fs.CheckAccess("10.0.0.1")
	assert.Equal(t, Allowed, result)
	assert.Contains(t, fs.lists.Whitelist, "10.0.0.1")
}

func TestFilterService_AddIP_Graylist(t *testing.T) {
	fs := &FilterService{
		trie:          NewIPTrie(),
		defaultPolicy: "deny",
		lists:         config.ListsConfig{},
	}

	err := fs.AddIP("172.16.0.1", "graylist")
	assert.NoError(t, err)

	result := fs.CheckAccess("172.16.0.1")
	assert.Equal(t, CaptchaRequired, result)
	assert.Contains(t, fs.lists.Graylist, "172.16.0.1")
}

func TestFilterService_AddIP_InvalidType(t *testing.T) {
	fs := &FilterService{
		trie: NewIPTrie(),
		lists: config.ListsConfig{},
	}

	err := fs.AddIP("5.5.5.5", "purplelist")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown list type")
}

func TestFilterService_AddIP_InvalidIP(t *testing.T) {
	fs := &FilterService{
		trie: NewIPTrie(),
		lists: config.ListsConfig{},
	}

	err := fs.AddIP("invalid-ip", "blacklist")
	assert.Error(t, err)
}

func TestFilterService_AddIP_WithCacheInvalidation(t *testing.T) {
	fs := &FilterService{
		trie:          NewIPTrie(),
		defaultPolicy: "deny",
		lists:         config.ListsConfig{},
	}

	cache, _ := NewIPCache(100, 60*time.Second)
	fs.cache = cache

	// Кэшируем отсутствие правила
	fs.CheckAccess("1.1.1.1")

	// Добавляем в whitelist
	fs.AddIP("1.1.1.1", "whitelist")

	// Теперь должно быть Allowed (кэш инвалидирован)
	result := fs.CheckAccess("1.1.1.1")
	assert.Equal(t, Allowed, result)
}

func TestFilterService_GetLists(t *testing.T) {
	fs := &FilterService{
		trie: NewIPTrie(),
		lists: config.ListsConfig{
			Whitelist: []string{"192.168.1.1", "10.0.0.1"},
			Blacklist: []string{"1.2.3.4"},
			Graylist:  []string{"172.16.0.1", "172.16.0.2"},
		},
	}

	lists := fs.GetLists()
	assert.Len(t, lists.Whitelist, 2)
	assert.Len(t, lists.Blacklist, 1)
	assert.Len(t, lists.Graylist, 2)
}

func TestFilterService_RemoveIP_Blacklist(t *testing.T) {
	fs := &FilterService{
		trie:          NewIPTrie(),
		defaultPolicy: "allow",
		lists: config.ListsConfig{
			Blacklist: []string{"1.2.3.4", "5.6.7.8"},
		},
	}
	fs.trie.Insert("1.2.3.4", Blacklist)
	fs.trie.Insert("5.6.7.8", Blacklist)

	// До удаления
	assert.Equal(t, Denied, fs.CheckAccess("1.2.3.4"))

	fs.RemoveIP("1.2.3.4", "blacklist")

	// После удаления — default allow
	assert.Equal(t, Allowed, fs.CheckAccess("1.2.3.4"))
	assert.NotContains(t, fs.lists.Blacklist, "1.2.3.4")
	assert.Contains(t, fs.lists.Blacklist, "5.6.7.8") // второй остался
}

func TestFilterService_RemoveIP_Whitelist(t *testing.T) {
	fs := &FilterService{
		trie:          NewIPTrie(),
		defaultPolicy: "deny",
		lists: config.ListsConfig{
			Whitelist: []string{"192.168.1.1"},
		},
	}
	fs.trie.Insert("192.168.1.1", Whitelist)

	assert.Equal(t, Allowed, fs.CheckAccess("192.168.1.1"))

	fs.RemoveIP("192.168.1.1", "whitelist")

	assert.Equal(t, Denied, fs.CheckAccess("192.168.1.1"))
	assert.Empty(t, fs.lists.Whitelist)
}

func TestFilterService_RemoveIP_Graylist(t *testing.T) {
	fs := &FilterService{
		trie:          NewIPTrie(),
		defaultPolicy: "deny",
		lists: config.ListsConfig{
			Graylist: []string{"172.16.0.1"},
		},
	}
	fs.trie.Insert("172.16.0.1", Graylist)

	assert.Equal(t, CaptchaRequired, fs.CheckAccess("172.16.0.1"))

	fs.RemoveIP("172.16.0.1", "graylist")

	assert.Equal(t, Denied, fs.CheckAccess("172.16.0.1"))
	assert.Empty(t, fs.lists.Graylist)
}

func TestFilterService_RemoveIP_NonExistent(t *testing.T) {
	fs := &FilterService{
		trie: NewIPTrie(),
		lists: config.ListsConfig{
			Blacklist: []string{"1.2.3.4"},
		},
	}

	// Не паникует при удалении несуществующего
	assert.NotPanics(t, func() {
		fs.RemoveIP("9.9.9.9", "blacklist")
	})
}

func TestFilterService_LoadLists(t *testing.T) {
	fs := &FilterService{
		trie:          NewIPTrie(),
		defaultPolicy: "deny",
	}

	lists := config.ListsConfig{
		Whitelist: []string{"10.0.0.0/8"},
		Blacklist: []string{"1.2.3.4"},
		Graylist:  []string{"172.16.0.0/12"},
	}

	fs.loadLists(lists)

	assert.Equal(t, Allowed, fs.CheckAccess("10.1.1.1"))
	assert.Equal(t, Denied, fs.CheckAccess("1.2.3.4"))
	assert.Equal(t, CaptchaRequired, fs.CheckAccess("172.16.1.1"))
}

func TestFilterService_ConcurrentCheckAccess(t *testing.T) {
	fs := &FilterService{
		trie:          NewIPTrie(),
		defaultPolicy: "deny",
	}
	fs.trie.Insert("10.0.0.0/8", Blacklist)
	fs.trie.Insert("192.168.0.0/16", Whitelist)

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

func TestFilterService_ConcurrentAddAndCheck(t *testing.T) {
	fs := &FilterService{
		trie:          NewIPTrie(),
		defaultPolicy: "deny",
		lists:         config.ListsConfig{},
	}

	var wg sync.WaitGroup

	// Писатели
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			ip := fmt.Sprintf("10.0.0.%d", n)
			fs.AddIP(ip, "whitelist")
		}(i)
	}

	// Читатели
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

	assert.Equal(t, Denied, fs.determineAccess(Blacklist, true))
	assert.Equal(t, Allowed, fs.determineAccess(Whitelist, true))
	assert.Equal(t, CaptchaRequired, fs.determineAccess(Graylist, true))
	assert.Equal(t, Denied, fs.determineAccess(0, false)) // default deny
}

func TestDetermineAccess_DefaultAllow(t *testing.T) {
	fs := &FilterService{defaultPolicy: "allow"}

	assert.Equal(t, Allowed, fs.determineAccess(0, false)) // default allow
}

func TestRemoveFromSlice(t *testing.T) {
	// Существующий элемент
	slice := []string{"a", "b", "c"}
	result := removeFromSlice(slice, "b")
	assert.Equal(t, []string{"a", "c"}, result)

	// Несуществующий элемент
	result = removeFromSlice(slice, "x")
	assert.Equal(t, slice, result)

	// Единственный элемент
	result = removeFromSlice([]string{"only"}, "only")
	assert.Empty(t, result)

	// Пустой слайс
	result = removeFromSlice([]string{}, "anything")
	assert.Empty(t, result)
}

func TestAccessResult_Constants(t *testing.T) {
	assert.Equal(t, AccessResult(0), Allowed)
	assert.Equal(t, AccessResult(1), Denied)
	assert.Equal(t, AccessResult(2), CaptchaRequired)
}

func TestListType_Constants(t *testing.T) {
	assert.Equal(t, ListType(0), Whitelist)
	assert.Equal(t, ListType(1), Blacklist)
	assert.Equal(t, ListType(2), Graylist)
}