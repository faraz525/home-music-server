package trades

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	imodels "github.com/faraz525/home-music-server/backend/internal/models"
)

// RequestTradeHandler handles trade requests
func RequestTradeHandler(manager *Manager) gin.HandlerFunc {
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

		var req imodels.TradeRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, imodels.APIResponse{
				Success: false,
				Error:   &imodels.APIError{Code: "invalid_request", Message: err.Error()},
			})
			return
		}

		err := manager.RequestTrade(c.Request.Context(), userID.(string), &req)
		if err != nil {
			statusCode := http.StatusInternalServerError
			errorCode := "trade_failed"

			if err.Error() == "crate not found" {
				statusCode = http.StatusNotFound
				errorCode = "crate_not_found"
			} else if err.Error() == "crate is not public" {
				statusCode = http.StatusForbidden
				errorCode = "crate_not_public"
			} else if err.Error() == "you already own this track" || err.Error() == "you already have this track" {
				statusCode = http.StatusConflict
				errorCode = "already_owned"
			} else if err.Error() == "track not found" {
				statusCode = http.StatusNotFound
				errorCode = "track_not_found"
			}

			c.JSON(statusCode, imodels.APIResponse{
				Success: false,
				Error:   &imodels.APIError{Code: errorCode, Message: err.Error()},
			})
			return
		}

		c.JSON(http.StatusOK, imodels.APIResponse{
			Success: true,
			Data:    gin.H{"message": "Trade completed successfully"},
		})
	}
}

// GetTradeHistoryHandler returns user's trade history
func GetTradeHistoryHandler(manager *Manager) gin.HandlerFunc {
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

		trades, total, err := manager.GetUserTradeHistory(c.Request.Context(), userID.(string), limit, offset)
		if err != nil {
			c.JSON(http.StatusInternalServerError, imodels.APIResponse{
				Success: false,
				Error:   &imodels.APIError{Code: "server_error", Message: "Failed to fetch trade history"},
			})
			return
		}

		hasNext := offset+limit < total

		c.JSON(http.StatusOK, imodels.APIResponse{
			Success: true,
			Data: gin.H{
				"trades":   trades,
				"total":    total,
				"limit":    limit,
				"offset":   offset,
				"has_next": hasNext,
			},
		})
	}
}

// GetAvailableTracksHandler returns tracks available for trading
func GetAvailableTracksHandler(manager *Manager) gin.HandlerFunc {
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

		trackIDs, err := manager.GetAvailableTracksForTrade(c.Request.Context(), userID.(string))
		if err != nil {
			c.JSON(http.StatusInternalServerError, imodels.APIResponse{
				Success: false,
				Error:   &imodels.APIError{Code: "server_error", Message: "Failed to fetch available tracks"},
			})
			return
		}

		c.JSON(http.StatusOK, imodels.APIResponse{
			Success: true,
			Data:    gin.H{"track_ids": trackIDs},
		})
	}
}
