package claude

import (
	"os"
	"path/filepath"
	"testing"
)

func TestClaudeDir(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	got := ClaudeDir()
	want := filepath.Join(home, ".claude")
	if got != want {
		t.Errorf("ClaudeDir() = %q, want %q", got, want)
	}
}

func TestTasksDir(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	sessionID := "7ba50137-65c8-4349-b420-cdce14c38d2a"
	got := TasksDir(sessionID)
	want := filepath.Join(home, ".claude", "tasks", sessionID)
	if got != want {
		t.Errorf("TasksDir(%q) = %q, want %q", sessionID, got, want)
	}
}

func TestDebugLogPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	sessionID := "7ba50137-65c8-4349-b420-cdce14c38d2a"
	got := DebugLogPath(sessionID)
	want := filepath.Join(home, ".claude", "debug", sessionID+".txt")
	if got != want {
		t.Errorf("DebugLogPath(%q) = %q, want %q", sessionID, got, want)
	}
}

func TestHistoryPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	got := HistoryPath()
	want := filepath.Join(home, ".claude", "history.jsonl")
	if got != want {
		t.Errorf("HistoryPath() = %q, want %q", got, want)
	}
}

func TestGlobalConfigPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	got := GlobalConfigPath()
	want := filepath.Join(home, ".claude.json")
	if got != want {
		t.Errorf("GlobalConfigPath() = %q, want %q", got, want)
	}
}

func TestEncodeProjectPath(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"/Users/yone/github/project", "-Users-yone-github-project"},
		{"/Users/yone/.claude", "-Users-yone--claude"},
		{"/Users/yone/.takt", "-Users-yone--takt"},
		{"/", "-"},
		{"no-slash", "no-slash"},
	}
	for _, tt := range tests {
		got := EncodeProjectPath(tt.input)
		if got != tt.want {
			t.Errorf("EncodeProjectPath(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestSubagentsDir(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	got := SubagentsDir("/Users/yone/github/project", "abc-123")
	want := filepath.Join(home, ".claude", "projects", "-Users-yone-github-project", "abc-123", "subagents")
	if got != want {
		t.Errorf("SubagentsDir() = %q, want %q", got, want)
	}
}

func TestProjectsDir(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	got := ProjectsDir()
	want := filepath.Join(home, ".claude", "projects")
	if got != want {
		t.Errorf("ProjectsDir() = %q, want %q", got, want)
	}
}

func TestFindSubagentsDirBySessionID(t *testing.T) {
	// Create a temporary directory structure to test glob search
	tmpDir := t.TempDir()
	// Override homeDir for this test
	origHome := homeDir
	homeDir = tmpDir
	defer func() { homeDir = origHome }()

	sessionID := "test-session-123"
	subagentsPath := filepath.Join(tmpDir, ".claude", "projects", "-test-project", sessionID, "subagents")
	if err := os.MkdirAll(subagentsPath, 0o755); err != nil {
		t.Fatal(err)
	}

	got, err := FindSubagentsDirBySessionID(sessionID)
	if err != nil {
		t.Fatalf("FindSubagentsDirBySessionID() error = %v", err)
	}
	if got != subagentsPath {
		t.Errorf("FindSubagentsDirBySessionID() = %q, want %q", got, subagentsPath)
	}

	// Test with non-existent session
	_, err = FindSubagentsDirBySessionID("non-existent")
	if err == nil {
		t.Error("FindSubagentsDirBySessionID() should return error for non-existent session")
	}
}
