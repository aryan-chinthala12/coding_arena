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
	gin.SetMode(gin.ReleaseMode)

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

	cfg, err := config.LoadJudgeConfig()
	if err != nil {
		log.Fatalf("failed to load judge config: %v", err)
	}

	adapt := adapter.New(b, cfg)
	handler.SetAdapter(adapt)

	r := gin.New()

	/*
		Only trust loopback by default.
		Set TRUSTED_PROXIES env var for your infra.
	*/
	trustedProxies := []string{"127.0.0.1", "::1"}
	if envProxies := os.Getenv("TRUSTED_PROXIES"); envProxies != "" {
		trustedProxies = strings.Split(envProxies, ",")
	}
	if err := r.SetTrustedProxies(trustedProxies); err != nil {
		log.Fatalf("failed to set trusted proxies: %v", err)
	}

	r.Use(gin.Recovery())
	r.Use(middleware.RequestLogger())
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.MaxBodySize(maxBodyBytes))

	/*
		CORS — set CORS_ORIGINS env var to a
		comma-separated list of allowed origins.
	*/
	corsConfig := middleware.DefaultCORSConfig()
	if envOrigins := os.Getenv("CORS_ORIGINS"); envOrigins != "" {
		corsConfig.AllowOrigins = strings.Split(envOrigins, ",")
	}
	r.Use(middleware.CORS(corsConfig))

	limiter := middleware.NewRateLimiter(ratePerSecond, rateBurst)
	r.Use(limiter.Middleware())

	/*
		Load valid API keys from env (comma-separated).
		In production, use a secrets manager.
	*/
	apiKeys := make(map[string]bool)
	if envKeys := os.Getenv("API_KEYS"); envKeys != "" {
		for _, k := range strings.Split(envKeys, ",") {
			key := strings.TrimSpace(k)
			if key != "" {
				apiKeys[key] = true
			}
		}
	}

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

	authed := r.Group("/")
	if len(apiKeys) > 0 {
		authed.Use(middleware.APIKeyAuth(apiKeys))
	} else {
		log.Println("[WARN] No API_KEYS configured — authentication is DISABLED. Set API_KEYS env var for production.")
	}
	authed.POST("/submit", handler.Submit)
	authed.POST("/run", handler.Run)

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

	go func() {
		log.Printf("[INFO] Backend starting on :%s (release mode)", port)
		log.Printf("[INFO] Bridge listening on %s for judge connections", bridgeAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("[INFO] Shutting down server...")

	/*
		Give in-flight requests up to 30 seconds to complete.
	*/
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("forced shutdown: %v", err)
	}
	log.Println("[INFO] Server exited cleanly")
}
