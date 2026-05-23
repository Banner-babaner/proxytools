package entity

type RateLimitStats struct {
	Tokens      float64 `json:"tokens"`
	RPS         float64 `json:"rps"`
	Connections int     `json:"connections"`
	MaxConns    int     `json:"max_conns"`
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