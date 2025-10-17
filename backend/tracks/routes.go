package tracks

import (
	"context"

	"github.com/faraz525/home-music-server/backend/playlists"
	"github.com/gin-gonic/gin"
)

// TradeReferenceChecker interface for checking track references
type TradeReferenceChecker interface {
	HasTrackReference(ctx context.Context, userID, trackID string) (bool, error)
}

// Routes registers track-related routes on the provided router group.
func Routes(m *Manager, pm *playlists.Manager, tradesRepo TradeReferenceChecker) func(*gin.RouterGroup) {
	return func(r *gin.RouterGroup) {
		g := r.Group("/tracks")
		g.POST("", UploadHandler(m, pm))
		g.GET("", ListHandler(m, pm))
		g.GET("/:id", GetHandler(m))
		g.GET("/:id/stream", StreamHandler(m, pm))
		g.GET("/:id/download", DownloadHandler(m, tradesRepo))
		g.DELETE("/:id", DeleteHandler(m))
	}
}
