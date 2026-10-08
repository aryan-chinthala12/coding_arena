package db

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type Submission struct {
	ID           string          `json:"id"`
	ProblemID    string          `json:"problem_id"`
	Language     string          `json:"language"`
	Source       string          `json:"-"`
	Verdict      string          `json:"verdict"`
	Points       float64         `json:"points"`
	TotalPoints  float64         `json:"total_points"`
	TotalTime    float64         `json:"total_time"`
	MaxMemory    int64           `json:"max_memory"`
	CompileError string          `json:"compile_error"`
	Cases        json.RawMessage `json:"cases"`
	IP           string          `json:"-"`
	CreatedAt    time.Time       `json:"created_at"`
}

func InsertSubmission(ctx context.Context, s *Submission) error {
	if Pool == nil {
		return nil
	}
	_, err := Pool.Exec(ctx, `
		INSERT INTO submissions (id, problem_id, language, source, verdict, points, total_points, total_time, max_memory, compile_error, cases, ip)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
		s.ID, s.ProblemID, s.Language, s.Source, s.Verdict,
		s.Points, s.TotalPoints, s.TotalTime, s.MaxMemory,
		s.CompileError, s.Cases, s.IP,
	)
	return err
}

func GetSubmissions(ctx context.Context, problem, verdict string, offset int) ([]Submission, error) {
	if Pool == nil {
		return []Submission{}, nil
	}

	query := `SELECT id, problem_id, language, verdict, points, total_points, total_time, max_memory, created_at FROM submissions`
	var conditions []string
	var args []interface{}
	argID := 1

	if problem != "" {
		conditions = append(conditions, fmt.Sprintf("problem_id = $%d", argID))
		args = append(args, problem)
		argID++
	}

	if verdict != "" {
		conditions = append(conditions, fmt.Sprintf("verdict = $%d", argID))
		args = append(args, verdict)
		argID++
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += fmt.Sprintf(
		" ORDER BY created_at DESC, id DESC LIMIT 100 OFFSET $%d",
		argID,
	)
	args = append(args, offset)

	rows, err := Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var submissions []Submission
	for rows.Next() {
		var s Submission
		err := rows.Scan(
			&s.ID,
			&s.ProblemID,
			&s.Language,
			&s.Verdict,
			&s.Points,
			&s.TotalPoints,
			&s.TotalTime,
			&s.MaxMemory,
			&s.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		submissions = append(submissions, s)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if submissions == nil {
		submissions = []Submission{}
	}

	return submissions, nil
}

func MarshalCases(cases any) []byte {
	b, err := json.Marshal(cases)
	if err != nil {
		return []byte("[]")
	}
	return b
}
