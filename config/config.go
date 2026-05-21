package config

import (
    "github.com/spf13/viper"
)





type Config struct {
    Server    ServerConfig    `mapstructure:"server"`
    IPFilter  IPFilterConfig  `mapstructure:"ip_filter"`
    RateLimit RateLimitConfig `mapstructure:"rate_limit"`
    Cache     CacheConfig     `mapstructure:"cache"`
    Logging   LoggingConfig   `mapstructure:"logging"`
}

type ServerConfig struct {
    Port     int    `mapstructure:"port"`
    Upstream string `mapstructure:"upstream"`
}

type IPFilterConfig struct {
    DefaultPolicy string      `mapstructure:"default_policy"`
    Lists         ListsConfig `mapstructure:"lists"`
    Cache         struct {
        Enabled bool `mapstructure:"enabled"`
        TTL     int  `mapstructure:"ttl"`
        MaxSize int  `mapstructure:"max_size"`
    } `mapstructure:"cache"`
    AutoReload bool `mapstructure:"auto_reload"`
}

type ListsConfig struct {
    Whitelist []string `mapstructure:"whitelist"`
    Blacklist []string `mapstructure:"blacklist"`
    Graylist  []string `mapstructure:"graylist"`
}

type RateLimitConfig struct {
    Enabled     bool `mapstructure:"enabled"`
    Default     RateLimitDefaults `mapstructure:"default"`
    PerIP       bool `mapstructure:"per_ip"`
    PerSubnet   bool `mapstructure:"per_subnet"`
}

type RateLimitDefaults struct {
    RPS         int `mapstructure:"rps"`
    RPM         int `mapstructure:"rpm"`
    RPH         int `mapstructure:"rph"`
    Connections int `mapstructure:"connections"`
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

type LoggingConfig struct {
    Level  string `mapstructure:"level"`
    Format string `mapstructure:"format"`
    Output string `mapstructure:"output"`
}

func Load(path string) (*Config, error) {
    v := viper.New()
    v.SetConfigFile(path)
    v.SetConfigType("yaml")
    
    if err := v.ReadInConfig(); err != nil {
        return nil, err
    }
    
    var cfg Config
    if err := v.Unmarshal(&cfg); err != nil {
        return nil, err
    }
    
    return &cfg, nil
}