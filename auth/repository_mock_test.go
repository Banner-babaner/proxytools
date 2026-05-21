// internal/auth/repository_mock_test.go
package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMockUserRepository_FindByUsername(t *testing.T) {
	repo := NewMockUserRepository()

	user, err := repo.FindByUsername("admin")
	assert.NoError(t, err)
	assert.Equal(t, "admin", user.Username)
	assert.Equal(t, "admin123", user.Password)
	assert.Equal(t, "admin", user.Role)
	assert.NotEmpty(t, user.ID)
}

func TestMockUserRepository_FindByUsername_NotFound(t *testing.T) {
	repo := NewMockUserRepository()

	_, err := repo.FindByUsername("nonexistent")
	assert.ErrorIs(t, err, ErrUserNotFound)
}

func TestMockUserRepository_FindByID(t *testing.T) {
	repo := NewMockUserRepository()

	admin, _ := repo.FindByUsername("admin")
	user, err := repo.FindByID(admin.ID)

	assert.NoError(t, err)
	assert.Equal(t, admin.Username, user.Username)
}

func TestMockUserRepository_FindByID_NotFound(t *testing.T) {
	repo := NewMockUserRepository()

	_, err := repo.FindByID("nonexistent-id")
	assert.ErrorIs(t, err, ErrUserNotFound)
}

func TestMockUserRepository_FindAll(t *testing.T) {
	repo := NewMockUserRepository()

	users, err := repo.FindAll()
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(users), 2) // минимум admin и user
}

func TestMockUserRepository_Create(t *testing.T) {
	repo := NewMockUserRepository()

	newUser := &User{
		Username: "newuser",
		Password: "pass",
		Role:     "user",
	}

	err := repo.Create(newUser)
	assert.NoError(t, err)
	assert.NotEmpty(t, newUser.ID)

	// Проверяем, что создался
	found, err := repo.FindByUsername("newuser")
	assert.NoError(t, err)
	assert.Equal(t, "newuser", found.Username)
}

func TestMockUserRepository_Create_Duplicate(t *testing.T) {
	repo := NewMockUserRepository()

	err := repo.Create(&User{Username: "admin", Password: "x", Role: "user"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestMockUserRepository_Delete(t *testing.T) {
	repo := NewMockUserRepository()

	admin, _ := repo.FindByUsername("admin")
	err := repo.Delete(admin.ID)
	assert.NoError(t, err)

	_, err = repo.FindByUsername("admin")
	assert.ErrorIs(t, err, ErrUserNotFound)
}

func TestMockUserRepository_Delete_NotFound(t *testing.T) {
	repo := NewMockUserRepository()

	err := repo.Delete("nonexistent")
	assert.ErrorIs(t, err, ErrUserNotFound)
}

func TestMockUserRepository_Concurrent(t *testing.T) {
	repo := NewMockUserRepository()

	done := make(chan bool)
	for i := 0; i < 50; i++ {
		go func() {
			_, _ = repo.FindByUsername("admin")
			done <- true
		}()
	}

	for i := 0; i < 50; i++ {
		<-done
	}
}