-- SoundCloud Sync Configuration (Singleton table for admin)
CREATE TABLE IF NOT EXISTS soundcloud_sync_config (
    id INTEGER PRIMARY KEY DEFAULT 1,
    oauth_token TEXT,
    owner_user_id TEXT NOT NULL,
    playlist_id TEXT,
    enabled BOOLEAN NOT NULL DEFAULT FALSE,
    last_sync_at DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (owner_user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (playlist_id) REFERENCES playlists(id) ON DELETE SET NULL,
    CHECK (id = 1)
);

-- Sync history for tracking sync runs
CREATE TABLE IF NOT EXISTS soundcloud_sync_history (
    id TEXT PRIMARY KEY,
    sync_started_at DATETIME NOT NULL,
    sync_completed_at DATETIME,
    tracks_added INTEGER DEFAULT 0,
    tracks_skipped INTEGER DEFAULT 0,
    error_message TEXT
);

CREATE INDEX IF NOT EXISTS idx_soundcloud_sync_history_started
    ON soundcloud_sync_history(sync_started_at DESC);

-- Track external IDs to avoid re-downloading
CREATE TABLE IF NOT EXISTS soundcloud_synced_tracks (
    id TEXT PRIMARY KEY,
    soundcloud_id TEXT NOT NULL UNIQUE,
    track_id TEXT NOT NULL,
    synced_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (track_id) REFERENCES tracks(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_soundcloud_synced_tracks_sc_id
    ON soundcloud_synced_tracks(soundcloud_id);
