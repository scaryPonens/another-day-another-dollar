package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// APIKeyAuth returns a Gin middleware that enforces X-API-Key header validation.
// If key is empty, the middleware is a no-op (auth disabled).
func APIKeyAuth(key string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if key == "" {
			c.Next()
			return
		}
		provided := strings.TrimSpace(c.GetHeader("X-API-Key"))
		if provided == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing X-API-Key header"})
			return
		}
		if provided != key {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "invalid API key"})
			return
		}
		c.Next()
	}
}
