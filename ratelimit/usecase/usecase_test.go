package usecase

import (
	"testing"

	"github.com/Banner-babaner/proxytools/ratelimit/entity"
	"github.com/Banner-babaner/proxytools/ratelimit/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newTestService(enabled bool) (*RateLimitService, *mocks.RateLimitRepository) {
	mockRepo := new(mocks.RateLimitRepository)
	svc := &RateLimitService{repo: mockRepo, enabled: enabled}
	return svc, mockRepo
}

func TestAllow_Enabled_True(t *testing.T) {
	svc, mockRepo := newTestService(true)
	mockRepo.On("Allow", "192.168.1.1").Return(true)
	assert.True(t, svc.Allow("192.168.1.1"))
}

func TestAllow_Enabled_False(t *testing.T) {
	svc, mockRepo := newTestService(true)
	mockRepo.On("Allow", "10.0.0.1").Return(false)
	assert.False(t, svc.Allow("10.0.0.1"))
}

func TestAllow_Disabled(t *testing.T) {
	svc, _ := newTestService(false)
	assert.True(t, svc.Allow("192.168.1.1"))
}

func TestIncrementConnections_Enabled_True(t *testing.T) {
	svc, mockRepo := newTestService(true)
	mockRepo.On("IncrementConnections", "192.168.1.1").Return(true)
	assert.True(t, svc.IncrementConnections("192.168.1.1"))
}

func TestIncrementConnections_Enabled_False(t *testing.T) {
	svc, mockRepo := newTestService(true)
	mockRepo.On("IncrementConnections", "10.0.0.1").Return(false)
	assert.False(t, svc.IncrementConnections("10.0.0.1"))
}

func TestIncrementConnections_Disabled(t *testing.T) {
	svc, _ := newTestService(false)
	assert.True(t, svc.IncrementConnections("192.168.1.1"))
}

func TestDecrementConnections_Enabled(t *testing.T) {
	svc, mockRepo := newTestService(true)
	mockRepo.On("DecrementConnections", "192.168.1.1").Return()
	svc.DecrementConnections("192.168.1.1")
	mockRepo.AssertCalled(t, "DecrementConnections", "192.168.1.1")
}

func TestDecrementConnections_Disabled(t *testing.T) {
	svc, mockRepo := newTestService(false)
	svc.DecrementConnections("192.168.1.1")
	mockRepo.AssertNotCalled(t, "DecrementConnections", mock.Anything)
}

func TestGetStats_Found(t *testing.T) {
	svc, mockRepo := newTestService(true)
	expected := &entity.RateLimitStats{Tokens: 5.0, RPS: 10.0, Connections: 2, MaxConns: 50}
	mockRepo.On("GetStats", "192.168.1.1").Return(expected)

	stats := svc.GetStats("192.168.1.1")
	assert.NotNil(t, stats)
	assert.Equal(t, 5.0, stats.Tokens)
	assert.Equal(t, 2, stats.Connections)
	assert.Equal(t, 50, stats.MaxConns)
}

func TestGetStats_NotFound(t *testing.T) {
	svc, mockRepo := newTestService(true)
	mockRepo.On("GetStats", "unknown").Return(nil)

	stats := svc.GetStats("unknown")
	assert.Nil(t, stats)
}