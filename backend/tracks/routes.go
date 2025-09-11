package tracks

import (
	"github.com/faraz525/home-music-server/backend/playlists"
	"github.com/gin-gonic/gin"
)

// Routes registers track-related routes on the provided router group.
func Routes(m *Manager, pm *playlists.Manager) func(*gin.RouterGroup) {
	return func(r *gin.RouterGroup) {
		g := r.Group("/tracks")
		g.POST("", UploadHandler(m))
		g.GET("", ListHandler(m, pm))
		g.GET("/:id", GetHandler(m))
		g.GET("/:id/stream", StreamHandler(m))
		g.DELETE("/:id", DeleteHandler(m))
	}
}
