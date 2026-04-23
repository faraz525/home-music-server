package spotify

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/faraz525/home-music-server/backend/internal/media/metadata"
	imodels "github.com/faraz525/home-music-server/backend/internal/models"
	"github.com/faraz525/home-music-server/backend/internal/storage"
	"github.com/faraz525/home-music-server/backend/monochrome"
	"github.com/faraz525/home-music-server/backend/playlists"
	"github.com/faraz525/home-music-server/backend/tracks"
	"github.com/faraz525/home-music-server/backend/utils"
)

const (
	SpotifyAuthURL  = "https://accounts.spotify.com/authorize"
	SpotifyTokenURL = "https://accounts.spotify.com/api/token"
	SpotifyAPIURL   = "https://api.spotify.com/v1"
)

type Manager struct {
	repo             *Repository
	tracksRepo       *tracks.Repository
	storage          storage.Storage
	extractor        metadata.Extractor
	playlistsManager *playlists.Manager
	dataDir          string
	clientID         string
	monochrome       *monochrome.Client // optional; nil = disabled, falls straight to yt-dlp
	syncMutex        sync.Mutex
}

// NewManager builds a Spotify sync manager. monoClient may be nil — when set,
// downloads try monochrome.tf (TIDAL FLAC) first and fall back to yt-dlp.
func NewManager(
	repo *Repository,
	tracksRepo *tracks.Repository,
	storage storage.Storage,
	extractor metadata.Extractor,
	pm *playlists.Manager,
	dataDir string,
	monoClient *monochrome.Client,
) *Manager {
	clientID := os.Getenv("SPOTIFY_CLIENT_ID")
	return &Manager{
		repo:             repo,
		tracksRepo:       tracksRepo,
		storage:          storage,
		extractor:        extractor,
		playlistsManager: pm,
		dataDir:          dataDir,
		clientID:         clientID,
		monochrome:       monoClient,
	}
}

func (m *Manager) GetClientID() string {
	return m.clientID
}

func (m *Manager) GetSyncConfig(ctx context.Context) (*SyncConfig, error) {
	return m.repo.GetSyncConfig(ctx)
}

func (m *Manager) SaveSyncConfig(ctx context.Context, ownerUserID string, enabled bool) error {
	cfg := &SyncConfig{
		OwnerUserID: ownerUserID,
		Enabled:     enabled,
	}

	existing, _ := m.repo.GetSyncConfig(ctx)
	if existing != nil {
		cfg.AccessToken = existing.AccessToken
		cfg.RefreshToken = existing.RefreshToken
		cfg.TokenExpiresAt = existing.TokenExpiresAt
		cfg.LikedSongsPlaylistID = existing.LikedSongsPlaylistID
	}

	return m.repo.UpsertSyncConfig(ctx, cfg)
}

func (m *Manager) GetSyncHistory(ctx context.Context, limit int) ([]*SyncHistory, error) {
	if limit <= 0 || limit > 50 {
		limit = 10
	}
	return m.repo.GetSyncHistory(ctx, limit)
}

// ExchangeCodeForToken exchanges an authorization code for access and refresh tokens using PKCE
func (m *Manager) ExchangeCodeForToken(ctx context.Context, code, codeVerifier, redirectURI, ownerUserID string) error {
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", redirectURI)
	data.Set("client_id", m.clientID)
	data.Set("code_verifier", codeVerifier)

	req, err := http.NewRequestWithContext(ctx, "POST", SpotifyTokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to exchange token: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("token exchange failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
		TokenType    string `json:"token_type"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return fmt.Errorf("failed to parse token response: %w", err)
	}

	expiresAt := time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)

	cfg := &SyncConfig{
		AccessToken:    &tokenResp.AccessToken,
		RefreshToken:   &tokenResp.RefreshToken,
		TokenExpiresAt: &expiresAt,
		OwnerUserID:    ownerUserID,
		Enabled:        false, // User must explicitly enable
	}

	// Preserve existing playlist ID if config exists
	existing, _ := m.repo.GetSyncConfig(ctx)
	if existing != nil {
		cfg.LikedSongsPlaylistID = existing.LikedSongsPlaylistID
		cfg.Enabled = existing.Enabled
	}

	return m.repo.UpsertSyncConfig(ctx, cfg)
}

// RefreshAccessToken refreshes the access token using the refresh token
func (m *Manager) RefreshAccessToken(ctx context.Context) error {
	cfg, err := m.repo.GetSyncConfig(ctx)
	if err != nil || cfg == nil || cfg.RefreshToken == nil {
		return fmt.Errorf("no refresh token available")
	}

	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", *cfg.RefreshToken)
	data.Set("client_id", m.clientID)

	req, err := http.NewRequestWithContext(ctx, "POST", SpotifyTokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create refresh request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to refresh token: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("token refresh failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return fmt.Errorf("failed to parse refresh response: %w", err)
	}

	expiresAt := time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)

	// If no new refresh token is provided, keep the old one
	refreshToken := *cfg.RefreshToken
	if tokenResp.RefreshToken != "" {
		refreshToken = tokenResp.RefreshToken
	}

	return m.repo.UpdateTokens(ctx, tokenResp.AccessToken, refreshToken, expiresAt)
}

// getValidToken returns a valid access token, refreshing if necessary
func (m *Manager) getValidToken(ctx context.Context) (string, error) {
	cfg, err := m.repo.GetSyncConfig(ctx)
	if err != nil || cfg == nil {
		return "", fmt.Errorf("spotify not configured")
	}

	if cfg.AccessToken == nil || *cfg.AccessToken == "" {
		return "", fmt.Errorf("no access token")
	}

	// Refresh if token expires in the next 5 minutes
	if cfg.TokenExpiresAt != nil && time.Until(*cfg.TokenExpiresAt) < 5*time.Minute {
		if err := m.RefreshAccessToken(ctx); err != nil {
			return "", fmt.Errorf("failed to refresh token: %w", err)
		}
		// Reload config after refresh
		cfg, _ = m.repo.GetSyncConfig(ctx)
	}

	return *cfg.AccessToken, nil
}

// SpotifyTrack represents a track from Spotify API
type SpotifyTrack struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Artists []struct {
		Name string `json:"name"`
	} `json:"artists"`
	Album struct {
		Name string `json:"name"`
	} `json:"album"`
	DurationMs   int `json:"duration_ms"`
	ExternalURLs struct {
		Spotify string `json:"spotify"`
	} `json:"external_urls"`
	ExternalIDs struct {
		ISRC string `json:"isrc"`
	} `json:"external_ids"`
}

// SpotifyPlaylist represents a playlist from Spotify API
type SpotifyPlaylist struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Tracks struct {
		Total int `json:"total"`
	} `json:"tracks"`
}

// FetchLikedSongs fetches the user's liked songs from Spotify
func (m *Manager) FetchLikedSongs(ctx context.Context, limit, offset int) ([]SpotifyTrack, int, error) {
	token, err := m.getValidToken(ctx)
	if err != nil {
		return nil, 0, err
	}

	url := fmt.Sprintf("%s/me/tracks?limit=%d&offset=%d", SpotifyAPIURL, limit, offset)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, 0, fmt.Errorf("spotify API error %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Items []struct {
			Track SpotifyTrack `json:"track"`
		} `json:"items"`
		Total int `json:"total"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, 0, err
	}

	tracks := make([]SpotifyTrack, len(result.Items))
	for i, item := range result.Items {
		tracks[i] = item.Track
	}

	return tracks, result.Total, nil
}

// FetchUserPlaylists fetches the user's playlists from Spotify
func (m *Manager) FetchUserPlaylists(ctx context.Context) ([]SpotifyPlaylist, error) {
	token, err := m.getValidToken(ctx)
	if err != nil {
		return nil, err
	}

	var allPlaylists []SpotifyPlaylist
	offset := 0
	limit := 50

	for {
		url := fmt.Sprintf("%s/me/playlists?limit=%d&offset=%d", SpotifyAPIURL, limit, offset)
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+token)

		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return nil, fmt.Errorf("spotify API error %d: %s", resp.StatusCode, string(body))
		}

		var result struct {
			Items []SpotifyPlaylist `json:"items"`
			Total int               `json:"total"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			resp.Body.Close()
			return nil, err
		}
		resp.Body.Close()

		allPlaylists = append(allPlaylists, result.Items...)

		if len(allPlaylists) >= result.Total {
			break
		}
		offset += limit
	}

	return allPlaylists, nil
}

// FetchPlaylistTracks fetches tracks from a specific playlist
func (m *Manager) FetchPlaylistTracks(ctx context.Context, playlistID string, limit, offset int) ([]SpotifyTrack, int, error) {
	token, err := m.getValidToken(ctx)
	if err != nil {
		return nil, 0, err
	}

	url := fmt.Sprintf("%s/playlists/%s/tracks?limit=%d&offset=%d", SpotifyAPIURL, playlistID, limit, offset)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, 0, fmt.Errorf("spotify API error %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Items []struct {
			Track SpotifyTrack `json:"track"`
		} `json:"items"`
		Total int `json:"total"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, 0, err
	}

	tracks := make([]SpotifyTrack, 0, len(result.Items))
	for _, item := range result.Items {
		// Skip nil tracks (can happen with local files)
		if item.Track.ID != "" {
			tracks = append(tracks, item.Track)
		}
	}

	return tracks, result.Total, nil
}

func (m *Manager) SyncLikes(ctx context.Context) error {
	if !m.syncMutex.TryLock() {
		fmt.Println("[Spotify] Sync already in progress, skipping")
		return nil
	}
	defer m.syncMutex.Unlock()

	fmt.Println("[Spotify] Starting sync...")
	started := time.Now()

	cfg, err := m.repo.GetSyncConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to get sync config: %w", err)
	}

	if cfg == nil {
		fmt.Println("[Spotify] Sync not configured, skipping")
		return nil
	}

	if !cfg.Enabled {
		fmt.Println("[Spotify] Sync disabled, skipping")
		return nil
	}

	if cfg.AccessToken == nil || *cfg.AccessToken == "" {
		return fmt.Errorf("access token not configured")
	}

	playlistID, err := m.ensureLikedSongsPlaylist(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to ensure playlist: %w", err)
	}

	tracksAdded := 0
	tracksSkipped := 0
	var trackIDs []string

	// Fetch liked songs with pagination
	offset := 0
	limit := 50
	for {
		spotifyTracks, total, err := m.FetchLikedSongs(ctx, limit, offset)
		if err != nil {
			errMsg := fmt.Sprintf("failed to fetch liked songs: %v", err)
			m.repo.RecordSync(ctx, started, tracksAdded, tracksSkipped, &errMsg)
			return fmt.Errorf(errMsg)
		}

		for _, st := range spotifyTracks {
			synced, err := m.repo.IsSpotifyTrackSynced(ctx, st.ID)
			if err != nil {
				fmt.Printf("[Spotify] Error checking if track synced: %v\n", err)
			}
			if synced {
				fmt.Printf("[Spotify] Skipping already synced track: %s\n", st.Name)
				tracksSkipped++
				continue
			}

			trackID, err := m.downloadAndImportTrack(ctx, cfg.OwnerUserID, st)
			if err != nil {
				fmt.Printf("[Spotify] Skipping track %s: %v\n", st.Name, err)
				tracksSkipped++
				continue
			}

			if err := m.repo.RecordSyncedTrack(ctx, st.ID, trackID); err != nil {
				fmt.Printf("[Spotify] Warning: failed to record synced track: %v\n", err)
			}

			trackIDs = append(trackIDs, trackID)
			tracksAdded++
			fmt.Printf("[Spotify] Imported: %s - %s\n", getArtistName(st), st.Name)
		}

		offset += limit
		if offset >= total {
			break
		}
	}

	if len(trackIDs) > 0 {
		req := &imodels.AddTracksToPlaylistRequest{TrackIDs: trackIDs}
		if err := m.playlistsManager.AddTracksToPlaylist(playlistID, cfg.OwnerUserID, req); err != nil {
			fmt.Printf("[Spotify] Warning: failed to add tracks to playlist: %v\n", err)
		}
	}

	m.repo.UpdateLastSync(ctx)
	m.repo.RecordSync(ctx, started, tracksAdded, tracksSkipped, nil)

	fmt.Printf("[Spotify] Sync complete: %d added, %d skipped\n", tracksAdded, tracksSkipped)
	return nil
}

func (m *Manager) ensureLikedSongsPlaylist(ctx context.Context, cfg *SyncConfig) (string, error) {
	if cfg.LikedSongsPlaylistID != nil && *cfg.LikedSongsPlaylistID != "" {
		return *cfg.LikedSongsPlaylistID, nil
	}

	desc := "Auto-synced from Spotify liked songs"
	isPublic := false
	req := &imodels.CreatePlaylistRequest{
		Name:        "Spotify Liked Songs",
		Description: &desc,
		IsPublic:    &isPublic,
	}

	playlist, err := m.playlistsManager.CreatePlaylist(cfg.OwnerUserID, req)
	if err != nil {
		return "", err
	}

	if err := m.repo.UpdateLikedSongsPlaylistID(ctx, playlist.ID); err != nil {
		fmt.Printf("[Spotify] Warning: failed to update playlist ID in config: %v\n", err)
	}

	fmt.Printf("[Spotify] Created playlist: %s (%s)\n", playlist.Name, playlist.ID)
	return playlist.ID, nil
}

func (m *Manager) downloadAndImportTrack(ctx context.Context, userID string, st SpotifyTrack) (string, error) {
	tmpDir := filepath.Join(m.dataDir, "tmp", fmt.Sprintf("spotify-%s", st.ID))
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Prefer monochrome (TIDAL FLAC) when configured and ISRC is known.
	// Any failure falls through to yt-dlp.
	if m.monochrome != nil && st.ExternalIDs.ISRC != "" {
		if err := m.tryMonochromeDownload(ctx, tmpDir, st); err != nil {
			fmt.Printf("[Spotify] monochrome download failed for %s (ISRC %s): %v — falling back to yt-dlp\n",
				st.Name, st.ExternalIDs.ISRC, err)
		}
	}

	// yt-dlp fallback only runs if monochrome produced no file.
	if existing, _ := os.ReadDir(tmpDir); len(existing) == 0 {
		searchQuery := fmt.Sprintf("%s %s", getArtistName(st), st.Name)
		outputTemplate := filepath.Join(tmpDir, "%(title)s.%(ext)s")

		cmd := exec.CommandContext(ctx, "yt-dlp",
			"--default-search", "ytsearch1",
			"-x", "--audio-format", "mp3",
			"--audio-quality", "0",
			"-o", outputTemplate,
			searchQuery,
		)

		output, err := cmd.CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("download failed: %v, output: %s", err, string(output))
		}
	}

	files, err := os.ReadDir(tmpDir)
	if err != nil || len(files) == 0 {
		return "", fmt.Errorf("no files downloaded")
	}

	downloadedFile := filepath.Join(tmpDir, files[0].Name())

	file, err := os.Open(downloadedFile)
	if err != nil {
		return "", fmt.Errorf("failed to open downloaded file: %w", err)
	}
	defer file.Close()

	filename := files[0].Name()
	trackID := utils.GenerateTrackID()

	relPath, size, contentType, err := m.storage.Save(ctx, userID, trackID, filename, file)
	if err != nil {
		return "", fmt.Errorf("failed to save file: %w", err)
	}

	fullPath, _ := m.storage.ResolveFullPath(relPath)
	md, err := m.extractor.Extract(ctx, fullPath)
	if err != nil {
		fmt.Printf("[Spotify] Warning: metadata extraction failed for %s: %v\n", filename, err)
		md = &metadata.AudioMetadata{}
	}

	track := &imodels.Track{
		OwnerUserID:      userID,
		OriginalFilename: filename,
		ContentType:      contentType,
		SizeBytes:        size,
		FilePath:         relPath,
		CreatedAt:        utils.Now(),
		UpdatedAt:        utils.Now(),
	}

	// Prefer Spotify metadata over extracted metadata
	title := st.Name
	track.Title = &title

	artist := getArtistName(st)
	if artist != "" {
		track.Artist = &artist
	} else if md.Artist != nil {
		track.Artist = md.Artist
	}

	album := st.Album.Name
	if album != "" {
		track.Album = &album
	} else if md.Album != nil {
		track.Album = md.Album
	}

	duration := float64(st.DurationMs) / 1000
	if duration > 0 {
		track.DurationSeconds = &duration
	} else if md.DurationSeconds > 0 {
		track.DurationSeconds = &md.DurationSeconds
	}

	if md.SampleRate != nil {
		track.SampleRate = md.SampleRate
	}
	if md.Bitrate != nil {
		track.Bitrate = md.Bitrate
	}

	track, err = m.tracksRepo.CreateTrack(ctx, track)
	if err != nil {
		m.storage.Delete(ctx, relPath)
		return "", fmt.Errorf("failed to create track record: %w", err)
	}

	return track.ID, nil
}

func getArtistName(st SpotifyTrack) string {
	if len(st.Artists) > 0 {
		names := make([]string, len(st.Artists))
		for i, a := range st.Artists {
			names[i] = a.Name
		}
		return strings.Join(names, ", ")
	}
	return ""
}

// tryMonochromeDownload attempts to resolve st to a TIDAL FLAC via monochrome
// and saves it into tmpDir. Duration is used to guard against wrong-version
// matches (clean edits, remasters) when multiple ISRC hits are returned.
func (m *Manager) tryMonochromeDownload(ctx context.Context, tmpDir string, st SpotifyTrack) error {
	matches, err := m.monochrome.SearchByISRC(ctx, st.ExternalIDs.ISRC)
	if err != nil {
		return fmt.Errorf("search: %w", err)
	}
	if len(matches) == 0 {
		return fmt.Errorf("no ISRC match")
	}

	spotifyDurSec := st.DurationMs / 1000
	best := matches[0]
	bestDiff := absInt(best.DurationSec - spotifyDurSec)
	for _, cand := range matches[1:] {
		d := absInt(cand.DurationSec - spotifyDurSec)
		if d < bestDiff {
			best = cand
			bestDiff = d
		}
	}
	if spotifyDurSec > 0 && bestDiff > 5 {
		return fmt.Errorf("duration mismatch: tidal=%ds spotify=%ds", best.DurationSec, spotifyDurSec)
	}

	info, err := m.monochrome.GetStreamInfo(ctx, best.TidalID, monochrome.QualityHiRes)
	if err != nil {
		return fmt.Errorf("stream info: %w", err)
	}
	// Guard against silent downgrades to AAC when the backend account has been
	// restricted — we only want this path for actual lossless FLAC; anything
	// else should fall back to yt-dlp.
	if !strings.EqualFold(info.Codec, "flac") {
		return fmt.Errorf("unexpected codec %q (quality=%s); want flac", info.Codec, info.Quality)
	}

	safeName := sanitizeFilename(st.Name)
	if safeName == "" {
		safeName = fmt.Sprintf("track_%d", best.TidalID)
	}
	destPath := filepath.Join(tmpDir, safeName+".flac")
	if err := m.monochrome.Download(ctx, info.URL, destPath); err != nil {
		return fmt.Errorf("download: %w", err)
	}

	fmt.Printf("[Spotify] monochrome: %s — tidal=%d quality=%s\n", st.Name, best.TidalID, info.Quality)
	return nil
}

// sanitizeFilename strips filesystem-reserved chars and control bytes; caps at 100 runes.
func sanitizeFilename(s string) string {
	s = strings.TrimSpace(s)
	var b strings.Builder
	for _, r := range s {
		switch {
		case r == '/' || r == '\\' || r == ':' || r == '*' || r == '?' || r == '"' || r == '<' || r == '>' || r == '|':
			b.WriteByte('_')
		case r < 0x20:
			// skip control chars
		default:
			b.WriteRune(r)
		}
	}
	out := b.String()
	if len(out) > 100 {
		out = out[:100]
	}
	return out
}

func absInt(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// GetSyncedPlaylists returns all synced Spotify playlists
func (m *Manager) GetSyncedPlaylists(ctx context.Context) ([]*SyncedPlaylist, error) {
	return m.repo.GetSyncedPlaylists(ctx)
}

func (m *Manager) Disconnect(ctx context.Context) error {
	cfg, err := m.repo.GetSyncConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to get sync config: %w", err)
	}
	if cfg == nil {
		return nil
	}

	cfg.AccessToken = nil
	cfg.RefreshToken = nil
	cfg.TokenExpiresAt = nil
	cfg.Enabled = false

	return m.repo.UpsertSyncConfig(ctx, cfg)
}
