package trades

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	idb "github.com/faraz525/home-music-server/backend/internal/db"
	imodels "github.com/faraz525/home-music-server/backend/internal/models"
	"github.com/google/uuid"
)

// Repository handles database operations for trades
type Repository struct {
	db *idb.DB
}

// NewRepository creates a new trade repository
func NewRepository(db *idb.DB) *Repository {
	return &Repository{db: db}
}

// CreateTrade creates a new trade transaction
func (r *Repository) CreateTrade(ctx context.Context, trade *imodels.TradeTransaction) error {
	if trade.ID == "" {
		trade.ID = uuid.New().String()
	}

	query := `
		INSERT INTO trade_transactions (
			id, requester_user_id, owner_user_id, crate_id, 
			requested_track_id, given_track_ids, trade_ratio, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.ExecContext(ctx, query,
		trade.ID,
		trade.RequesterUserID,
		trade.OwnerUserID,
		trade.CrateID,
		trade.RequestedTrackID,
		trade.GivenTrackIDs,
		trade.TradeRatio,
		time.Now(),
	)

	if err != nil {
		return fmt.Errorf("failed to create trade: %w", err)
	}

	return nil
}

// CreateTrackReference creates a reference to a track for a user
func (r *Repository) CreateTrackReference(ctx context.Context, ref *imodels.TrackReference) error {
	if ref.ID == "" {
		ref.ID = uuid.New().String()
	}

	query := `
		INSERT INTO track_references (
			id, user_id, track_id, source_user_id, acquired_via, created_at
		) VALUES (?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.ExecContext(ctx, query,
		ref.ID,
		ref.UserID,
		ref.TrackID,
		ref.SourceUserID,
		ref.AcquiredVia,
		time.Now(),
	)

	if err != nil {
		// Check if it's a unique constraint violation
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return fmt.Errorf("you already have this track")
		}
		return fmt.Errorf("failed to create track reference: %w", err)
	}

	return nil
}

// GetUserTradeHistory returns trade history for a user
func (r *Repository) GetUserTradeHistory(ctx context.Context, userID string, limit, offset int) ([]*imodels.TradeTransaction, int, error) {
	query := `
		SELECT id, requester_user_id, owner_user_id, crate_id, 
		       requested_track_id, given_track_ids, trade_ratio, created_at
		FROM trade_transactions
		WHERE requester_user_id = ? OR owner_user_id = ?
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`

	rows, err := r.db.QueryContext(ctx, query, userID, userID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get trade history: %w", err)
	}
	defer rows.Close()

	var trades []*imodels.TradeTransaction
	for rows.Next() {
		trade := &imodels.TradeTransaction{}
		err := rows.Scan(
			&trade.ID,
			&trade.RequesterUserID,
			&trade.OwnerUserID,
			&trade.CrateID,
			&trade.RequestedTrackID,
			&trade.GivenTrackIDs,
			&trade.TradeRatio,
			&trade.CreatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan trade: %w", err)
		}

		trades = append(trades, trade)
	}

	// Get total count
	countQuery := `
		SELECT COUNT(*) 
		FROM trade_transactions 
		WHERE requester_user_id = ? OR owner_user_id = ?
	`
	var total int
	err = r.db.QueryRowContext(ctx, countQuery, userID, userID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count trades: %w", err)
	}

	return trades, total, nil
}

// GetUserTracks returns tracks owned by a user (not traded)
func (r *Repository) GetUserTracks(ctx context.Context, userID string) ([]string, error) {
	query := `
		SELECT id FROM tracks WHERE owner_user_id = ?
	`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user tracks: %w", err)
	}
	defer rows.Close()

	var trackIDs []string
	for rows.Next() {
		var trackID string
		if err := rows.Scan(&trackID); err != nil {
			return nil, fmt.Errorf("failed to scan track: %w", err)
		}
		trackIDs = append(trackIDs, trackID)
	}

	return trackIDs, nil
}

// HasTrackReference checks if a user has a reference to a track
func (r *Repository) HasTrackReference(ctx context.Context, userID, trackID string) (bool, error) {
	query := `
		SELECT COUNT(*) FROM track_references 
		WHERE user_id = ? AND track_id = ?
	`

	var count int
	err := r.db.QueryRowContext(ctx, query, userID, trackID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check track reference: %w", err)
	}

	return count > 0, nil
}

// GetTrackOwner gets the owner of a track
func (r *Repository) GetTrackOwner(ctx context.Context, trackID string) (string, error) {
	query := `SELECT owner_user_id FROM tracks WHERE id = ?`
	
	var ownerID string
	err := r.db.QueryRowContext(ctx, query, trackID).Scan(&ownerID)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("track not found")
	}
	if err != nil {
		return "", fmt.Errorf("failed to get track owner: %w", err)
	}

	return ownerID, nil
}
