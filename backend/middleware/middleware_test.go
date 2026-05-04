package middleware

import (
	"io"
	"log"
	"testing"

	"github.com/gin-gonic/gin"
)

/*
	Integration tests for middleware stack.
	Verifies correct behavior when multiple middleware are combined,
	including execution ordering, error propagation, and the full
	production-like stack from main.go.
*/

func TestMiddlewareIntegration(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("RateLimiter_And_Auth", func(t *testing.T) {
		limiter := NewRateLimiter(1.0, 1)
		validKeys := map[string]bool{"test-key": true}

		router := setupTestRouter()
		router.Use(limiter.Middleware())
		router.Use(APIKeyAuth(validKeys))
		router.GET("/test", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })

		w1 := makeRequest(router, "GET", "/test", map[string]string{"X-API-Key": "test-key"})
		assertStatus(t, w1, 200)

		// Second request hits rate limiter before auth
		w2 := makeRequest(router, "GET", "/test", map[string]string{"X-API-Key": "test-key"})
		assertStatus(t, w2, 429)
	})

	t.Run("CORS_And_SecurityHeaders", func(t *testing.T) {
		corsConfig := CORSConfig{
			AllowOrigins: []string{"http://localhost:3000"},
			AllowMethods: []string{"GET", "POST"},
			AllowHeaders: []string{"Content-Type"},
			MaxAge:       3600,
		}

		router := setupTestRouter()
		router.Use(CORS(corsConfig))
		router.Use(SecurityHeaders())
		router.GET("/test", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })

		w := makeRequest(router, "GET", "/test", map[string]string{"Origin": "http://localhost:3000"})
		assertHeader(t, w, "Access-Control-Allow-Origin", "http://localhost:3000")
		assertHeaderExists(t, w, "X-Frame-Options")
		assertHeaderExists(t, w, "X-Content-Type-Options")
		assertHeaderExists(t, w, "Strict-Transport-Security")
	})

	t.Run("FullMiddlewareStack", func(t *testing.T) {
		limiter := NewRateLimiter(10.0, 20)
		validKeys := map[string]bool{"prod-key": true}
		corsConfig := CORSConfig{
			AllowOrigins: []string{"http://localhost:3000"},
			AllowMethods: []string{"GET", "POST"},
			AllowHeaders: []string{"Content-Type", "X-API-Key"},
			MaxAge:       3600,
		}

		// Discard log output during benchmark-style test to avoid noise
		log.SetOutput(io.Discard)
		defer log.SetOutput(nil)

		router := setupTestRouter()
		router.Use(gin.Recovery())
		router.Use(RequestLogger())
		router.Use(SecurityHeaders())
		router.Use(MaxBodySize(1024))
		router.Use(CORS(corsConfig))
		router.Use(limiter.Middleware())
		router.Use(APIKeyAuth(validKeys))
		router.GET("/test", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })

		w := makeRequest(router, "GET", "/test", map[string]string{
			"Origin":    "http://localhost:3000",
			"X-API-Key": "prod-key",
		})
		assertStatus(t, w, 200)
		assertHeader(t, w, "Access-Control-Allow-Origin", "http://localhost:3000")
		assertHeaderExists(t, w, "X-Frame-Options")
		assertHeaderExists(t, w, "Cache-Control")
	})

	t.Run("ErrorPropagation", func(t *testing.T) {
		validKeys := map[string]bool{"valid-key": true}

		router := setupTestRouter()
		router.Use(SecurityHeaders())
		router.Use(APIKeyAuth(validKeys))
		router.GET("/test", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })

		// Auth aborts, but security headers (before auth) should still be set
		w := makeRequest(router, "GET", "/test", nil)
		assertStatus(t, w, 401)
		assertHeaderExists(t, w, "X-Frame-Options")
		assertHeaderExists(t, w, "X-Content-Type-Options")
	})
}

/*
	TestMiddlewareOrder verifies that middleware execution order
	determines which middleware handles the request first.
*/
func TestMiddlewareOrder(t *testing.T) {
	t.Run("RateLimiter_Before_Auth", func(t *testing.T) {
		limiter := NewRateLimiter(1.0, 1)
		validKeys := map[string]bool{"test-key": true}

		router := setupTestRouter()
		router.Use(limiter.Middleware())
		router.Use(APIKeyAuth(validKeys))
		router.GET("/test", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })

		makeRequest(router, "GET", "/test", map[string]string{"X-API-Key": "test-key"})

		w := makeRequest(router, "GET", "/test", map[string]string{"X-API-Key": "test-key"})
		assertStatus(t, w, 429)
	})

	t.Run("Auth_Before_RateLimiter", func(t *testing.T) {
		limiter := NewRateLimiter(1.0, 1)
		validKeys := map[string]bool{"test-key": true}

		router := setupTestRouter()
		router.Use(APIKeyAuth(validKeys))
		router.Use(limiter.Middleware())
		router.GET("/test", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })

		w := makeRequest(router, "GET", "/test", nil)
		assertStatus(t, w, 401)
	})
}

func BenchmarkMiddlewareStack(b *testing.B) {
	limiter := NewRateLimiter(1000.0, 2000)
	validKeys := map[string]bool{"bench-key": true}
	corsConfig := CORSConfig{
		AllowOrigins: []string{"http://localhost:3000"},
		AllowMethods: []string{"GET", "POST"},
		AllowHeaders: []string{"Content-Type", "X-API-Key"},
		MaxAge:       3600,
	}

	// Discard log output so benchmark measures middleware, not I/O
	log.SetOutput(io.Discard)
	defer log.SetOutput(nil)

	router := setupTestRouter()
	router.Use(gin.Recovery())
	router.Use(RequestLogger())
	router.Use(SecurityHeaders())
	router.Use(MaxBodySize(1024))
	router.Use(CORS(corsConfig))
	router.Use(limiter.Middleware())
	router.Use(APIKeyAuth(validKeys))
	router.GET("/test", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		makeRequest(router, "GET", "/test", map[string]string{
			"Origin":    "http://localhost:3000",
			"X-API-Key": "bench-key",
		})
	}
}
