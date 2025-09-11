package tracks

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"os"

	"github.com/faraz525/home-music-server/backend/internal/media/metadata"
	imodels "github.com/faraz525/home-music-server/backend/internal/models"
	"github.com/faraz525/home-music-server/backend/internal/storage"
	"github.com/faraz525/home-music-server/backend/utils"
)

// Manager handles track business logic and API management
type Manager struct {
	repo      *Repository
	storage   storage.Storage
	extractor metadata.Extractor
}

// NewManager creates a new tracks manager
func NewManager(repo *Repository, storage storage.Storage, extractor metadata.Extractor) *Manager {
	return &Manager{repo: repo, storage: storage, extractor: extractor}
}

// UploadTrack handles track upload with file processing
func (m *Manager) UploadTrack(ctx context.Context, userID string, fileHeader *multipart.FileHeader, req *imodels.UploadTrackRequest) (*imodels.Track, error) {
	// Open uploaded file
	file, err := fileHeader.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open uploaded file: %w", err)
	}
	defer file.Close()

	// Validate file type
	contentType := fileHeader.Header.Get("Content-Type")
	if !utils.IsValidAudioType(contentType) {
		return nil, fmt.Errorf("invalid file type. Only WAV, AIFF, FLAC, MP3 supported")
	}

	// Check file size (2GB limit)
	if fileHeader.Size > 2*1024*1024*1024 {
		return nil, fmt.Errorf("file too large. Maximum size is 2GB")
	}

	// Generate track ID and save via storage
	trackID := utils.GenerateTrackID()
	filePath, size, storedContentType, err := m.storage.Save(ctx, userID, trackID, fileHeader.Filename, file.(io.Reader))
	if err != nil {
		return nil, fmt.Errorf("failed to store file: %w", err)
	}
	if storedContentType != "" {
		contentType = storedContentType
	}

	// Resolve full path for metadata extraction
	fullPath, _ := m.storage.ResolveFullPath(filePath)

	// Extract metadata from the audio file
	fmt.Printf("[CrateDrop] Starting metadata extraction for file: %s\n", fullPath)
	md, err := m.extractor.Extract(ctx, fullPath)
	if err != nil {
		fmt.Printf("[CrateDrop] Warning: metadata extraction failed: %v\n", err)
		md = &metadata.AudioMetadata{}
	}

	// Create track record from metadata and request data
	track := &imodels.Track{
		OwnerUserID:      userID,
		OriginalFilename: fileHeader.Filename,
		ContentType:      contentType,
		SizeBytes:        size,
		FilePath:         filePath,
		CreatedAt:        utils.Now(),
		UpdatedAt:        utils.Now(),
	}
	if md.DurationSeconds > 0 {
		d := md.DurationSeconds
		track.DurationSeconds = &d
	}
	track.Title = utils.StringToPtr(firstNonEmpty(req.Title, derefString(md.Title)))
	track.Artist = utils.StringToPtr(firstNonEmpty(req.Artist, derefString(md.Artist)))
	track.Album = utils.StringToPtr(firstNonEmpty(req.Album, derefString(md.Album)))
	if v := firstNonZero(req.Year, derefInt(md.Year)); v > 0 {
		track.Year = &v
	}
	if v := firstNonZero(req.SampleRate, derefInt(md.SampleRate)); v > 0 {
		track.SampleRate = &v
	}
	if v := firstNonZero(req.Bitrate, derefInt(md.Bitrate)); v > 0 {
		track.Bitrate = &v
	}

	// Persist
	fmt.Printf("[CrateDrop] Inserting track into database...\n")
	track, err = m.repo.CreateTrack(ctx, track)
	if err != nil {
		// Attempt cleanup
		_ = m.storage.Delete(ctx, filePath)
		fmt.Printf("[CrateDrop] Database insert failed: %v\n", err)
		return nil, fmt.Errorf("failed to save track metadata: %w", err)
	}
	fmt.Printf("[CrateDrop] Track successfully saved with ID: %s\n", track.ID)

	return track, nil
}

// OpenFile exposes storage Open for streaming
func (m *Manager) OpenFile(ctx context.Context, relativePath string) (storage.ReadSeekCloser, storage.FileInfo, error) {
	return m.storage.Open(ctx, relativePath)
}

// GetTracks retrieves tracks for a user with pagination
func (m *Manager) GetTracks(ctx context.Context, userID string, limit, offset int) (*imodels.TrackList, error) {
	tracks, err := m.repo.GetTracks(ctx, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch tracks: %w", err)
	}

	total, err := m.repo.GetTracksCount(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to count tracks: %w", err)
	}

	return utils.NewTrackList(tracks, total, limit, offset), nil
}

// GetAllTracks retrieves all tracks with search (admin only)
func (m *Manager) GetAllTracks(ctx context.Context, limit, offset int, searchQuery string) (*imodels.TrackList, error) {
	tracks, err := m.repo.GetAllTracks(ctx, limit, offset, searchQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch tracks: %w", err)
	}

	total, err := m.repo.GetAllTracksCount(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count tracks: %w", err)
	}

	return utils.NewTrackList(tracks, total, limit, offset), nil
}

// GetTrack retrieves a single track
func (m *Manager) GetTrack(ctx context.Context, trackID string) (*imodels.Track, error) {
	track, err := m.repo.GetTrackByID(ctx, trackID)
	if err != nil {
		return nil, fmt.Errorf("track not found: %w", err)
	}
	return track, nil
}

// DeleteTrack deletes a track and its file
func (m *Manager) DeleteTrack(ctx context.Context, trackID string) error {
	// Get track info first
	track, err := m.repo.GetTrackByID(ctx, trackID)
	if err != nil {
		return fmt.Errorf("track not found: %w", err)
	}

	// Delete file via storage
	if err := m.storage.Delete(ctx, track.FilePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	// Delete from database
	if err := m.repo.DeleteTrack(ctx, trackID); err != nil {
		return fmt.Errorf("failed to delete track record: %w", err)
	}

	return nil
}

// SearchTracks searches tracks for a user
func (m *Manager) SearchTracks(ctx context.Context, query, userID string, limit, offset int) (*imodels.TrackList, error) {
	tracks, err := m.repo.SearchTracks(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to search tracks: %w", err)
	}

	// For search results, we don't provide total count for simplicity
	// In production, you might want to implement this
	total := len(tracks) // This is not accurate for pagination

	return utils.NewTrackList(tracks, total, limit, offset), nil
}

// GetStreamInfo returns information needed for streaming
func (m *Manager) GetStreamInfo(ctx context.Context, trackID string) (*imodels.Track, error) {
	return m.repo.GetTrackByID(ctx, trackID)
}

// GetAvailableAPIs returns the list of available track APIs
func (m *Manager) GetAvailableAPIs() []string {
	return []string{
		"POST /api/tracks - Upload track (multipart)",
		"GET /api/tracks - List tracks (with search/pagination)",
		"GET /api/tracks/:id - Get track metadata",
		"GET /api/tracks/:id/stream - Stream audio (with Range support)",
		"DELETE /api/tracks/:id - Delete track",
	}
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func firstNonZero(a, b int) int {
	if a != 0 {
		return a
	}
	return b
}

func derefString(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func derefInt(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}
