package models

import "time"

type User struct {
    ID           string    `json:"id"`
    Email        string    `json:"email"`
    PasswordHash string    `json:"-"`
    Role         string    `json:"role"`
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

