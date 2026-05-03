package middleware

import (
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

/*
	Test helpers for middleware tests.
	Provides router setup, request builders, and assertion utilities
	so individual test files stay focused on behavior.
*/

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func makeRequest(router *gin.Engine, method, path string, headers map[string]string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, nil)
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	router.ServeHTTP(w, req)
	return w
}

func makeRequestWithBody(router *gin.Engine, method, path string, body string, headers map[string]string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	var bodyReader io.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, bodyReader)
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	router.ServeHTTP(w, req)
	return w
}

func assertStatus(t *testing.T, w *httptest.ResponseRecorder, expected int) {
	t.Helper()
	if w.Code != expected {
		t.Errorf("status code: expected %d, got %d", expected, w.Code)
	}
}

func assertHeader(t *testing.T, w *httptest.ResponseRecorder, key, expected string) {
	t.Helper()
	if actual := w.Header().Get(key); actual != expected {
		t.Errorf("header %s: expected %q, got %q", key, expected, actual)
	}
}

func assertHeaderExists(t *testing.T, w *httptest.ResponseRecorder, key string) {
	t.Helper()
	if w.Header().Get(key) == "" {
		t.Errorf("header %s: expected present, got empty", key)
	}
}

func assertHeaderNotExists(t *testing.T, w *httptest.ResponseRecorder, key string) {
	t.Helper()
	if v := w.Header().Get(key); v != "" {
		t.Errorf("header %s: expected absent, got %q", key, v)
	}
}
