package repository

import "ratelimit/entity"

type RateLimitRepository interface {
	Allow(ip string) bool
	IncrementConnections(ip string) bool
	DecrementConnections(ip string)
	GetStats(ip string) *entity.RateLimitStats
}