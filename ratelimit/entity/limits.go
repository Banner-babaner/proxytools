package entity

type RateLimitStats struct {
	Tokens      float64 `json:"tokens"`
	RPS         float64 `json:"rps"`
	Connections int     `json:"connections"`
	MaxConns    int     `json:"max_conns"`
}

type RateLimitConfig struct {
	Enabled bool
	RPS     int
	MaxConns int
}