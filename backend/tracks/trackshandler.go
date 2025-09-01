package tracks

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/faraz525/home-music-server/backend/models"
)

type UploadRequest struct {
	Title  string `form:"title"`
	Artist string `form:"artist"`
	Album  string `form:"album"`
}

func UploadHandler(manager *Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user ID from context
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "unauthorized", "message": "User not authenticated"}})
			return
		}

		// Get file from form
		file, header, err := c.Request.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "no_file", "message": "No file provided"}})
			return
		}
		defer file.Close()

		// Parse form data
		var req models.UploadTrackRequest
		if err := c.ShouldBind(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "invalid_form", "message": err.Error()}})
			return
		}

		track, err := manager.UploadTrack(userID.(string), header, &req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "upload_failed", "message": err.Error()}})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"track": track})
	}
}

func ListHandler(manager *Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		userRole, _ := c.Get("user_role")

		// Parse query parameters
		q := c.Query("q")
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

		var trackList *models.TrackList

		if userRole == "admin" && q != "" {
			// Admin can search all tracks
			trackList, err = manager.GetAllTracks(limit, offset, q)
		} else if userRole == "admin" {
			// Admin can see all tracks
			trackList, err = manager.GetAllTracks(limit, offset, "")
		} else if q != "" {
			// Regular users can search their tracks
			trackList, err = manager.SearchTracks(q, userID.(string), limit, offset)
		} else {
			// Regular users see only their tracks
			trackList, err = manager.GetTracks(userID.(string), limit, offset)
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

		track, err := manager.GetTrack(trackID)
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
		trackID := c.Param("id")
		userID, _ := c.Get("user_id")
		userRole, _ := c.Get("user_role")

		track, err := manager.GetStreamInfo(trackID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "track_not_found", "message": "Track not found"}})
			return
		}

		// Check ownership
		if userRole != "admin" && track.OwnerUserID != userID.(string) {
			c.JSON(http.StatusForbidden, gin.H{"error": gin.H{"code": "access_denied", "message": "Access denied"}})
			return
		}

		// Open file
		filePath := filepath.Join(os.Getenv("DATA_DIR"), track.FilePath)
		file, err := os.Open(filePath)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "server_error", "message": "Failed to open file"}})
			return
		}
		defer file.Close()

		// Get file info
		stat, err := file.Stat()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "server_error", "message": "Failed to get file info"}})
			return
		}

		// Handle range requests for seeking
		rangeHeader := c.GetHeader("Range")
		if rangeHeader != "" {
			handleRangeRequest(c, file, stat.Size(), track.ContentType, rangeHeader)
			return
		}

		// Full file response
		c.Header("Content-Type", track.ContentType)
		c.Header("Content-Length", strconv.FormatInt(stat.Size(), 10))
		c.Header("Accept-Ranges", "bytes")
		c.File(filePath)
	}
}

func DeleteHandler(manager *Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		trackID := c.Param("id")
		userID, _ := c.Get("user_id")
		userRole, _ := c.Get("user_role")

		track, err := manager.GetTrack(trackID)
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
		if err := manager.DeleteTrack(trackID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "server_error", "message": err.Error()}})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Track deleted successfully"})
	}
}

// handleRangeRequest handles HTTP range requests for audio streaming
func handleRangeRequest(c *gin.Context, file *os.File, fileSize int64, contentType, rangeHeader string) {
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
