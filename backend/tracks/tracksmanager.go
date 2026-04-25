package tracks

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

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

	// Pull embedded album art out into a sidecar BEFORE the sanitizer strips it
	// from the audio file. This preserves the art for display while keeping the
	// audio file small for fast streaming start-up.
	coverTmp, coverErr := extractEmbeddedCover(ctx, fullPath)
	if coverErr != nil {
		fmt.Printf("[CrateDrop] No embedded cover for %s (or extract failed): %v\n", trackID, coverErr)
	}
	if coverTmp != "" {
		defer os.Remove(coverTmp)
	}

	// Sanitize over-sized metadata blobs (e.g. embedded album art) that delay playback
	if updatedSize, err := m.sanitizeIfNeeded(ctx, contentType, fullPath); err != nil {
		fmt.Printf("[CrateDrop] Warning: failed to sanitize track %s: %v\n", trackID, err)
	} else if updatedSize > 0 {
		fmt.Printf("[CrateDrop] Sanitized track %s metadata. Original size: %d bytes, new size: %d bytes\n", trackID, size, updatedSize)
		size = updatedSize
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

	// Attach the cover we extracted before sanitize, if any.
	if coverTmp != "" {
		if err := m.attachExtractedCover(ctx, track, coverTmp); err != nil {
			fmt.Printf("[CrateDrop] Warning: failed to attach extracted cover for %s: %v\n", track.ID, err)
		}
	}

	return track, nil
}

// attachExtractedCover moves a tmp cover file into the track's storage directory
// and updates cover_path. Best-effort.
func (m *Manager) attachExtractedCover(ctx context.Context, track *imodels.Track, tmpPath string) error {
	src, err := os.Open(tmpPath)
	if err != nil {
		return fmt.Errorf("open tmp cover: %w", err)
	}
	defer src.Close()
	coverRel, err := SaveCoverSidecar(ctx, m.storage, track.FilePath, "image/jpeg", tmpPath, src)
	if err != nil {
		return err
	}
	if err := m.repo.UpdateCoverPath(ctx, track.ID, coverRel); err != nil {
		return fmt.Errorf("update cover path: %w", err)
	}
	return nil
}

// extractEmbeddedCover runs ffmpeg to pull the attached picture out of an audio file
// into a JPEG tmp file. Returns the tmp path on success, or an error when no picture
// is embedded / ffmpeg fails. The caller owns the returned tmp file.
func extractEmbeddedCover(ctx context.Context, audioPath string) (string, error) {
	tmpFile, err := os.CreateTemp("", "cratedrop-cover-*.jpg")
	if err != nil {
		return "", fmt.Errorf("create cover tmp: %w", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	// ffmpeg won't overwrite a file unless we pass -y; we just truncated it above
	// so it's empty. -an drops audio, default mapping selects the embedded picture.
	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-y",
		"-i", audioPath,
		"-an",
		"-frames:v", "1",
		tmpPath,
	)
	output, runErr := cmd.CombinedOutput()
	info, statErr := os.Stat(tmpPath)
	if runErr != nil || statErr != nil || info == nil || info.Size() == 0 {
		os.Remove(tmpPath)
		if runErr != nil {
			return "", fmt.Errorf("ffmpeg cover extract: %w (output: %s)", runErr, strings.TrimSpace(string(output)))
		}
		return "", fmt.Errorf("ffmpeg produced no cover")
	}
	return tmpPath, nil
}

// sanitizeIfNeeded strips excessive metadata (like multi-megabyte album art) from MP3s.
// Returns the new file size when sanitization occurs.
func (m *Manager) sanitizeIfNeeded(ctx context.Context, contentType, fullPath string) (int64, error) {
	// Check for various MP3 mime types
	if contentType != "audio/mpeg" && contentType != "audio/mp3" && contentType != "audio/x-mp3" {
		fmt.Printf("[CrateDrop] Skipping sanitization for non-MP3 content type: %s\n", contentType)
		return 0, nil
	}

	file, err := os.Open(fullPath)
	if err != nil {
		return 0, fmt.Errorf("failed to open file for sanitization: %w", err)
	}
	defer file.Close()

	header := make([]byte, 10)
	if _, err := io.ReadFull(file, header); err != nil {
		return 0, fmt.Errorf("failed to read ID3 header: %w", err)
	}
	if string(header[:3]) != "ID3" {
		fmt.Printf("[CrateDrop] Skipping sanitization: No ID3 header found\n")
		return 0, nil
	}

	tagSize := parseID3Size(header[6:10])
	const tagThreshold = 10 * 1024 // 10 KB (lowered from 512KB to ensure cover art is stripped)
	
	if tagSize <= tagThreshold {
		fmt.Printf("[CrateDrop] Skipping sanitization: Metadata size (%d bytes) is below threshold (%d bytes)\n", tagSize, tagThreshold)
		return 0, nil
	}

	tmpPath := fullPath + ".tmp.mp3"
	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-y",
		"-i", fullPath,
		"-map_metadata", "-1",
		"-vn",
		"-c:a", "copy",
		tmpPath,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return 0, fmt.Errorf("ffmpeg sanitize failed: %w (output: %s)", err, string(output))
	}

	backupPath := fullPath + ".original"
	if err := os.Rename(fullPath, backupPath); err != nil {
		os.Remove(tmpPath)
		return 0, fmt.Errorf("failed to backup original file: %w", err)
	}

	if err := os.Rename(tmpPath, fullPath); err != nil {
		_ = os.Rename(backupPath, fullPath)
		return 0, fmt.Errorf("failed to replace sanitized file: %w", err)
	}
	_ = os.Remove(backupPath)

	info, err := os.Stat(fullPath)
	if err != nil {
		return 0, fmt.Errorf("failed to stat sanitized file: %w", err)
	}

	return info.Size(), nil
}

// SanitizeExistingTracks iterates over all tracks and sanitizes them if needed.
// This is a maintenance task to fix existing tracks with large metadata.
func (m *Manager) SanitizeExistingTracks(ctx context.Context) error {
	fmt.Printf("[CrateDrop] Starting retroactive sanitization of all tracks...\n")
	
	// Get all tracks (pagination loop)
	limit := 100
	offset := 0
	totalSanitized := 0
	totalSkipped := 0
	totalErrors := 0

	for {
		tracks, err := m.repo.GetAllTracks(ctx, limit, offset, "")
		if err != nil {
			return fmt.Errorf("failed to fetch tracks for sanitization: %w", err)
		}
		if len(tracks) == 0 {
			break
		}

		for _, track := range tracks {
			fullPath, ok := m.storage.ResolveFullPath(track.FilePath)
			if !ok {
				fmt.Printf("[CrateDrop] Error resolving path for track %s\n", track.ID)
				totalErrors++
				continue
			}

			// Check if .original file already exists (means already sanitized)
			if _, statErr := os.Stat(fullPath + ".original"); statErr == nil {
				// fmt.Printf("[CrateDrop] Track %s already sanitized (backup exists)\n", track.ID)
				totalSkipped++
				continue
			}

			// Attempt sanitization
			// We pass the content type from DB.
			newSize, sanitizeErr := m.sanitizeIfNeeded(ctx, track.ContentType, fullPath)
			if sanitizeErr != nil {
				fmt.Printf("[CrateDrop] Error sanitizing track %s: %v\n", track.ID, sanitizeErr)
				totalErrors++
				continue
			}

			if newSize > 0 {
				// Update DB with new size
				// Note: We are not updating the DB record here to keep it simple, 
				// as the size in DB is mostly for display. But ideally we should.
				// For now, the file system size is what matters for streaming.
				totalSanitized++
			} else {
				totalSkipped++
			}
		}

		offset += limit
	}

	fmt.Printf("[CrateDrop] Sanitization complete. Sanitized: %d, Skipped: %d, Errors: %d\n", totalSanitized, totalSkipped, totalErrors)
	return nil
}

func parseID3Size(b []byte) int {
	size := 0
	for _, v := range b {
		size = (size << 7) | int(v&0x7F)
	}
	return size
}

// SaveCoverSidecar writes an image as a sibling `cover.<ext>` file next to the track
// and returns the relative cover path (relative to dataDir, mirroring track.FilePath).
// Use sourcePath to indicate the source filename or url so a sensible extension is chosen
// when contentType is empty or generic.
func SaveCoverSidecar(ctx context.Context, store storage.Storage, trackFilePath, contentType, sourcePath string, r io.Reader) (string, error) {
	ext := coverExt(contentType, sourcePath)
	parentRel := filepath.Dir(trackFilePath)
	coverRel := filepath.Join(parentRel, "cover"+ext)

	fullPath, ok := store.ResolveFullPath(coverRel)
	if !ok {
		return "", fmt.Errorf("failed to resolve cover path")
	}
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return "", fmt.Errorf("failed to create cover dir: %w", err)
	}

	tmp := fullPath + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return "", fmt.Errorf("failed to create cover tmp: %w", err)
	}
	if _, err := io.Copy(f, r); err != nil {
		f.Close()
		os.Remove(tmp)
		return "", fmt.Errorf("failed to write cover: %w", err)
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return "", fmt.Errorf("failed to close cover: %w", err)
	}
	if err := os.Rename(tmp, fullPath); err != nil {
		os.Remove(tmp)
		return "", fmt.Errorf("failed to rename cover: %w", err)
	}
	return coverRel, nil
}

// coverExt picks a file extension for the cover sidecar based on the image content
// type or, as a fallback, the source filename/URL extension.
func coverExt(contentType, sourcePath string) string {
	switch strings.ToLower(strings.TrimSpace(contentType)) {
	case "image/jpeg", "image/jpg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	}
	if sourcePath != "" {
		ext := strings.ToLower(filepath.Ext(sourcePath))
		switch ext {
		case ".jpg", ".jpeg":
			return ".jpg"
		case ".png":
			return ".png"
		case ".webp":
			return ".webp"
		}
	}
	return ".jpg"
}

// OpenFile exposes storage Open for streaming
func (m *Manager) OpenFile(ctx context.Context, relativePath string) (storage.ReadSeekCloser, storage.FileInfo, error) {
	return m.storage.Open(ctx, relativePath)
}

// ResolveFullPath exposes storage path resolution for download with metadata
func (m *Manager) ResolveFullPath(relativePath string) (string, bool) {
	return m.storage.ResolveFullPath(relativePath)
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

	// Delete cover sidecar if present (best-effort)
	if track.CoverPath != nil && *track.CoverPath != "" {
		if err := m.storage.Delete(ctx, *track.CoverPath); err != nil && !os.IsNotExist(err) {
			fmt.Printf("[CrateDrop] Warning: failed to delete cover for track %s: %v\n", trackID, err)
		}
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

// UpdateAnalysisOverride applies a user override to BPM and/or musical key.
// Caller has already validated ranges and authorized the request.
func (m *Manager) UpdateAnalysisOverride(ctx context.Context, trackID string, bpm *float64, musicalKey *string) error {
	return m.repo.UpdateAnalysisOverride(ctx, trackID, bpm, musicalKey)
}
