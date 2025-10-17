package playlists

import (
	"github.com/faraz525/home-music-server/backend/auth"
	"github.com/gin-gonic/gin"
)

// Routes registers playlist-related routes on the provided router group.
func Routes(m *Manager) func(*gin.RouterGroup) {
	return func(r *gin.RouterGroup) {
		// Authenticated playlist routes
		g := r.Group("/playlists")
		g.Use(auth.AuthMiddleware())
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

		// Public playlist routes (no auth required)
		publicCrates := r.Group("/crates")
		publicCrates.Use(auth.AuthMiddleware())
		publicCrates.GET("/public", GetPublicPlaylistsHandler(m))
		publicCrates.GET("/:id/public", GetPublicPlaylistTracksHandler(m))

		// User public playlists
		r.GET("/users/:username/crates", auth.AuthMiddleware(), GetUserPublicPlaylistsHandler(m))
	}
}
