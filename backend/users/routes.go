package users

import (
	"github.com/faraz525/home-music-server/backend/auth"
	"github.com/gin-gonic/gin"
)

// RegisterRoutes registers all user routes
func RegisterRoutes(router *gin.RouterGroup, manager *Manager) {
	// User search and profile routes (authenticated)
	router.GET("/users/search", auth.AuthMiddleware(), SearchUsersHandler(manager))
	router.GET("/users/:username", auth.AuthMiddleware(), GetUserByUsernameHandler(manager))
	router.PUT("/users/me/username", auth.AuthMiddleware(), UpdateUsernameHandler(manager))
}
