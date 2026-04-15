package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	// APIKeyHeader is the header name for the API key.
	APIKeyHeader = "X-API-Key"
)

// APIKeyAuth returns middleware that validates requests carry a valid API key.
// In production, keys should come from a database or secrets manager.
// This provides a pluggable foundation — swap ValidateKey for your backing store.
func APIKeyAuth(validKeys map[string]bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := strings.TrimSpace(c.GetHeader(APIKeyHeader))
		if key == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "missing API key: set the X-API-Key header",
			})
			return
		}

		if !validKeys[key] {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "invalid API key",
			})
			return
		}

		c.Next()
	}
}
