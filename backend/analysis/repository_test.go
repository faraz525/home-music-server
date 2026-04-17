package analysis

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// newTestDB sets up an in-memory SQLite with just enough schema for the repo.
func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	_, err = db.Exec(`
        CREATE TABLE tracks (
            id TEXT PRIMARY KEY,
            owner_user_id TEXT NOT NULL DEFAULT 'u1',
            original_filename TEXT NOT NULL DEFAULT 'f.wav',
            content_type TEXT NOT NULL DEFAULT 'audio/wav',
            size_bytes INTEGER NOT NULL DEFAULT 0,
            file_path TEXT NOT NULL DEFAULT '/x',
            created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
            updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
            bpm REAL,
            bpm_confidence REAL,
            musical_key TEXT,
            key_confidence REAL,
            analyzed_at DATETIME,
            analysis_status TEXT NOT NULL DEFAULT 'pending',
            analysis_error TEXT,
            analysis_retry_count INTEGER NOT NULL DEFAULT 0,
            next_retry_at DATETIME
        );
    `)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func seedPending(t *testing.T, db *sql.DB, id, filePath string) {
	t.Helper()
	_, err := db.Exec(`INSERT INTO tracks (id, file_path) VALUES (?, ?)`, id, filePath)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
}

func TestRepository_ClaimNextPending_ReturnsPending(t *testing.T) {
	db := newTestDB(t)
	seedPending(t, db, "t1", "/audio/t1.wav")

	repo := NewRepository(db)
	got, err := repo.ClaimNextPending(context.Background())
	if err != nil {
		t.Fatalf("ClaimNextPending: %v", err)
	}
	if got == nil || got.ID != "t1" {
		t.Fatalf("got %v, want track t1", got)
	}
	if got.FilePath != "/audio/t1.wav" {
		t.Errorf("FilePath = %q", got.FilePath)
	}
}

func TestRepository_ClaimNextPending_NoneAvailable(t *testing.T) {
	db := newTestDB(t)
	repo := NewRepository(db)
	got, err := repo.ClaimNextPending(context.Background())
	if err != nil {
		t.Fatalf("ClaimNextPending: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
}

func TestRepository_ClaimNextPending_SkipsFutureRetry(t *testing.T) {
	db := newTestDB(t)
	seedPending(t, db, "t1", "/a.wav")
	// Push t1's next_retry_at into the future
	_, _ = db.Exec(`UPDATE tracks SET next_retry_at = datetime('now', '+1 hour') WHERE id = 't1'`)

	repo := NewRepository(db)
	got, err := repo.ClaimNextPending(context.Background())
	if err != nil {
		t.Fatalf("ClaimNextPending: %v", err)
	}
	if got != nil {
		t.Fatalf("should skip track with future next_retry_at, got %v", got)
	}
}

func TestRepository_MarkAnalyzed_UpdatesFields(t *testing.T) {
	db := newTestDB(t)
	seedPending(t, db, "t1", "/a.wav")
	repo := NewRepository(db)

	camKey := "8A"
	err := repo.MarkAnalyzed(context.Background(), "t1", Result{
		BPM: 128.0, BPMConfidence: 0.8, Key: camKey, KeyConfidence: 0.78,
	})
	if err != nil {
		t.Fatalf("MarkAnalyzed: %v", err)
	}

	var status string
	var bpm sql.NullFloat64
	var key sql.NullString
	err = db.QueryRow(`SELECT analysis_status, bpm, musical_key FROM tracks WHERE id='t1'`).Scan(&status, &bpm, &key)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if status != "analyzed" {
		t.Errorf("status = %q", status)
	}
	if !bpm.Valid || bpm.Float64 != 128.0 {
		t.Errorf("bpm = %v", bpm)
	}
	if !key.Valid || key.String != "8A" {
		t.Errorf("key = %v", key)
	}
}

func TestRepository_RecordFailure_IncrementsAndBackoff(t *testing.T) {
	db := newTestDB(t)
	seedPending(t, db, "t1", "/a.wav")
	repo := NewRepository(db)

	// First failure -> still pending, next_retry_at ~10min out
	err := repo.RecordFailure(context.Background(), "t1", "boom", time.Now())
	if err != nil {
		t.Fatalf("RecordFailure: %v", err)
	}
	var status string
	var retryCount int
	var nextRetry sql.NullTime
	_ = db.QueryRow(`SELECT analysis_status, analysis_retry_count, next_retry_at FROM tracks WHERE id='t1'`).Scan(&status, &retryCount, &nextRetry)
	if status != "pending" {
		t.Errorf("status after 1st failure = %q, want pending", status)
	}
	if retryCount != 1 {
		t.Errorf("retry_count = %d, want 1", retryCount)
	}
	if !nextRetry.Valid {
		t.Fatalf("next_retry_at not set")
	}

	// Third failure -> terminal
	_ = repo.RecordFailure(context.Background(), "t1", "boom", time.Now())
	_ = repo.RecordFailure(context.Background(), "t1", "boom", time.Now())

	_ = db.QueryRow(`SELECT analysis_status, analysis_retry_count FROM tracks WHERE id='t1'`).Scan(&status, &retryCount)
	if status != "failed" {
		t.Errorf("status after 3rd failure = %q, want failed", status)
	}
	if retryCount != 3 {
		t.Errorf("retry_count = %d, want 3", retryCount)
	}
}

func TestRepository_RecordTerminalFailure_SetsFailedImmediately(t *testing.T) {
	db := newTestDB(t)
	seedPending(t, db, "t1", "/a.wav")
	repo := NewRepository(db)

	err := repo.RecordTerminalFailure(context.Background(), "t1", "file missing")
	if err != nil {
		t.Fatalf("RecordTerminalFailure: %v", err)
	}
	var status string
	_ = db.QueryRow(`SELECT analysis_status FROM tracks WHERE id='t1'`).Scan(&status)
	if status != "failed" {
		t.Errorf("status = %q, want failed", status)
	}
}
