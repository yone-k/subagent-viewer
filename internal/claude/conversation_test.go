package claude

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParseConversationFile_ValidEntries(t *testing.T) {
	path := filepath.Join("testdata", "subagents", "agent-test1.jsonl")
	entries, info, err := ParseConversationFile(path)
	if err != nil {
		t.Fatalf("ParseConversationFile() error = %v", err)
	}

	if len(entries) != 4 {
		t.Fatalf("got %d entries, want 4", len(entries))
	}

	// First entry: user with string content
	if entries[0].Type != EntryTypeUser {
		t.Errorf("entry[0].Type = %q, want %q", entries[0].Type, EntryTypeUser)
	}
	if len(entries[0].Content) != 1 || entries[0].Content[0].Text != "Implement the feature" {
		t.Errorf("entry[0].Content unexpected: %+v", entries[0].Content)
	}

	// Second entry: assistant with text + tool_use
	if entries[1].Type != EntryTypeAssistant {
		t.Errorf("entry[1].Type = %q, want %q", entries[1].Type, EntryTypeAssistant)
	}
	if len(entries[1].Content) != 2 {
		t.Fatalf("entry[1] got %d content blocks, want 2", len(entries[1].Content))
	}
	if entries[1].Content[0].Type != "text" || entries[1].Content[0].Text != "I'll implement this feature." {
		t.Errorf("entry[1].Content[0] unexpected: %+v", entries[1].Content[0])
	}
	if entries[1].Content[1].Type != "tool_use" || entries[1].Content[1].ToolName != "Read" {
		t.Errorf("entry[1].Content[1] unexpected: %+v", entries[1].Content[1])
	}

	// Third entry: user with tool_result
	if entries[2].Content[0].Type != "tool_result" {
		t.Errorf("entry[2].Content[0].Type = %q, want tool_result", entries[2].Content[0].Type)
	}

	// Fourth entry: assistant with thinking + text
	if entries[3].Content[0].Type != "thinking" {
		t.Errorf("entry[3].Content[0].Type = %q, want thinking", entries[3].Content[0].Type)
	}

	// SubagentInfo
	if info == nil {
		t.Fatal("info should not be nil")
	}
	if info.AgentID != "abc123" {
		t.Errorf("info.AgentID = %q, want %q", info.AgentID, "abc123")
	}
	if info.Slug != "implement-feature" {
		t.Errorf("info.Slug = %q, want %q", info.Slug, "implement-feature")
	}
	if info.Prompt != "Implement the feature" {
		t.Errorf("info.Prompt = %q, want %q", info.Prompt, "Implement the feature")
	}
	if info.EntryCount != 4 {
		t.Errorf("info.EntryCount = %d, want 4", info.EntryCount)
	}
}

func TestParseConversationFile_SkipsProgress(t *testing.T) {
	path := filepath.Join("testdata", "subagents", "agent-test2.jsonl")
	entries, info, err := ParseConversationFile(path)
	if err != nil {
		t.Fatalf("ParseConversationFile() error = %v", err)
	}

	// progress line should be skipped
	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2 (progress should be skipped)", len(entries))
	}

	if info.AgentID != "def456" {
		t.Errorf("info.AgentID = %q, want %q", info.AgentID, "def456")
	}
}

func TestParseConversationFile_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	emptyFile := filepath.Join(tmpDir, "empty.jsonl")
	if err := os.WriteFile(emptyFile, []byte{}, 0o644); err != nil {
		t.Fatal(err)
	}

	entries, info, err := ParseConversationFile(emptyFile)
	if err != nil {
		t.Fatalf("ParseConversationFile() error = %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("got %d entries, want 0", len(entries))
	}
	if info != nil {
		t.Error("info should be nil for empty file")
	}
}

func TestParseConversationFile_NonExistentFile(t *testing.T) {
	_, _, err := ParseConversationFile("/nonexistent/path.jsonl")
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestExtractAgentDescriptions(t *testing.T) {
	path := filepath.Join("testdata", "parent-conversation.jsonl")
	descriptions, err := ExtractAgentDescriptions(path)
	if err != nil {
		t.Fatalf("ExtractAgentDescriptions() error = %v", err)
	}

	if len(descriptions) != 4 {
		t.Fatalf("got %d descriptions, want 4", len(descriptions))
	}

	// First Agent tool_use: closed (has tool_result)
	if descriptions[0].Description != "Explore current repo structure" {
		t.Errorf("descriptions[0].Description = %q, want %q", descriptions[0].Description, "Explore current repo structure")
	}
	if descriptions[0].SubagentType != "Explore" {
		t.Errorf("descriptions[0].SubagentType = %q, want %q", descriptions[0].SubagentType, "Explore")
	}
	if descriptions[0].ToolUseID != "tool1" {
		t.Errorf("descriptions[0].ToolUseID = %q, want %q", descriptions[0].ToolUseID, "tool1")
	}
	if descriptions[0].Status != SubagentClosed {
		t.Errorf("descriptions[0].Status = %q, want %q", descriptions[0].Status, SubagentClosed)
	}

	// Second Agent tool_use: closed (has tool_result)
	if descriptions[1].Description != "Implement user auth" {
		t.Errorf("descriptions[1].Description = %q, want %q", descriptions[1].Description, "Implement user auth")
	}
	if descriptions[1].SubagentType != "general-task-executor" {
		t.Errorf("descriptions[1].SubagentType = %q, want %q", descriptions[1].SubagentType, "general-task-executor")
	}
	if descriptions[1].ToolUseID != "tool2" {
		t.Errorf("descriptions[1].ToolUseID = %q, want %q", descriptions[1].ToolUseID, "tool2")
	}
	if descriptions[1].Status != SubagentClosed {
		t.Errorf("descriptions[1].Status = %q, want %q", descriptions[1].Status, SubagentClosed)
	}

	// Third Agent tool_use: running (no tool_result)
	if descriptions[2].Description != "Refactor database layer" {
		t.Errorf("descriptions[2].Description = %q, want %q", descriptions[2].Description, "Refactor database layer")
	}
	if descriptions[2].SubagentType != "general-task-executor" {
		t.Errorf("descriptions[2].SubagentType = %q, want %q", descriptions[2].SubagentType, "general-task-executor")
	}
	if descriptions[2].ToolUseID != "toolu_running_test" {
		t.Errorf("descriptions[2].ToolUseID = %q, want %q", descriptions[2].ToolUseID, "toolu_running_test")
	}
	if descriptions[2].Status != SubagentRunning {
		t.Errorf("descriptions[2].Status = %q, want %q", descriptions[2].Status, SubagentRunning)
	}

	// Fourth Agent tool_use: background agent (has tool_result but run_in_background=true)
	// Should be running because tool_result is just a launch confirmation
	if descriptions[3].Description != "Background code review" {
		t.Errorf("descriptions[3].Description = %q, want %q", descriptions[3].Description, "Background code review")
	}
	if descriptions[3].SubagentType != "ai-antipattern-fixer" {
		t.Errorf("descriptions[3].SubagentType = %q, want %q", descriptions[3].SubagentType, "ai-antipattern-fixer")
	}
	if descriptions[3].ToolUseID != "toolu_bg_test" {
		t.Errorf("descriptions[3].ToolUseID = %q, want %q", descriptions[3].ToolUseID, "toolu_bg_test")
	}
	if descriptions[3].Status != SubagentRunning {
		t.Errorf("descriptions[3].Status = %q, want %q (background agent should remain running)", descriptions[3].Status, SubagentRunning)
	}
}

func TestExtractAgentDescriptions_AsyncLaunchFallback(t *testing.T) {
	// Test that even without run_in_background flag, async launch content
	// in tool_result prevents marking the agent as closed.
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "conversation.jsonl")

	lines := []string{
		`{"type":"assistant","message":{"content":[{"type":"tool_use","id":"toolu_no_flag","name":"Agent","input":{"description":"Some task","prompt":"Do something","subagent_type":"general-task-executor"}}]}}`,
		`{"type":"user","message":{"content":[{"type":"tool_result","tool_use_id":"toolu_no_flag","content":"Async agent launched successfully.\nagentId: xyz789"}]}}`,
	}
	content := strings.Join(lines, "\n") + "\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	descriptions, err := ExtractAgentDescriptions(path)
	if err != nil {
		t.Fatalf("ExtractAgentDescriptions() error = %v", err)
	}
	if len(descriptions) != 1 {
		t.Fatalf("got %d descriptions, want 1", len(descriptions))
	}
	if descriptions[0].Status != SubagentRunning {
		t.Errorf("Status = %q, want %q (async launch result should be treated as running)", descriptions[0].Status, SubagentRunning)
	}
}

func TestExtractAgentDescriptions_BackgroundAgentCompleted(t *testing.T) {
	// Background agent that has completed: tool_result is async launch,
	// but a later task-notification indicates completion via <tool-use-id>.
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "conversation.jsonl")

	lines := []string{
		// Agent launched in background
		`{"type":"assistant","message":{"content":[{"type":"tool_use","id":"toolu_bg1","name":"Agent","input":{"description":"BG task","prompt":"Do background work","subagent_type":"general-task-executor","run_in_background":true}}]}}`,
		// Immediate tool_result (launch confirmation)
		`{"type":"user","message":{"content":[{"type":"tool_result","tool_use_id":"toolu_bg1","content":"Async agent launched successfully.\nagentId: abc123"}]}}`,
		// Later: task-notification indicating completion
		`{"type":"user","message":{"content":[{"type":"tool_result","tool_use_id":"toolu_other","content":"some result\n<system-reminder>\nA background agent completed a task:\n<task-notification>\n<task-id>abc123</task-id>\n<tool-use-id>toolu_bg1</tool-use-id>\n<status>completed</status>\n<summary>Agent completed</summary>\n<result>Done</result>\n</task-notification>\n</system-reminder>"}]}}`,
	}
	content := strings.Join(lines, "\n") + "\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	descriptions, err := ExtractAgentDescriptions(path)
	if err != nil {
		t.Fatalf("ExtractAgentDescriptions() error = %v", err)
	}
	if len(descriptions) != 1 {
		t.Fatalf("got %d descriptions, want 1", len(descriptions))
	}
	if descriptions[0].Status != SubagentClosed {
		t.Errorf("Status = %q, want %q (background agent with task-notification should be closed)", descriptions[0].Status, SubagentClosed)
	}
}

func TestExtractAgentDescriptions_BackgroundAgentStillRunning(t *testing.T) {
	// Background agent without task-notification should remain running.
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "conversation.jsonl")

	lines := []string{
		`{"type":"assistant","message":{"content":[{"type":"tool_use","id":"toolu_bg2","name":"Agent","input":{"description":"Running task","prompt":"Still working","subagent_type":"general-task-executor","run_in_background":true}}]}}`,
		`{"type":"user","message":{"content":[{"type":"tool_result","tool_use_id":"toolu_bg2","content":"Async agent launched successfully.\nagentId: def456"}]}}`,
	}
	content := strings.Join(lines, "\n") + "\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	descriptions, err := ExtractAgentDescriptions(path)
	if err != nil {
		t.Fatalf("ExtractAgentDescriptions() error = %v", err)
	}
	if len(descriptions) != 1 {
		t.Fatalf("got %d descriptions, want 1", len(descriptions))
	}
	if descriptions[0].Status != SubagentRunning {
		t.Errorf("Status = %q, want %q (background agent without notification should be running)", descriptions[0].Status, SubagentRunning)
	}
}

func TestExtractAgentDescriptions_QueueOperationCompletion(t *testing.T) {
	// Background agent whose completion is signaled via a queue-operation line
	// (the actual Claude Code format) rather than a type=user message.
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "conversation.jsonl")

	lines := []string{
		// Agent launched in background
		`{"type":"assistant","message":{"content":[{"type":"tool_use","id":"toolu_bg3","name":"Agent","input":{"description":"BG queue task","prompt":"Do queue work","subagent_type":"general-task-executor","run_in_background":true}}]}}`,
		// Immediate tool_result (launch confirmation)
		`{"type":"user","message":{"content":[{"type":"tool_result","tool_use_id":"toolu_bg3","content":"Async agent launched successfully.\nagentId: xyz789"}]}}`,
		// Completion via queue-operation (real Claude Code format)
		`{"type":"queue-operation","operation":"enqueue","content":"<task-notification>\n<task-id>xyz789</task-id>\n<tool-use-id>toolu_bg3</tool-use-id>\n<status>completed</status>\n<summary>Agent completed</summary>\n</task-notification>","timestamp":"2025-01-01T00:00:00Z","sessionId":"test-session"}`,
	}
	content := strings.Join(lines, "\n") + "\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	descriptions, err := ExtractAgentDescriptions(path)
	if err != nil {
		t.Fatalf("ExtractAgentDescriptions() error = %v", err)
	}
	if len(descriptions) != 1 {
		t.Fatalf("got %d descriptions, want 1", len(descriptions))
	}
	if descriptions[0].Status != SubagentClosed {
		t.Errorf("Status = %q, want %q (queue-operation completion should mark agent as closed)", descriptions[0].Status, SubagentClosed)
	}
}

func TestExtractAgentDescriptions_FileNotFound(t *testing.T) {
	_, err := ExtractAgentDescriptions("/nonexistent/path/conversation.jsonl")
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestEnrichSubagentsWithDescriptions(t *testing.T) {
	// Simulate descriptions extracted from parent conversation
	descriptions := []AgentDescription{
		{Prompt: "Analyze the project structure and list all directories and key files. Focus on understanding the architecture.", Description: "Explore current repo structure", SubagentType: "Explore", ToolUseID: "tool1", Status: SubagentClosed},
		{Prompt: "Add user authentication using JWT tokens. Create the auth middleware and login endpoint.", Description: "Implement user auth", SubagentType: "general-task-executor", ToolUseID: "tool2", Status: SubagentClosed},
	}

	baseTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	agents := []SubagentInfo{
		{
			AgentID:   "agent1",
			Prompt:    truncateString("Analyze the project structure and list all directories and key files. Focus on understanding the architecture.", 120),
			CreatedAt: baseTime,
		},
		{
			AgentID:   "agent2",
			Prompt:    truncateString("Add user authentication using JWT tokens. Create the auth middleware and login endpoint.", 120),
			CreatedAt: baseTime.Add(1 * time.Hour),
		},
	}

	EnrichSubagentsWithDescriptions(agents, descriptions)

	// After enrichment, agents remain in CreatedAt ascending order (caller sorts for display)
	// agents[0] = agent1 (older), agents[1] = agent2 (newer)
	if agents[0].Description != "Explore current repo structure" {
		t.Errorf("agents[0].Description = %q, want %q", agents[0].Description, "Explore current repo structure")
	}
	if agents[0].SubagentType != "Explore" {
		t.Errorf("agents[0].SubagentType = %q, want %q", agents[0].SubagentType, "Explore")
	}
	if agents[0].Status != SubagentClosed {
		t.Errorf("agents[0].Status = %q, want %q", agents[0].Status, SubagentClosed)
	}
	if agents[1].Description != "Implement user auth" {
		t.Errorf("agents[1].Description = %q, want %q", agents[1].Description, "Implement user auth")
	}
	if agents[1].SubagentType != "general-task-executor" {
		t.Errorf("agents[1].SubagentType = %q, want %q", agents[1].SubagentType, "general-task-executor")
	}
	if agents[1].Status != SubagentClosed {
		t.Errorf("agents[1].Status = %q, want %q", agents[1].Status, SubagentClosed)
	}
}

func TestEnrichSubagentsWithDescriptions_ExactMatchPriority(t *testing.T) {
	// When the prompt matches exactly, it should be used even if
	// another description also has the same prefix.
	descriptions := []AgentDescription{
		{Prompt: "Fix the bug in the authentication module", Description: "prefix match", SubagentType: "prefix", Status: SubagentClosed},
		{Prompt: "Fix the bug", Description: "exact match", SubagentType: "exact", Status: SubagentRunning},
	}

	agents := []SubagentInfo{
		{
			AgentID: "agent1",
			Prompt:  "Fix the bug",
		},
	}

	EnrichSubagentsWithDescriptions(agents, descriptions)

	if agents[0].Description != "exact match" {
		t.Errorf("agents[0].Description = %q, want %q (exact match should be preferred)", agents[0].Description, "exact match")
	}
	if agents[0].SubagentType != "exact" {
		t.Errorf("agents[0].SubagentType = %q, want %q", agents[0].SubagentType, "exact")
	}
	if agents[0].Status != SubagentRunning {
		t.Errorf("agents[0].Status = %q, want %q", agents[0].Status, SubagentRunning)
	}
}

func TestEnrichSubagentsWithDescriptions_PrefixFallback(t *testing.T) {
	// When no exact match exists, prefix match should be used as fallback.
	longPrompt := "This is a very long prompt that exceeds one hundred and twenty characters so it will be truncated by ParseConversationEntries to test prefix matching behavior"
	descriptions := []AgentDescription{
		{Prompt: longPrompt, Description: "found via prefix", SubagentType: "prefix-type", Status: SubagentClosed},
	}

	agents := []SubagentInfo{
		{
			AgentID: "agent1",
			Prompt:  truncateString(longPrompt, 120),
		},
	}

	EnrichSubagentsWithDescriptions(agents, descriptions)

	if agents[0].Description != "found via prefix" {
		t.Errorf("agents[0].Description = %q, want %q", agents[0].Description, "found via prefix")
	}
}

func TestEnrichSubagentsWithDescriptions_NoMatch(t *testing.T) {
	descriptions := []AgentDescription{
		{Prompt: "Some prompt that does not match", Description: "Some description", SubagentType: "Explore", Status: SubagentClosed},
	}

	agents := []SubagentInfo{
		{
			AgentID: "agent1",
			Prompt:  "Completely different prompt text",
		},
	}

	EnrichSubagentsWithDescriptions(agents, descriptions)

	if agents[0].Description != "" {
		t.Errorf("agents[0].Description = %q, want empty string", agents[0].Description)
	}
}

func TestEnrichSubagentsWithDescriptions_DuplicatePrompt(t *testing.T) {
	// Two descriptions with the same prompt but different statuses
	descriptions := []AgentDescription{
		{Prompt: "Run the test suite", Description: "First test run", SubagentType: "general-task-executor", ToolUseID: "tool_a", Status: SubagentClosed},
		{Prompt: "Run the test suite", Description: "Second test run", SubagentType: "general-task-executor", ToolUseID: "tool_b", Status: SubagentRunning},
	}

	baseTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	agents := []SubagentInfo{
		{
			AgentID:   "agent1",
			Prompt:    "Run the test suite",
			CreatedAt: baseTime,
		},
		{
			AgentID:   "agent2",
			Prompt:    "Run the test suite",
			CreatedAt: baseTime.Add(1 * time.Hour),
		},
	}

	EnrichSubagentsWithDescriptions(agents, descriptions)

	// After enrichment, agents remain in CreatedAt ascending order (caller sorts for display)
	// agents[0] = agent1 (older, matched first description = Closed)
	// agents[1] = agent2 (newer, matched second description = Running)
	if agents[0].Description != "First test run" {
		t.Errorf("agents[0].Description = %q, want %q", agents[0].Description, "First test run")
	}
	if agents[0].Status != SubagentClosed {
		t.Errorf("agents[0].Status = %q, want %q", agents[0].Status, SubagentClosed)
	}

	if agents[1].Description != "Second test run" {
		t.Errorf("agents[1].Description = %q, want %q", agents[1].Description, "Second test run")
	}
	if agents[1].Status != SubagentRunning {
		t.Errorf("agents[1].Status = %q, want %q", agents[1].Status, SubagentRunning)
	}
}

func TestDiscoverSubagentFiles(t *testing.T) {
	t.Run("returns paths and ModTime from testdata", func(t *testing.T) {
		dir := filepath.Join("testdata", "subagents")
		files, err := DiscoverSubagentFiles(dir)
		if err != nil {
			t.Fatalf("DiscoverSubagentFiles() error = %v", err)
		}

		if len(files) != 2 {
			t.Fatalf("got %d files, want 2", len(files))
		}

		// Verify each file has a non-empty Path and non-zero CreatedAt
		for i, f := range files {
			if f.Path == "" {
				t.Errorf("files[%d].Path is empty", i)
			}
			if f.CreatedAt.IsZero() {
				t.Errorf("files[%d].CreatedAt is zero", i)
			}
		}
	})

	t.Run("excludes compact files", func(t *testing.T) {
		dir := t.TempDir()

		// Create a normal agent file
		normalContent := []byte(`{"type":"user","message":{"content":"hello"}}` + "\n")
		if err := os.WriteFile(filepath.Join(dir, "agent-normal.jsonl"), normalContent, 0644); err != nil {
			t.Fatal(err)
		}
		// Create a compact file (should be excluded)
		if err := os.WriteFile(filepath.Join(dir, "agent-compact-abc.jsonl"), normalContent, 0644); err != nil {
			t.Fatal(err)
		}

		files, err := DiscoverSubagentFiles(dir)
		if err != nil {
			t.Fatalf("DiscoverSubagentFiles() error = %v", err)
		}

		if len(files) != 1 {
			t.Fatalf("got %d files, want 1 (compact should be excluded)", len(files))
		}
		if !strings.Contains(files[0].Path, "agent-normal.jsonl") {
			t.Errorf("expected agent-normal.jsonl, got %s", files[0].Path)
		}
	})

	t.Run("sorted by CreatedAt descending", func(t *testing.T) {
		dir := t.TempDir()

		baseTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
		testFiles := []struct {
			name    string
			modTime time.Time
		}{
			{"agent-old.jsonl", baseTime},
			{"agent-mid.jsonl", baseTime.Add(1 * time.Hour)},
			{"agent-new.jsonl", baseTime.Add(2 * time.Hour)},
		}

		content := []byte(`{"type":"user","message":{"content":"task"}}` + "\n")
		for _, f := range testFiles {
			path := filepath.Join(dir, f.name)
			if err := os.WriteFile(path, content, 0644); err != nil {
				t.Fatal(err)
			}
			if err := os.Chtimes(path, f.modTime, f.modTime); err != nil {
				t.Fatal(err)
			}
		}

		files, err := DiscoverSubagentFiles(dir)
		if err != nil {
			t.Fatalf("DiscoverSubagentFiles() error = %v", err)
		}

		if len(files) != 3 {
			t.Fatalf("got %d files, want 3", len(files))
		}

		// Expect newest first
		wantOrder := []string{"agent-new.jsonl", "agent-mid.jsonl", "agent-old.jsonl"}
		for i, want := range wantOrder {
			got := filepath.Base(files[i].Path)
			if got != want {
				t.Errorf("files[%d] = %q, want %q", i, got, want)
			}
		}

		// Verify CreatedAt is actually descending
		for i := 1; i < len(files); i++ {
			if files[i].CreatedAt.After(files[i-1].CreatedAt) {
				t.Errorf("files[%d].CreatedAt (%v) is after files[%d].CreatedAt (%v)",
					i, files[i].CreatedAt, i-1, files[i-1].CreatedAt)
			}
		}
	})
}

func TestExtractAgentDescriptionsIncremental(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "conversation.jsonl")

	// Phase 1: Initial read with offset=0, cache=nil
	// Write an assistant line with Agent tool_use
	line1 := `{"type":"assistant","message":{"content":[{"type":"tool_use","id":"tool_inc_1","name":"Agent","input":{"description":"Explore repo","prompt":"Analyze the project","subagent_type":"Explore"}}]}}` + "\n"
	if err := os.WriteFile(path, []byte(line1), 0o644); err != nil {
		t.Fatal(err)
	}

	r, err := ExtractAgentDescriptionsIncremental(path, 0, nil)
	if err != nil {
		t.Fatalf("Phase 1: error = %v", err)
	}
	cache := r.Cache
	offset := r.Offset
	if len(cache.Descriptions) != 1 {
		t.Fatalf("Phase 1: got %d descriptions, want 1", len(cache.Descriptions))
	}
	if cache.Descriptions[0].Description != "Explore repo" {
		t.Errorf("Phase 1: Description = %q, want %q", cache.Descriptions[0].Description, "Explore repo")
	}
	if cache.Descriptions[0].Status != SubagentRunning {
		t.Errorf("Phase 1: Status = %q, want %q", cache.Descriptions[0].Status, SubagentRunning)
	}
	if offset != int64(len(line1)) {
		t.Errorf("Phase 1: offset = %d, want %d", offset, len(line1))
	}

	// Phase 2: Append tool_result, read incrementally
	line2 := `{"type":"user","message":{"content":[{"type":"tool_result","tool_use_id":"tool_inc_1","content":"Done"}]}}` + "\n"
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(line2); err != nil {
		f.Close()
		t.Fatal(err)
	}
	f.Close()

	r, err = ExtractAgentDescriptionsIncremental(path, offset, cache)
	if err != nil {
		t.Fatalf("Phase 2: error = %v", err)
	}
	cache = r.Cache
	offset = r.Offset
	if len(cache.Descriptions) != 1 {
		t.Fatalf("Phase 2: got %d descriptions, want 1", len(cache.Descriptions))
	}
	if cache.Descriptions[0].Status != SubagentClosed {
		t.Errorf("Phase 2: Status = %q, want %q (should be closed after tool_result)", cache.Descriptions[0].Status, SubagentClosed)
	}
	if offset != int64(len(line1)+len(line2)) {
		t.Errorf("Phase 2: offset = %d, want %d", offset, len(line1)+len(line2))
	}

	// Phase 3: Append new Agent tool_use, read incrementally
	line3 := `{"type":"assistant","message":{"content":[{"type":"tool_use","id":"tool_inc_2","name":"Agent","input":{"description":"Implement feature","prompt":"Add the new feature","subagent_type":"general-task-executor"}}]}}` + "\n"
	f, err = os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(line3); err != nil {
		f.Close()
		t.Fatal(err)
	}
	f.Close()

	r, err = ExtractAgentDescriptionsIncremental(path, offset, cache)
	if err != nil {
		t.Fatalf("Phase 3: error = %v", err)
	}
	cache = r.Cache
	offset = r.Offset
	if len(cache.Descriptions) != 2 {
		t.Fatalf("Phase 3: got %d descriptions, want 2", len(cache.Descriptions))
	}
	if cache.Descriptions[1].Description != "Implement feature" {
		t.Errorf("Phase 3: Descriptions[1].Description = %q, want %q", cache.Descriptions[1].Description, "Implement feature")
	}
	if cache.Descriptions[1].Status != SubagentRunning {
		t.Errorf("Phase 3: Descriptions[1].Status = %q, want %q", cache.Descriptions[1].Status, SubagentRunning)
	}
	if offset != int64(len(line1)+len(line2)+len(line3)) {
		t.Errorf("Phase 3: offset = %d, want %d", offset, len(line1)+len(line2)+len(line3))
	}
}

func TestExtractAgentDescriptionsIncremental_NilCache(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "conversation.jsonl")

	line := `{"type":"assistant","message":{"content":[{"type":"tool_use","id":"tool_nil","name":"Agent","input":{"description":"Task A","prompt":"Do task A","subagent_type":"Explore"}}]}}` + "\n"
	if err := os.WriteFile(path, []byte(line), 0o644); err != nil {
		t.Fatal(err)
	}

	r, err := ExtractAgentDescriptionsIncremental(path, 0, nil)
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if r.Cache == nil {
		t.Fatal("cache should not be nil after call with nil cache")
	}
	if r.Cache.completedIDs == nil {
		t.Error("cache.completedIDs should be initialized")
	}
	if r.Cache.backgroundIDs == nil {
		t.Error("cache.backgroundIDs should be initialized")
	}
	if len(r.Cache.Descriptions) != 1 {
		t.Fatalf("got %d descriptions, want 1", len(r.Cache.Descriptions))
	}
	if r.Offset != int64(len(line)) {
		t.Errorf("offset = %d, want %d", r.Offset, len(line))
	}
}

func TestExtractAgentDescriptionsIncremental_Truncate(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "conversation.jsonl")

	// Write initial content (2 lines)
	line1 := `{"type":"assistant","message":{"content":[{"type":"tool_use","id":"tool_trunc_1","name":"Agent","input":{"description":"Old task","prompt":"Old prompt","subagent_type":"Explore"}}]}}` + "\n"
	line2 := `{"type":"user","message":{"content":[{"type":"tool_result","tool_use_id":"tool_trunc_1","content":"Done"}]}}` + "\n"
	if err := os.WriteFile(path, []byte(line1+line2), 0o644); err != nil {
		t.Fatal(err)
	}

	// First read
	r, err := ExtractAgentDescriptionsIncremental(path, 0, nil)
	if err != nil {
		t.Fatalf("Phase 1: error = %v", err)
	}
	cache := r.Cache
	offset := r.Offset
	if len(cache.Descriptions) != 1 {
		t.Fatalf("Phase 1: got %d descriptions, want 1", len(cache.Descriptions))
	}
	if cache.Descriptions[0].Status != SubagentClosed {
		t.Errorf("Phase 1: Status = %q, want %q", cache.Descriptions[0].Status, SubagentClosed)
	}

	// Truncate: write shorter content (simulating file truncation/rewrite)
	newLine := `{"type":"assistant","message":{"content":[{"type":"tool_use","id":"tool_trunc_2","name":"Agent","input":{"description":"New task","prompt":"New prompt","subagent_type":"general-task-executor"}}]}}` + "\n"
	if err := os.WriteFile(path, []byte(newLine), 0o644); err != nil {
		t.Fatal(err)
	}

	// Read with old offset (larger than new file size) — should reset and re-parse
	r, err = ExtractAgentDescriptionsIncremental(path, offset, cache)
	if err != nil {
		t.Fatalf("Phase 2: error = %v", err)
	}
	cache = r.Cache
	offset = r.Offset
	// Cache should be reset: only new content
	if len(cache.Descriptions) != 1 {
		t.Fatalf("Phase 2: got %d descriptions, want 1 (cache should be reset)", len(cache.Descriptions))
	}
	if cache.Descriptions[0].Description != "New task" {
		t.Errorf("Phase 2: Description = %q, want %q", cache.Descriptions[0].Description, "New task")
	}
	if cache.Descriptions[0].ToolUseID != "tool_trunc_2" {
		t.Errorf("Phase 2: ToolUseID = %q, want %q", cache.Descriptions[0].ToolUseID, "tool_trunc_2")
	}
	if offset != int64(len(newLine)) {
		t.Errorf("Phase 2: offset = %d, want %d", offset, len(newLine))
	}
}

func TestExtractAgentDescriptionsIncremental_IncompleteLine(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "conversation.jsonl")

	// Write a complete line followed by an incomplete line (no newline at end)
	completeLine := `{"type":"assistant","message":{"content":[{"type":"tool_use","id":"tool_complete","name":"Agent","input":{"description":"Complete task","prompt":"Do complete task","subagent_type":"Explore"}}]}}` + "\n"
	incompleteLine := `{"type":"assistant","message":{"content":[{"type":"tool_use","id":"tool_incomplete","name":"Agent","input":{"description":"Incomplete task","prompt":"Do incomplete` // truncated, no closing
	if err := os.WriteFile(path, []byte(completeLine+incompleteLine), 0o644); err != nil {
		t.Fatal(err)
	}

	// First read: should parse the complete line, skip the incomplete one
	r, err := ExtractAgentDescriptionsIncremental(path, 0, nil)
	if err != nil {
		t.Fatalf("Phase 1: error = %v", err)
	}
	cache := r.Cache
	offset := r.Offset
	if len(cache.Descriptions) != 1 {
		t.Fatalf("Phase 1: got %d descriptions, want 1", len(cache.Descriptions))
	}
	if cache.Descriptions[0].Description != "Complete task" {
		t.Errorf("Phase 1: Description = %q, want %q", cache.Descriptions[0].Description, "Complete task")
	}
	// Offset should only include the complete line's bytes
	if offset != int64(len(completeLine)) {
		t.Errorf("Phase 1: offset = %d, want %d (should not include incomplete line)", offset, len(completeLine))
	}

	// Now complete the incomplete line by rewriting the file
	completedSecondLine := `{"type":"assistant","message":{"content":[{"type":"tool_use","id":"tool_incomplete","name":"Agent","input":{"description":"Incomplete task","prompt":"Do incomplete task fully","subagent_type":"general-task-executor"}}]}}` + "\n"
	if err := os.WriteFile(path, []byte(completeLine+completedSecondLine), 0o644); err != nil {
		t.Fatal(err)
	}

	// Second read: should parse the now-complete second line
	r, err = ExtractAgentDescriptionsIncremental(path, offset, cache)
	if err != nil {
		t.Fatalf("Phase 2: error = %v", err)
	}
	cache = r.Cache
	offset = r.Offset
	if len(cache.Descriptions) != 2 {
		t.Fatalf("Phase 2: got %d descriptions, want 2", len(cache.Descriptions))
	}
	if cache.Descriptions[1].Description != "Incomplete task" {
		t.Errorf("Phase 2: Descriptions[1].Description = %q, want %q", cache.Descriptions[1].Description, "Incomplete task")
	}
	if offset != int64(len(completeLine)+len(completedSecondLine)) {
		t.Errorf("Phase 2: offset = %d, want %d", offset, len(completeLine)+len(completedSecondLine))
	}
}

func TestDiscoverSubagents(t *testing.T) {
	dir := filepath.Join("testdata", "subagents")
	agents, err := DiscoverSubagents(dir)
	if err != nil {
		t.Fatalf("DiscoverSubagents() error = %v", err)
	}

	if len(agents) != 2 {
		t.Fatalf("got %d agents, want 2", len(agents))
	}

	// Verify both agents were found (order may vary due to glob)
	agentIDs := map[string]bool{}
	for _, a := range agents {
		agentIDs[a.AgentID] = true
	}
	if !agentIDs["abc123"] {
		t.Error("expected agent abc123 to be found")
	}
	if !agentIDs["def456"] {
		t.Error("expected agent def456 to be found")
	}
}

func TestDiscoverSubagents_SortedByNewestFirst(t *testing.T) {
	dir := t.TempDir()

	// Create agent files with known modification times
	// agent-old: oldest
	oldContent := `{"type":"user","message":{"content":"old task"},"isSidechain":true,"agentId":"old-agent","slug":"old","sessionId":"s1"}
{"type":"assistant","message":{"content":"done"},"isSidechain":true,"agentId":"old-agent","slug":"old","sessionId":"s1"}
`
	// agent-mid: middle
	midContent := `{"type":"user","message":{"content":"mid task"},"isSidechain":true,"agentId":"mid-agent","slug":"mid","sessionId":"s1"}
{"type":"assistant","message":{"content":"done"},"isSidechain":true,"agentId":"mid-agent","slug":"mid","sessionId":"s1"}
`
	// agent-new: newest
	newContent := `{"type":"user","message":{"content":"new task"},"isSidechain":true,"agentId":"new-agent","slug":"new","sessionId":"s1"}
{"type":"assistant","message":{"content":"done"},"isSidechain":true,"agentId":"new-agent","slug":"new","sessionId":"s1"}
`

	baseTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	files := []struct {
		name    string
		content string
		modTime time.Time
	}{
		{"agent-old.jsonl", oldContent, baseTime},
		{"agent-mid.jsonl", midContent, baseTime.Add(1 * time.Hour)},
		{"agent-new.jsonl", newContent, baseTime.Add(2 * time.Hour)},
	}

	for _, f := range files {
		path := filepath.Join(dir, f.name)
		if err := os.WriteFile(path, []byte(f.content), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(path, f.modTime, f.modTime); err != nil {
			t.Fatal(err)
		}
	}

	agents, err := DiscoverSubagents(dir)
	if err != nil {
		t.Fatalf("DiscoverSubagents() error = %v", err)
	}

	if len(agents) != 3 {
		t.Fatalf("got %d agents, want 3", len(agents))
	}

	// Expect newest first
	wantOrder := []string{"new-agent", "mid-agent", "old-agent"}
	for i, want := range wantOrder {
		if agents[i].AgentID != want {
			t.Errorf("agents[%d].AgentID = %q, want %q", i, agents[i].AgentID, want)
		}
	}
}
