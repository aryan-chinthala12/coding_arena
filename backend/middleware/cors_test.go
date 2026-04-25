package middleware

import (
	"testing"

	"github.com/gin-gonic/gin"
)

/**
 * Tests for CORS middleware.
 * Covers allowed/disallowed origins, preflight OPTIONS, missing origin,
 * empty allowlist, and Vary header propagation.
 */

var testCORSConfig = CORSConfig{
	AllowOrigins: []string{"http://localhost:3000"},
	AllowMethods: []string{"GET", "POST"},
	AllowHeaders: []string{"Content-Type"},
	MaxAge:       3600,
}

func TestCORS_AllowedOrigin(t *testing.T) {
	router := setupTestRouter()
	router.Use(CORS(testCORSConfig))
	router.GET("/test", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })

	w := makeRequest(router, "GET", "/test", map[string]string{"Origin": "http://localhost:3000"})
	assertHeader(t, w, "Access-Control-Allow-Origin", "http://localhost:3000")
	assertHeaderExists(t, w, "Access-Control-Allow-Methods")
	assertHeaderExists(t, w, "Access-Control-Allow-Headers")
	assertHeaderExists(t, w, "Vary")
}

func TestCORS_DisallowedOrigin(t *testing.T) {
	router := setupTestRouter()
	router.Use(CORS(testCORSConfig))
	router.GET("/test", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })

	w := makeRequest(router, "GET", "/test", map[string]string{"Origin": "http://evil.com"})
	assertHeaderNotExists(t, w, "Access-Control-Allow-Origin")
}

func TestCORS_PreflightRequest(t *testing.T) {
	router := setupTestRouter()
	router.Use(CORS(testCORSConfig))
	router.POST("/test", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })

	w := makeRequest(router, "OPTIONS", "/test", map[string]string{"Origin": "http://localhost:3000"})
	assertStatus(t, w, 204)
	assertHeader(t, w, "Access-Control-Allow-Origin", "http://localhost:3000")
}

func TestCORS_MissingOrigin(t *testing.T) {
	router := setupTestRouter()
	router.Use(CORS(testCORSConfig))
	router.GET("/test", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })

	w := makeRequest(router, "GET", "/test", nil)
	assertStatus(t, w, 200)
	assertHeaderNotExists(t, w, "Access-Control-Allow-Origin")
}

func TestCORS_EmptyAllowedOrigins(t *testing.T) {
	emptyConfig := CORSConfig{
		AllowOrigins: []string{},
		AllowMethods: []string{"GET", "POST"},
		AllowHeaders: []string{"Content-Type"},
		MaxAge:       3600,
	}
	router := setupTestRouter()
	router.Use(CORS(emptyConfig))
	router.GET("/test", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })

	w := makeRequest(router, "GET", "/test", map[string]string{"Origin": "http://localhost:3000"})
	assertHeaderNotExists(t, w, "Access-Control-Allow-Origin")
}

func TestCORS_VaryHeader(t *testing.T) {
	router := setupTestRouter()
	router.Use(CORS(testCORSConfig))
	router.GET("/test", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })

	w := makeRequest(router, "GET", "/test", map[string]string{"Origin": "http://localhost:3000"})
	assertHeader(t, w, "Vary", "Origin")
}
