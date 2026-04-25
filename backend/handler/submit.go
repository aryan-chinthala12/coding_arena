package handler

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"net/http"
	"regexp"

	"github.com/GCET-Open-Source-Foundation/coding_arena/backend/adapter"
	"github.com/GCET-Open-Source-Foundation/coding_arena/backend/model"
	"github.com/gin-gonic/gin"
)

var supportedLanguages = map[string]bool{
	"python": true,
	"cpp":    true,
	"c":      true,
	"java":   true,
	"go":     true,
}

// problemIDPattern validates problem IDs: lowercase alphanumeric + hyphens, 1-64 chars.
var problemIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9\-]{0,63}$`)

const (
	// maxCodeLength is the maximum allowed source code size in bytes (512 KB).
	maxCodeLength = 512 * 1024
	// maxProblemIDLength is the maximum length for a problem ID.
	maxProblemIDLength = 64
	// submissionIDBytes is the number of random bytes for submission IDs (16 bytes = 128 bits).
	submissionIDBytes = 16
)

// judgeAdapter is set by main.go at startup via SetAdapter.
var judgeAdapter *adapter.JudgeAdapter

// SetAdapter injects the judge adapter into the handler package.
func SetAdapter(a *adapter.JudgeAdapter) {
	judgeAdapter = a
}

// Submit handles POST /submit — accepts code, language, and problem ID.
func Submit(c *gin.Context) {
	var req model.SubmitRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "missing or invalid fields: code, language, and problem_id are required",
		})
		return
	}

	// --- Input validation ---

	// Validate code length
	if len(req.Code) > maxCodeLength {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "code exceeds maximum allowed size",
		})
		return
	}

	// Validate language against whitelist (no user input reflected in response)
	if !supportedLanguages[req.Language] {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "unsupported language",
		})
		return
	}

	// Validate problem_id format (prevent injection via path traversal, NoSQL, etc.)
	if len(req.ProblemID) > maxProblemIDLength || !problemIDPattern.MatchString(req.ProblemID) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid problem_id: must be 1-64 lowercase alphanumeric characters or hyphens",
		})
		return
	}

	// --- Generate collision-safe submission ID (128-bit / UUID-equivalent entropy) ---
	b := make([]byte, submissionIDBytes)
	if _, err := rand.Read(b); err != nil {
		log.Printf("[ERROR] failed to generate submission ID: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	submissionID := "sub_" + hex.EncodeToString(b)

	// Log the submission for audit trail (no code content logged to avoid data leaks)
	log.Printf("[INFO] submission received: id=%s language=%s problem=%s ip=%s",
		submissionID, req.Language, req.ProblemID, c.ClientIP())

	// --- Judge integration ---
	// If the judge adapter is available with a connected judge, grade the submission.
	// Otherwise, queue it (graceful degradation).
	resp := model.SubmitResponse{
		ID:        submissionID,
		ProblemID: req.ProblemID,
		Language:  req.Language,
	}

	if judgeAdapter != nil && judgeAdapter.Available() {
		// Send to judge synchronously (blocks until grading completes)
		judgeResult, err := judgeAdapter.Submit(adapter.SubmissionRequest{
			ProblemID:    req.ProblemID,
			Language:     req.Language,
			Source:       req.Code,
			ShortCircuit: false,
		})

		if err != nil {
			log.Printf("[ERROR] judge submission failed for %s: %v", submissionID, err)
			resp.Status = "judge_error"
			resp.Message = "judge grading failed, submission queued for retry"
			c.JSON(http.StatusAccepted, resp)
			return
		}

		// Convert adapter result to API model
		resp.Status = "graded"
		resp.Message = judgeResult.Status
		resp.Result = &model.JudgeResult{
			Verdict:      judgeResult.Status,
			CompileError: judgeResult.CompileError,
			TotalTime:    judgeResult.TotalTime,
			MaxMemory:    judgeResult.MaxMemory,
			Points:       judgeResult.Points,
			TotalPoints:  judgeResult.TotalPoints,
		}
		for _, cr := range judgeResult.Cases {
			resp.Result.Cases = append(resp.Result.Cases, model.JudgeCaseResult{
				Position: cr.Position,
				Status:   cr.Status,
				Time:     cr.Time,
				Memory:   cr.Memory,
				Points:   cr.Points,
				Total:    cr.Total,
				Feedback: cr.Feedback,
			})
		}

		log.Printf("[INFO] submission graded: id=%s verdict=%s points=%.1f/%.1f",
			submissionID, judgeResult.Status, judgeResult.Points, judgeResult.TotalPoints)

		c.JSON(http.StatusOK, resp)
		return
	}

	// No judge available — accept and queue
	resp.Status = "queued"
	resp.Message = "submission received, pending judge execution"
	c.JSON(http.StatusAccepted, resp)
}
