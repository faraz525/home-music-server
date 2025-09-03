package auth

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"

	"github.com/faraz525/home-music-server/backend/models"
	"github.com/faraz525/home-music-server/backend/utils"
)

type SignupRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

func SignupHandler(manager *Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req models.SignupRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "invalid_request", "message": err.Error()}})
			return
		}

		tokens, err := manager.Signup(&req)
		if err != nil {
			statusCode := http.StatusInternalServerError
			errorCode := "signup_failed"

			switch err.Error() {
			case "user with this email already exists":
				statusCode = http.StatusConflict
				errorCode = "user_exists"
			case "admin account already exists":
				statusCode = http.StatusForbidden
				errorCode = "admin_exists"
			}

			c.JSON(statusCode, gin.H{"error": gin.H{"code": errorCode, "message": err.Error()}})
			return
		}

		c.JSON(http.StatusCreated, tokens)
	}
}

func LoginHandler(manager *Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req models.LoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "invalid_request", "message": err.Error()}})
			return
		}

		tokens, err := manager.Login(&req)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "login_failed", "message": err.Error()}})
			return
		}

		c.JSON(http.StatusOK, tokens)
	}
}

func RefreshHandler(manager *Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get refresh token from cookie
		refreshToken, err := c.Cookie("refresh_token")
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "refresh_token_required", "message": "Refresh token required"}})
			return
		}

		tokens, err := manager.Refresh(refreshToken)
		if err != nil {
			statusCode := http.StatusUnauthorized
			if err.Error() == "refresh token expired" {
				statusCode = http.StatusUnauthorized
			}
			c.JSON(statusCode, gin.H{"error": gin.H{"code": "refresh_failed", "message": err.Error()}})
			return
		}

		// Set new refresh token in cookie
		domain := os.Getenv("BASE_URL")
		if domain == "" {
			domain = "localhost"
		}

		c.SetCookie("refresh_token", tokens.RefreshToken, int(utils.RefreshTokenDuration.Seconds()), "/", domain, false, true)

		c.JSON(http.StatusOK, tokens)
	}
}

func LogoutHandler(manager *Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get refresh token from cookie
		refreshToken, err := c.Cookie("refresh_token")
		if err == nil {
			manager.Logout(refreshToken)
		}

		// Clear refresh token cookie
		domain := os.Getenv("BASE_URL")
		if domain == "" {
			domain = "localhost"
		}

		c.SetCookie("refresh_token", "", -1, "/", domain, false, true)
		c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
	}
}

func MeHandler(manager *Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Dev bypass: return a mock user so frontend can proceed without DB lookup
		if os.Getenv("DEV_AUTH_BYPASS") == "1" {
			c.JSON(http.StatusOK, gin.H{
				"user": gin.H{
					"id":    "dev-user",
					"email": "dev@local",
					"role":  "admin",
				},
			})
			return
		}
		userID, _ := c.Get("user_id")
		user, err := manager.GetCurrentUser(userID.(string))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "server_error", "message": "Failed to get user"}})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"user": gin.H{
				"id":    user.ID,
				"email": user.Email,
				"role":  user.Role,
			},
		})
	}
}

func GetUsersHandler(manager *Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if user is admin (middleware should handle this, but double-check)
		userRole, _ := c.Get("user_role")
		if userRole != "admin" {
			c.JSON(http.StatusForbidden, gin.H{"error": gin.H{"code": "admin_required", "message": "Admin access required"}})
			return
		}

		users, err := manager.GetUsers()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "server_error", "message": "Failed to fetch users"}})
			return
		}

		c.JSON(http.StatusOK, gin.H{"users": users})
	}
}
