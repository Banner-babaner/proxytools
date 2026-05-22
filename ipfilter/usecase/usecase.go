package usecase

import (
    "fmt"
    "sync"
    "time"
    "github.com/Banner-babaner/proxytools/logger"
	"github.com/Banner-babaner/proxytools/ipfilter/entity"
	"github.com/Banner-babaner/proxytools/ipfilter/repository"
)



type FilterService struct {
	mu            sync.RWMutex
	builder       func() repository.IPListRepository
	repo          repository.IPListRepository
	cache         repository.IPCache
	defaultPolicy string
	lists         entity.ListsConfig
}




func NewFilterService(cfg entity.IPFilterConfig,
	repoBuilder func() repository.IPListRepository,
	cacheBuilder func(maxSize int, ttlSeconds time.Duration) (repository.IPCache, error)) (*FilterService, error) {
	fs := &FilterService{
		builder:       repoBuilder,
		repo:          repoBuilder(),
		defaultPolicy: cfg.DefaultPolicy,
		lists:         cfg.Lists,
	}

	if cfg.Cache.Enabled {
		cache, err := cacheBuilder(cfg.Cache.MaxSize, time.Duration(cfg.Cache.TTL)*time.Second)
		if err != nil {
			return nil, err
		}
		fs.cache = cache
	}

	fs.loadLists(cfg.Lists)

	return fs, nil
}

func (fs *FilterService) loadListsNoLock(lists entity.ListsConfig) {
	fs.repo = fs.builder()

	for _, ip := range lists.Blacklist {
		fs.repo.Insert(ip, entity.Blacklist)
	}

	for _, ip := range lists.Whitelist {
		fs.repo.Insert(ip, entity.Whitelist)
	}

	for _, ip := range lists.Graylist {
		fs.repo.Insert(ip, entity.Graylist)
	}

	logger.Info().
		Int("blacklist", len(lists.Blacklist)).
		Int("whitelist", len(lists.Whitelist)).
		Int("graylist", len(lists.Graylist)).
		Msg("IP lists loaded")
}

// loadLists с блокировкой
func (fs *FilterService) loadLists(lists entity.ListsConfig) {
	fs.mu.Lock()
	fs.loadListsNoLock(lists)
	fs.mu.Unlock()
}

// CheckAccess проверяет доступ для IP
func (fs *FilterService) CheckAccess(ip string) entity.AccessResult {
	if fs.cache != nil {
		if listType, hasRule, found := fs.cache.Get(ip); found {
			return fs.determineAccess(listType, hasRule)
		}
	}

	fs.mu.RLock()
	listType, hasRule := fs.repo.Search(ip)
	fs.mu.RUnlock()

	if fs.cache != nil {
		fs.cache.Set(ip, listType, hasRule)
	}

	return fs.determineAccess(listType, hasRule)
}

func (fs *FilterService) determineAccess(listType entity.ListType, hasRule bool) entity.AccessResult {
	if hasRule {
		switch listType {
		case entity.Blacklist:
			return entity.Denied
		case entity.Whitelist:
			return entity.Allowed
		case entity.Graylist:
			return entity.CaptchaRequired
		}
	}

	if fs.defaultPolicy == "allow" {
		return entity.Allowed
	}
	return entity.Denied
}

// AddIP добавляет IP в список
func (fs *FilterService) AddIP(ip string, listType string) error {
	var lt entity.ListType
	switch listType {
	case "whitelist":
		lt = entity.Whitelist
		fs.mu.Lock()
		fs.lists.Whitelist = append(fs.lists.Whitelist, ip)
		fs.mu.Unlock()
	case "blacklist":
		lt = entity.Blacklist
		fs.mu.Lock()
		fs.lists.Blacklist = append(fs.lists.Blacklist, ip)
		fs.mu.Unlock()
	case "graylist":
		lt = entity.Graylist
		fs.mu.Lock()
		fs.lists.Graylist = append(fs.lists.Graylist, ip)
		fs.mu.Unlock()
	default:
		return fmt.Errorf("unknown list type: %s", listType)
	}

	fs.mu.Lock()
	if err := fs.repo.Insert(ip, lt); err != nil {
		fs.mu.Unlock()
		return err
	}
	fs.mu.Unlock()

	if fs.cache != nil {
		fs.cache.Remove(ip)
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
		fs.cache.Remove(ip)
	}
}

// GetLists возвращает текущие списки
func (fs *FilterService) GetLists() entity.ListsConfig {
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