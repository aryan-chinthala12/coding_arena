package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/GCET-Open-Source-Foundation/coding_arena/backend/model"
	"github.com/gin-gonic/gin"
)

func setupRunRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/run", Run)
	return r
}

func TestRun_ValidRequestNoJudge(t *testing.T) {
	r := setupRunRouter()
	body := `{
		"source": "print('hello')",
		"language": "python",
		"problem_id": "hello-world"
	}`
	req := httptest.NewRequest(http.MethodPost, "/run", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	// Since judgeAdapter is nil in tests, it should return 503
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", w.Code)
	}
	
	var resp struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("expected JSON response body, got unmarshal error: %v; body=%q", err, w.Body.String())
	}
	if resp.Status != "unavailable" {
		t.Fatalf("expected status unavailable, got %v", resp.Status)
	}
}

func TestRun_MissingFields(t *testing.T) {
	r := setupRunRouter()
	body := `{"language":"python"}`
	req := httptest.NewRequest(http.MethodPost, "/run", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestRun_UnsupportedLanguage(t *testing.T) {
	r := setupRunRouter()
	body := `{"source":"print(1)","language":"brainfuck","problem_id":"two-sum"}`
	req := httptest.NewRequest(http.MethodPost, "/run", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestRun_CodeTooLarge(t *testing.T) {
	r := setupRunRouter()
	bigCode := strings.Repeat("x", maxCodeLength+1)
	payload := model.RunRequest{Source: bigCode, Language: "python", ProblemID: "test"}
	b, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/run", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for oversized code, got %d", w.Code)
	}
}

func TestRun_InvalidProblemID(t *testing.T) {
	r := setupRunRouter()
	body := `{"source":"print(1)","language":"python","problem_id":"../../etc/passwd"}`
	req := httptest.NewRequest(http.MethodPost, "/run", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid problem_id, got %d", w.Code)
	}
}
