package playlists

import (
	"fmt"

	imodels "github.com/faraz525/home-music-server/backend/internal/models"
)

// Manager handles business logic for playlists
type Manager struct {
	repo *Repository
}

// NewManager creates a new playlist manager
func NewManager(repo *Repository) *Manager {
	return &Manager{repo: repo}
}

// CreatePlaylist creates a new playlist for a user
func (m *Manager) CreatePlaylist(ownerUserID string, req *imodels.CreatePlaylistRequest) (*imodels.Playlist, error) {
	// Validate request
	if req.Name == "" {
		return nil, fmt.Errorf("playlist name cannot be empty")
	}

	if len(req.Name) > 100 {
		return nil, fmt.Errorf("playlist name cannot exceed 100 characters")
	}

	if req.Description != nil && len(*req.Description) > 500 {
		return nil, fmt.Errorf("playlist description cannot exceed 500 characters")
	}

	// Create the playlist
	playlist, err := m.repo.CreatePlaylist(ownerUserID, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create playlist: %w", err)
	}

	return playlist, nil
}

// GetUserPlaylists returns all playlists for a user, including virtual "Unsorted" crate
func (m *Manager) GetUserPlaylists(userID string, limit, offset int) (*imodels.PlaylistList, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	return m.repo.GetUserPlaylistsWithVirtual(userID, limit, offset)
}

// GetPlaylist returns a specific playlist with ownership validation
func (m *Manager) GetPlaylist(playlistID, requestingUserID string) (*imodels.Playlist, error) {
	// Handle virtual "unsorted" playlist
	if playlistID == "unsorted" {
		description := "Tracks not assigned to any crate"
		return &imodels.Playlist{
			ID:          "unsorted",
			OwnerUserID: requestingUserID,
			Name:        "Unsorted",
			Description: &description,
			IsDefault:   true,
		}, nil
	}

	playlist, err := m.repo.GetPlaylist(playlistID)
	if err != nil {
		return nil, err
	}

	// Check ownership
	if playlist.OwnerUserID != requestingUserID {
		return nil, fmt.Errorf("access denied: playlist belongs to another user")
	}

	return playlist, nil
}

// UpdatePlaylist updates a playlist with ownership validation
func (m *Manager) UpdatePlaylist(playlistID, requestingUserID string, req *imodels.UpdatePlaylistRequest) error {
	// First check ownership
	playlist, err := m.repo.GetPlaylist(playlistID)
	if err != nil {
		return err
	}

	if playlist.OwnerUserID != requestingUserID {
		return fmt.Errorf("access denied: playlist belongs to another user")
	}

	// Don't allow updating default playlists
	if playlist.IsDefault {
		return fmt.Errorf("cannot modify the default playlist")
	}

	// Validate request
	if req.Name == "" {
		return fmt.Errorf("playlist name cannot be empty")
	}

	if len(req.Name) > 100 {
		return fmt.Errorf("playlist name cannot exceed 100 characters")
	}

	if req.Description != nil && len(*req.Description) > 500 {
		return fmt.Errorf("playlist description cannot exceed 500 characters")
	}

	return m.repo.UpdatePlaylist(playlistID, req)
}

// DeletePlaylist deletes a playlist with ownership validation
func (m *Manager) DeletePlaylist(playlistID, requestingUserID string) error {
	// First check ownership
	playlist, err := m.repo.GetPlaylist(playlistID)
	if err != nil {
		return err
	}

	if playlist.OwnerUserID != requestingUserID {
		return fmt.Errorf("access denied: playlist belongs to another user")
	}

	// Don't allow deleting default playlists
	if playlist.IsDefault {
		return fmt.Errorf("cannot delete the default playlist")
	}

	return m.repo.DeletePlaylist(playlistID)
}

// EnsureDefaultPlaylist creates the default "Unsorted" playlist if it doesn't exist
func (m *Manager) EnsureDefaultPlaylist(userID string) (*imodels.Playlist, error) {
	// Try to get existing default playlist
	playlist, err := m.repo.GetDefaultPlaylist(userID)
	if err == nil {
		return playlist, nil
	}

	// Create new default playlist if it doesn't exist
	return m.repo.CreateDefaultPlaylist(userID)
}

// AddTracksToPlaylist adds tracks to a playlist with validation
func (m *Manager) AddTracksToPlaylist(playlistID, requestingUserID string, req *imodels.AddTracksToPlaylistRequest) error {
	// Validate request
	if len(req.TrackIDs) == 0 {
		return fmt.Errorf("no track IDs provided")
	}

	if len(req.TrackIDs) > 100 {
		return fmt.Errorf("cannot add more than 100 tracks at once")
	}

	// Check playlist ownership
	playlist, err := m.repo.GetPlaylist(playlistID)
	if err != nil {
		return err
	}

	if playlist.OwnerUserID != requestingUserID {
		return fmt.Errorf("access denied: playlist belongs to another user")
	}

	// TODO: Validate that tracks belong to the user
	// This would require checking track ownership in the tracks repository

	return m.repo.AddTracksToPlaylist(playlistID, req.TrackIDs)
}

// RemoveTracksFromPlaylist removes tracks from a playlist with validation
func (m *Manager) RemoveTracksFromPlaylist(playlistID, requestingUserID string, req *imodels.RemoveTracksFromPlaylistRequest) error {
	// Validate request
	if len(req.TrackIDs) == 0 {
		return fmt.Errorf("no track IDs provided")
	}

	if len(req.TrackIDs) > 100 {
		return fmt.Errorf("cannot remove more than 100 tracks at once")
	}

	// Check playlist ownership
	playlist, err := m.repo.GetPlaylist(playlistID)
	if err != nil {
		return err
	}

	if playlist.OwnerUserID != requestingUserID {
		return fmt.Errorf("access denied: playlist belongs to another user")
	}

	return m.repo.RemoveTracksFromPlaylist(playlistID, req.TrackIDs)
}

// GetPlaylistTracks returns tracks for a playlist with ownership validation
func (m *Manager) GetPlaylistTracks(playlistID, requestingUserID string, limit, offset int) (*imodels.PlaylistWithTracks, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	// Handle virtual "unsorted" playlist
	if playlistID == "unsorted" {
		trackList, err := m.repo.GetTracksNotInPlaylist(requestingUserID, limit, offset)
		if err != nil {
			return nil, err
		}

		description := "Tracks not assigned to any crate"
		return &imodels.PlaylistWithTracks{
			Playlist: &imodels.Playlist{
				ID:          "unsorted",
				OwnerUserID: requestingUserID,
				Name:        "Unsorted",
				Description: &description,
				IsDefault:   true,
			},
			Tracks:  trackList.Tracks,
			Total:   trackList.Total,
			Limit:   trackList.Limit,
			Offset:  trackList.Offset,
			HasNext: trackList.HasNext,
		}, nil
	}

	// Check playlist ownership
	playlist, err := m.repo.GetPlaylist(playlistID)
	if err != nil {
		return nil, err
	}

	if playlist.OwnerUserID != requestingUserID {
		return nil, fmt.Errorf("access denied: playlist belongs to another user")
	}

	return m.repo.GetPlaylistTracks(playlistID, limit, offset)
}

// GetUnsortedTracks returns tracks not in any playlist for a user
func (m *Manager) GetUnsortedTracks(userID string, limit, offset int) (*imodels.TrackList, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	return m.repo.GetTracksNotInPlaylist(userID, limit, offset)
}

// GetDefaultPlaylist returns the default playlist for a user
func (m *Manager) GetDefaultPlaylist(userID string) (*imodels.Playlist, error) {
	return m.repo.GetDefaultPlaylist(userID)
}

// CanAccessPlaylist checks if a user can access a playlist (for internal use)
func (m *Manager) CanAccessPlaylist(playlistID, userID string) error {
	playlist, err := m.repo.GetPlaylist(playlistID)
	if err != nil {
		return err
	}

	if playlist.OwnerUserID != userID {
		return fmt.Errorf("access denied")
	}

	return nil
}

// SearchUnsortedTracks searches tracks not in any playlist using FTS5
func (m *Manager) SearchUnsortedTracks(userID, query string, limit, offset int) (*imodels.TrackList, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	return m.repo.SearchTracksNotInPlaylist(userID, query, limit, offset)
}

// SearchPlaylistTracks searches within a specific playlist using FTS5
func (m *Manager) SearchPlaylistTracks(playlistID, userID, query string, limit, offset int) (*imodels.TrackList, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	// Check playlist ownership
	playlist, err := m.repo.GetPlaylist(playlistID)
	if err != nil {
		return nil, err
	}

	if playlist.OwnerUserID != userID {
		return nil, fmt.Errorf("access denied: playlist belongs to another user")
	}

	return m.repo.SearchPlaylistTracks(playlistID, query, limit, offset)
}

// GetPublicPlaylists returns all public playlists with pagination
func (m *Manager) GetPublicPlaylists(limit, offset int) ([]*imodels.PlaylistWithOwner, int, bool, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	playlists, total, err := m.repo.GetPublicPlaylists(limit, offset)
	if err != nil {
		return nil, 0, false, err
	}

	hasNext := offset+limit < total
	return playlists, total, hasNext, nil
}

// UpdatePlaylistVisibility updates the visibility of a playlist with ownership validation
func (m *Manager) UpdatePlaylistVisibility(playlistID, requestingUserID string, isPublic bool) error {
	// Check playlist ownership
	playlist, err := m.repo.GetPlaylist(playlistID)
	if err != nil {
		return err
	}

	if playlist.OwnerUserID != requestingUserID {
		return fmt.Errorf("access denied: playlist belongs to another user")
	}

	// Validate: Unsorted playlists cannot be made public
	if playlist.IsDefault && isPublic {
		return fmt.Errorf("unsorted crates cannot be made public")
	}

	return m.repo.UpdatePlaylistVisibility(playlistID, isPublic)
}
