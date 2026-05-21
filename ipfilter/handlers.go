// internal/ipfilter/handler.go
package ipfilter

import (
    "net/http"
    "sync"
    
    "github.com/gin-gonic/gin"
    "github.com/Banner-babaner/proxytools/logger"
)

var (
    filterService *FilterService
    handlerOnce   sync.Once
)

// SetFilterService устанавливает сервис для хендлеров
func SetFilterService(fs *FilterService) {
    handlerOnce.Do(func() {
        filterService = fs
    })
}

// GetAllowLists godoc
// @Summary Получить текущие списки доступа
// @Tags ip_filter
// @Produce json
// @Success 200 {object} config.ListsConfig
// @Router /ip_access/allowlists [get]
func GetAllowLists(c *gin.Context) {
    lists := filterService.GetLists()
    c.JSON(http.StatusOK, lists)
}

// AddToListRequest тело запроса на добавление
type AddToListRequest struct {
    IP       string `json:"ip" binding:"required"`
    ListType string `json:"list_type" binding:"required,oneof=whitelist blacklist graylist"`
}

// AddToList godoc
// @Summary Добавить IP в список
// @Tags ip_filter
// @Accept json
// @Produce json
// @Param request body AddToListRequest true "IP и тип списка"
// @Success 200 {object} map[string]string
// @Router /ip_access/allowlists [post]
func AddToList(c *gin.Context) {
    var req AddToListRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    if err := filterService.AddIP(req.IP, req.ListType); err != nil {
        logger.Error().Err(err).Str("ip", req.IP).Msg("Failed to add IP")
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    
    logger.Info().
        Str("ip", req.IP).
        Str("list", req.ListType).
        Msg("IP added to list")
    
    c.JSON(http.StatusOK, gin.H{"message": "IP added successfully"})
}

// RemoveFromList godoc
// @Summary Удалить IP из списка
// @Tags ip_filter
// @Param ip path string true "IP адрес"
// @Param list_type query string true "Тип списка"
// @Success 200 {object} map[string]string
// @Router /ip_access/allowlists/{ip} [delete]
func RemoveFromList(c *gin.Context) {
    ip := c.Param("ip")
    listType := c.Query("list_type")
    
    if listType == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "list_type is required"})
        return
    }
    
    filterService.RemoveIP(ip, listType)
    
    logger.Info().
        Str("ip", ip).
        Str("list", listType).
        Msg("IP removed from list")
    
    c.JSON(http.StatusOK, gin.H{"message": "IP removed successfully"})
}

// CheckAccess godoc
// @Summary Проверить доступ для IP
// @Tags ip_filter
// @Param ip query string true "IP адрес"
// @Success 200 {object} map[string]interface{}
// @Router /ip_access/check [get]
func CheckAccess(c *gin.Context) {
    ip := c.Query("ip")
    if ip == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "ip parameter is required"})
        return
    }
    
    result := filterService.CheckAccess(ip)
    
    var status string
    switch result {
    case Allowed:
        status = "allowed"
    case Denied:
        status = "denied"
    case CaptchaRequired:
        status = "captcha_required"
    }
    
    c.JSON(http.StatusOK, gin.H{
        "ip":     ip,
        "access": status,
    })
}