package auth

import (
	"context"
	"time"

	"github.com/faraz525/home-music-server/backend/internal/db"
	imodels "github.com/faraz525/home-music-server/backend/internal/models"
	"github.com/faraz525/home-music-server/backend/utils"
)

// Data handles authentication data operations
type Repository struct {
	db *db.DB
}

// NewRepository creates a new auth data
func NewRepository(db *db.DB) *Repository {
	return &Repository{db: db}
}

// CreateUser creates a new user in the database
func (r *Repository) CreateUser(ctx context.Context, email, passwordHash, role string) (*imodels.User, error) {
	id := utils.GenerateUserID()
	now := time.Now()

	_, err := r.db.ExecContext(ctx,
		"INSERT INTO users (id, email, password_hash, role, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
		id, email, passwordHash, role, now, now,
	)
	if err != nil {
		return nil, err
	}

	return &imodels.User{
		ID:           id,
		Email:        email,
		PasswordHash: passwordHash,
		Role:         role,
		CreatedAt:    now,
		UpdatedAt:    now,
	}, nil
}

// AdminExists checks if an admin user already exists
func (r *Repository) AdminExists(ctx context.Context) (bool, error) {
	var count int
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users WHERE role = 'admin'").Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// GetUserByEmail retrieves a user by email
func (r *Repository) GetUserByEmail(ctx context.Context, email string) (*imodels.User, error) {
	var user imodels.User
	err := r.db.QueryRowContext(ctx,
		"SELECT id, email, password_hash, role, created_at, updated_at FROM users WHERE email = ?",
		email,
	).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.Role, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetUserByID retrieves a user by ID
func (r *Repository) GetUserByID(ctx context.Context, id string) (*imodels.User, error) {
	var user imodels.User
	err := r.db.QueryRowContext(ctx,
		"SELECT id, email, password_hash, role, created_at, updated_at FROM users WHERE id = ?",
		id,
	).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.Role, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// CreateRefreshToken creates a new refresh token
func (r *Repository) CreateRefreshToken(ctx context.Context, userID, tokenHash string, expiresAt time.Time) (*imodels.RefreshToken, error) {
	id := utils.GenerateTokenID()
	now := time.Now()

	_, err := r.db.ExecContext(ctx,
		"INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at, created_at) VALUES (?, ?, ?, ?, ?)",
		id, userID, tokenHash, expiresAt, now,
	)
	if err != nil {
		return nil, err
	}

	return &imodels.RefreshToken{
		ID:        id,
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
		CreatedAt: now,
	}, nil
}

// GetRefreshTokenByHash retrieves a refresh token by its hash
func (r *Repository) GetRefreshTokenByHash(ctx context.Context, tokenHash string) (*imodels.RefreshToken, error) {
	var token imodels.RefreshToken
	err := r.db.QueryRowContext(ctx,
		"SELECT id, user_id, token_hash, expires_at, revoked_at, created_at FROM refresh_tokens WHERE token_hash = ?",
		tokenHash,
	).Scan(&token.ID, &token.UserID, &token.TokenHash, &token.ExpiresAt, &token.RevokedAt, &token.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &token, nil
}

// RevokeRefreshToken revokes a refresh token
func (r *Repository) RevokeRefreshToken(ctx context.Context, tokenID string) error {
	now := time.Now()
	_, err := r.db.ExecContext(ctx, "UPDATE refresh_tokens SET revoked_at = ? WHERE id = ?", now, tokenID)
	return err
}

// GetUsers retrieves all users (admin only)
func (r *Repository) GetUsers(ctx context.Context) ([]*imodels.User, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT id, email, role, created_at, updated_at FROM users ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*imodels.User
	for rows.Next() {
		var user imodels.User
		err := rows.Scan(&user.ID, &user.Email, &user.Role, &user.CreatedAt, &user.UpdatedAt)
		if err != nil {
			return nil, err
		}
		users = append(users, &user)
	}

	return users, rows.Err()
}
