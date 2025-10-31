package users

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	imodels "github.com/faraz525/home-music-server/backend/internal/models"
)

// SearchUsersHandler handles user search requests
func SearchUsersHandler(manager *Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		query := c.Query("q")
		if query == "" {
			c.JSON(http.StatusBadRequest, imodels.APIResponse{
				Success: false,
				Error:   &imodels.APIError{Code: "invalid_request", Message: "Search query is required"},
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

		// Check if user is admin
		userRole, _ := c.Get("user_role")
		isAdmin := userRole == "admin"

		users, total, err := manager.SearchUsers(c.Request.Context(), query, limit, offset, isAdmin)
		if err != nil {
			c.JSON(http.StatusInternalServerError, imodels.APIResponse{
				Success: false,
				Error:   &imodels.APIError{Code: "server_error", Message: "Failed to search users"},
			})
			return
		}

		hasNext := offset+limit < total

		c.JSON(http.StatusOK, imodels.APIResponse{
			Success: true,
			Data: gin.H{
				"users":    users,
				"total":    total,
				"limit":    limit,
				"offset":   offset,
				"has_next": hasNext,
			},
		})
	}
}

// GetUserByUsernameHandler handles get user by username requests
func GetUserByUsernameHandler(manager *Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		username := c.Param("username")

		user, err := manager.GetUserByUsername(c.Request.Context(), username)
		if err != nil {
			statusCode := http.StatusInternalServerError
			if err.Error() == "user not found" {
				statusCode = http.StatusNotFound
			}

			c.JSON(statusCode, imodels.APIResponse{
				Success: false,
				Error:   &imodels.APIError{Code: "user_not_found", Message: err.Error()},
			})
			return
		}

		c.JSON(http.StatusOK, imodels.APIResponse{
			Success: true,
			Data:    gin.H{"user": user},
		})
	}
}

// UpdateUsernameHandler handles username update requests
func UpdateUsernameHandler(manager *Manager) gin.HandlerFunc {
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

		var req imodels.UpdateUsernameRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, imodels.APIResponse{
				Success: false,
				Error:   &imodels.APIError{Code: "invalid_request", Message: err.Error()},
			})
			return
		}

		err := manager.UpdateUsername(c.Request.Context(), userID.(string), &req)
		if err != nil {
			statusCode := http.StatusInternalServerError
			errorCode := "update_failed"

			if err.Error() == "username already taken" {
				statusCode = http.StatusConflict
				errorCode = "username_taken"
			} else if err.Error() == "user not found" {
				statusCode = http.StatusNotFound
				errorCode = "user_not_found"
			}

			c.JSON(statusCode, imodels.APIResponse{
				Success: false,
				Error:   &imodels.APIError{Code: errorCode, Message: err.Error()},
			})
			return
		}

		c.JSON(http.StatusOK, imodels.APIResponse{
			Success: true,
			Data:    gin.H{"message": "Username updated successfully"},
		})
	}
}
