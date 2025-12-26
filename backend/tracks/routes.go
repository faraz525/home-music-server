package tracks

import (
	"github.com/faraz525/home-music-server/backend/auth" // Added import for auth
	"github.com/faraz525/home-music-server/backend/playlists"
	"github.com/gin-gonic/gin"
)

// Routes registers track-related routes on the provided router group.
func Routes(m *Manager, pm *playlists.Manager) func(*gin.RouterGroup) {
	return func(r *gin.RouterGroup) {
		g := r.Group("/tracks")
		g.POST("", UploadHandler(m, pm))
		g.GET("", ListHandler(m, pm))
		g.GET("/:id/stream", StreamHandler(m))
		g.GET("/:id/download", DownloadHandler(m))
		g.DELETE("/:id", DeleteHandler(m))
		g.GET("/:id", GetHandler(m))

	// Admin routes
	admin := g.Group("/admin")
	admin.Use(auth.AdminMiddleware())
	{
		admin.GET("", func(c *gin.Context) {
			// TODO: Admin dashboard stats
			c.JSON(200, gin.H{"message": "Admin access granted"})
		})
		admin.POST("/sanitize", SanitizeAllHandler(m))
	}
}
}
