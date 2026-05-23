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
    Enabled    bool        `mapstructure:"enabled"`
    DefaultTTL int         `mapstructure:"default_ttl"`
    MaxSize    int         `mapstructure:"max_size"`
    Rules      []CacheRule `mapstructure:"rules"`
}

type CacheRule struct {
    Path   string `mapstructure:"path"`
    Domain string `mapstructure:"domain"`
    TTL    int    `mapstructure:"ttl"`
}