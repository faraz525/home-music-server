package tracks

import (
	"context"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/faraz525/home-music-server/backend/playlists"
	"github.com/gin-gonic/gin"

	imodels "github.com/faraz525/home-music-server/backend/internal/models"
)

// ... (existing code)

// SanitizeAllHandler triggers retroactive sanitization for all tracks
func SanitizeAllHandler(manager *Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Run in background as it might take time
		go func() {
			if err := manager.SanitizeExistingTracks(context.Background()); err != nil {
				fmt.Printf("[CrateDrop] Background sanitization failed: %v\n", err)
			}
		}()

		c.JSON(http.StatusOK, gin.H{
			"message": "Sanitization started in background. Check logs for progress.",
		})
	}
}

type UploadRequest struct {
	Title  string `form:"title"`
	Artist string `form:"artist"`
	Album  string `form:"album"`
}

func UploadHandler(manager *Manager, playlistsManager *playlists.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user ID from context
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "unauthorized", "message": "User not authenticated"}})
			return
		}

		fmt.Printf("[CrateDrop] Upload request received for user: %v\n", userID)

		// Get file from form
		file, header, err := c.Request.FormFile("file")
		if err != nil {
			fmt.Printf("[CrateDrop] No file provided: %v\n", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "no_file", "message": "No file provided"}})
			return
		}
		defer file.Close()

		fmt.Printf("[CrateDrop] File received: %s (%d bytes)\n", header.Filename, header.Size)

		// Get playlist_id from form (optional)
		playlistID := c.PostForm("playlist_id")
		if playlistID != "" {
			fmt.Printf("[CrateDrop] Playlist ID specified: %s\n", playlistID)
		}

		// Parse form data
		var req imodels.UploadTrackRequest
		if err := c.ShouldBind(&req); err != nil {
			fmt.Printf("[CrateDrop] Form binding failed: %v\n", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "invalid_form", "message": err.Error()}})
			return
		}

		fmt.Printf("[CrateDrop] Form data: Title=%s, Artist=%s, Album=%s\n", req.Title, req.Artist, req.Album)

		track, err := manager.UploadTrack(c.Request.Context(), userID.(string), header, &req)
		if err != nil {
			fmt.Printf("[CrateDrop] Upload failed: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "upload_failed", "message": err.Error()}})
			return
		}

		fmt.Printf("[CrateDrop] Upload successful, track ID: %s\n", track.ID)

		// Add track to playlist if specified
		if playlistID != "" && playlistID != "unsorted" {
			fmt.Printf("[CrateDrop] Adding track to playlist: %s\n", playlistID)
			addReq := &imodels.AddTracksToPlaylistRequest{
				TrackIDs: []string{track.ID},
			}
			if err := playlistsManager.AddTracksToPlaylist(playlistID, userID.(string), addReq); err != nil {
				fmt.Printf("[CrateDrop] Warning: failed to add track to playlist: %v\n", err)
				// Don't fail the upload, just log the warning
			} else {
				fmt.Printf("[CrateDrop] Track successfully added to playlist\n")
			}
		}

		c.JSON(http.StatusCreated, gin.H{"track": track})
	}
}

func ListHandler(manager *Manager, playlistsManager *playlists.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")

		// Parse query parameters
		q := c.Query("q")
		playlistID := c.Query("playlist_id")
		limitStr := c.DefaultQuery("limit", "20")
		offsetStr := c.DefaultQuery("offset", "0")

		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit < 1 || limit > 100 {
			limit = 20
		}

		offset, err := strconv.Atoi(offsetStr)
		if err != nil || offset < 0 {
			offset = 0
		}

		var trackList *imodels.TrackList

		// Handle search queries (prioritize search if present)
		if q != "" {
			// If searching with a specific playlist/crate, filter results
			if playlistID != "" {
				if playlistID == "unsorted" {
					// Search within unsorted tracks only
					trackList, err = playlistsManager.SearchUnsortedTracks(userID.(string), q, limit, offset)
				} else {
					// Search within specific playlist
					trackList, err = playlistsManager.SearchPlaylistTracks(playlistID, userID.(string), q, limit, offset)
				}
			} else {
				// Search all user's tracks
				trackList, err = manager.SearchTracks(c.Request.Context(), q, userID.(string), limit, offset)
			}
		} else if playlistID != "" {
			// No search, just list tracks from playlist/crate
			if playlistID == "unsorted" {
				// Get tracks not in any playlist
				trackList, err = playlistsManager.GetUnsortedTracks(userID.(string), limit, offset)
			} else {
				// Get tracks from specific playlist
				playlistWithTracks, playlistErr := playlistsManager.GetPlaylistTracks(playlistID, userID.(string), limit, offset)
				if playlistErr != nil {
					err = playlistErr
				} else {
					// Convert to TrackList format
					trackList = &imodels.TrackList{
						Tracks:  playlistWithTracks.Tracks,
						Total:   playlistWithTracks.Total,
						Limit:   playlistWithTracks.Limit,
						Offset:  playlistWithTracks.Offset,
						HasNext: playlistWithTracks.HasNext,
					}
				}
			}
		} else {
			// No search, no playlist - show all user's tracks
			trackList, err = manager.GetTracks(c.Request.Context(), userID.(string), limit, offset)
		}

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "server_error", "message": "Failed to fetch tracks"}})
			return
		}

		c.JSON(http.StatusOK, trackList)
	}
}

func GetHandler(manager *Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		trackID := c.Param("id")
		userID, _ := c.Get("user_id")
		userRole, _ := c.Get("user_role")

		track, err := manager.GetTrack(c.Request.Context(), trackID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "track_not_found", "message": "Track not found"}})
			return
		}

		// Check ownership
		if userRole != "admin" && track.OwnerUserID != userID.(string) {
			c.JSON(http.StatusForbidden, gin.H{"error": gin.H{"code": "access_denied", "message": "Access denied"}})
			return
		}

		c.JSON(http.StatusOK, gin.H{"track": track})
	}
}

func StreamHandler(manager *Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()
		fmt.Printf("[StreamProfiler] Request started at %v\n", startTime)

		trackID := c.Param("id")
		userID, _ := c.Get("user_id")
		userRole, _ := c.Get("user_role")

		trackStart := time.Now()
		track, err := manager.GetStreamInfo(c.Request.Context(), trackID)
		fmt.Printf("[StreamProfiler] GetStreamInfo took: %v\n", time.Since(trackStart))
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "track_not_found", "message": "Track not found"}})
			return
		}

		// Check ownership
		if userRole != "admin" && track.OwnerUserID != userID.(string) {
			c.JSON(http.StatusForbidden, gin.H{"error": gin.H{"code": "access_denied", "message": "Access denied"}})
			return
		}

		// Open file via manager/storage
		openStart := time.Now()
		file, info, err := manager.OpenFile(c.Request.Context(), track.FilePath)
		fmt.Printf("[StreamProfiler] OpenFile took: %v\n", time.Since(openStart))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "server_error", "message": "Failed to open file"}})
			return
		}
		defer file.Close()

		// Handle range requests for seeking
		rangeHeader := c.GetHeader("Range")
		if rangeHeader != "" {
			handleRangeRequest(c, file, info.Size, track.ContentType, rangeHeader)
			fmt.Printf("[StreamProfiler] Total request time: %v\n", time.Since(startTime))
			return
		}

		// Default to sending first chunk for faster initial playback
		// This allows the audio element to start playing quickly without downloading the entire file
		defaultChunkSize := int64(512 * 1024) // 512 KB initial chunk to minimize startup latency
		chunkSize := defaultChunkSize
		if info.Size < defaultChunkSize {
			chunkSize = info.Size
		}

		c.Header("Content-Type", track.ContentType)
		c.Header("Content-Length", strconv.FormatInt(chunkSize, 10))
		c.Header("Content-Range", fmt.Sprintf("bytes 0-%d/%d", chunkSize-1, info.Size))
		c.Header("Accept-Ranges", "bytes")
		// Cache control for better performance - allow caching but revalidate
		c.Header("Cache-Control", "public, max-age=3600, must-revalidate")
		c.Status(http.StatusPartialContent)

		io.CopyN(c.Writer, file, chunkSize)
		fmt.Printf("[StreamProfiler] Total request time: %v\n", time.Since(startTime))
	}
}

// CoverHandler serves the cover-art sidecar image for a track.
// Mirrors the auth pattern in StreamHandler. Returns 404 when the track has no cover.
func CoverHandler(manager *Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		trackID := c.Param("id")
		userID, _ := c.Get("user_id")
		userRole, _ := c.Get("user_role")

		track, err := manager.GetStreamInfo(c.Request.Context(), trackID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "track_not_found", "message": "Track not found"}})
			return
		}

		if userRole != "admin" && track.OwnerUserID != userID.(string) {
			c.JSON(http.StatusForbidden, gin.H{"error": gin.H{"code": "access_denied", "message": "Access denied"}})
			return
		}

		if track.CoverPath == nil || *track.CoverPath == "" {
			c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "cover_not_found", "message": "No cover art for this track"}})
			return
		}

		file, info, err := manager.OpenFile(c.Request.Context(), *track.CoverPath)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "cover_not_found", "message": "Cover file missing"}})
			return
		}
		defer file.Close()

		ctype := mime.TypeByExtension(filepath.Ext(*track.CoverPath))
		if ctype == "" {
			ctype = "image/jpeg"
		}
		c.Header("Content-Type", ctype)
		c.Header("Content-Length", strconv.FormatInt(info.Size, 10))
		c.Header("Cache-Control", "private, max-age=86400, immutable")
		c.Status(http.StatusOK)
		io.Copy(c.Writer, file)
	}
}

// DownloadHandler handles file downloads with metadata embedded
func DownloadHandler(manager *Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		trackID := c.Param("id")
		userID, _ := c.Get("user_id")
		userRole, _ := c.Get("user_role")

		track, err := manager.GetTrack(c.Request.Context(), trackID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "track_not_found", "message": "Track not found"}})
			return
		}

		// Check ownership
		if userRole != "admin" && track.OwnerUserID != userID.(string) {
			c.JSON(http.StatusForbidden, gin.H{"error": gin.H{"code": "access_denied", "message": "Access denied"}})
			return
		}

		// Generate download filename from metadata
		downloadFilename := generateDownloadFilename(track)

		// For MP3s, re-inject metadata that was stripped during sanitization
		if isMP3ContentType(track.ContentType) && hasMetadata(track) {
			if err := streamWithMetadata(c, manager, track, downloadFilename); err != nil {
				fmt.Printf("[CrateDrop] Failed to stream with metadata: %v, falling back to direct download\n", err)
				streamDirectDownload(c, manager, track, downloadFilename)
			}
			return
		}

		// Non-MP3 or no metadata: serve directly
		streamDirectDownload(c, manager, track, downloadFilename)
	}
}

func DeleteHandler(manager *Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		trackID := c.Param("id")
		userID, _ := c.Get("user_id")
		userRole, _ := c.Get("user_role")

		track, err := manager.GetTrack(c.Request.Context(), trackID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "track_not_found", "message": "Track not found"}})
			return
		}

		// Check ownership
		if userRole != "admin" && track.OwnerUserID != userID.(string) {
			c.JSON(http.StatusForbidden, gin.H{"error": gin.H{"code": "access_denied", "message": "Access denied"}})
			return
		}

		// Delete track (manager handles both file and database deletion)
		if err := manager.DeleteTrack(c.Request.Context(), trackID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "server_error", "message": err.Error()}})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Track deleted successfully"})
	}
}


// isMP3ContentType checks if the content type is MP3
func isMP3ContentType(contentType string) bool {
	return contentType == "audio/mpeg" || contentType == "audio/mp3" || contentType == "audio/x-mp3"
}

// hasMetadata checks if the track has any metadata worth embedding
func hasMetadata(track *imodels.Track) bool {
	return (track.Title != nil && *track.Title != "") ||
		(track.Artist != nil && *track.Artist != "") ||
		(track.Album != nil && *track.Album != "")
}

// generateDownloadFilename creates a filename from track metadata
func generateDownloadFilename(track *imodels.Track) string {
	ext := getFileExtension(track.OriginalFilename, track.ContentType)

	// Build filename from metadata
	var parts []string

	if track.Artist != nil && *track.Artist != "" {
		parts = append(parts, sanitizeFilename(*track.Artist))
	}
	if track.Title != nil && *track.Title != "" {
		parts = append(parts, sanitizeFilename(*track.Title))
	}

	if len(parts) == 0 {
		// Fall back to original filename if no metadata
		return track.OriginalFilename
	}

	return strings.Join(parts, " - ") + ext
}

// getFileExtension extracts file extension from filename or content type
func getFileExtension(filename, contentType string) string {
	if idx := strings.LastIndex(filename, "."); idx != -1 {
		return filename[idx:]
	}
	// Fallback based on content type
	switch contentType {
	case "audio/mpeg", "audio/mp3", "audio/x-mp3":
		return ".mp3"
	case "audio/wav", "audio/x-wav":
		return ".wav"
	case "audio/flac", "audio/x-flac":
		return ".flac"
	case "audio/aiff", "audio/x-aiff":
		return ".aiff"
	default:
		return ""
	}
}

// sanitizeFilename removes characters that are problematic in filenames
func sanitizeFilename(s string) string {
	// Remove or replace problematic characters
	replacer := strings.NewReplacer(
		"/", "-",
		"\\", "-",
		":", "-",
		"*", "",
		"?", "",
		"\"", "'",
		"<", "",
		">", "",
		"|", "-",
	)
	result := replacer.Replace(s)
	// Trim spaces and limit length
	result = strings.TrimSpace(result)
	if len(result) > 100 {
		result = result[:100]
	}
	return result
}

// streamDirectDownload serves the file directly without metadata injection
func streamDirectDownload(c *gin.Context, manager *Manager, track *imodels.Track, filename string) {
	file, info, err := manager.OpenFile(c.Request.Context(), track.FilePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "server_error", "message": "Failed to open file"}})
		return
	}
	defer file.Close()

	c.Header("Content-Type", track.ContentType)
	c.Header("Content-Length", strconv.FormatInt(info.Size, 10))
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))

	io.Copy(c.Writer, file)
}

// streamWithMetadata uses ffmpeg to inject ID3 tags into the download stream
func streamWithMetadata(c *gin.Context, manager *Manager, track *imodels.Track, filename string) error {
	fullPath, ok := manager.ResolveFullPath(track.FilePath)
	if !ok {
		return fmt.Errorf("failed to resolve file path")
	}

	// Build ffmpeg command to inject metadata
	args := []string{
		"-i", fullPath,
		"-c:a", "copy", // Copy audio without re-encoding
	}

	// Add metadata tags
	if track.Title != nil && *track.Title != "" {
		args = append(args, "-metadata", fmt.Sprintf("title=%s", *track.Title))
	}
	if track.Artist != nil && *track.Artist != "" {
		args = append(args, "-metadata", fmt.Sprintf("artist=%s", *track.Artist))
	}
	if track.Album != nil && *track.Album != "" {
		args = append(args, "-metadata", fmt.Sprintf("album=%s", *track.Album))
	}
	if track.Genre != nil && *track.Genre != "" {
		args = append(args, "-metadata", fmt.Sprintf("genre=%s", *track.Genre))
	}
	if track.Year != nil && *track.Year > 0 {
		args = append(args, "-metadata", fmt.Sprintf("date=%d", *track.Year))
	}

	// Output to stdout as MP3
	args = append(args, "-f", "mp3", "pipe:1")

	cmd := exec.CommandContext(c.Request.Context(), "ffmpeg", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start ffmpeg: %w", err)
	}

	// Set headers - we can't know exact size with streaming, so omit Content-Length
	c.Header("Content-Type", "audio/mpeg")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	c.Header("Transfer-Encoding", "chunked")

	// Stream ffmpeg output to response
	_, copyErr := io.Copy(c.Writer, stdout)

	// Wait for ffmpeg to finish
	cmdErr := cmd.Wait()

	if copyErr != nil {
		return fmt.Errorf("failed to copy ffmpeg output: %w", copyErr)
	}
	if cmdErr != nil {
		return fmt.Errorf("ffmpeg failed: %w", cmdErr)
	}

	return nil
}

// handleRangeRequest handles HTTP range requests for audio streaming
func handleRangeRequest(c *gin.Context, file io.ReadSeeker, fileSize int64, contentType, rangeHeader string) {
	// Parse range header (e.g., "bytes=0-1023")
	rangeParts := strings.Split(strings.TrimPrefix(rangeHeader, "bytes="), "-")
	if len(rangeParts) != 2 {
		c.AbortWithStatus(http.StatusRequestedRangeNotSatisfiable)
		return
	}

	start, err := strconv.ParseInt(rangeParts[0], 10, 64)
	if err != nil {
		c.AbortWithStatus(http.StatusRequestedRangeNotSatisfiable)
		return
	}

	var end int64
	maxChunkSize := int64(512 * 1024)     // 512 KB max chunk size for subsequent requests
	initialChunkSize := int64(256 * 1024) // 256 KB for initial request (faster playback start)

	if rangeParts[1] == "" {
		// Browser requested open-ended range (bytes=start-), limit to chunk size
		// This prevents downloading entire file before audio starts playing
		requestedEnd := fileSize - 1
		var chunkSize int64
		if start == 0 {
			// Initial request - use smaller chunk for faster start
			chunkSize = initialChunkSize
		} else {
			// Subsequent requests - use larger chunk
			chunkSize = maxChunkSize
		}
		chunkEnd := start + chunkSize - 1
		if chunkEnd < requestedEnd {
			end = chunkEnd
		} else {
			end = requestedEnd
		}
	} else {
		end, err = strconv.ParseInt(rangeParts[1], 10, 64)
		if err != nil {
			c.AbortWithStatus(http.StatusRequestedRangeNotSatisfiable)
			return
		}
		// Limit explicit ranges to max chunk size as well
		requestedSize := end - start + 1
		if requestedSize > maxChunkSize {
			end = start + maxChunkSize - 1
		}
	}

	if start >= fileSize || end >= fileSize || start > end {
		c.AbortWithStatus(http.StatusRequestedRangeNotSatisfiable)
		return
	}

	contentLength := end - start + 1

	// Seek to start position
	if _, err := file.Seek(start, 0); err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	// Set response headers
	c.Header("Content-Type", contentType)
	c.Header("Content-Length", strconv.FormatInt(contentLength, 10))
	c.Header("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, fileSize))
	c.Header("Accept-Ranges", "bytes")
	// Cache control for better performance - allow caching but revalidate
	c.Header("Cache-Control", "public, max-age=3600, must-revalidate")
	c.Status(http.StatusPartialContent)

	// Stream the range with timing
	streamStart := time.Now()
	io.CopyN(c.Writer, file, contentLength)
	fmt.Printf("[StreamProfiler] Range request: bytes %d-%d, io.CopyN took: %v, size: %d bytes\n",
		start, end, time.Since(streamStart), contentLength)
}
