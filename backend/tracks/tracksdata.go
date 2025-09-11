package tracks

import (
	"context"

	"github.com/faraz525/home-music-server/backend/internal/db"
	imodels "github.com/faraz525/home-music-server/backend/internal/models"
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
	id := generateTrackID()

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

// GetAllTracks retrieves all tracks with optional search (admin only)
func (r *Repository) GetAllTracks(ctx context.Context, limit, offset int, searchQuery string) ([]*imodels.Track, error) {
	var query string
	var args []interface{}

	if searchQuery != "" {
		query = `SELECT id, owner_user_id, original_filename, content_type, size_bytes,
			duration_seconds, title, artist, album, genre, year, sample_rate, bitrate, file_path, created_at, updated_at
			FROM tracks WHERE title LIKE ? OR artist LIKE ? OR album LIKE ? OR genre LIKE ? OR original_filename LIKE ?
			ORDER BY created_at DESC LIMIT ? OFFSET ?`
		searchPattern := "%" + searchQuery + "%"
		args = []interface{}{searchPattern, searchPattern, searchPattern, searchPattern, searchPattern, limit, offset}
	} else {
		query = `SELECT id, owner_user_id, original_filename, content_type, size_bytes,
			duration_seconds, title, artist, album, genre, year, sample_rate, bitrate, file_path, created_at, updated_at
			FROM tracks ORDER BY created_at DESC LIMIT ? OFFSET ?`
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

// SearchTracks searches tracks by title, artist, album, genre, or filename
func (r *Repository) SearchTracks(ctx context.Context, query string, userID string, limit, offset int) ([]*imodels.Track, error) {
	searchPattern := "%" + query + "%"
	searchQuery := `SELECT id, owner_user_id, original_filename, content_type, size_bytes,
		duration_seconds, title, artist, album, genre, year, sample_rate, bitrate, file_path, created_at, updated_at
		FROM tracks WHERE owner_user_id = ? AND (title LIKE ? OR artist LIKE ? OR album LIKE ? OR genre LIKE ? OR original_filename LIKE ?)
		ORDER BY created_at DESC LIMIT ? OFFSET ?`

	rows, err := r.db.QueryContext(ctx, searchQuery, userID, searchPattern, searchPattern, searchPattern, searchPattern, searchPattern, limit, offset)
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
