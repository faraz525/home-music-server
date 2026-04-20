package tracks

import (
	"net/http"
	"regexp"

	"github.com/gin-gonic/gin"
)

type patchTrackRequest struct {
	BPM        *float64 `json:"bpm,omitempty"`
	MusicalKey *string  `json:"musical_key,omitempty"`
}

var camelotRE = regexp.MustCompile(`^(1[0-2]|[1-9])[AB]$`)

func isValidBPM(bpm float64) bool {
	return bpm >= 50 && bpm <= 250
}

func isValidCamelot(k string) bool {
	return camelotRE.MatchString(k)
}

// PatchHandler handles PATCH /api/tracks/:id for user overrides of BPM/key.
// `mgr` can be nil only in tests that exercise validation paths.
func PatchHandler(mgr *Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req patchTrackRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "invalid_body", "message": err.Error()}})
			return
		}
		if req.BPM == nil && req.MusicalKey == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "empty_patch", "message": "at least one of bpm, musical_key is required"}})
			return
		}
		if req.BPM != nil && !isValidBPM(*req.BPM) {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "invalid_bpm", "message": "bpm must be between 50 and 250"}})
			return
		}
		if req.MusicalKey != nil && !isValidCamelot(*req.MusicalKey) {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "invalid_key", "message": "musical_key must be Camelot notation like 8A or 12B"}})
			return
		}

		trackID := c.Param("id")
		if mgr == nil {
			c.Status(http.StatusNoContent)
			return
		}

		userIDAny, _ := c.Get("user_id")
		userID, _ := userIDAny.(string)
		track, err := mgr.GetTrack(c.Request.Context(), trackID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "not_found", "message": "track not found"}})
			return
		}
		if track.OwnerUserID != userID {
			c.JSON(http.StatusForbidden, gin.H{"error": gin.H{"code": "forbidden", "message": "not your track"}})
			return
		}

		if err := mgr.UpdateAnalysisOverride(c.Request.Context(), trackID, req.BPM, req.MusicalKey); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "update_failed", "message": err.Error()}})
			return
		}
		updated, err := mgr.GetTrack(c.Request.Context(), trackID)
		if err != nil {
			c.Status(http.StatusNoContent)
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": updated})
	}
}
