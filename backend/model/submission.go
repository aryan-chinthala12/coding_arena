package model

// SubmitRequest represents an incoming code submission.
type SubmitRequest struct {
	Code      string `json:"code" binding:"required"`
	Language  string `json:"language" binding:"required"`
	ProblemID string `json:"problem_id" binding:"required"`
}

// RunRequest represents an incoming run/test request from the frontend.
// The frontend sends "source" instead of "code" for run requests.
type RunRequest struct {
	Source      string  `json:"source" binding:"required"`
	Language    string  `json:"language" binding:"required"`
	ProblemID   string  `json:"problem_id" binding:"required"`
	CustomInput *string `json:"custom_input"`
}

// SubmitResponse represents the result of a submission.
type SubmitResponse struct {
	ID        string       `json:"id"`
	Status    string       `json:"status"`
	ProblemID string       `json:"problem_id"`
	Language  string       `json:"language"`
	Message   string       `json:"message,omitempty"`
	Result    *JudgeResult `json:"result,omitempty"`
}

// JudgeResult holds the detailed grading result from the DMOJ judge.
type JudgeResult struct {
	Verdict      string           `json:"verdict"`
	CompileError string           `json:"compile_error,omitempty"`
	Cases        []JudgeCaseResult `json:"cases,omitempty"`
	TotalTime    float64          `json:"total_time"`
	MaxMemory    int64            `json:"max_memory_kb"`
	Points       float64          `json:"points"`
	TotalPoints  float64          `json:"total_points"`
}

// JudgeCaseResult holds the result of a single test case.
type JudgeCaseResult struct {
	Position int     `json:"position"`
	Status   string  `json:"status"`
	Time     float64 `json:"time"`
	Memory   int64   `json:"memory_kb"`
	Points   float64 `json:"points"`
	Total    float64 `json:"total_points"`
	Feedback string  `json:"feedback,omitempty"`
}

// RunResponse represents the result of a run/test request.
type RunResponse struct {
	RunID     string         `json:"run_id"`
	Status    string         `json:"status"`
	Message   string         `json:"message"`
	TestCases []RunCaseResult `json:"test_cases,omitempty"`
}

// RunCaseResult holds a single test case result for the run endpoint.
type RunCaseResult struct {
	Name           string  `json:"name"`
	Status         string  `json:"status"`
	Time           float64 `json:"time"`
	MemoryKB       int64   `json:"memory_kb"`
	Input          string  `json:"input"`
	ExpectedOutput string  `json:"expected_output"`
	ActualOutput   string  `json:"actual_output"`
}
