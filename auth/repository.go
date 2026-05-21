package auth

import "errors"

var (
	ErrUserNotFound = errors.New("user not found")
)

type User struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

type UserRepository interface {
	FindByUsername(username string) (*User, error)
	FindByID(id string) (*User, error)
	FindAll() ([]*User, error)
	Create(user *User) error
	Delete(id string) error
}