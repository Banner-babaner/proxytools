package transport

import (
	"net/http"

	"github.com/Banner-babaner/proxytools/ratelimit/usecase"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *usecase.RateLimitService
}

func NewHandler(svc *usecase.RateLimitService) *Handler {
	return &Handler{service: svc}
}

func (h *Handler) GetStats(c *gin.Context) {
	ip := c.Query("ip")
	if ip == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ip required"})
		return
	}

	stats := h.service.GetStats(ip)
	if stats == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "ip not found"})
		return
	}

	c.JSON(http.StatusOK, stats)
}
