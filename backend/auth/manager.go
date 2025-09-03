package auth

import (
	"errors"
	"os"
	"strings"
	"time"

	"github.com/faraz525/home-music-server/backend/models"
	"github.com/faraz525/home-music-server/backend/utils"
	"github.com/gin-gonic/gin"
)

// Manager handles authentication business logic and API management
type Manager struct {
	repo          *Repository
	jwtSecret     string
	refreshSecret string
}

// NewManager creates a new auth manager
func NewManager(repo *Repository) (*Manager, error) {
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		return nil, errors.New("JWT_SECRET not set")
	}

	refreshSecret := os.Getenv("REFRESH_SECRET")
	if refreshSecret == "" {
		return nil, errors.New("REFRESH_SECRET not set")
	}

	return &Manager{
		repo:          repo,
		jwtSecret:     jwtSecret,
		refreshSecret: refreshSecret,
	}, nil
}

// Signup handles user registration
func (m *Manager) Signup(req *models.SignupRequest) (*models.Tokens, error) {
	// Normalize email
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	// Check if user already exists
	existingUser, err := m.repo.GetUserByEmail(req.Email)
	if err == nil && existingUser != nil {
		return nil, errors.New("user with this email already exists")
	}

	// Determine role based on email
	role := "user"
	const adminEmail = "farazq638@gmail.com"

	if req.Email == adminEmail {
		// Check if admin already exists
		adminExists, err := m.repo.AdminExists()
		if err != nil {
			return nil, errors.New("failed to check admin status")
		}
		if adminExists {
			return nil, errors.New("admin account already exists")
		}
		role = "admin"
	}

	// Hash password
	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		return nil, errors.New("failed to process password")
	}

	// Create user
	user, err := m.repo.CreateUser(req.Email, hashedPassword, role)
	if err != nil {
		return nil, errors.New("failed to create user")
	}

	// Generate tokens
	tokens, err := m.generateTokens(user.ID, user.Email, user.Role)
	if err != nil {
		return nil, errors.New("failed to generate tokens")
	}

	// Store refresh token hash
	refreshTokenHash := utils.HashRefreshToken(tokens.RefreshToken)
	expiresAt := time.Now().Add(utils.RefreshTokenDuration)
	_, err = m.repo.CreateRefreshToken(user.ID, refreshTokenHash, expiresAt)
	if err != nil {
		return nil, errors.New("failed to store refresh token")
	}

	return tokens, nil
}

// Login handles user authentication
func (m *Manager) Login(req *models.LoginRequest) (*models.Tokens, error) {
	// Normalize email
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	// Get user
	user, err := m.repo.GetUserByEmail(req.Email)
	if err != nil {
		return nil, errors.New("invalid email or password")
	}

	// Check password
	if !utils.CheckPasswordHash(req.Password, user.PasswordHash) {
		return nil, errors.New("invalid email or password")
	}

	// Generate tokens
	tokens, err := m.generateTokens(user.ID, user.Email, user.Role)
	if err != nil {
		return nil, errors.New("failed to generate tokens")
	}

	// Store refresh token hash
	refreshTokenHash := utils.HashRefreshToken(tokens.RefreshToken)
	expiresAt := time.Now().Add(utils.RefreshTokenDuration)
	_, err = m.repo.CreateRefreshToken(user.ID, refreshTokenHash, expiresAt)
	if err != nil {
		return nil, errors.New("failed to store refresh token")
	}

	return tokens, nil
}

// Refresh handles token refresh
func (m *Manager) Refresh(refreshToken string) (*models.Tokens, error) {
	// Hash the token to look it up
	tokenHash := utils.HashRefreshToken(refreshToken)

	// Get token from database
	storedToken, err := m.repo.GetRefreshTokenByHash(tokenHash)
	if err != nil {
		return nil, errors.New("invalid refresh token")
	}

	// Check if token is expired
	if time.Now().After(storedToken.ExpiresAt) {
		return nil, errors.New("refresh token expired")
	}

	// Check if token is revoked
	if storedToken.RevokedAt != nil {
		return nil, errors.New("refresh token revoked")
	}

	// Get user
	user, err := m.repo.GetUserByID(storedToken.UserID)
	if err != nil {
		return nil, errors.New("failed to get user")
	}

	// Revoke old refresh token
	err = m.repo.RevokeRefreshToken(storedToken.ID)
	if err != nil {
		return nil, errors.New("failed to revoke old token")
	}

	// Generate new tokens
	tokens, err := m.generateTokens(user.ID, user.Email, user.Role)
	if err != nil {
		return nil, errors.New("failed to generate tokens")
	}

	// Store new refresh token hash
	newTokenHash := utils.HashRefreshToken(tokens.RefreshToken)
	expiresAt := time.Now().Add(utils.RefreshTokenDuration)
	_, err = m.repo.CreateRefreshToken(user.ID, newTokenHash, expiresAt)
	if err != nil {
		return nil, errors.New("failed to store refresh token")
	}

	return tokens, nil
}

// Logout handles user logout
func (m *Manager) Logout(refreshToken string) error {
	if refreshToken == "" {
		return nil // Nothing to revoke
	}

	// Hash the token and revoke it
	tokenHash := utils.HashRefreshToken(refreshToken)
	storedToken, err := m.repo.GetRefreshTokenByHash(tokenHash)
	if err == nil {
		return m.repo.RevokeRefreshToken(storedToken.ID)
	}

	return nil // Token not found, nothing to do
}

// GetCurrentUser retrieves the current authenticated user
func (m *Manager) GetCurrentUser(userID string) (*models.User, error) {
	return m.repo.GetUserByID(userID)
}

// GetUsers retrieves all users (admin only)
func (m *Manager) GetUsers() ([]*models.User, error) {
	return m.repo.GetUsers()
}

// ValidateAccessToken validates an access token
func (m *Manager) ValidateAccessToken(tokenString string) (*utils.Claims, error) {
	return utils.ValidateAccessToken(tokenString)
}

// AuthMiddleware validates JWT tokens and adds user context
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Dev bypass: if explicitly enabled, trust a fixed user
		if os.Getenv("DEV_AUTH_BYPASS") == "1" {
			c.Set("user_id", "dev-user")
			c.Set("user_email", "dev@local")
			c.Set("user_role", "admin")
			c.Next()
			return
		}
		authHeader := c.GetHeader("Authorization")
		// Fallback to cookie for cases like <audio> tag or dev proxy
		if authHeader == "" {
			if cookie, err := c.Cookie("access_token"); err == nil && cookie != "" {
				authHeader = "Bearer " + cookie
			}
		}
		if authHeader == "" {
			c.JSON(401, gin.H{"error": gin.H{"code": "auth_required", "message": "Authorization header required"}})
			c.Abort()
			return
		}

		// Extract token from "Bearer <token>"
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(401, gin.H{"error": gin.H{"code": "invalid_auth_format", "message": "Authorization header must be Bearer <token>"}})
			c.Abort()
			return
		}

		tokenString := parts[1]

		// Validate token
		claims, err := utils.ValidateAccessToken(tokenString)
		if err != nil {
			if strings.Contains(err.Error(), "token is expired") {
				c.JSON(401, gin.H{"error": gin.H{"code": "token_expired", "message": "Access token expired"}})
			} else {
				c.JSON(401, gin.H{"error": gin.H{"code": "invalid_token", "message": "Invalid access token"}})
			}
			c.Abort()
			return
		}

		// Add user info to context
		c.Set("user_id", claims.UserID)
		c.Set("user_email", claims.Email)
		c.Set("user_role", claims.Role)

		c.Next()
	}
}

// AdminMiddleware ensures user has admin role
func AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("user_role")
		if !exists || role != "admin" {
			c.JSON(403, gin.H{"error": gin.H{"code": "admin_required", "message": "Admin access required"}})
			c.Abort()
			return
		}
		c.Next()
	}
}

// GetAvailableAPIs returns the list of available auth APIs
func (m *Manager) GetAvailableAPIs() []string {
	return []string{
		"POST /api/auth/signup - User registration",
		"POST /api/auth/login - User login",
		"POST /api/auth/refresh - Token refresh",
		"POST /api/auth/logout - User logout",
		"GET /api/me - Current user info",
		"GET /api/users - List users (admin only)",
	}
}

// generateTokens creates both access and refresh tokens
func (m *Manager) generateTokens(userID, email, role string) (*models.Tokens, error) {
	// Generate access token
	accessToken, err := utils.GenerateAccessToken(userID, email, role)
	if err != nil {
		return nil, err
	}

	// Generate refresh token
	refreshToken, err := utils.GenerateRefreshToken()
	if err != nil {
		return nil, err
	}

	return &models.Tokens{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User: map[string]interface{}{
			"id":    userID,
			"email": email,
			"role":  role,
		},
	}, nil
}
