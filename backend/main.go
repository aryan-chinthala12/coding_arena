package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/GCET-Open-Source-Foundation/coding_arena/backend/adapter"
	"github.com/GCET-Open-Source-Foundation/coding_arena/backend/bridge"
	"github.com/GCET-Open-Source-Foundation/coding_arena/backend/config"
	"github.com/GCET-Open-Source-Foundation/coding_arena/backend/handler"
	"github.com/GCET-Open-Source-Foundation/coding_arena/backend/middleware"
	"github.com/gin-gonic/gin"
)

const (
	// maxBodyBytes is the maximum allowed request body size (1 MB).
	maxBodyBytes = 1 << 20 // 1 MiB

	// Server timeouts to prevent Slowloris and connection exhaustion.
	readTimeout  = 10 * time.Second
	writeTimeout = 30 * time.Second
	idleTimeout  = 120 * time.Second

	// Rate limit: 10 requests/second with burst of 20.
	ratePerSecond = 10
	rateBurst     = 20
)

func main() {
	// --- Production mode ---
	gin.SetMode(gin.ReleaseMode)

	// --- DMOJ Judge Bridge ---
	bridgeAddr := os.Getenv("BRIDGE_ADDR")
	if bridgeAddr == "" {
		bridgeAddr = ":9999"
	}
	judgeID := os.Getenv("JUDGE_ID")
	if judgeID == "" {
		judgeID = "coding-arena"
	}
	judgeKey := os.Getenv("JUDGE_KEY")
	if judgeKey == "" {
		judgeKey = "changeme"
		log.Println("[WARN] Using default JUDGE_KEY — set JUDGE_KEY env var for production.")
	}

	b := bridge.New(bridgeAddr, judgeID, judgeKey)
	if err := b.Start(); err != nil {
		log.Fatalf("failed to start bridge: %v", err)
	}
	defer b.Stop()

	// Load Judge Config
	cfg, err := config.LoadJudgeConfig()
	if err != nil {
		log.Fatalf("failed to load judge config: %v", err)
	}

	// Create adapter and inject into handler
	adapt := adapter.New(b, cfg)
	handler.SetAdapter(adapt)

	// --- Router setup (no default middleware — we add our own) ---
	r := gin.New()

	// --- Trusted proxies (CVE-2020-28483 mitigation) ---
	// Only trust loopback by default. Set TRUSTED_PROXIES env var for your infra.
	trustedProxies := []string{"127.0.0.1", "::1"}
	if envProxies := os.Getenv("TRUSTED_PROXIES"); envProxies != "" {
		trustedProxies = strings.Split(envProxies, ",")
	}
	if err := r.SetTrustedProxies(trustedProxies); err != nil {
		log.Fatalf("failed to set trusted proxies: %v", err)
	}

	// --- Global middleware stack (order matters) ---
	r.Use(gin.Recovery())                      // Panic recovery (always first)
	r.Use(middleware.RequestLogger())           // Structured security logging
	r.Use(middleware.SecurityHeaders())         // Security response headers (HSTS, CSP, etc.)
	r.Use(middleware.MaxBodySize(maxBodyBytes)) // Request body size limit (DoS prevention)

	// CORS — set CORS_ORIGINS env var to a comma-separated list of allowed origins.
	corsConfig := middleware.DefaultCORSConfig()
	if envOrigins := os.Getenv("CORS_ORIGINS"); envOrigins != "" {
		corsConfig.AllowOrigins = strings.Split(envOrigins, ",")
	}
	r.Use(middleware.CORS(corsConfig))

	// Rate limiting
	limiter := middleware.NewRateLimiter(ratePerSecond, rateBurst)
	r.Use(limiter.Middleware())

	// --- API key authentication ---
	// Load valid API keys from env (comma-separated). In production, use a secrets manager.
	apiKeys := make(map[string]bool)
	if envKeys := os.Getenv("API_KEYS"); envKeys != "" {
		for _, k := range strings.Split(envKeys, ",") {
			key := strings.TrimSpace(k)
			if key != "" {
				apiKeys[key] = true
			}
		}
	}

	// --- Routes ---
	// Health check is unauthenticated (for load balancers / k8s probes)
	r.GET("/health", func(c *gin.Context) {
		judgeStatus := "disconnected"
		if adapt.Available() {
			judgeStatus = "connected"
		}
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
			"judge":  judgeStatus,
		})
	})

	// Authenticated routes
	authed := r.Group("/")
	if len(apiKeys) > 0 {
		authed.Use(middleware.APIKeyAuth(apiKeys))
	} else {
		log.Println("[WARN] No API_KEYS configured — authentication is DISABLED. Set API_KEYS env var for production.")
	}
	authed.POST("/submit", handler.Submit)
	authed.POST("/run", handler.Run)

	// --- HTTP server with explicit timeouts (Slowloris / connection exhaustion prevention) ---
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		IdleTimeout:  idleTimeout,
	}

	// --- Graceful shutdown ---
	go func() {
		log.Printf("[INFO] Backend starting on :%s (release mode)", port)
		log.Printf("[INFO] Bridge listening on %s for judge connections", bridgeAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("[INFO] Shutting down server...")

	// Give in-flight requests up to 30 seconds to complete
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("forced shutdown: %v", err)
	}
	log.Println("[INFO] Server exited cleanly")
}
