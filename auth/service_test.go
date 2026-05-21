// internal/auth/service_test.go
package auth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func setupService() *Service {
	repo := NewMockUserRepository()
	return NewService(repo, "test-secret-key", 1*time.Hour)
}

func TestService_Login_Success(t *testing.T) {
	svc := setupService()

	token, err := svc.Login("admin", "admin123")
	assert.NoError(t, err)
	assert.NotEmpty(t, token)
}

func TestService_Login_InvalidPassword(t *testing.T) {
	svc := setupService()

	_, err := svc.Login("admin", "wrongpassword")
	assert.ErrorIs(t, err, ErrInvalidCredentials)
}

func TestService_Login_UserNotFound(t *testing.T) {
	svc := setupService()

	_, err := svc.Login("ghost", "password")
	assert.ErrorIs(t, err, ErrInvalidCredentials)
}

func TestService_ValidateToken_Success(t *testing.T) {
	svc := setupService()

	token, _ := svc.Login("admin", "admin123")
	claims, err := svc.ValidateToken(token)

	assert.NoError(t, err)
	assert.Equal(t, "admin", claims.Username)
	assert.Equal(t, "admin", claims.Role)
	assert.NotEmpty(t, claims.UserID)
}

func TestService_ValidateToken_InvalidToken(t *testing.T) {
	svc := setupService()

	_, err := svc.ValidateToken("invalid.token.here")
	assert.Error(t, err)
}

func TestService_ValidateToken_ExpiredToken(t *testing.T) {
	repo := NewMockUserRepository()
	svc := NewService(repo, "secret", 1*time.Millisecond)

	token, _ := svc.Login("admin", "admin123")
	time.Sleep(10 * time.Millisecond)

	_, err := svc.ValidateToken(token)
	assert.Error(t, err)
}

func TestService_ValidateToken_WrongSecret(t *testing.T) {
	repo := NewMockUserRepository()
	svc1 := NewService(repo, "secret1", 1*time.Hour)
	svc2 := NewService(repo, "secret2", 1*time.Hour)

	token, _ := svc1.Login("admin", "admin123")
	_, err := svc2.ValidateToken(token)

	assert.Error(t, err)
}

func TestService_ValidateToken_UserDeleted(t *testing.T) {
	repo := NewMockUserRepository()
	svc := NewService(repo, "secret", 1*time.Hour)

	token, _ := svc.Login("admin", "admin123")

	// Удаляем пользователя
	admin, _ := repo.FindByUsername("admin")
	repo.Delete(admin.ID)

	_, err := svc.ValidateToken(token)
	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestService_CreateUser_Success(t *testing.T) {
	svc := setupService()

	user, err := svc.CreateUser("newadmin", "pass", "admin")
	assert.NoError(t, err)
	assert.Equal(t, "newadmin", user.Username)
	assert.Equal(t, "admin", user.Role)
}

func TestService_CreateUser_InvalidRole(t *testing.T) {
	svc := setupService()

	_, err := svc.CreateUser("user", "pass", "superadmin")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid role")
}

func TestService_GetUsers(t *testing.T) {
	svc := setupService()

	users, err := svc.GetUsers()
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(users), 2)
}

func TestService_GetUser_Success(t *testing.T) {
	svc := setupService()

	admin, _ := svc.repo.FindByUsername("admin")
	user, err := svc.GetUser(admin.ID)

	assert.NoError(t, err)
	assert.Equal(t, "admin", user.Username)
}

func TestService_GetUser_NotFound(t *testing.T) {
	svc := setupService()

	_, err := svc.GetUser("nonexistent")
	assert.ErrorIs(t, err, ErrUserNotFound)
}

func TestService_DeleteUser_Success(t *testing.T) {
	svc := setupService()

	admin, _ := svc.repo.FindByUsername("admin")
	err := svc.DeleteUser(admin.ID)
	assert.NoError(t, err)

	_, err = svc.repo.FindByUsername("admin")
	assert.ErrorIs(t, err, ErrUserNotFound)
}