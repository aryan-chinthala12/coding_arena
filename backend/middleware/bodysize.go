package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// MaxBodySize returns middleware that rejects requests with bodies larger than maxBytes.
// This prevents memory exhaustion from oversized payloads.
func MaxBodySize(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Body != nil {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		}
		c.Next()
	}
}
