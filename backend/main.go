package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/faraz525/home-music-server/backend/auth"
	"github.com/faraz525/home-music-server/backend/server"
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
	fmt.Printf("[CrateDrop] Repositories initialized\n")

	// Initialize managers
	authManager, err := auth.NewManager(authRepo)
	if err != nil {
		log.Fatalf("[CrateDrop] Failed to initialize auth manager: %v", err)
	}
	fmt.Printf("[CrateDrop] Auth manager initialized\n")

	tracksManager := tracks.NewManager(tracksRepo)
	fmt.Printf("[CrateDrop] Tracks manager initialized\n")

	// Initialize router and API group
	r, api := server.NewRouter()

	// Register feature-owned routes
	auth.Routes(authManager)(api)
	protected := api.Group("")
	protected.Use(auth.AuthMiddleware())
	tracks.Routes(tracksManager)(protected)

	addr := "0.0.0.0:8080"
	fmt.Printf("[CrateDrop] Server listening on http://%s\n", addr)
	fmt.Printf("[CrateDrop] API available at http://%s/api\n", addr)
	fmt.Printf("[CrateDrop] Health check at http://%s/api/healthz\n", addr)

	if err := r.Run(addr); err != nil {
		log.Fatalf("[CrateDrop] Server error: %v", err)
	}
}
