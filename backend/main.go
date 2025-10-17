package main

import (
	"fmt"
	"log"
	"path/filepath"
	"time"

	"github.com/faraz525/home-music-server/backend/auth"
	"github.com/faraz525/home-music-server/backend/internal/config"
	idb "github.com/faraz525/home-music-server/backend/internal/db"
	mlocal "github.com/faraz525/home-music-server/backend/internal/media/metadata/local"
	slocal "github.com/faraz525/home-music-server/backend/internal/storage/local"
	"github.com/faraz525/home-music-server/backend/playlists"
	"github.com/faraz525/home-music-server/backend/server"
	"github.com/faraz525/home-music-server/backend/tracks"
	"github.com/faraz525/home-music-server/backend/trades"
	"github.com/faraz525/home-music-server/backend/users"
)

func main() {
	// Enhanced startup logging
	fmt.Printf("[CrateDrop] Starting CrateDrop server at %s...\n", time.Now().Format("2006-01-02 15:04:05"))

	// Load config
	cfg := config.FromEnv()
	fmt.Printf("[CrateDrop] Using data directory: %s\n", cfg.DataDir)

	// Initialize database
	dbPath := filepath.Join(cfg.DataDir, "db", "cratedrop.sqlite")
	fmt.Printf("[CrateDrop] Database path: %s\n", dbPath)
	db, err := idb.New(cfg.DataDir)
	if err != nil {
		log.Fatalf("[CrateDrop] Failed to initialize database: %v", err)
	}
	defer db.Close()
	fmt.Printf("[CrateDrop] Database initialized successfully\n")

	// Initialize repositories
	authRepo := auth.NewRepository(db)
	tracksRepo := tracks.NewRepository(db)
	playlistsRepo := playlists.NewRepository(db)
	usersRepo := users.NewRepository(db)
	tradesRepo := trades.NewRepository(db)
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
	playlistsManager := playlists.NewManager(playlistsRepo)
	usersManager := users.NewManager(usersRepo)
	tradesManager := trades.NewManager(tradesRepo)
	
	// Set dependencies
	tradesManager.SetPlaylistGetter(playlistsRepo)
	
	fmt.Printf("[CrateDrop] Tracks, playlists, users, and trades managers initialized\n")

	// Initialize router and API group
	r, api := server.NewRouter()

	// Register feature-owned routes
	auth.Routes(authManager)(api)
	protected := api.Group("")
	protected.Use(auth.AuthMiddleware())
	tracks.Routes(tracksManager, playlistsManager, tradesRepo)(protected)
	playlists.Routes(playlistsManager)(protected)
	users.RegisterRoutes(api, usersManager)
	trades.RegisterRoutes(api, tradesManager)

	addr := "0.0.0.0:" + cfg.Port
	fmt.Printf("[CrateDrop] Server listening on http://%s\n", addr)
	fmt.Printf("[CrateDrop] API available at http://%s/api\n", addr)
	fmt.Printf("[CrateDrop] Health check at http://%s/api/healthz\n", addr)

	if err := r.Run(addr); err != nil {
		log.Fatalf("[CrateDrop] Server error: %v", err)
	}
}
