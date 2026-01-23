package soundcloud

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
	ID          int        `json:"id"`
	OAuthToken  *string    `json:"oauth_token,omitempty"`
	OwnerUserID string     `json:"owner_user_id"`
	PlaylistID  *string    `json:"playlist_id"`
	Enabled     bool       `json:"enabled"`
	LastSyncAt  *time.Time `json:"last_sync_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func (r *Repository) GetSyncConfig(ctx context.Context) (*SyncConfig, error) {
	var cfg SyncConfig
	var oauthToken, playlistID sql.NullString
	var lastSyncAt sql.NullTime

	err := r.db.QueryRowContext(ctx,
		`SELECT id, oauth_token, owner_user_id, playlist_id, enabled, last_sync_at, created_at, updated_at
         FROM soundcloud_sync_config WHERE id = 1`,
	).Scan(&cfg.ID, &oauthToken, &cfg.OwnerUserID, &playlistID, &cfg.Enabled, &lastSyncAt, &cfg.CreatedAt, &cfg.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if oauthToken.Valid {
		cfg.OAuthToken = &oauthToken.String
	}
	if playlistID.Valid {
		cfg.PlaylistID = &playlistID.String
	}
	if lastSyncAt.Valid {
		cfg.LastSyncAt = &lastSyncAt.Time
	}

	return &cfg, nil
}

func (r *Repository) UpsertSyncConfig(ctx context.Context, cfg *SyncConfig) error {
	now := time.Now()

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO soundcloud_sync_config (id, oauth_token, owner_user_id, playlist_id, enabled, created_at, updated_at)
         VALUES (1, ?, ?, ?, ?, ?, ?)
         ON CONFLICT(id) DO UPDATE SET
            oauth_token = excluded.oauth_token,
            owner_user_id = excluded.owner_user_id,
            playlist_id = excluded.playlist_id,
            enabled = excluded.enabled,
            updated_at = excluded.updated_at`,
		cfg.OAuthToken, cfg.OwnerUserID, cfg.PlaylistID, cfg.Enabled, now, now,
	)
	return err
}

func (r *Repository) UpdateLastSync(ctx context.Context) error {
	now := time.Now()
	_, err := r.db.ExecContext(ctx,
		`UPDATE soundcloud_sync_config SET last_sync_at = ?, updated_at = ? WHERE id = 1`,
		now, now,
	)
	return err
}

func (r *Repository) UpdatePlaylistID(ctx context.Context, playlistID string) error {
	now := time.Now()
	_, err := r.db.ExecContext(ctx,
		`UPDATE soundcloud_sync_config SET playlist_id = ?, updated_at = ? WHERE id = 1`,
		playlistID, now,
	)
	return err
}

func (r *Repository) RecordSync(ctx context.Context, started time.Time, tracksAdded, tracksSkipped int, errMsg *string) error {
	id := utils.GenerateID("sync")
	now := time.Now()

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO soundcloud_sync_history (id, sync_started_at, sync_completed_at, tracks_added, tracks_skipped, error_message)
         VALUES (?, ?, ?, ?, ?, ?)`,
		id, started, now, tracksAdded, tracksSkipped, errMsg,
	)
	return err
}

func (r *Repository) GetSyncHistory(ctx context.Context, limit int) ([]*SyncHistory, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, sync_started_at, sync_completed_at, tracks_added, tracks_skipped, error_message
         FROM soundcloud_sync_history ORDER BY sync_started_at DESC LIMIT ?`,
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

func (r *Repository) IsSoundCloudTrackSynced(ctx context.Context, soundcloudID string) (bool, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM soundcloud_synced_tracks WHERE soundcloud_id = ?`,
		soundcloudID,
	).Scan(&count)
	return count > 0, err
}

func (r *Repository) RecordSyncedTrack(ctx context.Context, soundcloudID, trackID string) error {
	id := utils.GenerateID("sct")
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO soundcloud_synced_tracks (id, soundcloud_id, track_id, synced_at)
         VALUES (?, ?, ?, ?)`,
		id, soundcloudID, trackID, time.Now(),
	)
	return err
}
