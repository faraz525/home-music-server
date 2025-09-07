package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/faraz525/home-music-server/backend/auth"
	"github.com/faraz525/home-music-server/backend/playlists"
	"github.com/faraz525/home-music-server/backend/tracks"
	"github.com/faraz525/home-music-server/backend/utils"
)

func main() {
	// Enhanced startup logging
	fmt.Printf("[CrateDrop] Starting CrateDrop server at %s...\n", time.Now().Format("2006-01-02 15:04:05"))

	// Initialize database
	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "/mnt/music/cratedrop"
	}
	fmt.Printf("[CrateDrop] Using data directory: %s\n", dataDir)

	db, err := utils.NewDB(dataDir)
	if err != nil {
		log.Fatalf("[CrateDrop] Failed to initialize database: %v", err)
	}
	defer db.Close()
	fmt.Printf("[CrateDrop] Database initialized successfully\n")

	// Initialize repositories
	authRepo := auth.NewRepository(db)
	tracksRepo := tracks.NewRepository(db)
	playlistsRepo := playlists.NewRepository(db)
	fmt.Printf("[CrateDrop] Repositories initialized\n")

	// Initialize managers
	authManager, err := auth.NewManager(authRepo)
	if err != nil {
		log.Fatalf("[CrateDrop] Failed to initialize auth manager: %v", err)
	}
	fmt.Printf("[CrateDrop] Auth manager initialized\n")

	tracksManager := tracks.NewManager(tracksRepo)
	fmt.Printf("[CrateDrop] Tracks manager initialized\n")

	playlistsManager := playlists.NewManager(playlistsRepo)
	fmt.Printf("[CrateDrop] Playlists manager initialized\n")

	// Set playlist creator for auth manager to create default playlists on signup
	authManager.SetPlaylistCreator(playlistsManager)
	fmt.Printf("[CrateDrop] Playlist creator set for auth manager\n")

	// Set playlist adder for tracks manager to add uploaded tracks to playlists
	tracksManager.SetPlaylistAdder(playlistsManager)
	fmt.Printf("[CrateDrop] Playlist adder set for tracks manager\n")

	// Initialize Gin router with custom logging
	r := gin.New()

	// Add custom logger middleware
	r.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		return fmt.Sprintf("[CrateDrop] %s | %3d | %13v | %15s | %-7s %s\n",
			param.TimeStamp.Format("2006/01/02 15:04:05"),
			param.StatusCode,
			param.Latency,
			param.ClientIP,
			param.Method,
			param.Path,
		)
	}))
	r.Use(gin.Recovery())

	// CORS middleware
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization")
		c.Header("Access-Control-Allow-Credentials", "true")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// Health check endpoints
	r.GET("/api/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"service": "cratedrop-backend",
			"version": "v0",
		})
	})
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"service": "cratedrop-backend",
			"version": "v0",
		})
	})

	// Auth routes
	api := r.Group("/api")
	{
		// Public auth routes
		authGroup := api.Group("/auth")
		{
			authGroup.POST("/signup", auth.SignupHandler(authManager))
			authGroup.POST("/login", auth.LoginHandler(authManager))
			authGroup.POST("/refresh", auth.RefreshHandler(authManager))
			authGroup.POST("/logout", auth.LogoutHandler(authManager))
		}

		// Protected routes
		protected := api.Group("")
		protected.Use(auth.AuthMiddleware())
		{
			protected.GET("/me", auth.MeHandler(authManager))

			// Admin-only routes
			adminRoutes := protected.Group("")
			adminRoutes.Use(auth.AdminMiddleware())
			{
				adminRoutes.GET("/users", auth.GetUsersHandler(authManager))
			}

			// Track routes
			trackRoutes := protected.Group("/tracks")
			{
				trackRoutes.POST("", tracks.UploadHandler(tracksManager))
				trackRoutes.GET("", tracks.ListHandler(tracksManager))
				trackRoutes.GET("/:id", tracks.GetHandler(tracksManager))
				trackRoutes.GET("/:id/stream", tracks.StreamHandler(tracksManager))
				trackRoutes.DELETE("/:id", tracks.DeleteHandler(tracksManager))
			}

			// Playlist routes
			playlistRoutes := protected.Group("/playlists")
			{
				playlistRoutes.POST("", playlists.CreatePlaylistHandler(playlistsManager))
				playlistRoutes.GET("", playlists.GetPlaylistsHandler(playlistsManager))
				playlistRoutes.GET("/:id", playlists.GetPlaylistHandler(playlistsManager))
				playlistRoutes.PUT("/:id", playlists.UpdatePlaylistHandler(playlistsManager))
				playlistRoutes.DELETE("/:id", playlists.DeletePlaylistHandler(playlistsManager))

				// Playlist track management
				playlistRoutes.POST("/:id/tracks", playlists.AddTracksToPlaylistHandler(playlistsManager))
				playlistRoutes.DELETE("/:id/tracks", playlists.RemoveTracksFromPlaylistHandler(playlistsManager))
				playlistRoutes.GET("/:id/tracks", playlists.GetPlaylistTracksHandler(playlistsManager))
			}

			// Unsorted tracks endpoint (tracks not in any playlist)
			protected.GET("/tracks/unsorted", playlists.GetUnsortedTracksHandler(playlistsManager))
		}
	}

	addr := "0.0.0.0:8080"
	fmt.Printf("[CrateDrop] Server listening on http://%s\n", addr)
	fmt.Printf("[CrateDrop] API available at http://%s/api\n", addr)
	fmt.Printf("[CrateDrop] Health check at http://%s/api/healthz\n", addr)

	if err := r.Run(addr); err != nil {
		log.Fatalf("[CrateDrop] Server error: %v", err)
	}
}
