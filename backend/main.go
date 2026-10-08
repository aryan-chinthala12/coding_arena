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
	"github.com/GCET-Open-Source-Foundation/coding_arena/backend/db"
	"github.com/GCET-Open-Source-Foundation/coding_arena/backend/handler"
	"github.com/GCET-Open-Source-Foundation/coding_arena/backend/middleware"
	"github.com/gin-gonic/gin"
)

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	gin.SetMode(gin.ReleaseMode)

	bridgeAddr := envOr("BRIDGE_ADDR", ":9999")
	judgeID := envOr("JUDGE_ID", "coding-arena")
	judgeKey := envOr("JUDGE_KEY", "changeme")
	if judgeKey == "changeme" {
		log.Println("[WARN] Using default JUDGE_KEY — set JUDGE_KEY env var for production.")
	}

	b := bridge.New(bridgeAddr, judgeID, judgeKey)
	if err := b.Start(); err != nil {
		log.Fatalf("failed to start bridge: %v", err)
	}
	defer b.Stop()

	if dbURL := os.Getenv("DATABASE_URL"); dbURL != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := db.Connect(ctx, dbURL); err != nil {
			log.Fatalf("database connection failed: %v", err)
		}
		defer db.Close()
		if err := db.Migrate(ctx); err != nil {
			log.Fatalf("database migration failed: %v", err)
		}
	} else {
		log.Println("[WARN] No DATABASE_URL set — submissions will not be persisted.")
	}

	cfg, err := config.LoadJudgeConfig()
	if err != nil {
		log.Fatalf("failed to load judge config: %v", err)
	}

	adapt := adapter.New(b, cfg)
	handler.SetAdapter(adapt)

	r := gin.New()

	trustedProxies := strings.Split(envOr("TRUSTED_PROXIES", "127.0.0.1,::1"), ",")
	if err := r.SetTrustedProxies(trustedProxies); err != nil {
		log.Fatalf("failed to set trusted proxies: %v", err)
	}

	r.Use(gin.Recovery())
	r.Use(middleware.RequestLogger())
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.MaxBodySize(1 << 20))

	corsConfig := middleware.DefaultCORSConfig()
	if origins := os.Getenv("CORS_ORIGINS"); origins != "" {
		corsConfig.AllowOrigins = strings.Split(origins, ",")
	}
	r.Use(middleware.CORS(corsConfig))

	limiter := middleware.NewRateLimiter(10, 20)
	r.Use(limiter.Middleware())

	apiKeys := make(map[string]bool)
	if raw := os.Getenv("API_KEYS"); raw != "" {
		for _, k := range strings.Split(raw, ",") {
			if key := strings.TrimSpace(k); key != "" {
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
		log.Println("[WARN] No API_KEYS configured — authentication is DISABLED.")
	}
	authed.POST("/submit", handler.Submit)
	authed.POST("/run", handler.Run)
	authed.GET("/submissions", handler.ListSubmissions)

	port := envOr("PORT", "8080")
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		log.Printf("[INFO] Backend starting on :%s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("[INFO] Shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("forced shutdown: %v", err)
	}
	log.Println("[INFO] Server exited cleanly")
}
