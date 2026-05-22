package infrastructure

import (
	"time"

	"github.com/Banner-babaner/proxytools/ipfilter/entity"
	lru "github.com/hashicorp/golang-lru/v2"
)

type cacheEntry struct {
    listType  entity.ListType
    hasRule   bool
    expiresAt time.Time
}

type IPCache struct {
    cache *lru.Cache[string, cacheEntry]
    ttl   time.Duration
}

func NewIPCache(maxSize int, ttl time.Duration) (*IPCache, error) {
    cache, err := lru.New[string, cacheEntry](maxSize)
    if err != nil {
        return nil, err
    }
    
    return &IPCache{
        cache: cache,
        ttl:   ttl,
    }, nil
}

func (c *IPCache) Remove(ip string){
    c.cache.Remove(ip)
}

func (c *IPCache) Get(ip string) (entity.ListType, bool, bool) {
    entry, ok := c.cache.Get(ip)
    if !ok {
        return 0, false, false
    }
    
    if time.Now().After(entry.expiresAt) {
        c.Remove(ip)
        return 0, false, false
    }
    
    return entry.listType, entry.hasRule, true
}

func (c *IPCache) Set(ip string, listType  entity.ListType, hasRule bool) {
    c.cache.Add(ip, cacheEntry{
        listType:  listType,
        hasRule:   hasRule,
        expiresAt: time.Now().Add(c.ttl),
    })
}