package transport

import (
	"net/http"

	"github.com/Banner-babaner/proxytools/auth/entity"
	"github.com/Banner-babaner/proxytools/auth/usecase"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *usecase.AuthService
}

func NewHandler(svc *usecase.AuthService) *Handler {
	return &Handler{service: svc}
}

func (h *Handler) Login(c *gin.Context) {
	var req entity.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.service.Login(req.Username, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	c.SetCookie("token", resp.Token, int(h.service.GetTTL().Seconds()), "/", "", false, true)
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) Me(c *gin.Context) {
	userID, _ := c.Get("user_id")
	username, _ := c.Get("username")
	role, _ := c.Get("role")
	c.JSON(http.StatusOK, gin.H{"user_id": userID, "username": username, "role": role})
}