package claude

import (
	"os"
	"path/filepath"
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

	if len(descriptions) != 3 {
		t.Fatalf("got %d descriptions, want 3", len(descriptions))
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
