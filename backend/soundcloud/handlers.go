package soundcloud

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handlers struct {
	manager *Manager
}

func NewHandlers(manager *Manager) *Handlers {
	return &Handlers{manager: manager}
}

func (h *Handlers) GetConfig(c *gin.Context) {
	cfg, err := h.manager.GetSyncConfig(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "internal_error", "message": err.Error()}})
		return
	}

	if cfg == nil {
		c.JSON(http.StatusOK, gin.H{
			"configured": false,
			"enabled":    false,
		})
		return
	}

	hasToken := cfg.OAuthToken != nil && *cfg.OAuthToken != ""

	c.JSON(http.StatusOK, gin.H{
		"configured":    hasToken,
		"enabled":       cfg.Enabled,
		"owner_user_id": cfg.OwnerUserID,
		"playlist_id":   cfg.PlaylistID,
		"last_sync_at":  cfg.LastSyncAt,
	})
}

type UpdateConfigRequest struct {
	OAuthToken  string `json:"oauth_token"`
	OwnerUserID string `json:"owner_user_id" binding:"required"`
	Enabled     bool   `json:"enabled"`
}

func (h *Handlers) UpdateConfig(c *gin.Context) {
	var req UpdateConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "invalid_request", "message": err.Error()}})
		return
	}

	if err := h.manager.SaveSyncConfig(c.Request.Context(), req.OAuthToken, req.OwnerUserID, req.Enabled); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "internal_error", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "config updated"})
}

func (h *Handlers) TriggerSync(c *gin.Context) {
	go func() {
		if err := h.manager.SyncLikes(context.Background()); err != nil {
			fmt.Printf("[SoundCloud] Manual sync failed: %v\n", err)
		}
	}()

	c.JSON(http.StatusAccepted, gin.H{"message": "sync triggered"})
}

func (h *Handlers) GetHistory(c *gin.Context) {
	history, err := h.manager.GetSyncHistory(c.Request.Context(), 20)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "internal_error", "message": err.Error()}})
		return
	}

	if history == nil {
		history = []*SyncHistory{}
	}

	c.JSON(http.StatusOK, gin.H{"history": history})
}
