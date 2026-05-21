// internal/config/config_test.go
package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoad_ValidConfig(t *testing.T) {
	// Создаём временный конфиг
	yaml := `
server:
  port: 8080
  upstream: "http://localhost:9000"
auth:
  secret_key: "test-key"
  token_ttl: 3600
ip_filter:
  default_policy: "deny"
rate_limit:
  enabled: true
cache:
  enabled: true
logging:
  level: "info"
  format: "json"
  output: "stdout"
`
	tmpfile, err := os.CreateTemp("", "config-*.yaml")
	assert.NoError(t, err)
	defer os.Remove(tmpfile.Name())

	_, err = tmpfile.WriteString(yaml)
	assert.NoError(t, err)
	tmpfile.Close()

	cfg, err := Load(tmpfile.Name())
	assert.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Equal(t, 8080, cfg.Server.Port)
	assert.Equal(t, "http://localhost:9000", cfg.Server.Upstream)
	assert.Equal(t, "deny", cfg.IPFilter.DefaultPolicy)
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := Load("nonexistent.yaml")
	assert.Error(t, err)
}