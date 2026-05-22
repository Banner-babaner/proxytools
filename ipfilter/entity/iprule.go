package entity

type ListType int

const (
	Whitelist ListType = iota
	Blacklist
	Graylist
)

type AccessResult int

const (
	Allowed AccessResult = iota
	Denied
	CaptchaRequired
)


type ListsConfig struct {
    Whitelist []string `mapstructure:"whitelist"`
    Blacklist []string `mapstructure:"blacklist"`
    Graylist  []string `mapstructure:"graylist"`
}

type CacheConfig struct {
    Enabled bool `mapstructure:"enabled"`
    TTL     int  `mapstructure:"ttl"`
    MaxSize int  `mapstructure:"max_size"`
}

type IPFilterConfig struct {
    DefaultPolicy string      `mapstructure:"default_policy"`
    Lists         ListsConfig `mapstructure:"lists"`
    Cache         CacheConfig `mapstructure:"cache"`
    AutoReload    bool        `mapstructure:"auto_reload"`
}