package config

import (
	"testing"
	"time"
)

func TestLoadJudgeConfig(t *testing.T) {
	tests := []struct {
		name          string
		timeLimitEnv  string
		memoryLimitEnv string
		wantErr       bool
		wantNil       bool
		expectedTime  time.Duration
		expectedMem   int
	}{
		{
			name:          "Valid Configuration",
			timeLimitEnv:  "10s",
			memoryLimitEnv: "1024",
			wantErr:       false,
			wantNil:       false,
			expectedTime:  10 * time.Second,
			expectedMem:   1024,
		},
		{
			name:          "Empty Configuration Fallback",
			timeLimitEnv:  "",
			memoryLimitEnv: "",
			wantErr:       false,
			wantNil:       true,
		},
		{
			name:          "Garbage Time Limit Input",
			timeLimitEnv:  "invalid",
			memoryLimitEnv: "1024",
			wantErr:       true,
		},
		{
			name:          "Garbage Memory Limit Input",
			timeLimitEnv:  "10s",
			memoryLimitEnv: "not_a_number",
			wantErr:       true,
		},
		{
			name:          "Negative Time Limit",
			timeLimitEnv:  "-10s",
			memoryLimitEnv: "1024",
			wantErr:       true,
		},
		{
			name:          "Zero Memory Limit",
			timeLimitEnv:  "10s",
			memoryLimitEnv: "0",
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("JUDGE_TIME_LIMIT", tt.timeLimitEnv)
			t.Setenv("JUDGE_MEMORY_LIMIT", tt.memoryLimitEnv)

			config, err := LoadJudgeConfig()
			if (err != nil) != tt.wantErr {
				t.Fatalf("LoadJudgeConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
			
			if tt.wantNil {
				if config != nil {
					t.Fatalf("LoadJudgeConfig() expected nil config, got %v", config)
				}
				return
			}
			
			if !tt.wantErr {
				if config.TimeLimit != tt.expectedTime {
					t.Errorf("TimeLimit = %v, want %v", config.TimeLimit, tt.expectedTime)
				}
				if config.MemoryLimit != tt.expectedMem {
					t.Errorf("MemoryLimit = %v, want %v", config.MemoryLimit, tt.expectedMem)
				}
			}
		})
	}
}
