package transport

import (
	"net/http"
	"time"

	"github.com/Banner-babaner/proxytools/monitor/usecase"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type Handler struct {
	service  *usecase.MetricsService
	upgrader websocket.Upgrader
}

func NewHandler(svc *usecase.MetricsService) *Handler {
	return &Handler{
		service: svc,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	}
}

func (h *Handler) GetMetrics(c *gin.Context) {
	c.JSON(http.StatusOK, h.service.GetStats())
}

func (h *Handler) DashboardWS(c *gin.Context) {
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if err := conn.WriteJSON(h.service.GetStats()); err != nil {
			break
		}
	}
}

func (h *Handler) DashboardHTML(c *gin.Context) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, DashboardHTML)
}