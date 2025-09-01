package tracks

import (
	"github.com/faraz525/home-music-server/backend/models"
	"github.com/faraz525/home-music-server/backend/utils"
)

// Repository handles track data operations
type Repository struct {
	db *utils.DB
}

// NewRepository creates a new tracks repository
func NewRepository(db *utils.DB) *Repository {
	return &Repository{db: db}
}

// CreateTrack creates a new track in the database
func (r *Repository) CreateTrack(track *models.Track) (*models.Track, error) {
	id := utils.GenerateTrackID()

	_, err := r.db.Exec(
		`INSERT INTO tracks (id, owner_user_id, original_filename, content_type, size_bytes,
			duration_seconds, title, artist, album, file_path, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, track.OwnerUserID, track.OriginalFilename, track.ContentType, track.SizeBytes,
		track.DurationSeconds, track.Title, track.Artist, track.Album, track.FilePath, track.CreatedAt, track.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	track.ID = id
	return track, nil
}

// GetTracks retrieves tracks for a user with pagination
func (r *Repository) GetTracks(userID string, limit, offset int) ([]*models.Track, error) {
	query := `SELECT id, owner_user_id, original_filename, content_type, size_bytes,
		duration_seconds, title, artist, album, file_path, created_at, updated_at
		FROM tracks WHERE owner_user_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?`

	rows, err := r.db.Query(query, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tracks []*models.Track
	for rows.Next() {
		var track models.Track
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

// GetAllTracks retrieves all tracks with optional search (admin only)
func (r *Repository) GetAllTracks(limit, offset int, searchQuery string) ([]*models.Track, error) {
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

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tracks []*models.Track
	for rows.Next() {
		var track models.Track
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

// GetTrackByID retrieves a single track by ID
func (r *Repository) GetTrackByID(trackID string) (*models.Track, error) {
	var track models.Track
	err := r.db.QueryRow(
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

// DeleteTrack deletes a track by ID
func (r *Repository) DeleteTrack(trackID string) error {
	_, err := r.db.Exec("DELETE FROM tracks WHERE id = ?", trackID)
	return err
}

// GetTracksCount returns the total count of tracks for a user
func (r *Repository) GetTracksCount(userID string) (int, error) {
	var count int
	err := r.db.QueryRow("SELECT COUNT(*) FROM tracks WHERE owner_user_id = ?", userID).Scan(&count)
	return count, err
}

// GetAllTracksCount returns the total count of all tracks
func (r *Repository) GetAllTracksCount() (int, error) {
	var count int
	err := r.db.QueryRow("SELECT COUNT(*) FROM tracks").Scan(&count)
	return count, err
}

// SearchTracks searches tracks by title, artist, or album
func (r *Repository) SearchTracks(query string, userID string, limit, offset int) ([]*models.Track, error) {
	searchPattern := "%" + query + "%"
	searchQuery := `SELECT id, owner_user_id, original_filename, content_type, size_bytes,
		duration_seconds, title, artist, album, file_path, created_at, updated_at
		FROM tracks WHERE owner_user_id = ? AND (title LIKE ? OR artist LIKE ? OR album LIKE ? OR original_filename LIKE ?)
		ORDER BY created_at DESC LIMIT ? OFFSET ?`

	rows, err := r.db.Query(searchQuery, userID, searchPattern, searchPattern, searchPattern, searchPattern, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tracks []*models.Track
	for rows.Next() {
		var track models.Track
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
