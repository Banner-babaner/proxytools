package repository

import "github.com/Banner-babaner/proxytools/cache/entity"

type CacheRepository interface {
	Get(key string) (*entity.CacheEntry, bool)
	Set(key string, entry *entity.CacheEntry)
	Delete(key string)
	DeleteByPrefix(prefix string) int
	DeleteByTag(tag string) int
	DeleteByPattern(pattern string) (int, error)
	Clear()
	Stats() entity.CacheStats
}