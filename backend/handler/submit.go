package handler

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"net/http"
	"regexp"

	"github.com/Aerosane/coding_arena/backend/model"
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

	// TODO: Task 5 — send submission to DMOJ judge-server
	// Adapter must map: code->source, language->DMOJ executor ID (PY3, CPP17, etc.)
	// and supply: time-limit, memory-limit, short-circuit, meta from problem config
	resp := model.SubmitResponse{
		ID:        submissionID,
		Status:    "queued",
		ProblemID: req.ProblemID,
		Language:  req.Language,
		Message:   "submission received, pending judge execution",
	}

	c.JSON(http.StatusAccepted, resp)
}
