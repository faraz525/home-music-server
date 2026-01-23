package server

import (
    "fmt"
    "os"

    "github.com/gin-gonic/gin"
)

// getAllowedOrigins returns the list of allowed CORS origins based on environment
func getAllowedOrigins() map[string]bool {
    origins := map[string]bool{
        "http://localhost":      true,
        "http://localhost:80":   true,
        "http://localhost:5173": true, // Vite dev server
        "http://127.0.0.1":      true,
        "http://127.0.0.1:80":   true,
        "http://127.0.0.1:5173": true,
    }

    // Add production BASE_URL if set
    if baseURL := os.Getenv("BASE_URL"); baseURL != "" {
        origins[baseURL] = true
    }

    return origins
}

// NewRouter constructs a Gin engine with common middleware and returns
// both the engine and the versioned API group.
func NewRouter() (*gin.Engine, *gin.RouterGroup) {
    r := gin.New()

    r.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
        return fmt.Sprintf("[CrateDrop] %s | %3d | %13v | %15s | %-7s %s\n",
            param.TimeStamp.Format("2006/01/02 15:04:05"),
            param.StatusCode,
            param.Latency,
            param.ClientIP,
            param.Method,
            param.Path,
        )
    }))
    r.Use(gin.Recovery())

    // CORS middleware with explicit origin allowlist
    allowedOrigins := getAllowedOrigins()
    r.Use(func(c *gin.Context) {
        origin := c.GetHeader("Origin")

        // Only set CORS headers if origin is in allowlist
        if allowedOrigins[origin] {
            c.Header("Access-Control-Allow-Origin", origin)
            c.Header("Access-Control-Allow-Credentials", "true")
        }

        c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
        c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization, Range")
        c.Header("Access-Control-Expose-Headers", "Content-Range, Accept-Ranges, Content-Length")

        if c.Request.Method == "OPTIONS" {
            c.AbortWithStatus(204)
            return
        }
        c.Next()
    })

    // Health checks
    r.GET("/api/healthz", func(c *gin.Context) {
        c.JSON(200, gin.H{"status": "ok", "service": "cratedrop-backend", "version": "v0"})
    })
    r.GET("/healthz", func(c *gin.Context) {
        c.JSON(200, gin.H{"status": "ok", "service": "cratedrop-backend", "version": "v0"})
    })

    // Versioned API group
    api := r.Group("/api")
    v1 := api.Group("/v1")

    // Maintain backward compatibility for /api while we migrate by returning v1 for now
    _ = v1

    return r, api
}

