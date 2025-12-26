-- CrateDrop v1 SQLite Schema (Optimized for Raspberry Pi)
-- Run this with: sqlite3 cratedrop.sqlite < schema.sql

-- Performance optimizations for Raspberry Pi with SSD
PRAGMA foreign_keys = ON;
PRAGMA journal_mode = WAL;
PRAGMA synchronous = NORMAL;
PRAGMA cache_size = -64000;        -- 64MB cache (adjust based on available RAM)
PRAGMA temp_store = MEMORY;        -- Keep temp tables in memory
PRAGMA mmap_size = 268435456;      -- 256MB memory-mapped I/O for faster reads
PRAGMA page_size = 4096;           -- Match typical SSD block size

-- Users table
CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT 'user' CHECK (role IN ('admin', 'user')),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Tracks table
CREATE TABLE IF NOT EXISTS tracks (
    id TEXT PRIMARY KEY,
    owner_user_id TEXT NOT NULL,
    original_filename TEXT NOT NULL,
    content_type TEXT NOT NULL,
    size_bytes INTEGER NOT NULL,
    duration_seconds REAL,
    title TEXT,
    artist TEXT,
    album TEXT,
    genre TEXT,
    year INTEGER,
    sample_rate INTEGER,
    bitrate INTEGER,
    file_path TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (owner_user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Refresh tokens table
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    token_hash TEXT NOT NULL,
    expires_at DATETIME NOT NULL,
    revoked_at DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Playlists table
CREATE TABLE IF NOT EXISTS playlists (
    id TEXT PRIMARY KEY,
    owner_user_id TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    is_public BOOLEAN NOT NULL DEFAULT TRUE,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (owner_user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Playlist tracks table (junction table)
CREATE TABLE IF NOT EXISTS playlist_tracks (
    id TEXT PRIMARY KEY,
    playlist_id TEXT NOT NULL,
    track_id TEXT NOT NULL,
    position INTEGER NOT NULL DEFAULT 0,
    added_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (playlist_id) REFERENCES playlists(id) ON DELETE CASCADE,
    FOREIGN KEY (track_id) REFERENCES tracks(id) ON DELETE CASCADE,
    UNIQUE(playlist_id, track_id)
);

-- ============================================================================
-- OPTIMIZED INDEXES for Raspberry Pi Performance
-- ============================================================================

-- User indexes
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);

-- Track indexes (composite indexes for common query patterns)
CREATE INDEX IF NOT EXISTS idx_tracks_owner_created ON tracks(owner_user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_tracks_created_at ON tracks(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_tracks_genre ON tracks(genre);

-- Playlist indexes
CREATE INDEX IF NOT EXISTS idx_playlists_owner_default ON playlists(owner_user_id, is_default);
CREATE INDEX IF NOT EXISTS idx_playlists_owner_created ON playlists(owner_user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_playlists_is_public ON playlists(is_public);
CREATE INDEX IF NOT EXISTS idx_playlists_public_not_default ON playlists(is_public, is_default);

-- Playlist tracks indexes
CREATE INDEX IF NOT EXISTS idx_playlist_tracks_playlist_position ON playlist_tracks(playlist_id, position);
CREATE INDEX IF NOT EXISTS idx_playlist_tracks_track ON playlist_tracks(track_id);

-- Refresh token indexes
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user ON refresh_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_cleanup ON refresh_tokens(expires_at, revoked_at);

-- ============================================================================
-- FULL-TEXT SEARCH (FTS5) - Much faster than LIKE queries on Pi
-- ============================================================================

-- Virtual table for full-text search on tracks
-- Using external content (separate storage) for reliability
CREATE VIRTUAL TABLE IF NOT EXISTS tracks_fts USING fts5(
    track_id UNINDEXED,
    title,
    artist,
    album,
    genre,
    original_filename
);

-- ============================================================================
-- TRIGGERS - Auto-update timestamps and FTS index
-- ============================================================================

-- Update timestamps on user changes
CREATE TRIGGER IF NOT EXISTS update_users_updated_at
    AFTER UPDATE ON users
    FOR EACH ROW
    WHEN NEW.updated_at = OLD.updated_at
BEGIN
    UPDATE users SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

-- Update timestamps on track changes
CREATE TRIGGER IF NOT EXISTS update_tracks_updated_at
    AFTER UPDATE ON tracks
    FOR EACH ROW
    WHEN NEW.updated_at = OLD.updated_at
BEGIN
    UPDATE tracks SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

-- Update timestamps on playlist changes
CREATE TRIGGER IF NOT EXISTS update_playlists_updated_at
    AFTER UPDATE ON playlists
    FOR EACH ROW
    WHEN NEW.updated_at = OLD.updated_at
BEGIN
    UPDATE playlists SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

-- Keep FTS5 index in sync with tracks table (external content mode)
CREATE TRIGGER IF NOT EXISTS tracks_fts_insert
    AFTER INSERT ON tracks
BEGIN
    INSERT INTO tracks_fts(track_id, title, artist, album, genre, original_filename)
    VALUES (NEW.id, NEW.title, NEW.artist, NEW.album, NEW.genre, NEW.original_filename);
END;

CREATE TRIGGER IF NOT EXISTS tracks_fts_update
    AFTER UPDATE ON tracks
BEGIN
    DELETE FROM tracks_fts WHERE track_id = OLD.id;
    INSERT INTO tracks_fts(track_id, title, artist, album, genre, original_filename)
    VALUES (NEW.id, NEW.title, NEW.artist, NEW.album, NEW.genre, NEW.original_filename);
END;

CREATE TRIGGER IF NOT EXISTS tracks_fts_delete
    AFTER DELETE ON tracks
BEGIN
    DELETE FROM tracks_fts WHERE track_id = OLD.id;
END;

-- ============================================================================
-- DATA INTEGRITY TRIGGERS
-- ============================================================================

-- Prevent multiple default playlists per user
CREATE TRIGGER IF NOT EXISTS prevent_multiple_defaults_insert
    BEFORE INSERT ON playlists
    WHEN NEW.is_default = TRUE
BEGIN
    SELECT CASE
        WHEN EXISTS(
            SELECT 1 FROM playlists 
            WHERE owner_user_id = NEW.owner_user_id 
            AND is_default = TRUE
        )
        THEN RAISE(ABORT, 'User already has a default playlist')
    END;
END;

CREATE TRIGGER IF NOT EXISTS prevent_multiple_defaults_update
    BEFORE UPDATE ON playlists
    WHEN NEW.is_default = TRUE AND OLD.is_default = FALSE
BEGIN
    SELECT CASE
        WHEN EXISTS(
            SELECT 1 FROM playlists 
            WHERE owner_user_id = NEW.owner_user_id 
            AND is_default = TRUE
            AND id != NEW.id
        )
        THEN RAISE(ABORT, 'User already has a default playlist')
    END;
END;

-- ============================================================================
-- MAINTENANCE VIEWS (Optional - for monitoring)
-- ============================================================================

-- View to check database health
CREATE VIEW IF NOT EXISTS db_stats AS
SELECT 
    (SELECT COUNT(*) FROM users) as total_users,
    (SELECT COUNT(*) FROM tracks) as total_tracks,
    (SELECT COUNT(*) FROM playlists) as total_playlists,
    (SELECT COUNT(*) FROM playlist_tracks) as total_playlist_tracks,
    (SELECT COUNT(*) FROM refresh_tokens WHERE revoked_at IS NULL AND expires_at > datetime('now')) as active_tokens,
    (SELECT SUM(size_bytes) FROM tracks) as total_storage_bytes,
    (SELECT ROUND(SUM(size_bytes) / 1024.0 / 1024.0 / 1024.0, 2) FROM tracks) as total_storage_gb;

