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

func TestListSubmissionsInvalidOffset(t *testing.T) {
	tests := []struct {
		name   string
		offset string
	}{
		{
			name:   "negative offset",
			offset: "-1",
		},
		{
			name:   "non-numeric offset",
			offset: "abc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := setupSubmissionsRouter()

			req := httptest.NewRequest(
				http.MethodGet,
				"/submissions?offset="+tt.offset,
				nil,
			)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d", w.Code)
			}
		})
	}
}

func TestListSubmissionsInvalidProblem(t *testing.T) {
	r := setupSubmissionsRouter()

	req := httptest.NewRequest(
		http.MethodGet,
		"/submissions?problem=INVALID_PROBLEM!",
		nil,
	)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}
