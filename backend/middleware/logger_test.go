package middleware

import (
	"bytes"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

/*
	Tests for RequestLogger middleware.
	Verifies log level (INFO for 2xx, WARN for 4xx/5xx) and log content
	(method, path, status code, IP, latency).
*/

func TestRequestLogger_SuccessfulRequest(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	router := setupTestRouter()
	router.Use(RequestLogger())
	router.GET("/test", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })

	makeRequest(router, "GET", "/test", nil)

	out := buf.String()
	if !strings.Contains(out, "[INFO]") {
		t.Errorf("expected [INFO] in log, got: %s", out)
	}
	if !strings.Contains(out, "GET") || !strings.Contains(out, "/test") {
		t.Errorf("expected method and path in log, got: %s", out)
	}
}

func TestRequestLogger_ClientError(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	router := setupTestRouter()
	router.Use(RequestLogger())
	router.GET("/test", func(c *gin.Context) { c.JSON(400, gin.H{"error": "bad request"}) })

	makeRequest(router, "GET", "/test", nil)

	if !strings.Contains(buf.String(), "[WARN]") {
		t.Errorf("expected [WARN] for 4xx, got: %s", buf.String())
	}
}

func TestRequestLogger_ServerError(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	router := setupTestRouter()
	router.Use(RequestLogger())
	router.GET("/test", func(c *gin.Context) { c.JSON(500, gin.H{"error": "internal error"}) })

	makeRequest(router, "GET", "/test", nil)

	if !strings.Contains(buf.String(), "[WARN]") {
		t.Errorf("expected [WARN] for 5xx, got: %s", buf.String())
	}
}

func TestRequestLogger_LogContent(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	router := setupTestRouter()
	router.Use(RequestLogger())
	router.POST("/api/test", func(c *gin.Context) { c.JSON(201, gin.H{"created": true}) })

	makeRequest(router, "POST", "/api/test", nil)

	out := buf.String()
	for _, field := range []string{"POST", "/api/test", "201", "ip="} {
		if !strings.Contains(out, field) {
			t.Errorf("expected %q in log, got: %s", field, out)
		}
	}
}
