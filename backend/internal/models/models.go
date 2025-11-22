package models

import "time"

type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Role         string    `json:"role"`
	StorageBytes *int64    `json:"storage_bytes,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

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

type RefreshToken struct {
	ID        string     `json:"-"`
	UserID    string     `json:"-"`
	TokenHash string     `json:"-"`
	ExpiresAt time.Time  `json:"-"`
	RevokedAt *time.Time `json:"-"`
	CreatedAt time.Time  `json:"-"`
}

type SignupRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type UploadTrackRequest struct {
	Title      string `form:"title"`
	Artist     string `form:"artist"`
	Album      string `form:"album"`
	Genre      string `form:"genre"`
	Year       int    `form:"year"`
	SampleRate int    `form:"sample_rate"`
	Bitrate    int    `form:"bitrate"`
}

type Tokens struct {
	AccessToken  string                 `json:"access_token"`
	RefreshToken string                 `json:"refresh_token"`
	User         map[string]interface{} `json:"user"`
}

type APIResponse struct {
	Success bool      `json:"success"`
	Data    any       `json:"data,omitempty"`
	Error   *APIError `json:"error,omitempty"`
}

type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type TrackList struct {
	Tracks  []*Track `json:"tracks"`
	Total   int      `json:"total"`
	Limit   int      `json:"limit"`
	Offset  int      `json:"offset"`
	HasNext bool     `json:"has_next"`
}

// Playlist represents a music playlist
type Playlist struct {
	ID          string    `json:"id"`
	OwnerUserID string    `json:"owner_user_id"`
	Name        string    `json:"name"`
	Description *string   `json:"description,omitempty"`
	IsDefault   bool      `json:"is_default"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// PlaylistTrack represents the relationship between a playlist and a track
type PlaylistTrack struct {
	ID         string    `json:"id"`
	PlaylistID string    `json:"playlist_id"`
	TrackID    string    `json:"track_id"`
	Position   int       `json:"position"`
	AddedAt    time.Time `json:"added_at"`
}

// PlaylistList represents a paginated list of playlists
type PlaylistList struct {
	Playlists []*Playlist `json:"playlists"`
	Total     int         `json:"total"`
	Limit     int         `json:"limit"`
	Offset    int         `json:"offset"`
	HasNext   bool        `json:"has_next"`
}

// PlaylistWithTracks represents a playlist with its associated tracks
type PlaylistWithTracks struct {
	Playlist *Playlist `json:"playlist"`
	Tracks   []*Track  `json:"tracks"`
	Total    int       `json:"total"`
	Limit    int       `json:"limit"`
	Offset   int       `json:"offset"`
	HasNext  bool      `json:"has_next"`
}

// CreatePlaylistRequest represents a request to create a playlist
type CreatePlaylistRequest struct {
	Name        string  `json:"name" binding:"required"`
	Description *string `json:"description,omitempty"`
}

// UpdatePlaylistRequest represents a request to update a playlist
type UpdatePlaylistRequest struct {
	Name        string  `json:"name" binding:"required"`
	Description *string `json:"description,omitempty"`
}

// AddTracksToPlaylistRequest represents a request to add tracks to a playlist
type AddTracksToPlaylistRequest struct {
	TrackIDs []string `json:"track_ids" binding:"required"`
}

// RemoveTracksFromPlaylistRequest represents a request to remove tracks from a playlist
type RemoveTracksFromPlaylistRequest struct {
	TrackIDs []string `json:"track_ids" binding:"required"`
}
