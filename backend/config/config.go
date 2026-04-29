package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type JudgeConfig struct {
	TimeLimit   time.Duration
	MemoryLimit int // in MB
}

// LoadJudgeConfig reads and parses judge configuration overrides from the
// JUDGE_TIME_LIMIT and JUDGE_MEMORY_LIMIT environment variables.
// It returns nil, nil if neither variable is set, leaves unspecified fields
// at their zero value, and returns an error if a provided value is malformed.
func LoadJudgeConfig() (*JudgeConfig, error) {
	timeStr := os.Getenv("JUDGE_TIME_LIMIT")
	memStr := os.Getenv("JUDGE_MEMORY_LIMIT")

	if timeStr == "" && memStr == "" {
		return nil, nil
	}

	config := &JudgeConfig{}

	if timeStr != "" {
		parsedTime, err := time.ParseDuration(timeStr)
		if err != nil {
			return nil, fmt.Errorf("invalid JUDGE_TIME_LIMIT: %w", err)
		}
		if parsedTime <= 0 {
			return nil, fmt.Errorf("invalid JUDGE_TIME_LIMIT: must be greater than 0")
		}
		config.TimeLimit = parsedTime
	}

	if memStr != "" {
		parsedMem, err := strconv.Atoi(memStr)
		if err != nil {
			return nil, fmt.Errorf("invalid JUDGE_MEMORY_LIMIT: %w", err)
		}
		if parsedMem <= 0 {
			return nil, fmt.Errorf("invalid JUDGE_MEMORY_LIMIT: must be a positive integer")
		}
		config.MemoryLimit = parsedMem
	}

	return config, nil
}
