package entity

import (
	"net/http"
	"time"
)

type CacheEntry struct {
	StatusCode int
	Headers    http.Header
	Body       []byte
	Size       int64
	CreatedAt  time.Time
	TTL        time.Duration
	Tags       []string
	Key        string
}

type CacheStats struct {
	Entries int     `json:"entries"`
	SizeMB  float64 `json:"size_mb"`
	MaxSize int64   `json:"max_size"`
	Enabled bool    `json:"enabled"`
}

type InvalidateRequest struct {
	Key     string   `json:"key"`
	Prefix  string   `json:"prefix"`
	Pattern string   `json:"pattern"`
	Tags    []string `json:"tags"`
	Clear   bool     `json:"clear_all"`
}

type CacheConfig struct {
	Enabled    bool
	DefaultTTL time.Duration
	MaxSize    int64
	Rules      []CacheRule
}

type CacheRule struct {
	Path   string
	Domain string
	TTL    time.Duration
}