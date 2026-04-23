package spotify

import (
	"context"
	"fmt"
	"time"
)

func StartSyncLoop(ctx context.Context, manager *Manager) {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	fmt.Println("[Spotify] Sync loop started (24h interval)")

	initialDelay := time.NewTimer(2 * time.Minute)
	defer initialDelay.Stop()

	select {
	case <-ctx.Done():
		fmt.Println("[Spotify] Sync loop cancelled before initial run")
		return
	case <-initialDelay.C:
		if err := manager.SyncLikes(ctx); err != nil {
			fmt.Printf("[Spotify] Initial sync failed: %v\n", err)
		}
	}

	for {
		select {
		case <-ctx.Done():
			fmt.Println("[Spotify] Sync loop stopped")
			return
		case <-ticker.C:
			if err := manager.SyncLikes(ctx); err != nil {
				fmt.Printf("[Spotify] Scheduled sync failed: %v\n", err)
			}
		}
	}
}
