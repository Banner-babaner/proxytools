// internal/monitor/handler.go
package monitor

import (
    "net/http"

    "github.com/gin-gonic/gin"
    "github.com/gorilla/websocket"
    "github.com/Banner-babaner/proxytools/logger"
)

var (
    metricsService *MetricsService
    upgrader       = websocket.Upgrader{
        CheckOrigin: func(r *http.Request) bool { return true },
    }
)

func SetMetricsService(ms *MetricsService) {
    metricsService = ms
}

// GetMetrics godoc
// @Summary Получить текущие метрики
// @Tags monitor
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /monitor/metrics [get]
func GetMetrics(c *gin.Context) {
    stats := metricsService.GetStats()
    c.JSON(http.StatusOK, stats)
}

// DashboardWS godoc
// @Summary WebSocket для real-time метрик
// @Tags monitor
// @Router /monitor/ws [get]
func DashboardWS(c *gin.Context) {
    conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
    if err != nil {
        logger.Error().Err(err).Msg("WebSocket upgrade failed")
        return
    }

    metricsService.AddWSClient(conn)
    defer metricsService.RemoveWSClient(conn)

    // Держим соединение открытым
    for {
        _, _, err := conn.ReadMessage()
        if err != nil {
            break
        }
    }
}

// DashboardHTML отдаёт страницу дашборда
func DashboardHTML(c *gin.Context) {
    c.Header("Content-Type", "text/html; charset=utf-8")
    c.String(http.StatusOK, dashboardHTML)
}