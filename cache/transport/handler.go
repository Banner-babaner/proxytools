package transport

import (
	"net/http"

	"github.com/Banner-babaner/proxytools/cache/entity"
	"github.com/Banner-babaner/proxytools/cache/usecase"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *usecase.CacheService
}

func NewHandler(svc *usecase.CacheService) *Handler {
	return &Handler{service: svc}
}

func (h *Handler) GetStats(c *gin.Context) {
	c.JSON(http.StatusOK, h.service.Stats())
}

func (h *Handler) Invalidate(c *gin.Context) {
	var req entity.InvalidateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	count, err := h.service.Invalidate(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"invalidated": count})
}