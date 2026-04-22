-- Track analysis fields for BPM + Musical Key detection
ALTER TABLE tracks ADD COLUMN bpm REAL;
ALTER TABLE tracks ADD COLUMN bpm_confidence REAL;
ALTER TABLE tracks ADD COLUMN musical_key TEXT;              -- Camelot notation, e.g. "8A"
ALTER TABLE tracks ADD COLUMN key_confidence REAL;
ALTER TABLE tracks ADD COLUMN analyzed_at DATETIME;
ALTER TABLE tracks ADD COLUMN analysis_status TEXT NOT NULL DEFAULT 'pending';
ALTER TABLE tracks ADD COLUMN analysis_error TEXT;
ALTER TABLE tracks ADD COLUMN analysis_retry_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE tracks ADD COLUMN next_retry_at DATETIME;

CREATE INDEX IF NOT EXISTS idx_tracks_analysis_status
    ON tracks(analysis_status, next_retry_at);
