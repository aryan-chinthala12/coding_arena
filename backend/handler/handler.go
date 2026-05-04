package handler

import (
	"net/http"
	"regexp"

	"github.com/GCET-Open-Source-Foundation/coding_arena/backend/adapter"
	"github.com/gin-gonic/gin"
)

var supportedLanguages = map[string]bool{
	"python": true,
	"cpp":    true,
	"c":      true,
	"java":   true,
	"go":     true,
}

var problemIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9\-]{0,63}$`)

const (
	maxCodeLength      = 512 * 1024
	maxProblemIDLength = 64
	submissionIDBytes  = 16
)

// judgeAdapter is set by main.go at startup via SetAdapter.
var judgeAdapter *adapter.JudgeAdapter

// SetAdapter injects the judge adapter into the handler package.
func SetAdapter(a *adapter.JudgeAdapter) {
	judgeAdapter = a
}

/*
	validateInput checks the common fields shared by both
	submit and run requests. Returns an error message string
	if validation fails, or empty string on success.
*/
func validateInput(source string, language string, problemID string) string {
	if len(source) > maxCodeLength {
		return "code exceeds maximum allowed size"
	}

	if !supportedLanguages[language] {
		return "unsupported language"
	}

	if len(problemID) > maxProblemIDLength || !problemIDPattern.MatchString(problemID) {
		return "invalid problem_id: must be 1-64 lowercase alphanumeric characters or hyphens"
	}

	return ""
}

// abortWithValidationError writes a 400 response and returns true if msg is non-empty.
func abortWithValidationError(c *gin.Context, msg string) bool {
	if msg == "" {
		return false
	}
	c.JSON(http.StatusBadRequest, gin.H{"error": msg})
	return true
}
