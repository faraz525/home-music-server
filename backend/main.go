package main

import (
	"fmt"
	"log"
	"time"

	"github.com/faraz525/home-music-server/backend/auth"
	"github.com/faraz525/home-music-server/backend/internal/config"
	idb "github.com/faraz525/home-music-server/backend/internal/db"
	mlocal "github.com/faraz525/home-music-server/backend/internal/media/metadata/local"
	"github.com/faraz525/home-music-server/backend/server"
	slocal "github.com/faraz525/home-music-server/backend/internal/storage/local"
	"github.com/faraz525/home-music-server/backend/tracks"
)

func main() {
	// Enhanced startup logging
	fmt.Printf("[CrateDrop] Starting CrateDrop server at %s...\n", time.Now().Format("2006-01-02 15:04:05"))

	// Load config
	cfg := config.FromEnv()
	fmt.Printf("[CrateDrop] Using data directory: %s\n", cfg.DataDir)

	// Initialize database
	db, err := idb.New(cfg.DataDir)
	if err != nil {
		log.Fatalf("[CrateDrop] Failed to initialize database: %v", err)
	}
	defer db.Close()
	fmt.Printf("[CrateDrop] Database initialized successfully\n")

	// Initialize repositories
	authRepo := auth.NewRepository(db)
	tracksRepo := tracks.NewRepository(db)
	fmt.Printf("[CrateDrop] Repositories initialized\n")

	// Initialize managers and infra
	authManager, err := auth.NewManager(authRepo)
	if err != nil {
		log.Fatalf("[CrateDrop] Failed to initialize auth manager: %v", err)
	}
	fmt.Printf("[CrateDrop] Auth manager initialized\n")

	storage := slocal.New(cfg.DataDir)
	extractor := mlocal.New()
	tracksManager := tracks.NewManager(tracksRepo, storage, extractor)
	fmt.Printf("[CrateDrop] Tracks manager initialized\n")

	// Initialize router and API group
	r, api := server.NewRouter()

	// Register feature-owned routes
	auth.Routes(authManager)(api)
	protected := api.Group("")
	protected.Use(auth.AuthMiddleware())
	tracks.Routes(tracksManager)(protected)

	addr := "0.0.0.0:" + cfg.Port
	fmt.Printf("[CrateDrop] Server listening on http://%s\n", addr)
	fmt.Printf("[CrateDrop] API available at http://%s/api\n", addr)
	fmt.Printf("[CrateDrop] Health check at http://%s/api/healthz\n", addr)

	if err := r.Run(addr); err != nil {
		log.Fatalf("[CrateDrop] Server error: %v", err)
	}
}
