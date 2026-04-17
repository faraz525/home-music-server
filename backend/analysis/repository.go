package analysis

import (
	"context"
	"database/sql"
	"time"
)

const maxRetries = 3

// backoffFor returns the time until the next retry attempt given how many
// failures have already been recorded (1-based).
func backoffFor(retryCount int) time.Duration {
	switch retryCount {
	case 1:
		return 10 * time.Minute
	case 2:
		return 1 * time.Hour
	default:
		return 6 * time.Hour
	}
}

// ClaimedTrack holds the minimum info needed to run analysis.
type ClaimedTrack struct {
	ID       string
	FilePath string
}

// Repository reads/writes analysis fields on the tracks table. It accepts a
// *sql.DB directly (not the project's *db.DB wrapper) so tests can use an
// in-memory SQLite.
type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// ClaimNextPending returns the next track eligible for analysis, or nil if
// none. Eligibility: analysis_status='pending' AND next_retry_at is null or
// in the past. Sorted by upload order so backfill drains oldest first.
func (r *Repository) ClaimNextPending(ctx context.Context) (*ClaimedTrack, error) {
	row := r.db.QueryRowContext(ctx, `
        SELECT id, file_path
        FROM tracks
        WHERE analysis_status = 'pending'
          AND (next_retry_at IS NULL OR next_retry_at <= datetime('now'))
        ORDER BY created_at ASC
        LIMIT 1
    `)
	var t ClaimedTrack
	err := row.Scan(&t.ID, &t.FilePath)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// MarkAnalyzed writes a successful result and flips status to 'analyzed'.
// If the key is empty (unknown), musical_key is set NULL but the row is still
// considered analyzed.
func (r *Repository) MarkAnalyzed(ctx context.Context, id string, res Result) error {
	var key interface{}
	if res.Key != "" {
		key = res.Key
	}
	_, err := r.db.ExecContext(ctx, `
        UPDATE tracks
        SET bpm = ?, bpm_confidence = ?, musical_key = ?, key_confidence = ?,
            analyzed_at = datetime('now'),
            analysis_status = 'analyzed',
            analysis_error = NULL,
            next_retry_at = NULL,
            updated_at = datetime('now')
        WHERE id = ?
    `, res.BPM, res.BPMConfidence, key, res.KeyConfidence, id)
	return err
}

// RecordFailure increments retry_count and schedules the next attempt. After
// maxRetries failures the status flips to 'failed' (terminal). Skips no-ops on
// rows that are already terminal ('failed') or user-overridden ('user_edited')
// so a late error from a cancelled analysis can't resurrect a closed track.
func (r *Repository) RecordFailure(ctx context.Context, id, errMsg string, now time.Time) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var current int
	var status string
	err = tx.QueryRowContext(ctx, `SELECT analysis_status, analysis_retry_count FROM tracks WHERE id=?`, id).Scan(&status, &current)
	if err != nil {
		return err
	}
	if status == "failed" || status == "user_edited" {
		return tx.Commit()
	}
	next := current + 1
	if next >= maxRetries {
		_, err = tx.ExecContext(ctx, `
            UPDATE tracks
            SET analysis_status = 'failed',
                analysis_retry_count = ?,
                analysis_error = ?,
                next_retry_at = NULL,
                updated_at = datetime('now')
            WHERE id = ?
        `, next, errMsg, id)
	} else {
		retryAt := now.Add(backoffFor(next))
		_, err = tx.ExecContext(ctx, `
            UPDATE tracks
            SET analysis_status = 'pending',
                analysis_retry_count = ?,
                analysis_error = ?,
                next_retry_at = ?,
                updated_at = datetime('now')
            WHERE id = ?
        `, next, errMsg, retryAt, id)
	}
	if err != nil {
		return err
	}
	return tx.Commit()
}

// RecordTerminalFailure flips a track straight to 'failed' with no retries.
// Use for non-recoverable errors (e.g. file missing on disk).
func (r *Repository) RecordTerminalFailure(ctx context.Context, id, errMsg string) error {
	_, err := r.db.ExecContext(ctx, `
        UPDATE tracks
        SET analysis_status = 'failed',
            analysis_error = ?,
            next_retry_at = NULL,
            updated_at = datetime('now')
        WHERE id = ?
    `, errMsg, id)
	return err
}
