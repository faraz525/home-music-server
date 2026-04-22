package analysis

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// StartLoop polls for pending tracks on the given interval. When ProcessOne
// reports a track was handled, the loop immediately drains the next one
// instead of waiting for the next tick — this matters during initial backfill,
// where a library of N tracks would otherwise take N*interval of idle wait.
//
// Stops when ctx is cancelled or when ProcessOne reports ErrBinaryMissing
// (binary won't appear without a server restart, so no point spinning).
func StartLoop(ctx context.Context, m *Manager, interval time.Duration) {
	fmt.Printf("[Analysis] Starting loop (interval=%s)\n", interval)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		if ctx.Err() != nil {
			fmt.Println("[Analysis] Loop stopped")
			return
		}

		processed, err := m.ProcessOne(ctx)
		if err != nil {
			if errors.Is(err, ErrBinaryMissing) {
				fmt.Println("[Analysis] essentia binary not available; stopping loop until restart")
				return
			}
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				fmt.Println("[Analysis] Loop stopped")
				return
			}
			fmt.Printf("[Analysis] ProcessOne error: %v\n", err)
		}

		// Drain mode: if we just processed a track, loop again without waiting.
		// If idle, block on the next tick (or shutdown).
		if processed {
			continue
		}
		select {
		case <-ctx.Done():
			fmt.Println("[Analysis] Loop stopped")
			return
		case <-ticker.C:
		}
	}
}
