package usecase

import (
	"errors"
	"time"

	"github.com/Banner-babaner/proxytools/auth/entity"
	"github.com/Banner-babaner/proxytools/auth/repository"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

type AuthService struct {
	repo      repository.UserRepository
	secretKey []byte
	ttl       time.Duration
}

func NewAuthService(cfg entity.AuthConfig, repoBuilder func() repository.UserRepository) *AuthService {
	return &AuthService{
		repo:      repoBuilder(),
		secretKey: []byte(cfg.SecretKey),
		ttl:       time.Duration(cfg.TokenTTL) * time.Second,
	}
}

func (s *AuthService) Login(username, password string) (*entity.LoginResponse, error) {
	user, err := s.repo.FindByUsername(username)
	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	if user.Password != password {
		return nil, errors.New("invalid credentials")
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
	tokenStr, _ := token.SignedString(s.secretKey)

	return &entity.LoginResponse{
		Token:    tokenStr,
		Username: user.Username,
		Role:     user.Role,
	}, nil
}

func (s *AuthService) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		return s.secretKey, nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}

	_, err = s.repo.FindByID(claims.UserID)
	if err != nil {
		return nil, errors.New("user not found")
	}

	return claims, nil
}

func (auth *AuthService) GetTTL() time.Duration{
	return  auth.ttl
}