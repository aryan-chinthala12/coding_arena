package middleware

import (
	"testing"

	"github.com/gin-gonic/gin"
)

/*
	Tests for APIKeyAuth middleware.
	Validates key presence, correctness, whitespace trimming, and case sensitivity.
*/

func TestAPIKeyAuth_ValidKey(t *testing.T) {
	validKeys := map[string]bool{"test-key-123": true, "another-key": true}
	router := setupTestRouter()
	router.Use(APIKeyAuth(validKeys))
	router.GET("/test", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })

	w := makeRequest(router, "GET", "/test", map[string]string{"X-API-Key": "test-key-123"})
	assertStatus(t, w, 200)
}

func TestAPIKeyAuth_InvalidKey(t *testing.T) {
	validKeys := map[string]bool{"test-key-123": true}
	router := setupTestRouter()
	router.Use(APIKeyAuth(validKeys))
	router.GET("/test", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })

	w := makeRequest(router, "GET", "/test", map[string]string{"X-API-Key": "wrong-key"})
	assertStatus(t, w, 403)
}

func TestAPIKeyAuth_MissingKey(t *testing.T) {
	validKeys := map[string]bool{"test-key-123": true}
	router := setupTestRouter()
	router.Use(APIKeyAuth(validKeys))
	router.GET("/test", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })

	w := makeRequest(router, "GET", "/test", nil)
	assertStatus(t, w, 401)
}

func TestAPIKeyAuth_WhitespacePadding(t *testing.T) {
	validKeys := map[string]bool{"test-key-123": true}
	router := setupTestRouter()
	router.Use(APIKeyAuth(validKeys))
	router.GET("/test", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })

	// Key with leading/trailing spaces should be trimmed by middleware
	w := makeRequest(router, "GET", "/test", map[string]string{"X-API-Key": "  test-key-123  "})
	assertStatus(t, w, 200)
}

func TestAPIKeyAuth_EmptyKey(t *testing.T) {
	validKeys := map[string]bool{"test-key-123": true}
	router := setupTestRouter()
	router.Use(APIKeyAuth(validKeys))
	router.GET("/test", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })

	w := makeRequest(router, "GET", "/test", map[string]string{"X-API-Key": ""})
	assertStatus(t, w, 401)
}

func TestAPIKeyAuth_CaseSensitive(t *testing.T) {
	validKeys := map[string]bool{"test-key-123": true}
	router := setupTestRouter()
	router.Use(APIKeyAuth(validKeys))
	router.GET("/test", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })

	w := makeRequest(router, "GET", "/test", map[string]string{"X-API-Key": "TEST-KEY-123"})
	assertStatus(t, w, 403)
}
