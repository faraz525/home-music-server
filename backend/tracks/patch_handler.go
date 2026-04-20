package tracks

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
)

type patchTrackRequest struct {
	BPM        *float64 `json:"bpm,omitempty"`
	MusicalKey *string  `json:"musical_key,omitempty"`
}

// BPM bounds: 50 covers slow ballads / downtempo; 250 covers drum & bass /
// hardcore. Tracks outside this range exist but are vanishingly rare in DJ
// libraries and usually indicate a half-time / double-time analysis error.
const (
	minBPM = 50
	maxBPM = 250
)

var camelotRE = regexp.MustCompile(`^(1[0-2]|[1-9])[AB]$`)

func isValidBPM(bpm float64) bool {
	return bpm >= minBPM && bpm <= maxBPM
}

func isValidCamelot(k string) bool {
	return camelotRE.MatchString(k)
}

// PatchHandler handles PATCH /api/tracks/:id for user overrides of BPM/key.
// `mgr` can be nil only in tests that exercise validation-only paths.
func PatchHandler(mgr *Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req patchTrackRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "invalid_body", "message": "request body is not valid JSON"}})
			return
		}
		if req.BPM == nil && req.MusicalKey == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "empty_patch", "message": "at least one of bpm, musical_key is required"}})
			return
		}
		if req.BPM != nil && !isValidBPM(*req.BPM) {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "invalid_bpm", "message": fmt.Sprintf("bpm must be between %d and %d", minBPM, maxBPM)}})
			return
		}
		if req.MusicalKey != nil {
			// Normalize to uppercase (DJs routinely type "8a"); validate after.
			upper := strings.ToUpper(*req.MusicalKey)
			req.MusicalKey = &upper
			if !isValidCamelot(*req.MusicalKey) {
				c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "invalid_key", "message": "musical_key must be Camelot notation like 8A or 12B"}})
				return
			}
		}

		trackID := c.Param("id")
		if mgr == nil {
			c.Status(http.StatusNoContent)
			return
		}

		userIDAny, _ := c.Get("user_id")
		userID, _ := userIDAny.(string)
		userRoleAny, _ := c.Get("user_role")
		userRole, _ := userRoleAny.(string)

		track, err := mgr.GetTrack(c.Request.Context(), trackID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "not_found", "message": "track not found"}})
			return
		}
		// Match the owner-or-admin pattern used by sibling handlers
		// (GetHandler, StreamHandler, DeleteHandler).
		if track.OwnerUserID != userID && userRole != "admin" {
			c.JSON(http.StatusForbidden, gin.H{"error": gin.H{"code": "forbidden", "message": "not your track"}})
			return
		}

		if err := mgr.UpdateAnalysisOverride(c.Request.Context(), trackID, req.BPM, req.MusicalKey); err != nil {
			// Don't leak SQL driver text to clients.
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "update_failed", "message": "failed to update track analysis"}})
			return
		}
		updated, err := mgr.GetTrack(c.Request.Context(), trackID)
		if err != nil {
			// Update succeeded; re-fetch failed. Report success — the client
			// can refresh if it needs the canonical row.
			c.JSON(http.StatusOK, gin.H{"success": true})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": updated})
	}
}
