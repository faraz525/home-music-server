package server

import (
    "fmt"

    "github.com/gin-gonic/gin"
)

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

    // Simple permissive CORS for now; can be replaced with gin-contrib/cors later
    r.Use(func(c *gin.Context) {
        c.Header("Access-Control-Allow-Origin", "*")
        c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
        c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization, Range")
        c.Header("Access-Control-Expose-Headers", "Content-Range, Accept-Ranges, Content-Length")
        c.Header("Access-Control-Allow-Credentials", "true")
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

