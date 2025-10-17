package users

import (
	"context"
	"database/sql"
	"fmt"

	idb "github.com/faraz525/home-music-server/backend/internal/db"
	imodels "github.com/faraz525/home-music-server/backend/internal/models"
)

// Repository handles database operations for users
type Repository struct {
	db *idb.DB
}

// NewRepository creates a new user repository
func NewRepository(db *idb.DB) *Repository {
	return &Repository{db: db}
}

// SearchUsers searches for users by username
func (r *Repository) SearchUsers(ctx context.Context, query string, limit, offset int) ([]*imodels.UserSearchResult, int, error) {
	searchQuery := `
		SELECT id, username, email
		FROM users
		WHERE username LIKE ? AND username IS NOT NULL
		ORDER BY username ASC
		LIMIT ? OFFSET ?
	`

	searchPattern := "%" + query + "%"
	rows, err := r.db.QueryContext(ctx, searchQuery, searchPattern, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to search users: %w", err)
	}
	defer rows.Close()

	var users []*imodels.UserSearchResult
	for rows.Next() {
		user := &imodels.UserSearchResult{}
		var username sql.NullString
		
		err := rows.Scan(&user.ID, &username, &user.Email)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan user: %w", err)
		}

		if username.Valid {
			user.Username = username.String
		}

		users = append(users, user)
	}

	// Get total count
	countQuery := `SELECT COUNT(*) FROM users WHERE username LIKE ? AND username IS NOT NULL`
	var total int
	err = r.db.QueryRowContext(ctx, countQuery, searchPattern).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count users: %w", err)
	}

	return users, total, nil
}

// GetUserByUsername retrieves a user by username
func (r *Repository) GetUserByUsername(ctx context.Context, username string) (*imodels.User, error) {
	query := `
		SELECT id, email, username, password_hash, role, created_at, updated_at
		FROM users
		WHERE username = ?
	`

	var user imodels.User
	err := r.db.QueryRowContext(ctx, query, username).Scan(
		&user.ID,
		&user.Email,
		&user.Username,
		&user.PasswordHash,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

// UpdateUsername updates a user's username
func (r *Repository) UpdateUsername(ctx context.Context, userID, username string) error {
	query := `
		UPDATE users
		SET username = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`

	result, err := r.db.ExecContext(ctx, query, username, userID)
	if err != nil {
		return fmt.Errorf("failed to update username: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}
