package ratelimit

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