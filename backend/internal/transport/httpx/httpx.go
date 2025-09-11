package httpx

import (
    "github.com/gin-gonic/gin"
)

func JSON(c *gin.Context, status int, data interface{}) {
    c.Status(status)
    c.Header("Content-Type", "application/json")
    c.JSON(status, data)
}

func Error(c *gin.Context, status int, code, message string) {
    c.Status(status)
    c.Header("Content-Type", "application/json")
    c.JSON(status, gin.H{"error": gin.H{"code": code, "message": message}})
}

