package auth

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"

	imodels "github.com/faraz525/home-music-server/backend/internal/models"
	"github.com/faraz525/home-music-server/backend/utils"
)

// getCookieDomain determines the appropriate domain for cookies based on BASE_URL
func getCookieDomain() string {
	baseURL := os.Getenv("BASE_URL")
	var domain string

	// For localhost/development, don't set a domain to allow access from any IP
	// For production, use the specified domain
	if baseURL == "" || baseURL == "http://localhost" || strings.Contains(baseURL, "localhost") {
		domain = "" // Empty domain allows cookies to work from any domain
	} else {
		domain = baseURL
		// Remove protocol from domain for cookie
		if strings.HasPrefix(domain, "http://") {
			domain = domain[7:]
		} else if strings.HasPrefix(domain, "https://") {
			domain = domain[8:]
		}
	}

	return domain
}

// setAuthCookies sets both access and refresh token cookies with the appropriate domain
func setAuthCookies(c *gin.Context, tokens *imodels.Tokens) {
	domain := getCookieDomain()
	c.SetCookie("refresh_token", tokens.RefreshToken, int(utils.RefreshTokenDuration.Seconds()), "/", domain, false, true)
	c.SetCookie("access_token", tokens.AccessToken, 60*15, "/", domain, false, true)
}

// clearAuthCookies clears both access and refresh token cookies
func clearAuthCookies(c *gin.Context) {
	domain := getCookieDomain()
	c.SetCookie("refresh_token", "", -1, "/", domain, false, true)
	c.SetCookie("access_token", "", -1, "/", domain, false, true)
}

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
		var req imodels.SignupRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "invalid_request", "message": err.Error()}})
			return
		}

		tokens, err := manager.Signup(c.Request.Context(), &req)
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

		setAuthCookies(c, tokens)

		c.JSON(http.StatusCreated, tokens)
	}
}

func LoginHandler(manager *Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req imodels.LoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "invalid_request", "message": err.Error()}})
			return
		}

		tokens, err := manager.Login(c.Request.Context(), &req)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "login_failed", "message": err.Error()}})
			return
		}

		setAuthCookies(c, tokens)

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

		tokens, err := manager.Refresh(c.Request.Context(), refreshToken)
		if err != nil {
			statusCode := http.StatusUnauthorized
			if err.Error() == "refresh token expired" {
				statusCode = http.StatusUnauthorized
			}
			c.JSON(statusCode, gin.H{"error": gin.H{"code": "refresh_failed", "message": err.Error()}})
			return
		}

		// Set new tokens in cookies
		setAuthCookies(c, tokens)

		c.JSON(http.StatusOK, tokens)
	}
}

func LogoutHandler(manager *Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get refresh token from cookie
		refreshToken, err := c.Cookie("refresh_token")
		if err == nil {
			manager.Logout(c.Request.Context(), refreshToken)
		}

		// Clear both cookies
		clearAuthCookies(c)
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
		user, err := manager.GetCurrentUser(c.Request.Context(), userID.(string))
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

		users, err := manager.GetUsers(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "server_error", "message": "Failed to fetch users"}})
			return
		}

		c.JSON(http.StatusOK, gin.H{"users": users})
	}
}
