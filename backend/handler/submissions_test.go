package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func setupSubmissionsRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.GET("/submissions", ListSubmissions)

	return r
}

func TestListSubmissions(t *testing.T) {
	r := setupSubmissionsRouter()

	req := httptest.NewRequest(http.MethodGet, "/submissions", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	expected := `{"submissions":[]}`
	if w.Body.String() != expected {
		t.Fatalf("expected %s, got %s", expected, w.Body.String())
	}
}
