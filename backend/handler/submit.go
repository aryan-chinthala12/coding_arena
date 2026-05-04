package handler

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"net/http"

	"github.com/GCET-Open-Source-Foundation/coding_arena/backend/adapter"
	"github.com/GCET-Open-Source-Foundation/coding_arena/backend/model"
	"github.com/gin-gonic/gin"
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

	if abortWithValidationError(c, validateInput(req.Code, req.Language, req.ProblemID)) {
		return
	}

	b := make([]byte, submissionIDBytes)
	if _, err := rand.Read(b); err != nil {
		log.Printf("[ERROR] failed to generate submission ID: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	submissionID := "sub_" + hex.EncodeToString(b)

	log.Printf("[INFO] submission received: id=%s language=%s problem=%s ip=%s",
		submissionID, req.Language, req.ProblemID, c.ClientIP())

	/*
		If the judge adapter is available with a connected judge, grade the submission.
		Otherwise, queue it (graceful degradation).
	*/
	resp := model.SubmitResponse{
		ID:        submissionID,
		ProblemID: req.ProblemID,
		Language:  req.Language,
	}

	if judgeAdapter != nil && judgeAdapter.Available() {
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

	resp.Status = "queued"
	resp.Message = "submission received, pending judge execution"
	c.JSON(http.StatusAccepted, resp)
}
