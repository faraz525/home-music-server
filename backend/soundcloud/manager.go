package soundcloud

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/faraz525/home-music-server/backend/internal/media/metadata"
	imodels "github.com/faraz525/home-music-server/backend/internal/models"
	"github.com/faraz525/home-music-server/backend/internal/storage"
	"github.com/faraz525/home-music-server/backend/playlists"
	"github.com/faraz525/home-music-server/backend/tracks"
	"github.com/faraz525/home-music-server/backend/utils"
)

type Manager struct {
	repo             *Repository
	tracksRepo       *tracks.Repository
	storage          storage.Storage
	extractor        metadata.Extractor
	playlistsManager *playlists.Manager
	dataDir          string
	syncMutex        sync.Mutex
}

func NewManager(
	repo *Repository,
	tracksRepo *tracks.Repository,
	storage storage.Storage,
	extractor metadata.Extractor,
	pm *playlists.Manager,
	dataDir string,
) *Manager {
	return &Manager{
		repo:             repo,
		tracksRepo:       tracksRepo,
		storage:          storage,
		extractor:        extractor,
		playlistsManager: pm,
		dataDir:          dataDir,
	}
}

type ManifestEntry struct {
	FilePath     string `json:"file_path"`
	Title        string `json:"title"`
	Artist       string `json:"artist"`
	Duration     int    `json:"duration"`
	SoundCloudID string `json:"soundcloud_id"`
}

func (m *Manager) GetSyncConfig(ctx context.Context) (*SyncConfig, error) {
	return m.repo.GetSyncConfig(ctx)
}

func (m *Manager) SaveSyncConfig(ctx context.Context, token, ownerUserID string, enabled bool) error {
	cfg := &SyncConfig{
		OwnerUserID: ownerUserID,
		Enabled:     enabled,
	}

	existing, _ := m.repo.GetSyncConfig(ctx)
	if existing != nil {
		cfg.PlaylistID = existing.PlaylistID
		if token == "" {
			cfg.OAuthToken = existing.OAuthToken
		} else {
			cfg.OAuthToken = &token
		}
	} else if token != "" {
		cfg.OAuthToken = &token
	}

	return m.repo.UpsertSyncConfig(ctx, cfg)
}

func (m *Manager) GetSyncHistory(ctx context.Context, limit int) ([]*SyncHistory, error) {
	if limit <= 0 || limit > 50 {
		limit = 10
	}
	return m.repo.GetSyncHistory(ctx, limit)
}

func (m *Manager) SyncLikes(ctx context.Context) error {
	if !m.syncMutex.TryLock() {
		fmt.Println("[SoundCloud] Sync already in progress, skipping")
		return nil
	}
	defer m.syncMutex.Unlock()

	fmt.Println("[SoundCloud] Starting sync...")
	started := time.Now()

	cfg, err := m.repo.GetSyncConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to get sync config: %w", err)
	}

	if cfg == nil {
		fmt.Println("[SoundCloud] Sync not configured, skipping")
		return nil
	}

	if !cfg.Enabled {
		fmt.Println("[SoundCloud] Sync disabled, skipping")
		return nil
	}

	if cfg.OAuthToken == nil || *cfg.OAuthToken == "" {
		return fmt.Errorf("oauth token not configured")
	}

	playlistID, err := m.ensurePlaylist(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to ensure playlist: %w", err)
	}

	tmpDir := filepath.Join(m.dataDir, "tmp", fmt.Sprintf("soundcloud-sync-%d", time.Now().UnixNano()))
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	scriptPath := filepath.Join(m.dataDir, "soundcloud", "downloader.py")
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		scriptPath = "soundcloud/downloader.py"
	}

	cmd := exec.CommandContext(ctx, "python3", scriptPath, *cfg.OAuthToken, tmpDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		errMsg := fmt.Sprintf("download failed: %v, output: %s", err, string(output))
		m.repo.RecordSync(ctx, started, 0, 0, &errMsg)
		return fmt.Errorf(errMsg)
	}

	fmt.Printf("[SoundCloud] Download script output:\n%s\n", string(output))

	manifestPath := filepath.Join(tmpDir, "manifest.json")
	manifestFile, err := os.Open(manifestPath)
	if err != nil {
		errMsg := fmt.Sprintf("failed to open manifest: %v", err)
		m.repo.RecordSync(ctx, started, 0, 0, &errMsg)
		return fmt.Errorf(errMsg)
	}
	defer manifestFile.Close()

	var manifest []ManifestEntry
	if err := json.NewDecoder(manifestFile).Decode(&manifest); err != nil {
		errMsg := fmt.Sprintf("failed to parse manifest: %v", err)
		m.repo.RecordSync(ctx, started, 0, 0, &errMsg)
		return fmt.Errorf(errMsg)
	}

	tracksAdded := 0
	tracksSkipped := 0
	var trackIDs []string

	for _, entry := range manifest {
		if entry.SoundCloudID != "" {
			synced, err := m.repo.IsSoundCloudTrackSynced(ctx, entry.SoundCloudID)
			if err != nil {
				fmt.Printf("[SoundCloud] Error checking if track synced: %v\n", err)
			}
			if synced {
				fmt.Printf("[SoundCloud] Skipping already synced track: %s\n", entry.Title)
				tracksSkipped++
				continue
			}
		}

		trackID, err := m.importTrack(ctx, cfg.OwnerUserID, entry)
		if err != nil {
			fmt.Printf("[SoundCloud] Skipping track %s: %v\n", entry.Title, err)
			tracksSkipped++
			continue
		}

		if entry.SoundCloudID != "" {
			if err := m.repo.RecordSyncedTrack(ctx, entry.SoundCloudID, trackID); err != nil {
				fmt.Printf("[SoundCloud] Warning: failed to record synced track: %v\n", err)
			}
		}

		trackIDs = append(trackIDs, trackID)
		tracksAdded++
		fmt.Printf("[SoundCloud] Imported: %s\n", entry.Title)
	}

	if len(trackIDs) > 0 {
		req := &imodels.AddTracksToPlaylistRequest{TrackIDs: trackIDs}
		if err := m.playlistsManager.AddTracksToPlaylist(playlistID, cfg.OwnerUserID, req); err != nil {
			fmt.Printf("[SoundCloud] Warning: failed to add tracks to playlist: %v\n", err)
		}
	}

	m.repo.UpdateLastSync(ctx)
	m.repo.RecordSync(ctx, started, tracksAdded, tracksSkipped, nil)

	fmt.Printf("[SoundCloud] Sync complete: %d added, %d skipped\n", tracksAdded, tracksSkipped)
	return nil
}

func (m *Manager) ensurePlaylist(ctx context.Context, cfg *SyncConfig) (string, error) {
	if cfg.PlaylistID != nil && *cfg.PlaylistID != "" {
		return *cfg.PlaylistID, nil
	}

	desc := "Auto-synced from SoundCloud likes"
	isPublic := false
	req := &imodels.CreatePlaylistRequest{
		Name:        "SoundCloud Likes",
		Description: &desc,
		IsPublic:    &isPublic,
	}

	playlist, err := m.playlistsManager.CreatePlaylist(cfg.OwnerUserID, req)
	if err != nil {
		return "", err
	}

	if err := m.repo.UpdatePlaylistID(ctx, playlist.ID); err != nil {
		fmt.Printf("[SoundCloud] Warning: failed to update playlist ID in config: %v\n", err)
	}

	fmt.Printf("[SoundCloud] Created playlist: %s (%s)\n", playlist.Name, playlist.ID)
	return playlist.ID, nil
}

func (m *Manager) importTrack(ctx context.Context, userID string, entry ManifestEntry) (string, error) {
	file, err := os.Open(entry.FilePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	filename := filepath.Base(entry.FilePath)
	trackID := utils.GenerateTrackID()

	relPath, size, contentType, err := m.storage.Save(ctx, userID, trackID, filename, file)
	if err != nil {
		return "", fmt.Errorf("failed to save file: %w", err)
	}

	fullPath, _ := m.storage.ResolveFullPath(relPath)
	md, err := m.extractor.Extract(ctx, fullPath)
	if err != nil {
		fmt.Printf("[SoundCloud] Warning: metadata extraction failed for %s: %v\n", filename, err)
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

	if entry.Title != "" {
		track.Title = &entry.Title
	} else if md.Title != nil {
		track.Title = md.Title
	}

	if entry.Artist != "" {
		track.Artist = &entry.Artist
	} else if md.Artist != nil {
		track.Artist = md.Artist
	}

	if md.Album != nil {
		track.Album = md.Album
	}

	if entry.Duration > 0 {
		d := float64(entry.Duration)
		track.DurationSeconds = &d
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

func (m *Manager) TestDownload(ctx context.Context, url string) (string, error) {
	cfg, err := m.repo.GetSyncConfig(ctx)
	if err != nil || cfg == nil || cfg.OAuthToken == nil || *cfg.OAuthToken == "" {
		return "", fmt.Errorf("oauth token not configured")
	}

	tmpDir := filepath.Join(m.dataDir, "tmp", fmt.Sprintf("sc-test-%d", time.Now().UnixNano()))
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	cmd := exec.CommandContext(ctx, "yt-dlp",
		"--cookies-from-browser", "chrome",
		"-x", "--audio-format", "mp3",
		"-o", filepath.Join(tmpDir, "%(title)s.%(ext)s"),
		url,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("download failed: %v, output: %s", err, string(output))
	}

	files, _ := os.ReadDir(tmpDir)
	if len(files) == 0 {
		return "", fmt.Errorf("no files downloaded")
	}

	downloadedFile := filepath.Join(tmpDir, files[0].Name())
	destDir := filepath.Join(m.dataDir, "downloads")
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create downloads dir: %w", err)
	}
	destFile := filepath.Join(destDir, files[0].Name())

	src, err := os.Open(downloadedFile)
	if err != nil {
		return "", fmt.Errorf("failed to open downloaded file: %w", err)
	}
	defer src.Close()

	dst, err := os.Create(destFile)
	if err != nil {
		return "", fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return "", fmt.Errorf("failed to copy file: %w", err)
	}

	return destFile, nil
}
