package auth

import "github.com/gin-gonic/gin"

// Routes registers the auth-related routes on the provided router group.
func Routes(m *Manager) func(*gin.RouterGroup) {
    return func(r *gin.RouterGroup) {
        g := r.Group("/auth")
        g.POST("/signup", SignupHandler(m))
        g.POST("/login", LoginHandler(m))
        g.POST("/refresh", RefreshHandler(m))
        g.POST("/logout", LogoutHandler(m))

        // Protected
        protected := r.Group("")
        protected.Use(AuthMiddleware())
        protected.GET("/me", MeHandler(m))

        admin := protected.Group("")
        admin.Use(AdminMiddleware())
        admin.GET("/users", GetUsersHandler(m))
    }
}

