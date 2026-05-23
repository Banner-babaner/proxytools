// transport/handler.go
package transport

import (
	"net/http"

	"github.com/Banner-babaner/proxytools/ipfilter/usecase"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *usecase.FilterService
}

func NewHandler(svc *usecase.FilterService) *Handler {
	return &Handler{service: svc}
}

func (h *Handler) GetAllowLists(c *gin.Context) {
	c.JSON(http.StatusOK, h.service.GetLists())
}

func (h *Handler) AddToList(c *gin.Context) {
	var req struct {
		IP       string `json:"ip" binding:"required"`
		ListType string `json:"list_type" binding:"required,oneof=whitelist blacklist graylist"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.AddIP(req.IP, req.ListType); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "IP added"})
}

func (h *Handler) RemoveFromList(c *gin.Context) {
	ip := c.Param("ip")
	listType := c.Query("list_type")

	if listType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "list_type is required"})
		return
	}

	h.service.RemoveIP(ip, listType)
	c.JSON(http.StatusOK, gin.H{"message": "IP removed"})
}

func (h *Handler) CheckAccess(c *gin.Context) {
	ip := c.Query("ip")
	if ip == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ip required"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"ip":     ip,
		"access": h.service.CheckAccess(ip),
	})
}