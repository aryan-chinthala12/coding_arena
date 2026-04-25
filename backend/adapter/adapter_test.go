package adapter

import (
	"testing"

	"github.com/GCET-Open-Source-Foundation/coding_arena/backend/bridge"
)

func TestResolveExecutor(t *testing.T) {
	tests := []struct {
		language    string
		expectedID  string
		expectError bool
	}{
		{"python", "PY3", false},
		{"cpp", "CPP17", false},
		{"c", "C11", false},
		{"java", "JAVA8", false},
		{"go", "GO", false},
		{"unknown", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.language, func(t *testing.T) {
			id, err := resolveExecutor(tt.language)
			if tt.expectError {
				if err == nil {
					t.Fatalf("expected error for language %s, got none", tt.language)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error for language %s: %v", tt.language, err)
				}
				if id != tt.expectedID {
					t.Fatalf("expected id %s, got %s", tt.expectedID, id)
				}
			}
		})
	}
}

func TestMapResult(t *testing.T) {
	raw := &bridge.SubmissionResult{
		Status:       "AC",
		CompileError: "",
		TotalTime:    1.5,
		MaxMemory:    1024,
		Points:       10,
		TotalPoints:  10,
		Cases: []bridge.CaseResult{
			{Position: 1, Status: bridge.StatusAC, Time: 0.5, Memory: 512, Points: 5, Total: 5, Feedback: "Good"},
			{Position: 2, Status: bridge.StatusWA, Time: 1.0, Memory: 1024, Points: 5, Total: 5, Feedback: "Bad"},
		},
	}

	mapped := mapResult(raw)

	if mapped.Status != "AC" {
		t.Errorf("expected status AC, got %s", mapped.Status)
	}
	if mapped.TotalTime != 1.5 {
		t.Errorf("expected total time 1.5, got %f", mapped.TotalTime)
	}
	if mapped.MaxMemory != 1024 {
		t.Errorf("expected max memory 1024, got %d", mapped.MaxMemory)
	}
	if len(mapped.Cases) != 2 {
		t.Fatalf("expected 2 cases, got %d", len(mapped.Cases))
	}

	if mapped.Cases[0].Status != "AC" {
		t.Errorf("expected case 1 status AC, got %s", mapped.Cases[0].Status)
	}
	if mapped.Cases[1].Status != "WA" {
		t.Errorf("expected case 2 status WA, got %s", mapped.Cases[1].Status)
	}
}
