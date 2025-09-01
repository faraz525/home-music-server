package auth

import (
	"time"

	"github.com/faraz525/home-music-server/backend/models"
	"github.com/faraz525/home-music-server/backend/utils"
)

// Repository handles authentication data operations
type Repository struct {
	db *utils.DB
}

// NewRepository creates a new auth repository
func NewRepository(db *utils.DB) *Repository {
	return &Repository{db: db}
}

// CreateUser creates a new user in the database
func (r *Repository) CreateUser(email, passwordHash, role string) (*models.User, error) {
	id := utils.GenerateUserID()
	now := time.Now()

	_, err := r.db.Exec(
		"INSERT INTO users (id, email, password_hash, role, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
		id, email, passwordHash, role, now, now,
	)
	if err != nil {
		return nil, err
	}

	return &models.User{
		ID:           id,
		Email:        email,
		PasswordHash: passwordHash,
		Role:         role,
		CreatedAt:    now,
		UpdatedAt:    now,
	}, nil
}

// AdminExists checks if an admin user already exists
func (r *Repository) AdminExists() (bool, error) {
	var count int
	err := r.db.QueryRow("SELECT COUNT(*) FROM users WHERE role = 'admin'").Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// GetUserByEmail retrieves a user by email
func (r *Repository) GetUserByEmail(email string) (*models.User, error) {
	var user models.User
	err := r.db.QueryRow(
		"SELECT id, email, password_hash, role, created_at, updated_at FROM users WHERE email = ?",
		email,
	).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.Role, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetUserByID retrieves a user by ID
func (r *Repository) GetUserByID(id string) (*models.User, error) {
	var user models.User
	err := r.db.QueryRow(
		"SELECT id, email, password_hash, role, created_at, updated_at FROM users WHERE id = ?",
		id,
	).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.Role, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// CreateRefreshToken creates a new refresh token
func (r *Repository) CreateRefreshToken(userID, tokenHash string, expiresAt time.Time) (*models.RefreshToken, error) {
	id := utils.GenerateTokenID()
	now := time.Now()

	_, err := r.db.Exec(
		"INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at, created_at) VALUES (?, ?, ?, ?, ?)",
		id, userID, tokenHash, expiresAt, now,
	)
	if err != nil {
		return nil, err
	}

	return &models.RefreshToken{
		ID:        id,
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
		CreatedAt: now,
	}, nil
}

// GetRefreshTokenByHash retrieves a refresh token by its hash
func (r *Repository) GetRefreshTokenByHash(tokenHash string) (*models.RefreshToken, error) {
	var token models.RefreshToken
	err := r.db.QueryRow(
		"SELECT id, user_id, token_hash, expires_at, revoked_at, created_at FROM refresh_tokens WHERE token_hash = ?",
		tokenHash,
	).Scan(&token.ID, &token.UserID, &token.TokenHash, &token.ExpiresAt, &token.RevokedAt, &token.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &token, nil
}

// RevokeRefreshToken revokes a refresh token
func (r *Repository) RevokeRefreshToken(tokenID string) error {
	now := time.Now()
	_, err := r.db.Exec("UPDATE refresh_tokens SET revoked_at = ? WHERE id = ?", now, tokenID)
	return err
}

// GetUsers retrieves all users (admin only)
func (r *Repository) GetUsers() ([]*models.User, error) {
	rows, err := r.db.Query("SELECT id, email, role, created_at, updated_at FROM users ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		var user models.User
		err := rows.Scan(&user.ID, &user.Email, &user.Role, &user.CreatedAt, &user.UpdatedAt)
		if err != nil {
			return nil, err
		}
		users = append(users, &user)
	}

	return users, rows.Err()
}
