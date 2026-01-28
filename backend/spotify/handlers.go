package spotify

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

	clientID := h.manager.GetClientID()

	if cfg == nil {
		c.JSON(http.StatusOK, gin.H{
			"configured": false,
			"enabled":    false,
			"client_id":  clientID,
		})
		return
	}

	hasToken := cfg.AccessToken != nil && *cfg.AccessToken != ""

	c.JSON(http.StatusOK, gin.H{
		"configured":              hasToken,
		"enabled":                 cfg.Enabled,
		"owner_user_id":           cfg.OwnerUserID,
		"liked_songs_playlist_id": cfg.LikedSongsPlaylistID,
		"last_sync_at":            cfg.LastSyncAt,
		"client_id":               clientID,
	})
}

type UpdateConfigRequest struct {
	Enabled bool `json:"enabled"`
}

func (h *Handlers) UpdateConfig(c *gin.Context) {
	var req UpdateConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "invalid_request", "message": err.Error()}})
		return
	}

	// Get the user ID from the context (set by auth middleware)
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "unauthorized", "message": "user not authenticated"}})
		return
	}

	if err := h.manager.SaveSyncConfig(c.Request.Context(), userID.(string), req.Enabled); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "internal_error", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "config updated"})
}

type TokenExchangeRequest struct {
	Code         string `json:"code" binding:"required"`
	CodeVerifier string `json:"code_verifier" binding:"required"`
	RedirectURI  string `json:"redirect_uri" binding:"required"`
}

func (h *Handlers) ExchangeToken(c *gin.Context) {
	var req TokenExchangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "invalid_request", "message": err.Error()}})
		return
	}

	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "unauthorized", "message": "user not authenticated"}})
		return
	}

	if err := h.manager.ExchangeCodeForToken(c.Request.Context(), req.Code, req.CodeVerifier, req.RedirectURI, userID.(string)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "token_exchange_failed", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "connected to Spotify"})
}

func (h *Handlers) Disconnect(c *gin.Context) {
	if err := h.manager.Disconnect(c.Request.Context()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "internal_error", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "disconnected from Spotify"})
}

func (h *Handlers) TriggerSync(c *gin.Context) {
	go func() {
		if err := h.manager.SyncLikes(context.Background()); err != nil {
			fmt.Printf("[Spotify] Manual sync failed: %v\n", err)
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

func (h *Handlers) GetPlaylists(c *gin.Context) {
	playlists, err := h.manager.FetchUserPlaylists(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "internal_error", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"playlists": playlists})
}

func (h *Handlers) GetSyncedPlaylists(c *gin.Context) {
	playlists, err := h.manager.GetSyncedPlaylists(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "internal_error", "message": err.Error()}})
		return
	}

	if playlists == nil {
		playlists = []*SyncedPlaylist{}
	}

	c.JSON(http.StatusOK, gin.H{"playlists": playlists})
}
