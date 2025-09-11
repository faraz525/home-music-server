package playlists

import (
	"database/sql"
	"fmt"

	idb "github.com/faraz525/home-music-server/backend/internal/db"
	imodels "github.com/faraz525/home-music-server/backend/internal/models"
	"github.com/google/uuid"
)

// Repository handles database operations for playlists
type Repository struct {
	db *idb.DB
}

// NewRepository creates a new playlist repository
func NewRepository(db *idb.DB) *Repository {
	return &Repository{db: db}
}

// CreatePlaylist creates a new playlist for a user
func (r *Repository) CreatePlaylist(ownerUserID string, req *imodels.CreatePlaylistRequest) (*imodels.Playlist, error) {
	id := uuid.New().String()

	// Debug: Check if user exists
	var userCount int
	err := r.db.QueryRow("SELECT COUNT(*) FROM users WHERE id = ?", ownerUserID).Scan(&userCount)
	if err != nil {
		return nil, fmt.Errorf("failed to check user: %w", err)
	}
	if userCount == 0 {
		return nil, fmt.Errorf("user not found: %s", ownerUserID)
	}

	query := `
		INSERT INTO playlists (id, owner_user_id, name, description, is_default)
		VALUES (?, ?, ?, ?, ?)
	`

	var description interface{}
	if req.Description != nil {
		description = *req.Description
	} else {
		description = nil
	}

	_, err = r.db.DB.Exec(query, id, ownerUserID, req.Name, description, false)
	if err != nil {
		return nil, fmt.Errorf("failed to create playlist: %w", err)
	}

	return r.GetPlaylist(id)
}

// GetUserPlaylists returns all playlists for a user
func (r *Repository) GetUserPlaylists(userID string, limit, offset int) (*imodels.PlaylistList, error) {
	query := `
		SELECT id, owner_user_id, name, description, is_default, created_at, updated_at
		FROM playlists
		WHERE owner_user_id = ?
		ORDER BY is_default DESC, created_at DESC
		LIMIT ? OFFSET ?
	`

	rows, err := r.db.Query(query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get user playlists: %w", err)
	}
	defer rows.Close()

	var playlists []*imodels.Playlist
	for rows.Next() {
		playlist := &imodels.Playlist{}
		var description sql.NullString

		err := rows.Scan(
			&playlist.ID,
			&playlist.OwnerUserID,
			&playlist.Name,
			&description,
			&playlist.IsDefault,
			&playlist.CreatedAt,
			&playlist.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan playlist: %w", err)
		}

		if description.Valid {
			playlist.Description = &description.String
		}

		playlists = append(playlists, playlist)
	}

	// Get total count
	countQuery := `SELECT COUNT(*) FROM playlists WHERE owner_user_id = ?`
	var total int
	err = r.db.QueryRow(countQuery, userID).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to get playlist count: %w", err)
	}

	hasNext := offset+limit < total

	return &imodels.PlaylistList{
		Playlists: playlists,
		Total:     total,
		Limit:     limit,
		Offset:    offset,
		HasNext:   hasNext,
	}, nil
}

// GetPlaylist returns a specific playlist by ID
func (r *Repository) GetPlaylist(playlistID string) (*imodels.Playlist, error) {
	query := `
		SELECT id, owner_user_id, name, description, is_default, created_at, updated_at
		FROM playlists
		WHERE id = ?
	`

	playlist := &imodels.Playlist{}
	var description sql.NullString

	err := r.db.QueryRow(query, playlistID).Scan(
		&playlist.ID,
		&playlist.OwnerUserID,
		&playlist.Name,
		&description,
		&playlist.IsDefault,
		&playlist.CreatedAt,
		&playlist.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("playlist not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get playlist: %w", err)
	}

	if description.Valid {
		playlist.Description = &description.String
	}

	return playlist, nil
}

// UpdatePlaylist updates a playlist's information
func (r *Repository) UpdatePlaylist(playlistID string, req *imodels.UpdatePlaylistRequest) error {
	query := `
		UPDATE playlists
		SET name = ?, description = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`

	var description interface{}
	if req.Description != nil {
		description = *req.Description
	} else {
		description = nil
	}

	result, err := r.db.Exec(query, req.Name, description, playlistID)
	if err != nil {
		return fmt.Errorf("failed to update playlist: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("playlist not found")
	}

	return nil
}

// DeletePlaylist deletes a playlist and all its track associations
func (r *Repository) DeletePlaylist(playlistID string) error {
	// Start transaction
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Delete playlist-track associations first (cascade will handle this, but being explicit)
	_, err = tx.Exec("DELETE FROM playlist_tracks WHERE playlist_id = ?", playlistID)
	if err != nil {
		return fmt.Errorf("failed to delete playlist tracks: %w", err)
	}

	// Delete the playlist
	result, err := tx.Exec("DELETE FROM playlists WHERE id = ?", playlistID)
	if err != nil {
		return fmt.Errorf("failed to delete playlist: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("playlist not found")
	}

	return tx.Commit()
}

// CreateDefaultPlaylist creates the default "Unsorted" playlist for a user
func (r *Repository) CreateDefaultPlaylist(ownerUserID string) (*imodels.Playlist, error) {
	id := uuid.New().String()
	name := "Unsorted"
	description := "Tracks not assigned to any playlist"

	query := `
		INSERT INTO playlists (id, owner_user_id, name, description, is_default)
		VALUES (?, ?, ?, ?, ?)
	`

	_, err := r.db.Exec(query, id, ownerUserID, name, description, true)
	if err != nil {
		return nil, fmt.Errorf("failed to create default playlist: %w", err)
	}

	return r.GetPlaylist(id)
}

// GetDefaultPlaylist returns the default playlist for a user
func (r *Repository) GetDefaultPlaylist(userID string) (*imodels.Playlist, error) {
	query := `
		SELECT id, owner_user_id, name, description, is_default, created_at, updated_at
		FROM playlists
		WHERE owner_user_id = ? AND is_default = true
	`

	playlist := &imodels.Playlist{}
	var description sql.NullString

	err := r.db.QueryRow(query, userID).Scan(
		&playlist.ID,
		&playlist.OwnerUserID,
		&playlist.Name,
		&description,
		&playlist.IsDefault,
		&playlist.CreatedAt,
		&playlist.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("default playlist not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get default playlist: %w", err)
	}

	if description.Valid {
		playlist.Description = &description.String
	}

	return playlist, nil
}

// AddTracksToPlaylist adds multiple tracks to a playlist
func (r *Repository) AddTracksToPlaylist(playlistID string, trackIDs []string) error {
	if len(trackIDs) == 0 {
		return nil
	}

	// Start transaction
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Prepare the insert statement
	stmt, err := tx.Prepare(`
		INSERT OR IGNORE INTO playlist_tracks (id, playlist_id, track_id)
		VALUES (?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	// Insert each track
	for _, trackID := range trackIDs {
		id := uuid.New().String()
		_, err = stmt.Exec(id, playlistID, trackID)
		if err != nil {
			return fmt.Errorf("failed to add track %s to playlist: %w", trackID, err)
		}
	}

	return tx.Commit()
}

// RemoveTracksFromPlaylist removes multiple tracks from a playlist
func (r *Repository) RemoveTracksFromPlaylist(playlistID string, trackIDs []string) error {
	if len(trackIDs) == 0 {
		return nil
	}

	// Create placeholders for the IN clause
	placeholders := make([]string, len(trackIDs))
	args := make([]interface{}, len(trackIDs)+1)
	args[0] = playlistID

	for i, trackID := range trackIDs {
		placeholders[i] = "?"
		args[i+1] = trackID
	}

	query := fmt.Sprintf(`
		DELETE FROM playlist_tracks
		WHERE playlist_id = ? AND track_id IN (%s)
	`, fmt.Sprintf("%s", placeholders[0], placeholders[1:]))

	result, err := r.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to remove tracks from playlist: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no tracks found to remove")
	}

	return nil
}

// GetPlaylistTracks returns tracks for a specific playlist with pagination
func (r *Repository) GetPlaylistTracks(playlistID string, limit, offset int) (*imodels.PlaylistWithTracks, error) {
	// First get the playlist
	playlist, err := r.GetPlaylist(playlistID)
	if err != nil {
		return nil, err
	}

	// Get tracks for this playlist
	query := `
		SELECT t.id, t.owner_user_id, t.original_filename, t.content_type, t.size_bytes,
		       t.duration_seconds, t.title, t.artist, t.album, t.genre, t.year,
		       t.sample_rate, t.bitrate, t.file_path, t.created_at, t.updated_at,
		       pt.added_at
		FROM tracks t
		INNER JOIN playlist_tracks pt ON t.id = pt.track_id
		WHERE pt.playlist_id = ?
		ORDER BY pt.added_at DESC
		LIMIT ? OFFSET ?
	`

	rows, err := r.db.Query(query, playlistID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get playlist tracks: %w", err)
	}
	defer rows.Close()

	var tracks []*imodels.Track
	for rows.Next() {
		var track imodels.Track
		err := rows.Scan(
			&track.ID,
			&track.OwnerUserID,
			&track.OriginalFilename,
			&track.ContentType,
			&track.SizeBytes,
			&track.DurationSeconds,
			&track.Title,
			&track.Artist,
			&track.Album,
			&track.Genre,
			&track.Year,
			&track.SampleRate,
			&track.Bitrate,
			&track.FilePath,
			&track.CreatedAt,
			&track.UpdatedAt,
			&track.CreatedAt, // We'll reuse this field for added_at
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan track: %w", err)
		}

		tracks = append(tracks, &track)
	}

	// Get total count
	countQuery := `SELECT COUNT(*) FROM playlist_tracks WHERE playlist_id = ?`
	var total int
	err = r.db.QueryRow(countQuery, playlistID).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to get playlist track count: %w", err)
	}

	hasNext := offset+limit < total

	return &imodels.PlaylistWithTracks{
		Playlist: playlist,
		Tracks:   tracks,
		Total:    total,
		Limit:    limit,
		Offset:   offset,
		HasNext:  hasNext,
	}, nil
}

// GetTracksNotInPlaylist returns tracks that are not in any playlist for a user
func (r *Repository) GetTracksNotInPlaylist(userID string, limit, offset int) (*imodels.TrackList, error) {
	query := `
		SELECT t.id, t.owner_user_id, t.original_filename, t.content_type, t.size_bytes,
		       t.duration_seconds, t.title, t.artist, t.album, t.genre, t.year,
		       t.sample_rate, t.bitrate, t.file_path, t.created_at, t.updated_at
		FROM tracks t
		WHERE t.owner_user_id = ?
		AND t.id NOT IN (
			SELECT pt.track_id FROM playlist_tracks pt
			INNER JOIN playlists p ON pt.playlist_id = p.id
			WHERE p.owner_user_id = ?
		)
		ORDER BY t.created_at DESC
		LIMIT ? OFFSET ?
	`

	rows, err := r.db.Query(query, userID, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get tracks not in playlist: %w", err)
	}
	defer rows.Close()

	var tracks []*imodels.Track
	for rows.Next() {
		var track imodels.Track
		err := rows.Scan(
			&track.ID,
			&track.OwnerUserID,
			&track.OriginalFilename,
			&track.ContentType,
			&track.SizeBytes,
			&track.DurationSeconds,
			&track.Title,
			&track.Artist,
			&track.Album,
			&track.Genre,
			&track.Year,
			&track.SampleRate,
			&track.Bitrate,
			&track.FilePath,
			&track.CreatedAt,
			&track.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan track: %w", err)
		}

		tracks = append(tracks, &track)
	}

	// Get total count
	countQuery := `
		SELECT COUNT(*) FROM tracks t
		WHERE t.owner_user_id = ?
		AND t.id NOT IN (
			SELECT pt.track_id FROM playlist_tracks pt
			INNER JOIN playlists p ON pt.playlist_id = p.id
			WHERE p.owner_user_id = ?
		)
	`
	var total int
	err = r.db.QueryRow(countQuery, userID, userID).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to get track count: %w", err)
	}

	hasNext := offset+limit < total

	return &imodels.TrackList{
		Tracks:  tracks,
		Total:   total,
		Limit:   limit,
		Offset:  offset,
		HasNext: hasNext,
	}, nil
}
