package trades

import (
	"context"
	"fmt"
	"strings"

	imodels "github.com/faraz525/home-music-server/backend/internal/models"
)

// PlaylistGetter interface for getting playlist info
type PlaylistGetter interface {
	GetPlaylist(playlistID string) (*imodels.Playlist, error)
}

// Manager handles trade business logic
type Manager struct {
	repo           *Repository
	playlistGetter PlaylistGetter
}

// NewManager creates a new trade manager
func NewManager(repo *Repository) *Manager {
	return &Manager{repo: repo}
}

// SetPlaylistGetter sets the playlist getter
func (m *Manager) SetPlaylistGetter(pg PlaylistGetter) {
	m.playlistGetter = pg
}

// RequestTrade handles a trade request
func (m *Manager) RequestTrade(ctx context.Context, requesterUserID string, req *imodels.TradeRequest) error {
	// Validate request
	if len(req.OfferTrackIDs) == 0 {
		return fmt.Errorf("no tracks offered")
	}

	// Prevent too many tracks in one trade (Pi resource constraint)
	if len(req.OfferTrackIDs) > 10 {
		return fmt.Errorf("cannot offer more than 10 tracks in a single trade")
	}

	// Check for duplicate tracks in offer
	trackSet := make(map[string]bool)
	for _, trackID := range req.OfferTrackIDs {
		if trackSet[trackID] {
			return fmt.Errorf("duplicate track in offer: %s", trackID)
		}
		trackSet[trackID] = true
	}

	// Get the crate to check trade ratio and ownership
	if m.playlistGetter == nil {
		return fmt.Errorf("playlist getter not set")
	}

	playlist, err := m.playlistGetter.GetPlaylist(req.CrateID)
	if err != nil {
		return fmt.Errorf("crate not found")
	}

	// Check if crate is public
	if !playlist.IsPublic {
		return fmt.Errorf("crate is not public")
	}

	// Get track owner
	trackOwnerID, err := m.repo.GetTrackOwner(ctx, req.TrackID)
	if err != nil {
		return err
	}

	// Check if requester is trying to trade for their own track
	if requesterUserID == trackOwnerID {
		return fmt.Errorf("you already own this track")
	}

	// Check if user already has this track
	hasRef, err := m.repo.HasTrackReference(ctx, requesterUserID, req.TrackID)
	if err != nil {
		return err
	}
	if hasRef {
		return fmt.Errorf("you already have this track")
	}

	// Validate trade ratio
	requiredTracks := playlist.TradeRatioGive
	if len(req.OfferTrackIDs) < requiredTracks {
		return fmt.Errorf("you need to offer at least %d track(s) for this trade", requiredTracks)
	}

	// Validate that offered tracks belong to the requester
	userTracks, err := m.repo.GetUserTracks(ctx, requesterUserID)
	if err != nil {
		return err
	}

	userTrackMap := make(map[string]bool)
	for _, trackID := range userTracks {
		userTrackMap[trackID] = true
	}

	for _, offerTrackID := range req.OfferTrackIDs {
		if !userTrackMap[offerTrackID] {
			return fmt.Errorf("you don't own track %s", offerTrackID)
		}
	}

	// Create trade transaction
	trade := &imodels.TradeTransaction{
		RequesterUserID:  requesterUserID,
		OwnerUserID:      trackOwnerID,
		CrateID:          req.CrateID,
		RequestedTrackID: req.TrackID,
		GivenTrackIDs:    strings.Join(req.OfferTrackIDs, ","),
		TradeRatio:       fmt.Sprintf("%d:%d", playlist.TradeRatioGive, playlist.TradeRatioTake),
	}

	err = m.repo.CreateTrade(ctx, trade)
	if err != nil {
		return err
	}

	// Create track reference for requester
	ref := &imodels.TrackReference{
		UserID:       requesterUserID,
		TrackID:      req.TrackID,
		SourceUserID: trackOwnerID,
		AcquiredVia:  "trade",
	}

	err = m.repo.CreateTrackReference(ctx, ref)
	if err != nil {
		return err
	}

	// Create track references for owner (for the tracks they received)
	for _, offerTrackID := range req.OfferTrackIDs {
		ownerRef := &imodels.TrackReference{
			UserID:       trackOwnerID,
			TrackID:      offerTrackID,
			SourceUserID: requesterUserID,
			AcquiredVia:  "trade",
		}

		// Ignore errors if owner already has the track
		_ = m.repo.CreateTrackReference(ctx, ownerRef)
	}

	return nil
}

// GetUserTradeHistory returns trade history for a user
func (m *Manager) GetUserTradeHistory(ctx context.Context, userID string, limit, offset int) ([]*imodels.TradeTransaction, int, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	return m.repo.GetUserTradeHistory(ctx, userID, limit, offset)
}

// GetAvailableTracksForTrade returns tracks that a user can offer in trades
func (m *Manager) GetAvailableTracksForTrade(ctx context.Context, userID string) ([]string, error) {
	return m.repo.GetUserTracks(ctx, userID)
}
