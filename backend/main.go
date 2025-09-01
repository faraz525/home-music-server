package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"

	"github.com/faraz525/home-music-server/backend/auth"
	"github.com/faraz525/home-music-server/backend/tracks"
	"github.com/faraz525/home-music-server/backend/utils"
)

func main() {
	// Initialize database
	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "/mnt/music/cratedrop"
	}

	db, err := utils.NewDB(dataDir)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Initialize repositories
	authRepo := auth.NewRepository(db)
	tracksRepo := tracks.NewRepository(db)

	// Initialize managers
	authManager, err := auth.NewManager(authRepo)
	if err != nil {
		log.Fatalf("Failed to initialize auth manager: %v", err)
	}

	tracksManager := tracks.NewManager(tracksRepo)

	// Initialize Gin router
	r := gin.Default()

	// CORS middleware
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization")
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
		}
	}

	addr := "0.0.0.0:8080"
	log.Printf("Starting CrateDrop server on %s...", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
