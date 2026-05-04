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
func Run(c *gin.Context) {
	var req model.RunRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "missing or invalid fields: source, language, and problem_id are required",
		})
		return
	}

	if abortWithValidationError(c, validateInput(req.Source, req.Language, req.ProblemID)) {
		return
	}

	b := make([]byte, submissionIDBytes)
	if _, err := rand.Read(b); err != nil {
		log.Printf("[ERROR] failed to generate run ID: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	runID := "run_" + hex.EncodeToString(b)

	log.Printf("[INFO] run request: id=%s language=%s problem=%s ip=%s",
		runID, req.Language, req.ProblemID, c.ClientIP())

	if judgeAdapter != nil && judgeAdapter.Available() {
		judgeResult, err := judgeAdapter.Submit(adapter.SubmissionRequest{
			ProblemID:    req.ProblemID,
			Language:     req.Language,
			Source:       req.Source,
			ShortCircuit: false,
		})

		if err != nil {
			log.Printf("[ERROR] run failed for %s: %v", runID, err)
			c.JSON(http.StatusInternalServerError, model.RunResponse{
				RunID:   runID,
				Status:  "error",
				Message: "judge execution failed",
			})
			return
		}

		testCases := make([]model.RunCaseResult, 0, len(judgeResult.Cases))
		for _, cr := range judgeResult.Cases {
			testCases = append(testCases, model.RunCaseResult{
				Name:         fmt.Sprintf("Test Case %d", cr.Position),
				Status:       cr.Status,
				Time:         cr.Time,
				MemoryKB:     cr.Memory,
				ActualOutput: cr.Feedback,
			})
		}

		c.JSON(http.StatusOK, model.RunResponse{
			RunID:     runID,
			Status:    judgeResult.Status,
			Message:   judgeResult.Status,
			TestCases: testCases,
		})
		return
	}

	c.JSON(http.StatusServiceUnavailable, model.RunResponse{
		RunID:   runID,
		Status:  "unavailable",
		Message: "no judge connected — run unavailable",
	})
}
