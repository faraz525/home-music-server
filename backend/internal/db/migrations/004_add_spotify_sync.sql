-- Spotify Sync Configuration (Singleton table for admin)
CREATE TABLE IF NOT EXISTS spotify_sync_config (
    id INTEGER PRIMARY KEY DEFAULT 1,
    access_token TEXT,
    refresh_token TEXT,
    token_expires_at DATETIME,
    owner_user_id TEXT NOT NULL,
    liked_songs_playlist_id TEXT,
    enabled BOOLEAN NOT NULL DEFAULT FALSE,
    last_sync_at DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (owner_user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (liked_songs_playlist_id) REFERENCES playlists(id) ON DELETE SET NULL,
    CHECK (id = 1)
);

-- Sync history for tracking sync runs
CREATE TABLE IF NOT EXISTS spotify_sync_history (
    id TEXT PRIMARY KEY,
    sync_started_at DATETIME NOT NULL,
    sync_completed_at DATETIME,
    tracks_added INTEGER DEFAULT 0,
    tracks_skipped INTEGER DEFAULT 0,
    error_message TEXT
);

CREATE INDEX IF NOT EXISTS idx_spotify_sync_history_started
    ON spotify_sync_history(sync_started_at DESC);

-- Track external IDs to avoid re-downloading
CREATE TABLE IF NOT EXISTS spotify_synced_tracks (
    id TEXT PRIMARY KEY,
    spotify_id TEXT NOT NULL UNIQUE,
    track_id TEXT NOT NULL,
    synced_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (track_id) REFERENCES tracks(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_spotify_synced_tracks_sp_id
    ON spotify_synced_tracks(spotify_id);

-- Spotify playlists that are being synced
CREATE TABLE IF NOT EXISTS spotify_synced_playlists (
    id TEXT PRIMARY KEY,
    spotify_playlist_id TEXT NOT NULL UNIQUE,
    local_playlist_id TEXT NOT NULL,
    name TEXT NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    last_sync_at DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (local_playlist_id) REFERENCES playlists(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_spotify_synced_playlists_sp_id
    ON spotify_synced_playlists(spotify_playlist_id);
