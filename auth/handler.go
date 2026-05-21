package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/Banner-babaner/proxytools/logger"
)

var authService *Service

func SetAuthService(s *Service) {
	authService = s
}

// LoginRequest тело запроса на логин
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse ответ с токеном
type LoginResponse struct {
	Token    string `json:"token"`
	Username string `json:"username"`
	Role     string `json:"role"`
}


func Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	token, err := authService.Login(req.Username, req.Password)
	if err != nil {
		logger.Warn().
			Str("username", req.Username).
			Err(err).
			Msg("Login failed")

		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}


	claims, _ := authService.ValidateToken(token)

	logger.Info().
		Str("username", req.Username).
		Str("role", claims.Role).
		Msg("User logged in")

	c.JSON(http.StatusOK, LoginResponse{
		Token:    token,
		Username: claims.Username,
		Role:     claims.Role,
	})
}


type CreateUserRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Role     string `json:"role" binding:"required,oneof=admin user"`
}


func CreateUser(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := authService.CreateUser(req.Username, req.Password, req.Role)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	logger.Info().
		Str("username", user.Username).
		Str("role", user.Role).
		Msg("User created")

	c.JSON(http.StatusCreated, user)
}


func GetUsers(c *gin.Context) {
	users, err := authService.GetUsers()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, users)
}