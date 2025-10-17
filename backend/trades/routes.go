package trades

import (
	"github.com/faraz525/home-music-server/backend/auth"
	"github.com/gin-gonic/gin"
)

// RegisterRoutes registers all trade routes
func RegisterRoutes(router *gin.RouterGroup, manager *Manager) {
	// Trade routes (authenticated)
	router.POST("/trades/request", auth.AuthMiddleware(), RequestTradeHandler(manager))
	router.GET("/trades/history", auth.AuthMiddleware(), GetTradeHistoryHandler(manager))
	router.GET("/trades/available", auth.AuthMiddleware(), GetAvailableTracksHandler(manager))
}
