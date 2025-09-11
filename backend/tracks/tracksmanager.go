package tracks

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"

	imodels "github.com/faraz525/home-music-server/backend/internal/models"
	"github.com/faraz525/home-music-server/backend/utils"
)

// Manager handles track business logic and API management
type Manager struct {
	repo *Repository
}

// NewManager creates a new tracks manager
func NewManager(repo *Repository) *Manager {
	return &Manager{repo: repo}
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

	// Generate track ID and file path
	trackID := utils.GenerateTrackID()
	filename := fmt.Sprintf("%s%s", trackID, utils.GetFileExtension(fileHeader.Filename))
	filePath := utils.BuildTrackFilePath(userID, trackID, filename)

	// Ensure directory exists
	fullPath := filepath.Join(os.Getenv("DATA_DIR"), filePath)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create upload directory: %w", err)
	}

	// Save file temporarily
	tempPath := fullPath + ".tmp"
	tempFile, err := os.Create(tempPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tempPath) // Clean up temp file on error

	if _, err := io.Copy(tempFile, file); err != nil {
		tempFile.Close()
		return nil, fmt.Errorf("failed to save file: %w", err)
	}
	tempFile.Close()

	// Move temp file to final location
	if err := os.Rename(tempPath, fullPath); err != nil {
		return nil, fmt.Errorf("failed to finalize upload: %w", err)
	}

	// Extract metadata from the audio file
	fmt.Printf("[CrateDrop] Starting metadata extraction for file: %s\n", fullPath)
	metadata, err := utils.ExtractMetadata(fullPath)
	if err != nil {
		// Log the error but don't fail - continue with basic info
		fmt.Printf("[CrateDrop] Warning: failed to extract metadata: %v\n", err)
		metadata = &utils.AudioMetadata{} // Empty metadata as fallback
	}

	// Create track record from metadata and request data
	track := utils.CreateTrackFromMetadata(metadata, userID, fileHeader.Filename, contentType, filePath, fileHeader.Size, req)
	fmt.Printf("[CrateDrop] Created track record: ID=%s, Title=%v, Artist=%v, Album=%v\n",
		track.ID, track.Title, track.Artist, track.Album)

	fmt.Printf("[CrateDrop] Inserting track into database...\n")
	track, err = m.repo.CreateTrack(ctx, track)
	if err != nil {
		// Clean up file if database insert fails
		os.Remove(fullPath)
		fmt.Printf("[CrateDrop] Database insert failed: %v\n", err)
		return nil, fmt.Errorf("failed to save track metadata: %w", err)
	}
	fmt.Printf("[CrateDrop] Track successfully saved with ID: %s\n", track.ID)

	return track, nil
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

	// Delete file
	filePath := filepath.Join(os.Getenv("DATA_DIR"), track.FilePath)
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
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
