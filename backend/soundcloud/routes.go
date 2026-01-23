package soundcloud

import (
	"github.com/gin-gonic/gin"
)

func Routes(manager *Manager) func(*gin.RouterGroup) {
	return func(rg *gin.RouterGroup) {
		handlers := NewHandlers(manager)

		sc := rg.Group("/soundcloud")
		{
			sc.GET("/config", handlers.GetConfig)
			sc.PUT("/config", handlers.UpdateConfig)
			sc.POST("/sync", handlers.TriggerSync)
			sc.GET("/history", handlers.GetHistory)
		}
	}
}
