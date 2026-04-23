package spotify

import (
	"context"
	"database/sql"
	"time"

	"github.com/faraz525/home-music-server/backend/internal/db"
	"github.com/faraz525/home-music-server/backend/utils"
)

type Repository struct {
	db *db.DB
}

func NewRepository(db *db.DB) *Repository {
	return &Repository{db: db}
}

type SyncConfig struct {
	ID                   int        `json:"id"`
	AccessToken          *string    `json:"access_token,omitempty"`
	RefreshToken         *string    `json:"refresh_token,omitempty"`
	TokenExpiresAt       *time.Time `json:"token_expires_at,omitempty"`
	OwnerUserID          string     `json:"owner_user_id"`
	LikedSongsPlaylistID *string    `json:"liked_songs_playlist_id"`
	Enabled              bool       `json:"enabled"`
	PlaylistPattern      *string    `json:"playlist_pattern,omitempty"`
	LastSyncAt           *time.Time `json:"last_sync_at"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

func (r *Repository) GetSyncConfig(ctx context.Context) (*SyncConfig, error) {
	var cfg SyncConfig
	var accessToken, refreshToken, playlistID, playlistPattern sql.NullString
	var tokenExpiresAt, lastSyncAt sql.NullTime

	err := r.db.QueryRowContext(ctx,
		`SELECT id, access_token, refresh_token, token_expires_at, owner_user_id,
                liked_songs_playlist_id, enabled, playlist_pattern, last_sync_at, created_at, updated_at
         FROM spotify_sync_config WHERE id = 1`,
	).Scan(&cfg.ID, &accessToken, &refreshToken, &tokenExpiresAt, &cfg.OwnerUserID,
		&playlistID, &cfg.Enabled, &playlistPattern, &lastSyncAt, &cfg.CreatedAt, &cfg.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if accessToken.Valid {
		cfg.AccessToken = &accessToken.String
	}
	if refreshToken.Valid {
		cfg.RefreshToken = &refreshToken.String
	}
	if tokenExpiresAt.Valid {
		cfg.TokenExpiresAt = &tokenExpiresAt.Time
	}
	if playlistID.Valid {
		cfg.LikedSongsPlaylistID = &playlistID.String
	}
	if playlistPattern.Valid {
		cfg.PlaylistPattern = &playlistPattern.String
	}
	if lastSyncAt.Valid {
		cfg.LastSyncAt = &lastSyncAt.Time
	}

	return &cfg, nil
}

func (r *Repository) UpsertSyncConfig(ctx context.Context, cfg *SyncConfig) error {
	now := time.Now()

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO spotify_sync_config (id, access_token, refresh_token, token_expires_at, owner_user_id,
                                          liked_songs_playlist_id, enabled, playlist_pattern, created_at, updated_at)
         VALUES (1, ?, ?, ?, ?, ?, ?, ?, ?, ?)
         ON CONFLICT(id) DO UPDATE SET
            access_token = excluded.access_token,
            refresh_token = excluded.refresh_token,
            token_expires_at = excluded.token_expires_at,
            owner_user_id = excluded.owner_user_id,
            liked_songs_playlist_id = excluded.liked_songs_playlist_id,
            enabled = excluded.enabled,
            playlist_pattern = excluded.playlist_pattern,
            updated_at = excluded.updated_at`,
		cfg.AccessToken, cfg.RefreshToken, cfg.TokenExpiresAt, cfg.OwnerUserID,
		cfg.LikedSongsPlaylistID, cfg.Enabled, cfg.PlaylistPattern, now, now,
	)
	return err
}

func (r *Repository) GetSyncedTrackID(ctx context.Context, spotifyID string) (string, error) {
	var trackID string
	err := r.db.QueryRowContext(ctx,
		`SELECT track_id FROM spotify_synced_tracks WHERE spotify_id = ?`,
		spotifyID,
	).Scan(&trackID)
	return trackID, err
}

func (r *Repository) UpdateTokens(ctx context.Context, accessToken, refreshToken string, expiresAt time.Time) error {
	now := time.Now()
	_, err := r.db.ExecContext(ctx,
		`UPDATE spotify_sync_config SET access_token = ?, refresh_token = ?, token_expires_at = ?, updated_at = ? WHERE id = 1`,
		accessToken, refreshToken, expiresAt, now,
	)
	return err
}

func (r *Repository) UpdateLastSync(ctx context.Context) error {
	now := time.Now()
	_, err := r.db.ExecContext(ctx,
		`UPDATE spotify_sync_config SET last_sync_at = ?, updated_at = ? WHERE id = 1`,
		now, now,
	)
	return err
}

func (r *Repository) UpdateLikedSongsPlaylistID(ctx context.Context, playlistID string) error {
	now := time.Now()
	_, err := r.db.ExecContext(ctx,
		`UPDATE spotify_sync_config SET liked_songs_playlist_id = ?, updated_at = ? WHERE id = 1`,
		playlistID, now,
	)
	return err
}

func (r *Repository) RecordSync(ctx context.Context, started time.Time, tracksAdded, tracksSkipped int, errMsg *string) error {
	id := utils.GenerateID("sync")
	now := time.Now()

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO spotify_sync_history (id, sync_started_at, sync_completed_at, tracks_added, tracks_skipped, error_message)
         VALUES (?, ?, ?, ?, ?, ?)`,
		id, started, now, tracksAdded, tracksSkipped, errMsg,
	)
	return err
}

func (r *Repository) GetSyncHistory(ctx context.Context, limit int) ([]*SyncHistory, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, sync_started_at, sync_completed_at, tracks_added, tracks_skipped, error_message
         FROM spotify_sync_history ORDER BY sync_started_at DESC LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []*SyncHistory
	for rows.Next() {
		var h SyncHistory
		var completedAt sql.NullTime
		var errMsg sql.NullString

		err := rows.Scan(&h.ID, &h.StartedAt, &completedAt, &h.TracksAdded, &h.TracksSkipped, &errMsg)
		if err != nil {
			return nil, err
		}

		if completedAt.Valid {
			h.CompletedAt = &completedAt.Time
		}
		if errMsg.Valid {
			h.ErrorMessage = &errMsg.String
		}

		history = append(history, &h)
	}

	return history, rows.Err()
}

type SyncHistory struct {
	ID            string     `json:"id"`
	StartedAt     time.Time  `json:"started_at"`
	CompletedAt   *time.Time `json:"completed_at,omitempty"`
	TracksAdded   int        `json:"tracks_added"`
	TracksSkipped int        `json:"tracks_skipped"`
	ErrorMessage  *string    `json:"error_message,omitempty"`
}

func (r *Repository) IsSpotifyTrackSynced(ctx context.Context, spotifyID string) (bool, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM spotify_synced_tracks WHERE spotify_id = ?`,
		spotifyID,
	).Scan(&count)
	return count > 0, err
}

func (r *Repository) RecordSyncedTrack(ctx context.Context, spotifyID, trackID string) error {
	id := utils.GenerateID("spt")
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO spotify_synced_tracks (id, spotify_id, track_id, synced_at)
         VALUES (?, ?, ?, ?)`,
		id, spotifyID, trackID, time.Now(),
	)
	return err
}

// Synced playlist management
type SyncedPlaylist struct {
	ID                string     `json:"id"`
	SpotifyPlaylistID string     `json:"spotify_playlist_id"`
	LocalPlaylistID   string     `json:"local_playlist_id"`
	Name              string     `json:"name"`
	Enabled           bool       `json:"enabled"`
	LastSyncAt        *time.Time `json:"last_sync_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
}

func (r *Repository) GetSyncedPlaylists(ctx context.Context) ([]*SyncedPlaylist, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, spotify_playlist_id, local_playlist_id, name, enabled, last_sync_at, created_at
         FROM spotify_synced_playlists ORDER BY name ASC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var playlists []*SyncedPlaylist
	for rows.Next() {
		var p SyncedPlaylist
		var lastSyncAt sql.NullTime

		err := rows.Scan(&p.ID, &p.SpotifyPlaylistID, &p.LocalPlaylistID, &p.Name, &p.Enabled, &lastSyncAt, &p.CreatedAt)
		if err != nil {
			return nil, err
		}

		if lastSyncAt.Valid {
			p.LastSyncAt = &lastSyncAt.Time
		}

		playlists = append(playlists, &p)
	}

	return playlists, rows.Err()
}

func (r *Repository) AddSyncedPlaylist(ctx context.Context, spotifyID, localID, name string) error {
	id := utils.GenerateID("spp")
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO spotify_synced_playlists (id, spotify_playlist_id, local_playlist_id, name, enabled, created_at)
         VALUES (?, ?, ?, ?, TRUE, ?)
         ON CONFLICT(spotify_playlist_id) DO UPDATE SET name = excluded.name`,
		id, spotifyID, localID, name, time.Now(),
	)
	return err
}

func (r *Repository) UpdateSyncedPlaylistLastSync(ctx context.Context, spotifyID string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE spotify_synced_playlists SET last_sync_at = ? WHERE spotify_playlist_id = ?`,
		time.Now(), spotifyID,
	)
	return err
}

func (r *Repository) SetSyncedPlaylistEnabled(ctx context.Context, spotifyID string, enabled bool) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE spotify_synced_playlists SET enabled = ? WHERE spotify_playlist_id = ?`,
		enabled, spotifyID,
	)
	return err
}

func (r *Repository) DeleteSyncedPlaylist(ctx context.Context, spotifyID string) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM spotify_synced_playlists WHERE spotify_playlist_id = ?`,
		spotifyID,
	)
	return err
}
