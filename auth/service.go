// internal/auth/service.go
package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidToken       = errors.New("invalid token")
	ErrExpiredToken       = errors.New("token expired")
)

// Claims JWT claims
type Claims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// Service сервис авторизации
type Service struct {
	repo      UserRepository
	secretKey []byte
	ttl       time.Duration
}

func NewService(repo UserRepository, secretKey string, ttl time.Duration) *Service {
	return &Service{
		repo:      repo,
		secretKey: []byte(secretKey),
		ttl:       ttl,
	}
}

// Login аутентифицирует пользователя и возвращает JWT
func (s *Service) Login(username, password string) (string, error) {
	user, err := s.repo.FindByUsername(username)
	if err != nil {
		return "", ErrInvalidCredentials
	}

	if user.Password != password {
		return "", ErrInvalidCredentials
	}

	claims := Claims{
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "proxy",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secretKey)
}

// ValidateToken проверяет JWT и возвращает claims
func (s *Service) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return s.secretKey, nil
	})

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	// Проверяем, что пользователь ещё существует в БД
	_, err = s.repo.FindByID(claims.UserID)
	if err != nil {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// GetUser возвращает пользователя по ID
func (s *Service) GetUser(id string) (*User, error) {
	return s.repo.FindByID(id)
}

// GetUsers возвращает всех пользователей (только для админов)
func (s *Service) GetUsers() ([]*User, error) {
	return s.repo.FindAll()
}

// CreateUser создаёт нового пользователя (только для админов)
func (s *Service) CreateUser(username, password, role string) (*User, error) {
	if role != "admin" && role != "user" {
		return nil, errors.New("invalid role: must be 'admin' or 'user'")
	}

	user := &User{
		Username: username,
		Password: password,
		Role:     role,
	}

	err := s.repo.Create(user)
	if err != nil {
		return nil, err
	}

	return user, nil
}

// DeleteUser удаляет пользователя (только для админов)
func (s *Service) DeleteUser(id string) error {
	return s.repo.Delete(id)
}