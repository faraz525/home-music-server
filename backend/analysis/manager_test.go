package analysis

import (
	"context"
	"database/sql"
	"errors"
	"testing"
)

// fakeAnalyzer returns canned results or errors for tests.
type fakeAnalyzer struct {
	result Result
	err    error
	calls  int
}

func (f *fakeAnalyzer) Analyze(ctx context.Context, path string) (Result, error) {
	f.calls++
	return f.result, f.err
}

func TestManager_ProcessOne_NoPending(t *testing.T) {
	db := newTestDB(t)
	repo := NewRepository(db)
	fa := &fakeAnalyzer{}
	m := NewManager(repo, fa)

	processed, err := m.ProcessOne(context.Background())
	if err != nil {
		t.Fatalf("ProcessOne: %v", err)
	}
	if processed {
		t.Error("expected processed=false when nothing to do")
	}
	if fa.calls != 0 {
		t.Errorf("analyzer called %d times, want 0", fa.calls)
	}
}

func TestManager_ProcessOne_Success(t *testing.T) {
	db := newTestDB(t)
	seedPending(t, db, "t1", "/a.wav")
	repo := NewRepository(db)
	fa := &fakeAnalyzer{result: Result{BPM: 128, BPMConfidence: 0.8, Key: "8A", KeyConfidence: 0.7}}
	m := NewManager(repo, fa)

	processed, err := m.ProcessOne(context.Background())
	if err != nil {
		t.Fatalf("ProcessOne: %v", err)
	}
	if !processed {
		t.Error("expected processed=true")
	}

	var status string
	var bpm sql.NullFloat64
	_ = db.QueryRow(`SELECT analysis_status, bpm FROM tracks WHERE id='t1'`).Scan(&status, &bpm)
	if status != "analyzed" || bpm.Float64 != 128 {
		t.Errorf("status=%q bpm=%v", status, bpm)
	}
}

func TestManager_ProcessOne_RetryableFailure(t *testing.T) {
	db := newTestDB(t)
	seedPending(t, db, "t1", "/a.wav")
	repo := NewRepository(db)
	fa := &fakeAnalyzer{err: ErrTimeout}
	m := NewManager(repo, fa)

	_, err := m.ProcessOne(context.Background())
	if err != nil {
		t.Fatalf("ProcessOne: %v", err)
	}

	var status string
	var retryCount int
	_ = db.QueryRow(`SELECT analysis_status, analysis_retry_count FROM tracks WHERE id='t1'`).Scan(&status, &retryCount)
	if status != "pending" {
		t.Errorf("status=%q, want pending after 1st retryable failure", status)
	}
	if retryCount != 1 {
		t.Errorf("retry_count=%d, want 1", retryCount)
	}
}

func TestManager_ProcessOne_TerminalFailureOnMissingFile(t *testing.T) {
	db := newTestDB(t)
	seedPending(t, db, "t1", "/a.wav")
	repo := NewRepository(db)
	fa := &fakeAnalyzer{err: ErrFileMissing}
	m := NewManager(repo, fa)

	_, err := m.ProcessOne(context.Background())
	if err != nil {
		t.Fatalf("ProcessOne: %v", err)
	}

	var status string
	var retryCount int
	_ = db.QueryRow(`SELECT analysis_status, analysis_retry_count FROM tracks WHERE id='t1'`).Scan(&status, &retryCount)
	if status != "failed" {
		t.Errorf("status=%q, want failed (terminal)", status)
	}
	if retryCount != 0 {
		t.Errorf("retry_count=%d, want 0 for terminal failure", retryCount)
	}
}

func TestManager_ProcessOne_BinaryMissingReturnsError(t *testing.T) {
	db := newTestDB(t)
	seedPending(t, db, "t1", "/a.wav")
	repo := NewRepository(db)
	fa := &fakeAnalyzer{err: ErrBinaryMissing}
	m := NewManager(repo, fa)

	// ErrBinaryMissing means config is broken; surface it rather than burning
	// retry budget on every track in the library.
	_, err := m.ProcessOne(context.Background())
	if !errors.Is(err, ErrBinaryMissing) {
		t.Fatalf("want ErrBinaryMissing, got %v", err)
	}

	var status string
	var retryCount int
	_ = db.QueryRow(`SELECT analysis_status, analysis_retry_count FROM tracks WHERE id='t1'`).Scan(&status, &retryCount)
	if status != "pending" || retryCount != 0 {
		t.Errorf("status=%q retry=%d, want untouched", status, retryCount)
	}
}
