package claude

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var (
	homeDir     string
	homeDirOnce sync.Once
)

func ensureHomeDir() {
	homeDirOnce.Do(func() {
		var err error
		homeDir, err = os.UserHomeDir()
		if err != nil {
			homeDir = ""
		}
	})
}

// ClaudeDir returns the path to ~/.claude
func ClaudeDir() string {
	ensureHomeDir()
	return filepath.Join(homeDir, ".claude")
}

// HistoryPath returns the path to ~/.claude/history.jsonl
func HistoryPath() string {
	return filepath.Join(ClaudeDir(), "history.jsonl")
}

// TasksDir returns the path to ~/.claude/tasks/{sessionID}
func TasksDir(sessionID string) string {
	return filepath.Join(ClaudeDir(), "tasks", sessionID)
}

// DebugLogPath returns the path to ~/.claude/debug/{sessionID}.txt
func DebugLogPath(sessionID string) string {
	return filepath.Join(ClaudeDir(), "debug", sessionID+".txt")
}

// FileHistoryDir returns the path to ~/.claude/file-history/{sessionID}
func FileHistoryDir(sessionID string) string {
	return filepath.Join(ClaudeDir(), "file-history", sessionID)
}

// GlobalConfigPath returns the path to ~/.claude.json
func GlobalConfigPath() string {
	ensureHomeDir()
	return filepath.Join(homeDir, ".claude.json")
}

// ProjectsDir returns the path to ~/.claude/projects
func ProjectsDir() string {
	return filepath.Join(ClaudeDir(), "projects")
}

// EncodeProjectPath encodes a project path for use in the projects directory.
// It replaces "/" and "." with "-" to match Claude Code's encoding.
func EncodeProjectPath(projectPath string) string {
	s := strings.ReplaceAll(projectPath, "/", "-")
	s = strings.ReplaceAll(s, ".", "-")
	return s
}

// SubagentsDir returns the path to the subagents directory for a given project and session.
func SubagentsDir(projectPath, sessionID string) string {
	encoded := EncodeProjectPath(projectPath)
	return filepath.Join(ProjectsDir(), encoded, sessionID, "subagents")
}

// ParentConversationPath returns the path to the parent conversation JSONL file
// for a given project and session.
func ParentConversationPath(projectPath, sessionID string) string {
	encoded := EncodeProjectPath(projectPath)
	return filepath.Join(ProjectsDir(), encoded, sessionID+".jsonl")
}

// FindSubagentsDirBySessionID searches for a subagents directory by session ID
// across all project directories. Returns the path if found, or an error.
func FindSubagentsDirBySessionID(sessionID string) (string, error) {
	pattern := filepath.Join(ProjectsDir(), "*", sessionID, "subagents")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return "", fmt.Errorf("glob search for subagents dir: %w", err)
	}
	if len(matches) == 0 {
		return "", fmt.Errorf("subagents directory not found for session %s", sessionID)
	}
	return matches[0], nil
}
