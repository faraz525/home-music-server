package utils

import (
	"fmt"
	"path/filepath"
	"time"

	imodels "github.com/faraz525/home-music-server/backend/internal/models"
)

// GenerateID generates a unique ID with a prefix
func GenerateID(prefix string) string {
	return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
}

// GenerateUserID generates a unique user ID
func GenerateUserID() string {
	return GenerateID("user")
}

// GenerateTrackID generates a unique track ID
func GenerateTrackID() string {
	return GenerateID("track")
}

// GenerateTokenID generates a unique token ID
func GenerateTokenID() string {
	return GenerateID("token")
}

// StringToPtr converts a string to a pointer, returning nil for empty strings
func StringToPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// IsValidAudioType checks if the content type is a supported audio format
func IsValidAudioType(contentType string) bool {
	validTypes := []string{
		"audio/wav", "audio/wave", "audio/x-wav",
		"audio/aiff", "audio/x-aiff",
		"audio/flac", "audio/x-flac",
		"audio/mpeg", "audio/mp3",
	}
	for _, validType := range validTypes {
		if contentType == validType {
			return true
		}
	}
	return false
}

// GetFileExtension extracts the file extension from a filename
func GetFileExtension(filename string) string {
	return filepath.Ext(filename)
}

// BuildTrackFilePath builds the file path for a track
func BuildTrackFilePath(userID, trackID, filename string) string {
	ext := GetFileExtension(filename)
	return filepath.Join("library", userID, trackID, trackID+ext)
}

// NewAPIResponse creates a new API response
func NewAPIResponse(success bool, data interface{}, err *imodels.APIError) *imodels.APIResponse {
	return &imodels.APIResponse{
		Success: success,
		Data:    data,
		Error:   err,
	}
}

// NewAPIError creates a new API error
func NewAPIError(code, message string) *imodels.APIError {
	return &imodels.APIError{
		Code:    code,
		Message: message,
	}
}

// NewTrackList creates a new paginated track list
func NewTrackList(tracks []*imodels.Track, total, limit, offset int) *imodels.TrackList {
	hasNext := offset+limit < total
	return &imodels.TrackList{
		Tracks:  tracks,
		Total:   total,
		Limit:   limit,
		Offset:  offset,
		HasNext: hasNext,
	}
}

// Now centralizes time access for testability
func Now() time.Time { return time.Now() }
