package middleware

import (
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

/**
 * Tests for MaxBodySize middleware.
 * Handlers must read the body (GetRawData) for MaxBytesReader to trigger,
 * otherwise the limit is never enforced.
 */

func TestMaxBodySize_UnderLimit(t *testing.T) {
	router := setupTestRouter()
	router.Use(MaxBodySize(1024))
	router.POST("/test", func(c *gin.Context) {
		if _, err := c.GetRawData(); err != nil {
			c.JSON(400, gin.H{"error": "request body too large"})
			return
		}
		c.JSON(200, gin.H{"ok": true})
	})

	body := strings.Repeat("a", 512)
	w := makeRequestWithBody(router, "POST", "/test", body, map[string]string{"Content-Type": "text/plain"})
	assertStatus(t, w, 200)
}

func TestMaxBodySize_AtLimit(t *testing.T) {
	router := setupTestRouter()
	router.Use(MaxBodySize(1024))
	router.POST("/test", func(c *gin.Context) {
		if _, err := c.GetRawData(); err != nil {
			c.JSON(400, gin.H{"error": "request body too large"})
			return
		}
		c.JSON(200, gin.H{"ok": true})
	})

	body := strings.Repeat("a", 1024)
	w := makeRequestWithBody(router, "POST", "/test", body, map[string]string{"Content-Type": "text/plain"})
	assertStatus(t, w, 200)
}

func TestMaxBodySize_OverLimit(t *testing.T) {
	router := setupTestRouter()
	router.Use(MaxBodySize(1024))
	router.POST("/test", func(c *gin.Context) {
		if _, err := c.GetRawData(); err != nil {
			c.JSON(400, gin.H{"error": "request body too large"})
			return
		}
		c.JSON(200, gin.H{"ok": true})
	})

	body := strings.Repeat("a", 2048)
	w := makeRequestWithBody(router, "POST", "/test", body, map[string]string{"Content-Type": "text/plain"})
	assertStatus(t, w, 400)
}

func TestMaxBodySize_NoBody(t *testing.T) {
	router := setupTestRouter()
	router.Use(MaxBodySize(1024))
	router.GET("/test", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })

	w := makeRequest(router, "GET", "/test", nil)
	assertStatus(t, w, 200)
}
