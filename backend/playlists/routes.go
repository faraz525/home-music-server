package playlists

import "github.com/gin-gonic/gin"

// Routes registers playlist-related routes on the provided router group.
func Routes(m *Manager) func(*gin.RouterGroup) {
	return func(r *gin.RouterGroup) {
		g := r.Group("/playlists")
		g.POST("", CreatePlaylistHandler(m))
		g.GET("", GetPlaylistsHandler(m))
		g.GET("/:id", GetPlaylistHandler(m))
		g.PUT("/:id", UpdatePlaylistHandler(m))
		g.DELETE("/:id", DeletePlaylistHandler(m))

		// Playlist track management
		g.POST("/:id/tracks", AddTracksToPlaylistHandler(m))
		g.DELETE("/:id/tracks", RemoveTracksFromPlaylistHandler(m))
		g.GET("/:id/tracks", GetPlaylistTracksHandler(m))

		// Special endpoint for unsorted tracks
		g.GET("/unsorted", GetUnsortedTracksHandler(m))
	}
}
