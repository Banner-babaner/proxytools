package auth

import (
	"errors"

	"github.com/gin-gonic/gin"
)

var (
	ErrUserNotAuthenticated = errors.New("user not authenticated")
)


// GetCurrentUser возвращает текущего пользователя из контекста
func GetCurrentUser(c *gin.Context) (userID, username, role string, err error) {
	userIDRaw, exists := c.Get("user_id")
	if !exists {
		return "", "", "", ErrUserNotAuthenticated
	}
	userID, ok := userIDRaw.(string)
	if !ok {
		return "", "", "", errors.New("invalid user_id type")
	}

	usernameRaw, exists := c.Get("username")
	if !exists {
		return "", "", "", ErrUserNotAuthenticated
	}
	username, ok = usernameRaw.(string)
	if !ok {
		return "", "", "", errors.New("invalid username type")
	}

	roleRaw, exists := c.Get("role")
	if !exists {
		return "", "", "", ErrUserNotAuthenticated
	}
	role, ok = roleRaw.(string)
	if !ok {
		return "", "", "", errors.New("invalid role type")
	}

	return userID, username, role, nil
}