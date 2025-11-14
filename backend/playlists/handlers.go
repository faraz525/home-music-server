package playlists

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	imodels "github.com/faraz525/home-music-server/backend/internal/models"
)

// CreatePlaylistHandler creates a new playlist
func CreatePlaylistHandler(manager *Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user ID from context
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, imodels.APIResponse{
				Success: false,
				Error:   &imodels.APIError{Code: "unauthorized", Message: "User not authenticated"},
			})
			return
		}

		var req imodels.CreatePlaylistRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, imodels.APIResponse{
				Success: false,
				Error:   &imodels.APIError{Code: "invalid_request", Message: err.Error()},
			})
			return
		}

		playlist, err := manager.CreatePlaylist(userID.(string), &req)
		if err != nil {
			c.JSON(http.StatusBadRequest, imodels.APIResponse{
				Success: false,
				Error:   &imodels.APIError{Code: "create_failed", Message: err.Error()},
			})
			return
		}

		c.JSON(http.StatusCreated, imodels.APIResponse{
			Success: true,
			Data:    playlist,
		})
	}
}

// GetPlaylistsHandler returns all playlists for the current user
func GetPlaylistsHandler(manager *Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user ID from context
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, imodels.APIResponse{
				Success: false,
				Error:   &imodels.APIError{Code: "unauthorized", Message: "User not authenticated"},
			})
			return
		}

		// Parse pagination parameters
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

		playlists, err := manager.GetUserPlaylists(userID.(string), limit, offset)
		if err != nil {
			c.JSON(http.StatusInternalServerError, imodels.APIResponse{
				Success: false,
				Error:   &imodels.APIError{Code: "server_error", Message: "Failed to fetch playlists"},
			})
			return
		}

		c.JSON(http.StatusOK, imodels.APIResponse{
			Success: true,
			Data:    playlists,
		})
	}
}

// GetPlaylistHandler returns a specific playlist
func GetPlaylistHandler(manager *Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user ID from context
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, imodels.APIResponse{
				Success: false,
				Error:   &imodels.APIError{Code: "unauthorized", Message: "User not authenticated"},
			})
			return
		}

		playlistID := c.Param("id")

		playlist, err := manager.GetPlaylist(playlistID, userID.(string))
		if err != nil {
			statusCode := http.StatusInternalServerError
			if err.Error() == "playlist not found" {
				statusCode = http.StatusNotFound
			} else if err.Error() == "access denied: playlist belongs to another user" {
				statusCode = http.StatusForbidden
			}

			c.JSON(statusCode, imodels.APIResponse{
				Success: false,
				Error:   &imodels.APIError{Code: "access_denied", Message: err.Error()},
			})
			return
		}

		c.JSON(http.StatusOK, imodels.APIResponse{
			Success: true,
			Data:    playlist,
		})
	}
}

// UpdatePlaylistHandler updates a playlist
func UpdatePlaylistHandler(manager *Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user ID from context
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, imodels.APIResponse{
				Success: false,
				Error:   &imodels.APIError{Code: "unauthorized", Message: "User not authenticated"},
			})
			return
		}

		playlistID := c.Param("id")

		var req imodels.UpdatePlaylistRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, imodels.APIResponse{
				Success: false,
				Error:   &imodels.APIError{Code: "invalid_request", Message: err.Error()},
			})
			return
		}

		err := manager.UpdatePlaylist(playlistID, userID.(string), &req)
		if err != nil {
			statusCode := http.StatusInternalServerError
			if err.Error() == "playlist not found" {
				statusCode = http.StatusNotFound
			} else if err.Error() == "access denied: playlist belongs to another user" ||
				err.Error() == "cannot modify the default playlist" {
				statusCode = http.StatusForbidden
			}

			c.JSON(statusCode, imodels.APIResponse{
				Success: false,
				Error:   &imodels.APIError{Code: "update_failed", Message: err.Error()},
			})
			return
		}

		c.JSON(http.StatusOK, imodels.APIResponse{
			Success: true,
			Data:    map[string]string{"message": "Playlist updated successfully"},
		})
	}
}

// DeletePlaylistHandler deletes a playlist
func DeletePlaylistHandler(manager *Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user ID from context
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, imodels.APIResponse{
				Success: false,
				Error:   &imodels.APIError{Code: "unauthorized", Message: "User not authenticated"},
			})
			return
		}

		playlistID := c.Param("id")

		err := manager.DeletePlaylist(playlistID, userID.(string))
		if err != nil {
			statusCode := http.StatusInternalServerError
			if err.Error() == "playlist not found" {
				statusCode = http.StatusNotFound
			} else if err.Error() == "access denied: playlist belongs to another user" ||
				err.Error() == "cannot delete the default playlist" {
				statusCode = http.StatusForbidden
			}

			c.JSON(statusCode, imodels.APIResponse{
				Success: false,
				Error:   &imodels.APIError{Code: "delete_failed", Message: err.Error()},
			})
			return
		}

		c.JSON(http.StatusOK, imodels.APIResponse{
			Success: true,
			Data:    map[string]string{"message": "Playlist deleted successfully"},
		})
	}
}

// AddTracksToPlaylistHandler adds tracks to a playlist
func AddTracksToPlaylistHandler(manager *Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user ID from context
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, imodels.APIResponse{
				Success: false,
				Error:   &imodels.APIError{Code: "unauthorized", Message: "User not authenticated"},
			})
			return
		}

		playlistID := c.Param("id")

		var req imodels.AddTracksToPlaylistRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, imodels.APIResponse{
				Success: false,
				Error:   &imodels.APIError{Code: "invalid_request", Message: err.Error()},
			})
			return
		}

		err := manager.AddTracksToPlaylist(playlistID, userID.(string), &req)
		if err != nil {
			statusCode := http.StatusInternalServerError
			if err.Error() == "playlist not found" {
				statusCode = http.StatusNotFound
			} else if err.Error() == "access denied: playlist belongs to another user" {
				statusCode = http.StatusForbidden
			}

			c.JSON(statusCode, imodels.APIResponse{
				Success: false,
				Error:   &imodels.APIError{Code: "add_tracks_failed", Message: err.Error()},
			})
			return
		}

		c.JSON(http.StatusOK, imodels.APIResponse{
			Success: true,
			Data:    map[string]string{"message": "Tracks added to playlist successfully"},
		})
	}
}

// RemoveTracksFromPlaylistHandler removes tracks from a playlist
func RemoveTracksFromPlaylistHandler(manager *Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user ID from context
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, imodels.APIResponse{
				Success: false,
				Error:   &imodels.APIError{Code: "unauthorized", Message: "User not authenticated"},
			})
			return
		}

		playlistID := c.Param("id")

		var req imodels.RemoveTracksFromPlaylistRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, imodels.APIResponse{
				Success: false,
				Error:   &imodels.APIError{Code: "invalid_request", Message: err.Error()},
			})
			return
		}

		err := manager.RemoveTracksFromPlaylist(playlistID, userID.(string), &req)
		if err != nil {
			statusCode := http.StatusInternalServerError
			if err.Error() == "no tracks found to remove" {
				statusCode = http.StatusNotFound
			} else if err.Error() == "access denied: playlist belongs to another user" {
				statusCode = http.StatusForbidden
			}

			c.JSON(statusCode, imodels.APIResponse{
				Success: false,
				Error:   &imodels.APIError{Code: "remove_tracks_failed", Message: err.Error()},
			})
			return
		}

		c.JSON(http.StatusOK, imodels.APIResponse{
			Success: true,
			Data:    map[string]string{"message": "Tracks removed from playlist successfully"},
		})
	}
}

// GetPlaylistTracksHandler returns tracks for a specific playlist
func GetPlaylistTracksHandler(manager *Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user ID from context
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, imodels.APIResponse{
				Success: false,
				Error:   &imodels.APIError{Code: "unauthorized", Message: "User not authenticated"},
			})
			return
		}

		playlistID := c.Param("id")

		// Parse pagination parameters
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

		playlistTracks, err := manager.GetPlaylistTracks(playlistID, userID.(string), limit, offset)
		if err != nil {
			statusCode := http.StatusInternalServerError
			if err.Error() == "playlist not found" {
				statusCode = http.StatusNotFound
			} else if err.Error() == "access denied: playlist belongs to another user" {
				statusCode = http.StatusForbidden
			}

			c.JSON(statusCode, imodels.APIResponse{
				Success: false,
				Error:   &imodels.APIError{Code: "access_denied", Message: err.Error()},
			})
			return
		}

		c.JSON(http.StatusOK, imodels.APIResponse{
			Success: true,
			Data:    playlistTracks,
		})
	}
}

// GetUnsortedTracksHandler returns tracks not assigned to any playlist
func GetUnsortedTracksHandler(manager *Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user ID from context
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, imodels.APIResponse{
				Success: false,
				Error:   &imodels.APIError{Code: "unauthorized", Message: "User not authenticated"},
			})
			return
		}

		// Parse pagination parameters
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

		tracks, err := manager.GetUnsortedTracks(userID.(string), limit, offset)
		if err != nil {
			c.JSON(http.StatusInternalServerError, imodels.APIResponse{
				Success: false,
				Error:   &imodels.APIError{Code: "server_error", Message: "Failed to fetch unsorted tracks"},
			})
			return
		}

		c.JSON(http.StatusOK, imodels.APIResponse{
			Success: true,
			Data:    tracks,
		})
	}
}
