// internal/auth/handler.go
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

// Login godoc
// @Summary Войти в систему
// @Tags auth
// @Accept json
// @Produce json
// @Param request body LoginRequest true "Логин и пароль"
// @Success 200 {object} LoginResponse
// @Failure 401 {object} map[string]string
// @Router /auth/login [post]
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

	// Получаем пользователя для ответа
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

// CreateUserRequest запрос на создание пользователя
type CreateUserRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Role     string `json:"role" binding:"required,oneof=admin user"`
}

// CreateUser godoc
// @Summary Создать пользователя (только для админов)
// @Tags auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateUserRequest true "Данные пользователя"
// @Success 201 {object} User
// @Router /auth/users [post]
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

// GetUsers godoc
// @Summary Получить список пользователей (только для админов)
// @Tags auth
// @Produce json
// @Security BearerAuth
// @Success 200 {array} User
// @Router /auth/users [get]
func GetUsers(c *gin.Context) {
	users, err := authService.GetUsers()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, users)
}