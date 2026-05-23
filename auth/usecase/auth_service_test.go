package usecase

import (
	"testing"
	"time"
	"errors"

	"github.com/Banner-babaner/proxytools/auth/entity"
	"github.com/Banner-babaner/proxytools/auth/mocks"

	"github.com/stretchr/testify/assert"
)

func newTestAuthService() (*AuthService, *mocks.UserRepository) {
	mockRepo := new(mocks.UserRepository)

	svc := &AuthService{
		repo:      mockRepo,
		secretKey: []byte("test-secret"),
		ttl:       time.Hour,
	}

	return svc, mockRepo
}

func TestLogin_Success(t *testing.T) {
	svc, mockRepo := newTestAuthService()

	mockRepo.On("FindByUsername", "admin").Return(&entity.User{
		ID:       "1",
		Username: "admin",
		Password: "admin123",
		Role:     "admin",
	}, nil)

	resp, err := svc.Login("admin", "admin123")
	assert.NoError(t, err)
	assert.NotEmpty(t, resp.Token)
	assert.Equal(t, "admin", resp.Username)
	assert.Equal(t, "admin", resp.Role)
}

func TestLogin_InvalidPassword(t *testing.T) {
	svc, mockRepo := newTestAuthService()

	mockRepo.On("FindByUsername", "admin").Return(&entity.User{
		Password: "admin123",
	}, nil)

	_, err := svc.Login("admin", "wrong")
	assert.Error(t, err)
}

func TestLogin_UserNotFound(t *testing.T) {
	svc, mockRepo := newTestAuthService()

	mockRepo.On("FindByUsername", "ghost").Return(nil, errors.New("not found"))

	_, err := svc.Login("ghost", "pass")
	assert.Error(t, err)
}

func TestValidateToken_Success(t *testing.T) {
	svc, mockRepo := newTestAuthService()

	mockRepo.On("FindByUsername", "admin").Return(&entity.User{
		ID:       "1",
		Username: "admin",
		Password: "admin123",
		Role:     "admin",
	}, nil)
	mockRepo.On("FindByID", "1").Return(&entity.User{ID: "1"}, nil)

	resp, _ := svc.Login("admin", "admin123")

	claims, err := svc.ValidateToken(resp.Token)
	assert.NoError(t, err)
	assert.Equal(t, "admin", claims.Username)
	assert.Equal(t, "admin", claims.Role)
}

func TestValidateToken_Invalid(t *testing.T) {
	svc, _ := newTestAuthService()

	_, err := svc.ValidateToken("invalid.token")
	assert.Error(t, err)
}

func TestValidateToken_UserDeleted(t *testing.T) {
	svc, mockRepo := newTestAuthService()

	mockRepo.On("FindByUsername", "admin").Return(&entity.User{
		ID:       "1",
		Username: "admin",
		Password: "admin123",
		Role:     "admin",
	}, nil)
	mockRepo.On("FindByID", "1").Return(nil, errors.New("not found")).Once()

	resp, _ := svc.Login("admin", "admin123")

	_, err := svc.ValidateToken(resp.Token)
	assert.Error(t, err)
}