package utils

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

const (
	AccessTokenDuration  = 15 * time.Minute
	RefreshTokenDuration = 30 * 24 * time.Hour // 30 days
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token expired")
)

type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

type Tokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	User         gin.H  `json:"user"`
}

// GenerateTokens creates both access and refresh tokens for a user
func GenerateTokens(userID, email, role string) (*Tokens, error) {
	// Generate access token
	accessToken, err := GenerateAccessToken(userID, email, role)
	if err != nil {
		return nil, err
	}

	// Generate refresh token
	refreshToken, err := GenerateRefreshToken()
	if err != nil {
		return nil, err
	}

	return &Tokens{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User: gin.H{
			"id":    userID,
			"email": email,
			"role":  role,
		},
	}, nil
}

// GenerateAccessToken creates a JWT access token
func GenerateAccessToken(userID, email, role string) (string, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return "", errors.New("JWT_SECRET not set")
	}

	claims := Claims{
		UserID: userID,
		Email:  email,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(AccessTokenDuration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "cratedrop",
			Subject:   userID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// GenerateRefreshToken creates a random refresh token
func GenerateRefreshToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// HashRefreshToken creates a SHA-256 hash of the refresh token for storage
func HashRefreshToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// ValidateAccessToken validates and parses a JWT access token
func ValidateAccessToken(tokenString string) (*Claims, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return nil, errors.New("JWT_SECRET not set")
	}

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, ErrInvalidToken
}

// AuthMiddleware validates JWT tokens and adds user context
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "auth_required", "message": "Authorization header required"}})
			c.Abort()
			return
		}

		// Extract token from "Bearer <token>"
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "invalid_auth_format", "message": "Authorization header must be Bearer <token>"}})
			c.Abort()
			return
		}

		tokenString := parts[1]

		// Validate token
		claims, err := ValidateAccessToken(tokenString)
		if err != nil {
			if errors.Is(err, jwt.ErrTokenExpired) {
				c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "token_expired", "message": "Access token expired"}})
			} else {
				c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "invalid_token", "message": "Invalid access token"}})
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
			c.JSON(http.StatusForbidden, gin.H{"error": gin.H{"code": "admin_required", "message": "Admin access required"}})
			c.Abort()
			return
		}
		c.Next()
	}
}

// OwnerMiddleware ensures user owns the resource or is admin
func OwnerMiddleware(resourceUserID string) gin.HandlerFunc {
	return func(c *gin.Context) {
		currentUserID, _ := c.Get("user_id")
		currentUserRole, _ := c.Get("user_role")

		if currentUserRole == "admin" || currentUserID == resourceUserID {
			c.Next()
			return
		}

		c.JSON(http.StatusForbidden, gin.H{"error": gin.H{"code": "access_denied", "message": "Access denied"}})
		c.Abort()
	}
}

// HashPassword hashes a password using bcrypt
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckPasswordHash compares a password with its hash
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
