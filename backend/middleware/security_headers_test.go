package middleware

import (
	"testing"

	"github.com/gin-gonic/gin"
)

/*
	Tests for SecurityHeaders middleware.
	Verifies all 8 security headers are present with correct values,
	including on error (404) responses.
*/

func TestSecurityHeaders_AllPresent(t *testing.T) {
	router := setupTestRouter()
	router.Use(SecurityHeaders())
	router.GET("/test", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })

	w := makeRequest(router, "GET", "/test", nil)

	headers := []string{
		"X-Content-Type-Options", "X-Frame-Options", "X-XSS-Protection",
		"Referrer-Policy", "Content-Security-Policy", "Permissions-Policy",
		"Strict-Transport-Security", "Cache-Control",
	}
	for _, h := range headers {
		assertHeaderExists(t, w, h)
	}
}

func TestSecurityHeaders_CorrectValues(t *testing.T) {
	router := setupTestRouter()
	router.Use(SecurityHeaders())
	router.GET("/test", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })

	w := makeRequest(router, "GET", "/test", nil)

	assertHeader(t, w, "X-Content-Type-Options", "nosniff")
	assertHeader(t, w, "X-Frame-Options", "DENY")
	assertHeader(t, w, "X-XSS-Protection", "1; mode=block")
	assertHeader(t, w, "Referrer-Policy", "strict-origin-when-cross-origin")
	assertHeader(t, w, "Content-Security-Policy", "default-src 'self'")
	assertHeader(t, w, "Permissions-Policy", "interest-cohort=()")
	assertHeader(t, w, "Strict-Transport-Security", "max-age=31536000; includeSubDomains")
	assertHeader(t, w, "Cache-Control", "no-store")
}

func TestSecurityHeaders_OnErrorResponse(t *testing.T) {
	router := setupTestRouter()
	router.Use(SecurityHeaders())
	// No routes — triggers 404

	w := makeRequest(router, "GET", "/nonexistent", nil)
	assertStatus(t, w, 404)

	// Security headers should still be present on error responses
	for _, h := range []string{
		"X-Content-Type-Options", "X-Frame-Options", "X-XSS-Protection",
		"Referrer-Policy", "Content-Security-Policy", "Permissions-Policy",
		"Strict-Transport-Security", "Cache-Control",
	} {
		assertHeaderExists(t, w, h)
	}
}
