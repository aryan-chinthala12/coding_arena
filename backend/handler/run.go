package handler

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"

	"github.com/GCET-Open-Source-Foundation/coding_arena/backend/adapter"
	"github.com/GCET-Open-Source-Foundation/coding_arena/backend/model"
	"github.com/gin-gonic/gin"
)

// Run handles POST /run — runs code against sample test cases.
// Functionally similar to Submit but uses the "run" semantic for the frontend.
func Run(c *gin.Context) {
	var req model.RunRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "missing or invalid fields: source, language, and problem_id are required",
		})
		return
	}

	// --- Input validation (same rules as Submit) ---

	if len(req.Source) > maxCodeLength {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "code exceeds maximum allowed size",
		})
		return
	}

	if !supportedLanguages[req.Language] {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "unsupported language",
		})
		return
	}

	if len(req.ProblemID) > maxProblemIDLength || !problemIDPattern.MatchString(req.ProblemID) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid problem_id: must be 1-64 lowercase alphanumeric characters or hyphens",
		})
		return
	}

	// Generate run ID
	b := make([]byte, submissionIDBytes)
	if _, err := rand.Read(b); err != nil {
		log.Printf("[ERROR] failed to generate run ID: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	runID := "run_" + hex.EncodeToString(b)

	log.Printf("[INFO] run request: id=%s language=%s problem=%s ip=%s",
		runID, req.Language, req.ProblemID, c.ClientIP())

	// If judge is available, grade it
	if judgeAdapter != nil && judgeAdapter.Available() {
		judgeResult, err := judgeAdapter.Submit(adapter.SubmissionRequest{
			ProblemID:    req.ProblemID,
			Language:     req.Language,
			Source:       req.Source,
			ShortCircuit: false,
		})

		if err != nil {
			log.Printf("[ERROR] run failed for %s: %v", runID, err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"run_id":  runID,
				"status":  "error",
				"message": "judge execution failed",
			})
			return
		}

		// Convert to frontend-expected format
		testCases := make([]gin.H, 0, len(judgeResult.Cases))
		for _, cr := range judgeResult.Cases {
			testCases = append(testCases, gin.H{
				"name":            fmt.Sprintf("Test Case %d", cr.Position),
				"status":          cr.Status,
				"time":            cr.Time,
				"memory_kb":       cr.Memory,
				"input":           "",
				"expected_output": "",
				"actual_output":   cr.Feedback,
			})
		}

		c.JSON(http.StatusOK, gin.H{
			"run_id":     runID,
			"status":     judgeResult.Status,
			"message":    judgeResult.Status,
			"test_cases": testCases,
			"timestamp":  0,
		})
		return
	}

	// No judge — return a helpful message
	c.JSON(http.StatusServiceUnavailable, gin.H{
		"run_id":  runID,
		"status":  "unavailable",
		"message": "no judge connected — run unavailable",
	})
}
