package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Aerosane/coding_arena/backend/model"
	"github.com/gin-gonic/gin"
)

func setupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/submit", Submit)
	return r
}

func TestSubmit_ValidPython(t *testing.T) {
	r := setupRouter()
	body := `{
		"code": "def two_sum(nums, target):\n    seen = {}\n    for i, n in enumerate(nums):\n        diff = target - n\n        if diff in seen:\n            return [seen[diff], i]\n        seen[n] = i\n    return []",
		"language": "python",
		"problem_id": "two-sum"
	}`
	req := httptest.NewRequest(http.MethodPost, "/submit", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", w.Code)
	}
	var resp model.SubmitResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Status != "queued" {
		t.Fatalf("expected status queued, got %s", resp.Status)
	}
	if resp.Language != "python" {
		t.Fatalf("expected language python, got %s", resp.Language)
	}
	if !strings.HasPrefix(resp.ID, "sub_") {
		t.Fatalf("expected ID prefix sub_, got %s", resp.ID)
	}
	// 128-bit ID = 16 bytes = 32 hex chars + "sub_" prefix = 36 chars total
	if len(resp.ID) != 36 {
		t.Fatalf("expected 128-bit ID (36 chars with prefix), got %d chars: %s", len(resp.ID), resp.ID)
	}
}

func TestSubmit_ValidCpp(t *testing.T) {
	r := setupRouter()
	code := `#include <vector>
#include <unordered_map>
using namespace std;

class Solution {
public:
    vector<int> twoSum(vector<int>& nums, int target) {
        unordered_map<int, int> seen;
        for (int i = 0; i < nums.size(); i++) {
            int diff = target - nums[i];
            if (seen.count(diff)) return {seen[diff], i};
            seen[nums[i]] = i;
        }
        return {};
    }
};`
	payload := model.SubmitRequest{Code: code, Language: "cpp", ProblemID: "two-sum"}
	b, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/submit", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSubmit_ValidJava(t *testing.T) {
	r := setupRouter()
	code := `import java.util.*;

class Solution {
    public int[] twoSum(int[] nums, int target) {
        Map<Integer, Integer> map = new HashMap<>();
        for (int i = 0; i < nums.length; i++) {
            int complement = target - nums[i];
            if (map.containsKey(complement)) {
                return new int[]{map.get(complement), i};
            }
            map.put(nums[i], i);
        }
        throw new IllegalArgumentException("No solution");
    }
}`
	payload := model.SubmitRequest{Code: code, Language: "java", ProblemID: "two-sum"}
	b, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/submit", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSubmit_ValidGo(t *testing.T) {
	r := setupRouter()
	code := `package main

func twoSum(nums []int, target int) []int {
	seen := make(map[int]int)
	for i, n := range nums {
		if j, ok := seen[target-n]; ok {
			return []int{j, i}
		}
		seen[n] = i
	}
	return nil
}`
	payload := model.SubmitRequest{Code: code, Language: "go", ProblemID: "two-sum"}
	b, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/submit", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSubmit_LargePayload(t *testing.T) {
	r := setupRouter()
	// Simulate a large but valid submission (~20KB of code, under 512KB limit)
	var sb strings.Builder
	sb.WriteString("def solve():\n")
	for i := 0; i < 500; i++ {
		sb.WriteString("    x = x + 1  # line of computation\n")
	}
	payload := model.SubmitRequest{Code: sb.String(), Language: "python", ProblemID: "large-input"}
	b, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/submit", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", w.Code, w.Body.String())
	}
}

// --- Security-specific tests ---

func TestSubmit_CodeTooLarge(t *testing.T) {
	r := setupRouter()
	// Generate code exceeding 512 KB
	bigCode := strings.Repeat("x", 512*1024+1)
	payload := model.SubmitRequest{Code: bigCode, Language: "python", ProblemID: "test"}
	b, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/submit", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for oversized code, got %d", w.Code)
	}
}

func TestSubmit_InvalidProblemID_PathTraversal(t *testing.T) {
	r := setupRouter()
	body := `{"code":"print(1)","language":"python","problem_id":"../../etc/passwd"}`
	req := httptest.NewRequest(http.MethodPost, "/submit", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for path traversal problem_id, got %d", w.Code)
	}
}

func TestSubmit_InvalidProblemID_Injection(t *testing.T) {
	r := setupRouter()
	body := `{"code":"print(1)","language":"python","problem_id":"test'; DROP TABLE--"}`
	req := httptest.NewRequest(http.MethodPost, "/submit", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for injection problem_id, got %d", w.Code)
	}
}

func TestSubmit_InvalidProblemID_TooLong(t *testing.T) {
	r := setupRouter()
	longID := strings.Repeat("a", 65)
	payload := model.SubmitRequest{Code: "print(1)", Language: "python", ProblemID: longID}
	b, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/submit", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for too-long problem_id, got %d", w.Code)
	}
}

func TestSubmit_InvalidProblemID_UpperCase(t *testing.T) {
	r := setupRouter()
	body := `{"code":"print(1)","language":"python","problem_id":"TwoSum"}`
	req := httptest.NewRequest(http.MethodPost, "/submit", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for uppercase problem_id, got %d", w.Code)
	}
}

func TestSubmit_UnsupportedLanguageNoReflection(t *testing.T) {
	r := setupRouter()
	body := `{"code":"print(1)","language":"<script>alert(1)</script>","problem_id":"two-sum"}`
	req := httptest.NewRequest(http.MethodPost, "/submit", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	// Ensure user input is NOT reflected in the response
	if strings.Contains(w.Body.String(), "<script>") {
		t.Fatal("user input was reflected in error response — XSS risk")
	}
}

// --- Existing negative tests ---

func TestSubmit_MissingCode(t *testing.T) {
	r := setupRouter()
	body := `{"language":"python","problem_id":"two-sum"}`
	req := httptest.NewRequest(http.MethodPost, "/submit", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestSubmit_MissingLanguage(t *testing.T) {
	r := setupRouter()
	body := `{"code":"print(1)","problem_id":"two-sum"}`
	req := httptest.NewRequest(http.MethodPost, "/submit", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestSubmit_MissingProblemID(t *testing.T) {
	r := setupRouter()
	body := `{"code":"print(1)","language":"python"}`
	req := httptest.NewRequest(http.MethodPost, "/submit", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestSubmit_UnsupportedLanguage(t *testing.T) {
	r := setupRouter()
	body := `{"code":"print(1)","language":"brainfuck","problem_id":"two-sum"}`
	req := httptest.NewRequest(http.MethodPost, "/submit", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestSubmit_EmptyBody(t *testing.T) {
	r := setupRouter()
	req := httptest.NewRequest(http.MethodPost, "/submit", strings.NewReader(""))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestSubmit_MalformedJSON(t *testing.T) {
	r := setupRouter()
	req := httptest.NewRequest(http.MethodPost, "/submit", strings.NewReader("{bad json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestSubmit_UniqueIDs(t *testing.T) {
	r := setupRouter()
	body := `{"code":"print(1)","language":"python","problem_id":"two-sum"}`
	ids := make(map[string]bool)

	for i := 0; i < 100; i++ {
		req := httptest.NewRequest(http.MethodPost, "/submit", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		var resp model.SubmitResponse
		json.Unmarshal(w.Body.Bytes(), &resp)
		if ids[resp.ID] {
			t.Fatalf("duplicate submission ID: %s", resp.ID)
		}
		ids[resp.ID] = true
	}
}

func TestSubmit_AllLanguages(t *testing.T) {
	r := setupRouter()
	langs := []string{"python", "cpp", "c", "java", "go"}

	for _, lang := range langs {
		t.Run(lang, func(t *testing.T) {
			payload := model.SubmitRequest{Code: "code", Language: lang, ProblemID: "test"}
			b, _ := json.Marshal(payload)
			req := httptest.NewRequest(http.MethodPost, "/submit", bytes.NewReader(b))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			if w.Code != http.StatusAccepted {
				t.Fatalf("language %s: expected 202, got %d", lang, w.Code)
			}
		})
	}
}

func BenchmarkSubmit(b *testing.B) {
	r := setupRouter()
	body := `{"code":"def solve(n):\n    return n * 2","language":"python","problem_id":"two-sum"}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/submit", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
	}
}

func BenchmarkSubmit_LargePayload(b *testing.B) {
	r := setupRouter()
	var sb strings.Builder
	sb.WriteString("def solve():\n")
	for i := 0; i < 500; i++ {
		sb.WriteString("    x = x + 1\n")
	}
	payload := model.SubmitRequest{Code: sb.String(), Language: "python", ProblemID: "heavy"}
	body, _ := json.Marshal(payload)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/submit", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
	}
}
