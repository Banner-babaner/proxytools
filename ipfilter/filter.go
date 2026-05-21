// internal/ipfilter/filter.go
package ipfilter

import (
    "fmt"
    "sync"
    "time"
	"github.com/Banner-babaner/proxytools/config"
    "github.com/Banner-babaner/proxytools/logger"
)

type AccessResult int

const (
	Allowed AccessResult = iota
	Denied
	CaptchaRequired
)

type FilterService struct {
	mu            sync.RWMutex
	trie          *IPTrie
	cache         *IPCache
	defaultPolicy string
	lists         config.ListsConfig
}

func NewFilterService(cfg config.IPFilterConfig) (*FilterService, error) {
	fs := &FilterService{
		trie:          NewIPTrie(),
		defaultPolicy: cfg.DefaultPolicy,
		lists:         cfg.Lists,
	}

	if cfg.Cache.Enabled {
		cache, err := NewIPCache(cfg.Cache.MaxSize, time.Duration(cfg.Cache.TTL)*time.Second)
		if err != nil {
			return nil, err
		}
		fs.cache = cache
	}

	fs.loadLists(cfg.Lists)

	return fs, nil
}

// loadLists НЕ блокирует мьютекс — вызывающий должен сам это сделать
func (fs *FilterService) loadListsNoLock(lists config.ListsConfig) {
	fs.trie = NewIPTrie()

	for _, ip := range lists.Blacklist {
		fs.trie.Insert(ip, Blacklist)
	}

	for _, ip := range lists.Whitelist {
		fs.trie.Insert(ip, Whitelist)
	}

	for _, ip := range lists.Graylist {
		fs.trie.Insert(ip, Graylist)
	}

	logger.Info().
		Int("blacklist", len(lists.Blacklist)).
		Int("whitelist", len(lists.Whitelist)).
		Int("graylist", len(lists.Graylist)).
		Msg("IP lists loaded")
}

// loadLists с блокировкой
func (fs *FilterService) loadLists(lists config.ListsConfig) {
	fs.loadListsNoLock(lists)
}

// CheckAccess проверяет доступ для IP
func (fs *FilterService) CheckAccess(ip string) AccessResult {
	if fs.cache != nil {
		if listType, hasRule, found := fs.cache.Get(ip); found {
			return fs.determineAccess(listType, hasRule)
		}
	}

	fs.mu.RLock()
	listType, hasRule := fs.trie.Search(ip)
	fs.mu.RUnlock()

	if fs.cache != nil {
		fs.cache.Set(ip, listType, hasRule)
	}

	return fs.determineAccess(listType, hasRule)
}

func (fs *FilterService) determineAccess(listType ListType, hasRule bool) AccessResult {
	if hasRule {
		switch listType {
		case Blacklist:
			return Denied
		case Whitelist:
			return Allowed
		case Graylist:
			return CaptchaRequired
		}
	}

	if fs.defaultPolicy == "allow" {
		return Allowed
	}
	return Denied
}

// AddIP добавляет IP в список
func (fs *FilterService) AddIP(ip string, listType string) error {
	var lt ListType
	switch listType {
	case "whitelist":
		lt = Whitelist
		fs.mu.Lock()
		fs.lists.Whitelist = append(fs.lists.Whitelist, ip)
		fs.mu.Unlock()
	case "blacklist":
		lt = Blacklist
		fs.mu.Lock()
		fs.lists.Blacklist = append(fs.lists.Blacklist, ip)
		fs.mu.Unlock()
	case "graylist":
		lt = Graylist
		fs.mu.Lock()
		fs.lists.Graylist = append(fs.lists.Graylist, ip)
		fs.mu.Unlock()
	default:
		return fmt.Errorf("unknown list type: %s", listType)
	}

	fs.mu.Lock()
	if err := fs.trie.Insert(ip, lt); err != nil {
		fs.mu.Unlock()
		return err
	}
	fs.mu.Unlock()

	if fs.cache != nil {
		fs.cache.cache.Remove(ip)
	}

	return nil
}

// RemoveIP удаляет IP из списка
func (fs *FilterService) RemoveIP(ip string, listType string) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	// Удаляем из конфигурации
	switch listType {
	case "whitelist":
		fs.lists.Whitelist = removeFromSlice(fs.lists.Whitelist, ip)
	case "blacklist":
		fs.lists.Blacklist = removeFromSlice(fs.lists.Blacklist, ip)
	case "graylist":
		fs.lists.Graylist = removeFromSlice(fs.lists.Graylist, ip)
	}

	// Перестраиваем дерево без повторной блокировки
	fs.loadListsNoLock(fs.lists)

	// Инвалидируем кэш
	if fs.cache != nil {
		fs.cache.cache.Remove(ip)
	}
}

// GetLists возвращает текущие списки
func (fs *FilterService) GetLists() config.ListsConfig {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	return fs.lists
}

func removeFromSlice(slice []string, item string) []string {
	for i, v := range slice {
		if v == item {
			return append(slice[:i], slice[i+1:]...)
		}
	}
	return slice
}