package claude

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// setupTestBase creates a temporary directory structure mimicking ~/.claude
func setupTestBase(t *testing.T) string {
	t.Helper()
	base := t.TempDir()

	// Create history.jsonl
	entries := []HistoryEntry{
		{Display: "プロジェクトを分析して", Timestamp: 1772326237190, Project: "/test/project-a", SessionID: "session-1"},
		{Display: "テストを実行して", Timestamp: 1772326337190, Project: "/test/project-a", SessionID: "session-1"},
		{Display: "/help", Timestamp: 1772326437190, Project: "/test/project-a", SessionID: "session-1"},
		{Display: "バグを修正して", Timestamp: 1772325237190, Project: "/test/project-b", SessionID: "session-2"},
		{Display: "コミットして", Timestamp: 1772326537190, Project: "/test/project-a", SessionID: "session-3"},
	}
	historyPath := filepath.Join(base, "history.jsonl")
	f, err := os.Create(historyPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		data, _ := json.Marshal(e)
		f.Write(data)
		f.Write([]byte("\n"))
	}
	f.Close()

	// Create tasks dir for session-1
	tasksDir := filepath.Join(base, "tasks", "session-1")
	if err := os.MkdirAll(tasksDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tasksDir, "1.json"), []byte(`{"id":"1","subject":"test","status":"pending","blocks":[],"blockedBy":[]}`), 0644); err != nil {
		t.Fatal(err)
	}

	// Create debug log for session-1
	debugDir := filepath.Join(base, "debug")
	if err := os.MkdirAll(debugDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(debugDir, "session-1.txt"), []byte("2026-03-01T00:00:00.000Z [DEBUG] test\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create file-history dir for session-1
	fhDir := filepath.Join(base, "file-history", "session-1")
	if err := os.MkdirAll(fhDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(fhDir, "abcd1234abcd1234@v1"), []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create .claude.json (note: in real usage this is ~/.claude.json, but for tests we put it at basePath/../.claude.json)
	// We'll use a separate configPath parameter approach
	configPath := filepath.Join(base, "claude.json")
	config := map[string]interface{}{
		"projects": map[string]interface{}{
			"/test/project-a": map[string]interface{}{
				"lastSessionId":         "session-1",
				"lastCost":              1.5,
				"lastDuration":          600000,
				"lastTotalInputTokens":  100000,
				"lastTotalOutputTokens": 20000,
				"lastModelUsage": map[string]interface{}{
					"claude-sonnet-4-20250514": map[string]interface{}{
						"inputTokens":  100000,
						"outputTokens": 20000,
					},
				},
			},
		},
	}
	configData, err := json.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, configData, 0644); err != nil {
		t.Fatal(err)
	}

	return base
}

func TestDiscoverSessions_GroupsBySessionID(t *testing.T) {
	base := setupTestBase(t)
	sessions, err := DiscoverSessions(base, filepath.Join(base, "claude.json"))
	if err != nil {
		t.Fatalf("DiscoverSessions() error = %v", err)
	}
	// session-1 appears 3 times in history but should be grouped into 1
	sessionIDs := make(map[string]bool)
	for _, s := range sessions {
		if sessionIDs[s.SessionID] {
			t.Errorf("duplicate session ID: %s", s.SessionID)
		}
		sessionIDs[s.SessionID] = true
	}
	if len(sessions) != 3 {
		t.Errorf("DiscoverSessions() returned %d sessions, want 3", len(sessions))
	}
}

func TestDiscoverSessions_SortByTimestamp(t *testing.T) {
	base := setupTestBase(t)
	sessions, err := DiscoverSessions(base, filepath.Join(base, "claude.json"))
	if err != nil {
		t.Fatalf("DiscoverSessions() error = %v", err)
	}
	for i := 1; i < len(sessions); i++ {
		if sessions[i].Timestamp > sessions[i-1].Timestamp {
			t.Errorf("sessions not sorted descending: %d > %d at index %d", sessions[i].Timestamp, sessions[i-1].Timestamp, i)
		}
	}
}

func TestDiscoverSessions_FirstInput(t *testing.T) {
	base := setupTestBase(t)
	sessions, err := DiscoverSessions(base, filepath.Join(base, "claude.json"))
	if err != nil {
		t.Fatalf("DiscoverSessions() error = %v", err)
	}
	for _, s := range sessions {
		if s.SessionID == "session-1" {
			// First non-command input should be "プロジェクトを分析して" (not "/help")
			if s.FirstInput != "プロジェクトを分析して" {
				t.Errorf("session-1 FirstInput = %q, want \"プロジェクトを分析して\"", s.FirstInput)
			}
			return
		}
	}
	t.Error("session-1 not found")
}

func TestDiscoverSessions_FirstInput_AllCommands(t *testing.T) {
	base := t.TempDir()
	// Create history with only command entries
	entries := []HistoryEntry{
		{Display: "/help", Timestamp: 1000, Project: "/test", SessionID: "cmd-session"},
		{Display: "/commit", Timestamp: 1001, Project: "/test", SessionID: "cmd-session"},
	}
	historyPath := filepath.Join(base, "history.jsonl")
	f, err := os.Create(historyPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		data, _ := json.Marshal(e)
		f.Write(data)
		f.Write([]byte("\n"))
	}
	f.Close()

	sessions, err := DiscoverSessions(base, filepath.Join(base, "claude.json"))
	if err != nil {
		t.Fatalf("DiscoverSessions() error = %v", err)
	}
	for _, s := range sessions {
		if s.SessionID == "cmd-session" {
			if s.FirstInput != "" {
				t.Errorf("FirstInput = %q, want empty string for all-commands session", s.FirstInput)
			}
			return
		}
	}
	t.Error("cmd-session not found")
}

func TestDiscoverSessions_HasTasks(t *testing.T) {
	base := setupTestBase(t)
	sessions, err := DiscoverSessions(base, filepath.Join(base, "claude.json"))
	if err != nil {
		t.Fatalf("DiscoverSessions() error = %v", err)
	}
	for _, s := range sessions {
		if s.SessionID == "session-1" {
			if !s.HasTasks {
				t.Error("session-1 HasTasks should be true")
			}
		}
		if s.SessionID == "session-2" {
			if s.HasTasks {
				t.Error("session-2 HasTasks should be false")
			}
		}
	}
}

func TestDiscoverSessions_HasDebugLog(t *testing.T) {
	base := setupTestBase(t)
	sessions, err := DiscoverSessions(base, filepath.Join(base, "claude.json"))
	if err != nil {
		t.Fatalf("DiscoverSessions() error = %v", err)
	}
	for _, s := range sessions {
		if s.SessionID == "session-1" {
			if !s.HasDebugLog {
				t.Error("session-1 HasDebugLog should be true")
			}
		}
		if s.SessionID == "session-2" {
			if s.HasDebugLog {
				t.Error("session-2 HasDebugLog should be false")
			}
		}
	}
}

func TestDiscoverSessions_HasFileHistory(t *testing.T) {
	base := setupTestBase(t)
	sessions, err := DiscoverSessions(base, filepath.Join(base, "claude.json"))
	if err != nil {
		t.Fatalf("DiscoverSessions() error = %v", err)
	}
	for _, s := range sessions {
		if s.SessionID == "session-1" {
			if !s.HasFileHistory {
				t.Error("session-1 HasFileHistory should be true")
			}
		}
		if s.SessionID == "session-2" {
			if s.HasFileHistory {
				t.Error("session-2 HasFileHistory should be false")
			}
		}
	}
}

func TestDiscoverSessions_StatsAttachment(t *testing.T) {
	base := setupTestBase(t)
	sessions, err := DiscoverSessions(base, filepath.Join(base, "claude.json"))
	if err != nil {
		t.Fatalf("DiscoverSessions() error = %v", err)
	}
	// session-1 and session-3 are both project-a, should both have stats
	for _, s := range sessions {
		if s.SessionID == "session-1" || s.SessionID == "session-3" {
			if s.Stats == nil {
				t.Errorf("session %s Stats should not be nil (project-a has stats)", s.SessionID)
			}
		} else if s.SessionID == "session-2" {
			if s.Stats != nil {
				t.Errorf("session-2 Stats should be nil (project-b has no stats)")
			}
		}
	}
}

func TestDiscoverSessions_EmptyHistory(t *testing.T) {
	base := t.TempDir()
	historyPath := filepath.Join(base, "history.jsonl")
	os.WriteFile(historyPath, []byte{}, 0644)

	sessions, err := DiscoverSessions(base, filepath.Join(base, "claude.json"))
	if err != nil {
		t.Fatalf("DiscoverSessions() error = %v", err)
	}
	if len(sessions) != 0 {
		t.Errorf("DiscoverSessions() returned %d sessions, want 0", len(sessions))
	}
}

func TestBuildSessionInfo(t *testing.T) {
	base := setupTestBase(t)
	configPath := filepath.Join(base, "claude.json")

	t.Run("session with matching project sets Project field", func(t *testing.T) {
		// session-1 is lastSessionId for /test/project-a in the test fixture
		info := BuildSessionInfo(base, configPath, "session-1")
		if info.SessionID != "session-1" {
			t.Errorf("SessionID = %q, want %q", info.SessionID, "session-1")
		}
		if info.Project != "/test/project-a" {
			t.Errorf("Project = %q, want %q", info.Project, "/test/project-a")
		}
		if info.Stats == nil {
			t.Error("Stats should not be nil for session-1 (project-a has stats)")
		}
		if !info.HasTasks {
			t.Error("HasTasks should be true for session-1")
		}
		if !info.HasDebugLog {
			t.Error("HasDebugLog should be true for session-1")
		}
		if !info.HasFileHistory {
			t.Error("HasFileHistory should be true for session-1")
		}
	})

	t.Run("session without matching project leaves Project empty", func(t *testing.T) {
		// session-2 is not a lastSessionId for any project in the test fixture
		info := BuildSessionInfo(base, configPath, "session-2")
		if info.SessionID != "session-2" {
			t.Errorf("SessionID = %q, want %q", info.SessionID, "session-2")
		}
		if info.Project != "" {
			t.Errorf("Project = %q, want empty string", info.Project)
		}
		if info.Stats != nil {
			t.Error("Stats should be nil for session-2 (not a lastSessionId)")
		}
		if info.HasTasks {
			t.Error("HasTasks should be false for session-2")
		}
		if info.HasDebugLog {
			t.Error("HasDebugLog should be false for session-2")
		}
		if info.HasFileHistory {
			t.Error("HasFileHistory should be false for session-2")
		}
	})

	t.Run("unknown session has all fields empty/false", func(t *testing.T) {
		info := BuildSessionInfo(base, configPath, "nonexistent")
		if info.Project != "" {
			t.Errorf("Project = %q, want empty string", info.Project)
		}
		if info.Stats != nil {
			t.Error("Stats should be nil for nonexistent session")
		}
		if info.HasTasks || info.HasDebugLog || info.HasFileHistory {
			t.Error("capabilities should all be false for nonexistent session")
		}
	})
}
