package handler

import (
	"net/http"
	"strconv"

	"github.com/GCET-Open-Source-Foundation/coding_arena/backend/db"
	"github.com/gin-gonic/gin"
)

// ListSubmissions handles GET /submissions?problem=...&verdict=...
func ListSubmissions(c *gin.Context) {
	problem := c.Query("problem")
	verdict := c.Query("verdict")

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
	submissions, err := db.GetSubmissions(c.Request.Context(), problem, verdict, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to retrieve submissions",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"submissions": submissions,
	})
}
