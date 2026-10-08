package handler

import (
	"net/http"
	"strconv"

	"github.com/GCET-Open-Source-Foundation/coding_arena/backend/db"
	"github.com/GCET-Open-Source-Foundation/coding_arena/backend/model"
	"github.com/gin-gonic/gin"
)

// ListSubmissions handles GET /submissions?problem=...&verdict=...
func ListSubmissions(c *gin.Context) {
	problem := c.Query("problem")
	verdict := c.Query("verdict")

	if problem != "" && !problemIDPattern.MatchString(problem) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid problem: must be 1-64 lowercase alphanumeric characters or hyphens",
		})
		return
	}

	offset := 0
	if value := c.Query("offset"); value != "" {
		var err error
		offset, err = strconv.Atoi(value)
		if err != nil || offset < 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "offset must be a non-negative integer",
			})
			return
		}
	}

	submissions, err := db.GetSubmissions(
		c.Request.Context(),
		problem,
		verdict,
		offset,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to retrieve submissions",
		})
		return
	}

	response := make([]model.SubmissionResponse, 0, len(submissions))
	for _, submission := range submissions {
		response = append(response, model.SubmissionResponse{
			ID:          submission.ID,
			ProblemID:   submission.ProblemID,
			Language:    submission.Language,
			Verdict:     submission.Verdict,
			Points:      submission.Points,
			TotalPoints: submission.TotalPoints,
			TotalTime:   submission.TotalTime,
			MaxMemory:   submission.MaxMemory,
			CreatedAt:   submission.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"submissions": response,
	})
}
