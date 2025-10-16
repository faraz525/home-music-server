package users

import (
	"context"
	"fmt"
	"strings"

	imodels "github.com/faraz525/home-music-server/backend/internal/models"
)

// Manager handles user business logic
type Manager struct {
	repo *Repository
}

// NewManager creates a new user manager
func NewManager(repo *Repository) *Manager {
	return &Manager{repo: repo}
}

// SearchUsers searches for users by username
func (m *Manager) SearchUsers(ctx context.Context, query string, limit, offset int, isAdmin bool) ([]*imodels.UserSearchResult, int, error) {
	// Normalize search query
	query = strings.ToLower(strings.TrimSpace(query))
	
	if query == "" {
		return []*imodels.UserSearchResult{}, 0, nil
	}

	users, total, err := m.repo.SearchUsers(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	// Hide email from non-admin users
	if !isAdmin {
		for _, user := range users {
			user.Email = ""
		}
	}

	return users, total, nil
}

// GetUserByUsername retrieves a user profile by username
func (m *Manager) GetUserByUsername(ctx context.Context, username string) (*imodels.User, error) {
	user, err := m.repo.GetUserByUsername(ctx, username)
	if err != nil {
		return nil, err
	}

	// Don't expose password hash
	user.PasswordHash = ""

	return user, nil
}

// UpdateUsername updates a user's username
func (m *Manager) UpdateUsername(ctx context.Context, userID string, req *imodels.UpdateUsernameRequest) error {
	// Normalize username
	username := strings.ToLower(strings.TrimSpace(req.Username))

	// Validate username format (alphanumeric, underscores, hyphens)
	if !isValidUsername(username) {
		return fmt.Errorf("username can only contain letters, numbers, underscores, and hyphens")
	}

	err := m.repo.UpdateUsername(ctx, userID, username)
	if err != nil {
		// Check if it's a unique constraint violation
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return fmt.Errorf("username already taken")
		}
		return err
	}

	return nil
}

// isValidUsername checks if username contains only valid characters
func isValidUsername(username string) bool {
	if len(username) < 3 || len(username) > 30 {
		return false
	}

	for _, char := range username {
		if !((char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '_' || char == '-') {
			return false
		}
	}

	return true
}
