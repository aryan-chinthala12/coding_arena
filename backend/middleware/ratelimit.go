package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type client struct {
	tokens    float64
	lastSeen  time.Time
}

// RateLimiter implements a token-bucket rate limiter per client IP.
type RateLimiter struct {
	mu       sync.Mutex
	clients  map[string]*client
	rate     float64 // tokens per second
	burst    int     // max tokens (burst capacity)
	cleanup  time.Duration
}

// NewRateLimiter creates a rate limiter. rate is requests/second, burst is max burst size.
func NewRateLimiter(rate float64, burst int) *RateLimiter {
	rl := &RateLimiter{
		clients: make(map[string]*client),
		rate:    rate,
		burst:   burst,
		cleanup: 5 * time.Minute,
	}
	go rl.cleanupLoop()
	return rl
}

// cleanupLoop removes stale entries to prevent memory leaks.
func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(rl.cleanup)
	defer ticker.Stop()
	for range ticker.C {
		rl.mu.Lock()
		for ip, c := range rl.clients {
			if time.Since(c.lastSeen) > rl.cleanup {
				delete(rl.clients, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// allow checks whether the given IP is allowed to make a request.
func (rl *RateLimiter) allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	c, exists := rl.clients[ip]
	now := time.Now()

	if !exists {
		rl.clients[ip] = &client{
			tokens:   float64(rl.burst) - 1,
			lastSeen: now,
		}
		return true
	}

	// Refill tokens based on elapsed time
	elapsed := now.Sub(c.lastSeen).Seconds()
	c.tokens += elapsed * rl.rate
	if c.tokens > float64(rl.burst) {
		c.tokens = float64(rl.burst)
	}
	c.lastSeen = now

	if c.tokens >= 1 {
		c.tokens--
		return true
	}

	return false
}

// Middleware returns a Gin middleware that enforces rate limiting.
func (rl *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		if !rl.allow(ip) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "rate limit exceeded, please try again later",
			})
			return
		}
		c.Next()
	}
}
