package spotify

import (
	"github.com/gin-gonic/gin"
)

func Routes(manager *Manager) func(*gin.RouterGroup) {
	return func(rg *gin.RouterGroup) {
		handlers := NewHandlers(manager)

		sp := rg.Group("/spotify")
		{
			sp.GET("/config", handlers.GetConfig)
			sp.PUT("/config", handlers.UpdateConfig)
			sp.POST("/token", handlers.ExchangeToken)
			sp.POST("/disconnect", handlers.Disconnect)
			sp.POST("/sync", handlers.TriggerSync)
			sp.GET("/history", handlers.GetHistory)
			sp.GET("/playlists", handlers.GetPlaylists)
			sp.GET("/synced-playlists", handlers.GetSyncedPlaylists)
		}
	}
}
