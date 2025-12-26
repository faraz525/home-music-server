-- Migration: Add is_public column to playlists table
-- This handles existing databases that were created before the community feature

-- Add is_public column if it doesn't exist
-- SQLite doesn't support IF NOT EXISTS for ALTER TABLE, so we'll handle this gracefully
ALTER TABLE playlists ADD COLUMN is_public BOOLEAN NOT NULL DEFAULT TRUE;

-- Set unsorted crates (is_default=TRUE) to private
UPDATE playlists SET is_public = FALSE WHERE is_default = TRUE;
