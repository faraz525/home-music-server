package analysis

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// StartLoop polls for pending tracks on the given interval. Stops when ctx
// is cancelled. If Analyze returns ErrBinaryMissing, the loop backs off to
// a longer wait (since no amount of retrying will help until the binary is
// installed and the server restarted).
func StartLoop(ctx context.Context, m *Manager, interval time.Duration) {
	fmt.Printf("[Analysis] Starting loop (interval=%s)\n", interval)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	backoff := time.Duration(0)
	for {
		select {
		case <-ctx.Done():
			fmt.Println("[Analysis] Loop stopped")
			return
		case <-ticker.C:
			if backoff > 0 {
				// Consume this tick to honor backoff.
				backoff -= interval
				if backoff < 0 {
					backoff = 0
				}
				continue
			}
			processed, err := m.ProcessOne(ctx)
			if err != nil {
				if errors.Is(err, ErrBinaryMissing) {
					fmt.Println("[Analysis] essentia binary not available; backing off for 5 minutes")
					backoff = 5 * time.Minute
					continue
				}
				fmt.Printf("[Analysis] ProcessOne error: %v\n", err)
			}
			_ = processed
		}
	}
}
