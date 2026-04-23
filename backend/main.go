package main

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/faraz525/home-music-server/backend/analysis"
	"github.com/faraz525/home-music-server/backend/auth"
	"github.com/faraz525/home-music-server/backend/internal/config"
	idb "github.com/faraz525/home-music-server/backend/internal/db"
	mlocal "github.com/faraz525/home-music-server/backend/internal/media/metadata/local"
	slocal "github.com/faraz525/home-music-server/backend/internal/storage/local"
	"github.com/faraz525/home-music-server/backend/monochrome"
	"github.com/faraz525/home-music-server/backend/playlists"
	"github.com/faraz525/home-music-server/backend/server"
	"github.com/faraz525/home-music-server/backend/soundcloud"
	"github.com/faraz525/home-music-server/backend/spotify"
	"github.com/faraz525/home-music-server/backend/tracks"
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
	fmt.Printf("[CrateDrop] Tracks and playlists managers initialized\n")

	// Initialize SoundCloud sync manager
	soundcloudRepo := soundcloud.NewRepository(db)
	soundcloudManager := soundcloud.NewManager(
		soundcloudRepo,
		tracksRepo,
		storage,
		extractor,
		playlistsManager,
		cfg.DataDir,
	)
	fmt.Printf("[CrateDrop] SoundCloud sync manager initialized\n")

	// Initialize monochrome.tf client (optional — enables FLAC downloads via TIDAL)
	var monoClient *monochrome.Client
	if cfg.MonochromeAPIURL != "" {
		hosts := strings.Split(cfg.MonochromeAPIURL, ",")
		monoClient = monochrome.NewClient(hosts, 120*time.Second)
		fmt.Printf("[CrateDrop] Monochrome client enabled (%d backend(s): %v)\n",
			len(monoClient.Hosts()), monoClient.Hosts())
	} else {
		fmt.Printf("[CrateDrop] Monochrome client disabled (set MONOCHROME_API_URL to enable; comma-separated list supported)\n")
	}

	// Initialize Spotify sync manager
	spotifyRepo := spotify.NewRepository(db)
	spotifyManager := spotify.NewManager(
		spotifyRepo,
		tracksRepo,
		storage,
		extractor,
		playlistsManager,
		cfg.DataDir,
		monoClient,
	)
	fmt.Printf("[CrateDrop] Spotify sync manager initialized\n")

	// Initialize analysis (BPM + key detection)
	analysisRepo := analysis.NewRepository(db.DB)
	analyzer := analysis.NewAnalyzer(90 * time.Second)
	analysisManager := analysis.NewManager(analysisRepo, analyzer)
	if analysis.BinaryAvailable() {
		fmt.Printf("[CrateDrop] Analysis worker enabled (essentia binary found)\n")
	} else {
		fmt.Printf("[CrateDrop] WARNING: streaming_extractor_music not on PATH — analysis disabled\n")
	}

	// Initialize router and API group
	r, api := server.NewRouter()

	// Register feature-owned routes
	auth.Routes(authManager)(api)
	protected := api.Group("")
	protected.Use(auth.AuthMiddleware())
	tracks.Routes(tracksManager, playlistsManager)(protected)
	playlists.Routes(playlistsManager)(protected)
	soundcloud.Routes(soundcloudManager)(protected)
	spotify.Routes(spotifyManager)(protected)

	// Start sync loops in background
	ctx := context.Background()
	go soundcloud.StartSyncLoop(ctx, soundcloudManager)
	go spotify.StartSyncLoop(ctx, spotifyManager)

	if analysis.BinaryAvailable() {
		go analysis.StartLoop(ctx, analysisManager, 10*time.Second)
	}

	addr := "0.0.0.0:" + cfg.Port
	fmt.Printf("[CrateDrop] Server listening on http://%s\n", addr)
	fmt.Printf("[CrateDrop] API available at http://%s/api\n", addr)
	fmt.Printf("[CrateDrop] Health check at http://%s/api/healthz\n", addr)

	if err := r.Run(addr); err != nil {
		log.Fatalf("[CrateDrop] Server error: %v", err)
	}
}
