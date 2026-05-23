package repository

import "github.com/Banner-babaner/proxytools/auth/entity"

type UserRepository interface {
	FindByUsername(username string) (*entity.User, error)
	FindByID(id string) (*entity.User, error)
	Create(user *entity.User) error
	Delete(id string) error
}