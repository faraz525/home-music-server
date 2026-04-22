package tracks

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestValidateKey(t *testing.T) {
	valid := []string{"1A", "12B", "7A", "9B"}
	for _, k := range valid {
		if !isValidCamelot(k) {
			t.Errorf("isValidCamelot(%q) = false, want true", k)
		}
	}
	invalid := []string{"", "0A", "13A", "1C", "1a", "A1", "100A"}
	for _, k := range invalid {
		if isValidCamelot(k) {
			t.Errorf("isValidCamelot(%q) = true, want false", k)
		}
	}
}

func TestValidateBPM(t *testing.T) {
	if !isValidBPM(120) {
		t.Error("isValidBPM(120) = false, want true")
	}
	if isValidBPM(49) {
		t.Error("isValidBPM(49) = true, want false")
	}
	if isValidBPM(251) {
		t.Error("isValidBPM(251) = true, want false")
	}
}

// TestPatchHandler_RejectsInvalidBody verifies the handler rejects malformed
// payloads with a 400 response. Uses a nil manager since we don't reach the
// DB for validation-only tests.
func TestPatchHandler_RejectsInvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.PATCH("/api/tracks/:id", PatchHandler(nil))

	req := httptest.NewRequest("PATCH", "/api/tracks/abc", strings.NewReader(`{"bpm": 9999}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestPatchHandler_RejectsEmptyBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.PATCH("/api/tracks/:id", PatchHandler(nil))

	req := httptest.NewRequest("PATCH", "/api/tracks/abc", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Errorf("status = %d, want 400 (empty body)", w.Code)
	}
}
