package transport

import (
	"net/http"
	"sync"

	"ipfilter/entity"
	"ipfilter/usecase"
	"github.com/Banner-babaner/proxytools/logger"
	"github.com/gin-gonic/gin"
)

var (
    filterService *usecase.FilterService
    handlerOnce   sync.Once
)


func SetFilterService(fs *usecase.FilterService) {
    handlerOnce.Do(func() {
        filterService = fs
    })
}

func GetAllowLists(c *gin.Context) {
    lists := filterService.GetLists()
    c.JSON(http.StatusOK, lists)
}


type AddToListRequest struct {
    IP       string `json:"ip" binding:"required"`
    ListType string `json:"list_type" binding:"required,oneof=whitelist blacklist graylist"`
}


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

func CheckAccess(c *gin.Context) {
    ip := c.Query("ip")
    if ip == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "ip parameter is required"})
        return
    }
    
    result := filterService.CheckAccess(ip)
    
    var status string
    switch result {
    case entity.Allowed:
        status = "allowed"
    case entity.Denied:
        status = "denied"
    case entity.CaptchaRequired:
        status = "captcha_required"
    }
    
    c.JSON(http.StatusOK, gin.H{
        "ip":     ip,
        "access": status,
    })
}