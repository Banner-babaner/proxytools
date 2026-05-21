
package auth

import (
	"fmt"
	"sync"
	"github.com/google/uuid"
)


type MockUserRepository struct {
	mu    sync.RWMutex
	users map[string]*User//username
}

func NewMockUserRepository() *MockUserRepository {
	repo := &MockUserRepository{
		users: make(map[string]*User),
	}

	repo.Create(&User{
		ID:       uuid.New().String(),
		Username: "admin",
		Password: "admin123",
		Role:     "admin",
	})
	repo.Create(&User{
		ID:       uuid.New().String(),
		Username: "user",
		Password: "user123",
		Role:     "user",
	})

	return repo
}

func (r *MockUserRepository) FindByUsername(username string) (*User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	user, exists := r.users[username]
	if !exists {
		return nil, ErrUserNotFound
	}

	return &User{
		ID:       user.ID,
		Username: user.Username,
		Password: user.Password,
		Role:     user.Role,
	}, nil
}

func (r *MockUserRepository) FindByID(id string) (*User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, user := range r.users {
		if user.ID == id {
			return &User{
				ID:       user.ID,
				Username: user.Username,
				Password: user.Password,
				Role:     user.Role,
			}, nil
		}
	}

	return nil, ErrUserNotFound
}

func (r *MockUserRepository) FindAll() ([]*User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	users := make([]*User, 0, len(r.users))
	for _, user := range r.users {
		users = append(users, &User{
			ID:       user.ID,
			Username: user.Username,
			Password: user.Password,
			Role:     user.Role,
		})
	}

	return users, nil
}

func (r *MockUserRepository) Create(user *User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.users[user.Username]; exists {
		return fmt.Errorf("user %s already exists", user.Username)
	}

	if user.ID == "" {
		user.ID = uuid.New().String()
	}

	r.users[user.Username] = user
	return nil
}

func (r *MockUserRepository) Delete(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for username, user := range r.users {
		if user.ID == id {
			delete(r.users, username)
			return nil
		}
	}

	return ErrUserNotFound
}