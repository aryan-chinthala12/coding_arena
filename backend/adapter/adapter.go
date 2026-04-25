package adapter

import (
	"fmt"
	"log"

	"github.com/GCET-Open-Source-Foundation/coding_arena/backend/bridge"
)

// LanguageMap maps frontend language names to DMOJ executor IDs.
// Uses a fallback chain per language — the adapter tries the first available executor.
// JAVA8 is used as primary since tier1/tier2 images only ship Java 8;
// tier3 images include JAVA (Java 9+) and GO.
var LanguageMap = map[string][]string{
	"python": {"PY3"},
	"cpp":    {"CPP17", "CPP14", "CPP11"},
	"c":      {"C11", "C"},
	"java":   {"JAVA8", "JAVA"},
	"go":     {"GO"},
}

// resolveExecutor picks the first executor ID from the fallback chain.
func resolveExecutor(language string) (string, error) {
	chain, ok := LanguageMap[language]
	if !ok || len(chain) == 0 {
		return "", fmt.Errorf("unsupported language: %s", language)
	}
	// For now, use the first in the chain.
	// TODO: query judge for available executors and pick the first match.
	return chain[0], nil
}

// DefaultTimeLimit is the default time limit in seconds per test case.
const DefaultTimeLimit = 2.0

// DefaultMemoryLimit is the default memory limit in kilobytes (256 MB).
const DefaultMemoryLimit = 262144

// JudgeAdapter translates between the backend API and the DMOJ bridge.
type JudgeAdapter struct {
	bridge *bridge.Bridge
}

// New creates a new JudgeAdapter backed by the given bridge.
func New(b *bridge.Bridge) *JudgeAdapter {
	return &JudgeAdapter{bridge: b}
}

// Available returns true if at least one judge is connected.
func (a *JudgeAdapter) Available() bool {
	return a.bridge.HasJudge()
}

// SubmissionRequest holds the parameters for a judge submission.
type SubmissionRequest struct {
	ProblemID    string
	Language     string  // frontend name: "python", "cpp", etc.
	Source       string
	TimeLimit    float64 // seconds; 0 = use default
	MemoryLimit  int64   // kilobytes; 0 = use default
	ShortCircuit bool
}

// SubmissionResult is the processed result returned to the handler.
type SubmissionResult struct {
	Status       string       `json:"status"`
	CompileError string       `json:"compile_error,omitempty"`
	Cases        []CaseResult `json:"cases,omitempty"`
	TotalTime    float64      `json:"total_time"`
	MaxMemory    int64        `json:"max_memory_kb"`
	Points       float64      `json:"points"`
	TotalPoints  float64      `json:"total_points"`
}

// CaseResult is a single test case result for the API response.
type CaseResult struct {
	Position int     `json:"position"`
	Status   string  `json:"status"`
	Time     float64 `json:"time"`
	Memory   int64   `json:"memory_kb"`
	Points   float64 `json:"points"`
	Total    float64 `json:"total_points"`
	Feedback string  `json:"feedback,omitempty"`
}

// Submit sends a submission to the DMOJ judge and blocks until the result is ready.
func (a *JudgeAdapter) Submit(req SubmissionRequest) (*SubmissionResult, error) {
	// Map language
	executorID, err := resolveExecutor(req.Language)
	if err != nil {
		return nil, err
	}

	// Apply defaults
	timeLimit := req.TimeLimit
	if timeLimit <= 0 {
		timeLimit = DefaultTimeLimit
	}
	memoryLimit := req.MemoryLimit
	if memoryLimit <= 0 {
		memoryLimit = DefaultMemoryLimit
	}

	log.Printf("[ADAPTER] Submitting problem=%s lang=%s->%s time=%.1fs mem=%dKB",
		req.ProblemID, req.Language, executorID, timeLimit, memoryLimit)

	// Send to bridge (blocks until grading completes)
	raw, err := a.bridge.Submit(req.ProblemID, executorID, req.Source, timeLimit, memoryLimit, req.ShortCircuit)
	if err != nil {
		return nil, fmt.Errorf("judge submission failed: %w", err)
	}

	result := mapResult(raw)

	log.Printf("[ADAPTER] Result: status=%s points=%.1f/%.1f time=%.3fs mem=%dKB cases=%d",
		result.Status, result.Points, result.TotalPoints, result.TotalTime, result.MaxMemory, len(result.Cases))

	return result, nil
}

func mapResult(raw *bridge.SubmissionResult) *SubmissionResult {
	result := &SubmissionResult{
		Status:       raw.Status,
		CompileError: raw.CompileError,
		TotalTime:    raw.TotalTime,
		MaxMemory:    raw.MaxMemory,
		Points:       raw.Points,
		TotalPoints:  raw.TotalPoints,
	}

	for _, c := range raw.Cases {
		result.Cases = append(result.Cases, CaseResult{
			Position: c.Position,
			Status:   bridge.StatusName(c.Status),
			Time:     c.Time,
			Memory:   c.Memory,
			Points:   c.Points,
			Total:    c.Total,
			Feedback: c.Feedback,
		})
	}

	return result
}
