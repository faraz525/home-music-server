package models

import (
	"time"
)

// User represents a user account
type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Role         string    `json:"role"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Track represents an uploaded music track
type Track struct {
	ID               string    `json:"id"`
	OwnerUserID      string    `json:"owner_user_id"`
	OriginalFilename string    `json:"original_filename"`
	ContentType      string    `json:"content_type"`
	SizeBytes        int64     `json:"size_bytes"`
	DurationSeconds  *float64  `json:"duration_seconds,omitempty"`
	Title            *string   `json:"title,omitempty"`
	Artist           *string   `json:"artist,omitempty"`
	Album            *string   `json:"album,omitempty"`
	Genre            *string   `json:"genre,omitempty"`
	Year             *int      `json:"year,omitempty"`
	SampleRate       *int      `json:"sample_rate,omitempty"`
	Bitrate          *int      `json:"bitrate,omitempty"`
	FilePath         string    `json:"file_path"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// RefreshToken represents a refresh token for session management
type RefreshToken struct {
	ID        string     `json:"-"`
	UserID    string     `json:"-"`
	TokenHash string     `json:"-"`
	ExpiresAt time.Time  `json:"-"`
	RevokedAt *time.Time `json:"-"`
	CreatedAt time.Time  `json:"-"`
}

// SignupRequest represents a signup request
type SignupRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

// LoginRequest represents a login request
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// UploadTrackRequest represents a track upload request
type UploadTrackRequest struct {
	Title      string  `form:"title"`
	Artist     string  `form:"artist"`
	Album      string  `form:"album"`
	Genre      string  `form:"genre"`
	Year       int     `form:"year"`
	SampleRate int     `form:"sample_rate"`
	Bitrate    int     `form:"bitrate"`
	PlaylistID *string `form:"playlist_id"` // Optional playlist to add track to
}

// Tokens represents the response containing access and refresh tokens
type Tokens struct {
	AccessToken  string                 `json:"access_token"`
	RefreshToken string                 `json:"refresh_token"`
	User         map[string]interface{} `json:"user"`
}

// APIResponse represents a standard API response
type APIResponse struct {
	Success bool      `json:"success"`
	Data    any       `json:"data,omitempty"`
	Error   *APIError `json:"error,omitempty"`
}

// APIError represents an API error response
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// TrackList represents a paginated list of tracks
type TrackList struct {
	Tracks  []*Track `json:"tracks"`
	Total   int      `json:"total"`
	Limit   int      `json:"limit"`
	Offset  int      `json:"offset"`
	HasNext bool     `json:"has_next"`
}

// Playlist represents a user-created playlist
type Playlist struct {
	ID          string    `json:"id"`
	OwnerUserID string    `json:"owner_user_id"`
	Name        string    `json:"name"`
	Description *string   `json:"description,omitempty"`
	IsDefault   bool      `json:"is_default"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// PlaylistTrack represents the junction between playlists and tracks
type PlaylistTrack struct {
	ID         string    `json:"id"`
	PlaylistID string    `json:"playlist_id"`
	TrackID    string    `json:"track_id"`
	AddedAt    time.Time `json:"added_at"`
}

// PlaylistWithTracks represents a playlist with its associated tracks
type PlaylistWithTracks struct {
	Playlist Playlist `json:"playlist"`
	Tracks   []*Track `json:"tracks"`
	Total    int      `json:"total"`
	Limit    int      `json:"limit"`
	Offset   int      `json:"offset"`
	HasNext  bool     `json:"has_next"`
}

// PlaylistList represents a paginated list of playlists
type PlaylistList struct {
	Playlists []*Playlist `json:"playlists"`
	Total     int         `json:"total"`
	Limit     int         `json:"limit"`
	Offset    int         `json:"offset"`
	HasNext   bool        `json:"has_next"`
}

// Playlist API Request/Response types
type CreatePlaylistRequest struct {
	Name        string  `json:"name" binding:"required,min=1,max=100"`
	Description *string `json:"description,omitempty"`
}

type UpdatePlaylistRequest struct {
	Name        string  `json:"name" binding:"required,min=1,max=100"`
	Description *string `json:"description,omitempty"`
}

type AddTracksToPlaylistRequest struct {
	TrackIDs []string `json:"track_ids" binding:"required,min=1"`
}

type RemoveTracksFromPlaylistRequest struct {
	TrackIDs []string `json:"track_ids" binding:"required,min=1"`
}
