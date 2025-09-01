package database

import (
	"fmt"
	"time"
)

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

// User operations - moved to auth repository for better separation

func (db *DB) GetUserByEmail(email string) (*User, error) {
	var user User
	err := db.QueryRow(
		"SELECT id, email, password_hash, role, created_at, updated_at FROM users WHERE email = ?",
		email,
	).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.Role, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (db *DB) GetUserByID(id string) (*User, error) {
	var user User
	err := db.QueryRow(
		"SELECT id, email, password_hash, role, created_at, updated_at FROM users WHERE id = ?",
		id,
	).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.Role, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// Track operations
func (db *DB) CreateTrack(track *Track) (*Track, error) {
	id := generateID()
	now := time.Now()

	_, err := db.Exec(
		`INSERT INTO tracks (id, owner_user_id, original_filename, content_type, size_bytes,
			duration_seconds, title, artist, album, file_path, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, track.OwnerUserID, track.OriginalFilename, track.ContentType, track.SizeBytes,
		track.DurationSeconds, track.Title, track.Artist, track.Album, track.FilePath, now, now,
	)
	if err != nil {
		return nil, err
	}

	track.ID = id
	track.CreatedAt = now
	track.UpdatedAt = now

	return track, nil
}

func (db *DB) GetTracks(userID string, limit, offset int) ([]*Track, error) {
	query := `SELECT id, owner_user_id, original_filename, content_type, size_bytes,
		duration_seconds, title, artist, album, file_path, created_at, updated_at
		FROM tracks WHERE owner_user_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?`

	rows, err := db.Query(query, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tracks []*Track
	for rows.Next() {
		var track Track
		err := rows.Scan(
			&track.ID, &track.OwnerUserID, &track.OriginalFilename, &track.ContentType, &track.SizeBytes,
			&track.DurationSeconds, &track.Title, &track.Artist, &track.Album, &track.FilePath,
			&track.CreatedAt, &track.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		tracks = append(tracks, &track)
	}

	return tracks, rows.Err()
}

func (db *DB) GetAllTracks(limit, offset int, searchQuery string) ([]*Track, error) {
	var query string
	var args []interface{}

	if searchQuery != "" {
		query = `SELECT id, owner_user_id, original_filename, content_type, size_bytes,
			duration_seconds, title, artist, album, file_path, created_at, updated_at
			FROM tracks WHERE title LIKE ? OR artist LIKE ? OR album LIKE ? OR original_filename LIKE ?
			ORDER BY created_at DESC LIMIT ? OFFSET ?`
		searchPattern := "%" + searchQuery + "%"
		args = []interface{}{searchPattern, searchPattern, searchPattern, searchPattern, limit, offset}
	} else {
		query = `SELECT id, owner_user_id, original_filename, content_type, size_bytes,
			duration_seconds, title, artist, album, file_path, created_at, updated_at
			FROM tracks ORDER BY created_at DESC LIMIT ? OFFSET ?`
		args = []interface{}{limit, offset}
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tracks []*Track
	for rows.Next() {
		var track Track
		err := rows.Scan(
			&track.ID, &track.OwnerUserID, &track.OriginalFilename, &track.ContentType, &track.SizeBytes,
			&track.DurationSeconds, &track.Title, &track.Artist, &track.Album, &track.FilePath,
			&track.CreatedAt, &track.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		tracks = append(tracks, &track)
	}

	return tracks, rows.Err()
}

func (db *DB) GetTrackByID(trackID string) (*Track, error) {
	var track Track
	err := db.QueryRow(
		`SELECT id, owner_user_id, original_filename, content_type, size_bytes,
		duration_seconds, title, artist, album, file_path, created_at, updated_at
		FROM tracks WHERE id = ?`,
		trackID,
	).Scan(
		&track.ID, &track.OwnerUserID, &track.OriginalFilename, &track.ContentType, &track.SizeBytes,
		&track.DurationSeconds, &track.Title, &track.Artist, &track.Album, &track.FilePath,
		&track.CreatedAt, &track.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &track, nil
}

func (db *DB) DeleteTrack(trackID string) error {
	_, err := db.Exec("DELETE FROM tracks WHERE id = ?", trackID)
	return err
}

// Refresh token operations
func (db *DB) CreateRefreshToken(userID, tokenHash string, expiresAt time.Time) (*RefreshToken, error) {
	id := generateID()
	now := time.Now()

	_, err := db.Exec(
		"INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at, created_at) VALUES (?, ?, ?, ?, ?)",
		id, userID, tokenHash, expiresAt, now,
	)
	if err != nil {
		return nil, err
	}

	return &RefreshToken{
		ID:        id,
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
		CreatedAt: now,
	}, nil
}

func (db *DB) GetRefreshTokenByHash(tokenHash string) (*RefreshToken, error) {
	var token RefreshToken
	err := db.QueryRow(
		"SELECT id, user_id, token_hash, expires_at, revoked_at, created_at FROM refresh_tokens WHERE token_hash = ?",
		tokenHash,
	).Scan(&token.ID, &token.UserID, &token.TokenHash, &token.ExpiresAt, &token.RevokedAt, &token.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &token, nil
}

func (db *DB) RevokeRefreshToken(tokenID string) error {
	now := time.Now()
	_, err := db.Exec("UPDATE refresh_tokens SET revoked_at = ? WHERE id = ?", now, tokenID)
	return err
}

// Helper function to generate IDs
func generateID() string {
	return fmt.Sprintf("id_%d", time.Now().UnixNano())
}
