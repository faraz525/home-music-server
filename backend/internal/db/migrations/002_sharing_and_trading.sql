-- CrateDrop Sharing & Trading Feature Migration
-- Adds user search, crate privacy, and trading system

-- ============================================================================
-- USER ENHANCEMENTS - Add username for user discovery
-- ============================================================================

ALTER TABLE users ADD COLUMN username TEXT UNIQUE;

-- Index for fast username lookups
CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);

-- ============================================================================
-- CRATE PRIVACY & TRADE CONFIGURATION
-- ============================================================================

-- Add privacy and trade configuration to playlists (crates)
ALTER TABLE playlists ADD COLUMN is_public BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE playlists ADD COLUMN trade_ratio_give INTEGER NOT NULL DEFAULT 1;
ALTER TABLE playlists ADD COLUMN trade_ratio_take INTEGER NOT NULL DEFAULT 1;

-- Index for finding public crates
CREATE INDEX IF NOT EXISTS idx_playlists_public ON playlists(is_public);
CREATE INDEX IF NOT EXISTS idx_playlists_public_owner ON playlists(is_public, owner_user_id);

-- ============================================================================
-- TRACK REFERENCES - For traded songs (references instead of copies)
-- ============================================================================

CREATE TABLE IF NOT EXISTS track_references (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    track_id TEXT NOT NULL,
    source_user_id TEXT NOT NULL,
    acquired_via TEXT NOT NULL DEFAULT 'trade',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (track_id) REFERENCES tracks(id) ON DELETE CASCADE,
    FOREIGN KEY (source_user_id) REFERENCES users(id) ON DELETE CASCADE,
    UNIQUE(user_id, track_id)
);

CREATE INDEX IF NOT EXISTS idx_track_refs_user ON track_references(user_id);
CREATE INDEX IF NOT EXISTS idx_track_refs_track ON track_references(track_id);
CREATE INDEX IF NOT EXISTS idx_track_refs_source ON track_references(source_user_id);

-- ============================================================================
-- TRADE TRANSACTIONS - History of all trades
-- ============================================================================

CREATE TABLE IF NOT EXISTS trade_transactions (
    id TEXT PRIMARY KEY,
    requester_user_id TEXT NOT NULL,
    owner_user_id TEXT NOT NULL,
    crate_id TEXT NOT NULL,
    requested_track_id TEXT NOT NULL,
    given_track_ids TEXT NOT NULL,
    trade_ratio TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (requester_user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (owner_user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (crate_id) REFERENCES playlists(id) ON DELETE CASCADE,
    FOREIGN KEY (requested_track_id) REFERENCES tracks(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_trades_requester ON trade_transactions(requester_user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_trades_owner ON trade_transactions(owner_user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_trades_created ON trade_transactions(created_at DESC);
