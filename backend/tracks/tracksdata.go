package tracks

import (
	"context"
	"strings"

	"github.com/faraz525/home-music-server/backend/internal/db"
	imodels "github.com/faraz525/home-music-server/backend/internal/models"
	"github.com/faraz525/home-music-server/backend/utils"
)

// Data handles track data operations
type Repository struct {
	db *db.DB
}

// NewRepository creates a new tracks data
func NewRepository(db *db.DB) *Repository {
	return &Repository{db: db}
}

// CreateTrack creates a new track in the database
func (r *Repository) CreateTrack(ctx context.Context, track *imodels.Track) (*imodels.Track, error) {
	id := utils.GenerateTrackID()

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO tracks (id, owner_user_id, original_filename, content_type, size_bytes,
			duration_seconds, title, artist, album, genre, year, sample_rate, bitrate, file_path, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, track.OwnerUserID, track.OriginalFilename, track.ContentType, track.SizeBytes,
		track.DurationSeconds, track.Title, track.Artist, track.Album, track.Genre, track.Year,
		track.SampleRate, track.Bitrate, track.FilePath, track.CreatedAt, track.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	track.ID = id
	return track, nil
}

// GetTracks retrieves tracks for a user with pagination
func (r *Repository) GetTracks(ctx context.Context, userID string, limit, offset int) ([]*imodels.Track, error) {
	query := `SELECT id, owner_user_id, original_filename, content_type, size_bytes,
		duration_seconds, title, artist, album, genre, year, sample_rate, bitrate, file_path, created_at, updated_at
		FROM tracks WHERE owner_user_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?`

	rows, err := r.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tracks []*imodels.Track
	for rows.Next() {
		var track imodels.Track
		err := rows.Scan(
			&track.ID, &track.OwnerUserID, &track.OriginalFilename, &track.ContentType, &track.SizeBytes,
			&track.DurationSeconds, &track.Title, &track.Artist, &track.Album, &track.Genre, &track.Year,
			&track.SampleRate, &track.Bitrate, &track.FilePath, &track.CreatedAt, &track.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		tracks = append(tracks, &track)
	}

	return tracks, rows.Err()
}

// GetAllTracks retrieves all tracks with optional FTS5 search (admin only)
func (r *Repository) GetAllTracks(ctx context.Context, limit, offset int, searchQuery string) ([]*imodels.Track, error) {
	var query string
	var args []interface{}

	if searchQuery != "" {
		// Use FTS5 for search
		query = `
			SELECT t.id, t.owner_user_id, t.original_filename, t.content_type, t.size_bytes,
				t.duration_seconds, t.title, t.artist, t.album, t.genre, t.year, 
				t.sample_rate, t.bitrate, t.file_path, t.created_at, t.updated_at
			FROM tracks t
			INNER JOIN tracks_fts fts ON t.id = fts.track_id
			WHERE fts MATCH ?
			ORDER BY fts.rank, t.created_at DESC
			LIMIT ? OFFSET ?
		`
		ftsQuery := prepareFTS5Query(searchQuery)
		args = []interface{}{ftsQuery, limit, offset}
	} else {
		// No search, just list all
		query = `
			SELECT id, owner_user_id, original_filename, content_type, size_bytes,
				duration_seconds, title, artist, album, genre, year, sample_rate, bitrate, 
				file_path, created_at, updated_at
			FROM tracks 
			ORDER BY created_at DESC 
			LIMIT ? OFFSET ?
		`
		args = []interface{}{limit, offset}
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tracks []*imodels.Track
	for rows.Next() {
		var track imodels.Track
		err := rows.Scan(
			&track.ID, &track.OwnerUserID, &track.OriginalFilename, &track.ContentType, &track.SizeBytes,
			&track.DurationSeconds, &track.Title, &track.Artist, &track.Album, &track.Genre, &track.Year,
			&track.SampleRate, &track.Bitrate, &track.FilePath, &track.CreatedAt, &track.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		tracks = append(tracks, &track)
	}

	return tracks, rows.Err()
}

// GetTrackByID retrieves a single track by ID
func (r *Repository) GetTrackByID(ctx context.Context, trackID string) (*imodels.Track, error) {
	var track imodels.Track
	err := r.db.QueryRowContext(ctx,
		`SELECT id, owner_user_id, original_filename, content_type, size_bytes,
		duration_seconds, title, artist, album, genre, year, sample_rate, bitrate, file_path, created_at, updated_at
		FROM tracks WHERE id = ?`,
		trackID,
	).Scan(
		&track.ID, &track.OwnerUserID, &track.OriginalFilename, &track.ContentType, &track.SizeBytes,
		&track.DurationSeconds, &track.Title, &track.Artist, &track.Album, &track.Genre, &track.Year,
		&track.SampleRate, &track.Bitrate, &track.FilePath, &track.CreatedAt, &track.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &track, nil
}

// DeleteTrack deletes a track by ID
func (r *Repository) DeleteTrack(ctx context.Context, trackID string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM tracks WHERE id = ?", trackID)
	return err
}

// GetTracksCount returns the total count of tracks for a user
func (r *Repository) GetTracksCount(ctx context.Context, userID string) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM tracks WHERE owner_user_id = ?", userID).Scan(&count)
	return count, err
}

// GetAllTracksCount returns the total count of all tracks
func (r *Repository) GetAllTracksCount(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM tracks").Scan(&count)
	return count, err
}

// SearchTracks searches tracks by title, artist, album, genre, or filename using FTS5
// FTS5 provides much faster full-text search than LIKE, especially on Raspberry Pi
func (r *Repository) SearchTracks(ctx context.Context, query string, userID string, limit, offset int) ([]*imodels.Track, error) {
	// FTS5 query - searches across all indexed fields
	// Use double quotes for phrase matching or just the term for any word matching
	searchQuery := `
		SELECT t.id, t.owner_user_id, t.original_filename, t.content_type, t.size_bytes,
			t.duration_seconds, t.title, t.artist, t.album, t.genre, t.year, 
			t.sample_rate, t.bitrate, t.file_path, t.created_at, t.updated_at
		FROM tracks t
		INNER JOIN tracks_fts fts ON t.id = fts.track_id
		WHERE t.owner_user_id = ? 
		AND fts MATCH ?
		ORDER BY fts.rank, t.created_at DESC
		LIMIT ? OFFSET ?
	`

	// Escape special FTS5 characters and prepare the query
	// FTS5 uses OR by default for multiple terms
	ftsQuery := prepareFTS5Query(query)

	// Debug logging
	println("[SearchTracks] Original query:", query)
	println("[SearchTracks] FTS5 query:", ftsQuery)
	println("[SearchTracks] UserID:", userID)

	rows, err := r.db.QueryContext(ctx, searchQuery, userID, ftsQuery, limit, offset)
	if err != nil {
		println("[SearchTracks] ERROR:", err.Error())
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tracks []*imodels.Track
	for rows.Next() {
		var track imodels.Track
		err := rows.Scan(
			&track.ID, &track.OwnerUserID, &track.OriginalFilename, &track.ContentType, &track.SizeBytes,
			&track.DurationSeconds, &track.Title, &track.Artist, &track.Album, &track.Genre, &track.Year,
			&track.SampleRate, &track.Bitrate, &track.FilePath, &track.CreatedAt, &track.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		tracks = append(tracks, &track)
	}

	return tracks, rows.Err()
}

// prepareFTS5Query escapes and formats a user query for FTS5
func prepareFTS5Query(query string) string {
	// Remove special FTS5 operators that could cause syntax errors
	// and prepare for full-text search
	query = strings.TrimSpace(query)
	if query == "" {
		return "*"
	}

	// Remove or escape special FTS5 characters that can cause issues
	// These characters have special meaning in FTS5: & " - ( ) : ^
	specialChars := map[string]string{
		"&": "", // Remove & as it's an operator
		"(": "", // Remove parentheses
		")": "",
		":": "",  // Remove colon (column specifier)
		"^": "",  // Remove caret
		"-": " ", // Convert dash to space (so "on-off" becomes "on off")
	}

	for old, new := range specialChars {
		query = strings.ReplaceAll(query, old, new)
	}

	// Normalize multiple spaces to single space
	query = strings.Join(strings.Fields(query), " ")

	// Add prefix matching with wildcard (*) to enable "as you type" search
	// This allows "pres" to match "pressure", "pres*" matches any word starting with "pres"
	// Split by spaces to handle multiple terms
	terms := strings.Fields(query)
	for i, term := range terms {
		// Don't add * if the term already has special FTS5 operators
		if !strings.Contains(term, "*") && !strings.Contains(term, `"`) {
			terms[i] = term + "*"
		}
	}

	// Join terms with AND so ALL terms must match (not just any one)
	// "feel the vibration" becomes "feel* AND the* AND vibration*"
	// "on & on" becomes "on* AND on*" (& is removed, both "on" terms must match)
	return strings.Join(terms, " AND ")
}
