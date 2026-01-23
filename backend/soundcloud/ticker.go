package soundcloud

import (
	"context"
	"fmt"
	"time"
)

func StartSyncLoop(ctx context.Context, manager *Manager) {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	fmt.Println("[SoundCloud] Sync loop started (24h interval)")

	initialDelay := time.NewTimer(1 * time.Minute)
	defer initialDelay.Stop()

	select {
	case <-ctx.Done():
		fmt.Println("[SoundCloud] Sync loop cancelled before initial run")
		return
	case <-initialDelay.C:
		if err := manager.SyncLikes(ctx); err != nil {
			fmt.Printf("[SoundCloud] Initial sync failed: %v\n", err)
		}
	}

	for {
		select {
		case <-ctx.Done():
			fmt.Println("[SoundCloud] Sync loop stopped")
			return
		case <-ticker.C:
			if err := manager.SyncLikes(ctx); err != nil {
				fmt.Printf("[SoundCloud] Scheduled sync failed: %v\n", err)
			}
		}
	}
}
