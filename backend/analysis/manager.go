package analysis

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// analyzer is the narrow interface the manager needs — lets tests swap in a fake.
type analyzer interface {
	Analyze(ctx context.Context, audioPath string) (Result, error)
}

type Manager struct {
	repo     *Repository
	analyzer analyzer
	now      func() time.Time
}

func NewManager(repo *Repository, a analyzer) *Manager {
	return &Manager{repo: repo, analyzer: a, now: time.Now}
}

// ProcessOne claims the next pending track (if any) and analyzes it.
// Returns (processed, err). `processed` is true iff a track was claimed;
// err is only returned for surface-breaking conditions like ErrBinaryMissing.
// Per-track analysis failures are recorded in the DB and do NOT bubble up.
func (m *Manager) ProcessOne(ctx context.Context) (bool, error) {
	claim, err := m.repo.ClaimNextPending(ctx)
	if err != nil {
		return false, fmt.Errorf("claim next pending: %w", err)
	}
	if claim == nil {
		return false, nil
	}

	result, analyzeErr := m.analyzer.Analyze(ctx, claim.FilePath)
	if analyzeErr != nil {
		// Surface ErrBinaryMissing so the ticker can back off instead of
		// marching through every track as failed.
		if errors.Is(analyzeErr, ErrBinaryMissing) {
			return false, analyzeErr
		}
		if errors.Is(analyzeErr, ErrFileMissing) {
			if err := m.repo.RecordTerminalFailure(ctx, claim.ID, analyzeErr.Error()); err != nil {
				fmt.Printf("[analysis] record terminal failure for %s: %v\n", claim.ID, err)
			}
			return true, nil
		}
		// Timeout, malformed output, exec failure — all retryable.
		if err := m.repo.RecordFailure(ctx, claim.ID, analyzeErr.Error(), m.now()); err != nil {
			fmt.Printf("[analysis] record failure for %s: %v\n", claim.ID, err)
		}
		return true, nil
	}

	if err := m.repo.MarkAnalyzed(ctx, claim.ID, result); err != nil {
		fmt.Printf("[analysis] mark analyzed for %s: %v\n", claim.ID, err)
		return true, nil
	}
	return true, nil
}
