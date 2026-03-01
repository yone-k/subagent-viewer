package claude

import (
	"encoding/json"
	"fmt"
	"os"
)

// ModelUsage represents per-model token usage statistics.
type ModelUsage struct {
	InputTokens              int64 `json:"inputTokens"`
	OutputTokens             int64 `json:"outputTokens"`
	CacheCreationInputTokens int64 `json:"cacheCreationInputTokens"`
	CacheReadInputTokens     int64 `json:"cacheReadInputTokens"`
}

// ProjectStats represents statistics for a project from .claude.json
type ProjectStats struct {
	LastSessionID         string                `json:"lastSessionId"`
	LastCost              float64               `json:"lastCost"`
	LastDuration          int64                 `json:"lastDuration"`
	LastTotalInputTokens  int64                 `json:"lastTotalInputTokens"`
	LastTotalOutputTokens int64                 `json:"lastTotalOutputTokens"`
	LastModelUsage        map[string]ModelUsage `json:"lastModelUsage"`
}

type globalConfig struct {
	// Pointer values for json.Unmarshal compatibility
	Projects map[string]*ProjectStats `json:"projects"`
}

// loadGlobalConfig reads and parses the global config file (.claude.json).
// Returns the projects map (never nil) and any error encountered.
func loadGlobalConfig(configPath string) (map[string]*ProjectStats, error) {
	// .claude.json is typically small (< 100KB); full read is acceptable
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}
	var config globalConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parsing config JSON: %w", err)
	}
	if config.Projects == nil {
		return make(map[string]*ProjectStats), nil
	}
	return config.Projects, nil
}

// LoadProjectStats loads project statistics from the global config file.
// Returns nil (not error) if the project is not found in config.
func LoadProjectStats(configPath, projectPath string) (*ProjectStats, error) {
	projects, err := loadGlobalConfig(configPath)
	if err != nil {
		return nil, err
	}
	stats, ok := projects[projectPath]
	if !ok {
		return nil, nil
	}
	return stats, nil
}
