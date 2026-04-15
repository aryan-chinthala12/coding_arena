package middleware

import (
	"github.com/gin-gonic/gin"
)

// SecurityHeaders adds standard security headers to every response.
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Prevent MIME-type sniffing
		c.Header("X-Content-Type-Options", "nosniff")

		// Prevent clickjacking
		c.Header("X-Frame-Options", "DENY")

		// Enable XSS filtering in legacy browsers
		c.Header("X-XSS-Protection", "1; mode=block")

		// Only send referrer for same-origin requests
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		// Restrict resource loading to same origin + inline styles (adjust as needed)
		c.Header("Content-Security-Policy", "default-src 'self'")

		// Opt out of FLoC / Topics API tracking
		c.Header("Permissions-Policy", "interest-cohort=()")

		// Enforce HTTPS (1 year, including subdomains)
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")

		// Prevent caching of API responses containing sensitive data
		c.Header("Cache-Control", "no-store")

		c.Next()
	}
}
