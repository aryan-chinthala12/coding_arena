package middleware

import (
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

/*
	Tests for RateLimiter middleware.
	Covers first request, burst capacity, exceeding limits, token refill,
	and concurrent access under the race detector.
*/

func TestRateLimiter_FirstRequest(t *testing.T) {
	limiter := NewRateLimiter(10.0, 20)
	router := setupTestRouter()
	router.Use(limiter.Middleware())
	router.GET("/test", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })

	w := makeRequest(router, "GET", "/test", nil)
	assertStatus(t, w, 200)
}

func TestRateLimiter_BurstCapacity(t *testing.T) {
	limiter := NewRateLimiter(1.0, 5)
	router := setupTestRouter()
	router.Use(limiter.Middleware())
	router.GET("/test", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })

	for i := 0; i < 5; i++ {
		w := makeRequest(router, "GET", "/test", nil)
		if w.Code != 200 {
			t.Errorf("request %d: expected 200, got %d", i+1, w.Code)
		}
	}
}

func TestRateLimiter_ExceedsLimit(t *testing.T) {
	limiter := NewRateLimiter(1.0, 2)
	router := setupTestRouter()
	router.Use(limiter.Middleware())
	router.GET("/test", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })

	// Exhaust burst
	for i := 0; i < 2; i++ {
		makeRequest(router, "GET", "/test", nil)
	}

	w := makeRequest(router, "GET", "/test", nil)
	assertStatus(t, w, 429)
}

func TestRateLimiter_TokenRefill(t *testing.T) {
	limiter := NewRateLimiter(2.0, 2)
	router := setupTestRouter()
	router.Use(limiter.Middleware())
	router.GET("/test", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })

	// Exhaust burst
	for i := 0; i < 2; i++ {
		makeRequest(router, "GET", "/test", nil)
	}
	assertStatus(t, makeRequest(router, "GET", "/test", nil), 429)

	// Wait for refill (2 req/s → 1 token in 500ms, using 600ms for margin)
	time.Sleep(600 * time.Millisecond)

	assertStatus(t, makeRequest(router, "GET", "/test", nil), 200)
}

func TestRateLimiter_Concurrency(t *testing.T) {
	limiter := NewRateLimiter(10.0, 20)
	router := setupTestRouter()
	router.Use(limiter.Middleware())
	router.GET("/test", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })

	var wg sync.WaitGroup
	var mu sync.Mutex
	successCount, rateLimitedCount := 0, 0

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w := makeRequest(router, "GET", "/test", nil)
			mu.Lock()
			switch w.Code {
			case 200:
				successCount++
			case 429:
				rateLimitedCount++
			}
			mu.Unlock()
		}()
	}
	wg.Wait()

	if successCount > 20 {
		t.Errorf("expected at most 20 successful requests, got %d", successCount)
	}
	if successCount+rateLimitedCount != 50 {
		t.Errorf("expected 50 total responses, got %d", successCount+rateLimitedCount)
	}
	t.Logf("concurrent: %d succeeded, %d rate-limited", successCount, rateLimitedCount)
}
