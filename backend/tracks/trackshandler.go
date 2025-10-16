package tracks

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/faraz525/home-music-server/backend/playlists"
	"github.com/gin-gonic/gin"

	imodels "github.com/faraz525/home-music-server/backend/internal/models"
)

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

func StreamHandler(manager *Manager, playlistsManager *playlists.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		trackID := c.Param("id")
		userID, _ := c.Get("user_id")
		userRole, _ := c.Get("user_role")

		track, err := manager.GetStreamInfo(c.Request.Context(), trackID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "track_not_found", "message": "Track not found"}})
			return
		}

		// Check access: admin OR owner OR track is in public playlist
		hasAccess := false
		if userRole == "admin" || track.OwnerUserID == userID.(string) {
			hasAccess = true
		} else {
			// Check if track is in a public playlist
			inPublic, err := playlistsManager.IsTrackInPublicPlaylist(trackID)
			if err == nil && inPublic {
				hasAccess = true
			}
		}

		if !hasAccess {
			c.JSON(http.StatusForbidden, gin.H{"error": gin.H{"code": "access_denied", "message": "Access denied"}})
			return
		}

		// Open file via manager/storage
		file, info, err := manager.OpenFile(c.Request.Context(), track.FilePath)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "server_error", "message": "Failed to open file"}})
			return
		}
		defer file.Close()

		// Handle range requests for seeking
		rangeHeader := c.GetHeader("Range")
		if rangeHeader != "" {
			handleRangeRequest(c, file, info.Size, track.ContentType, rangeHeader)
			return
		}

		// Full file response
		c.Header("Content-Type", track.ContentType)
		c.Header("Content-Length", strconv.FormatInt(info.Size, 10))
		c.Header("Accept-Ranges", "bytes")
		io.Copy(c.Writer, file)
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

// DownloadHandler handles track download requests
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

		// Only allow download if user owns the track (original or has reference)
		// For now, only allow if user is owner or admin
		// TODO: Check track_references table for traded tracks
		if userRole != "admin" && track.OwnerUserID != userID.(string) {
			c.JSON(http.StatusForbidden, gin.H{"error": gin.H{"code": "access_denied", "message": "You can only download tracks you own"}})
			return
		}

		// Open file via manager/storage
		file, info, err := manager.OpenFile(c.Request.Context(), track.FilePath)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "server_error", "message": "Failed to open file"}})
			return
		}
		defer file.Close()

		// Set headers for download
		filename := track.OriginalFilename
		if filename == "" {
			filename = trackID + ".mp3" // fallback
		}

		c.Header("Content-Type", track.ContentType)
		c.Header("Content-Length", strconv.FormatInt(info.Size, 10))
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))

		// Stream the file
		io.Copy(c.Writer, file)
	}
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
	if rangeParts[1] == "" {
		end = fileSize - 1
	} else {
		end, err = strconv.ParseInt(rangeParts[1], 10, 64)
		if err != nil {
			c.AbortWithStatus(http.StatusRequestedRangeNotSatisfiable)
			return
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
	c.Status(http.StatusPartialContent)

	// Stream the range
	io.CopyN(c.Writer, file, contentLength)
}
