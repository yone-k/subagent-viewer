package watcher

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/yone/subagent-viewer/internal/claude"
)

func writeAgentFile(t *testing.T, dir, filename, content string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, filename), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

const testAgentLine1 = `{"type":"user","message":{"content":"Hello agent"},"isSidechain":true,"agentId":"agent1","slug":"test-agent","sessionId":"sess1"}` + "\n"
const testAgentLine2 = `{"type":"assistant","message":{"content":[{"type":"text","text":"Hello! I can help."}]},"isSidechain":true,"agentId":"agent1","slug":"test-agent","sessionId":"sess1"}` + "\n"

func TestConversationWatcher_InitialLoad(t *testing.T) {
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "subagents")
	writeAgentFile(t, subDir, "agent-test1.jsonl", testAgentLine1+testAgentLine2)

	collector := newMsgCollector()
	program := newTestProgram(collector)
	defer program.Quit()

	cw := NewConversationWatcher(subDir, "", program, "", nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go cw.Start(ctx)

	// Should receive SubagentsDiscoveredMsg
	msg, ok := collector.waitForMsg(3 * time.Second)
	if !ok {
		t.Fatal("timed out waiting for SubagentsDiscoveredMsg")
	}
	discovered, ok := msg.(SubagentsDiscoveredMsg)
	if !ok {
		t.Fatalf("expected SubagentsDiscoveredMsg, got %T", msg)
	}
	if len(discovered.Agents) != 1 {
		t.Fatalf("got %d agents, want 1", len(discovered.Agents))
	}
	if discovered.Agents[0].AgentID != "agent1" {
		t.Errorf("AgentID = %q, want %q", discovered.Agents[0].AgentID, "agent1")
	}

	// Should receive ConversationUpdatedMsg
	msg, ok = collector.waitForMsg(3 * time.Second)
	if !ok {
		t.Fatal("timed out waiting for ConversationUpdatedMsg")
	}
	updated, ok := msg.(ConversationUpdatedMsg)
	if !ok {
		t.Fatalf("expected ConversationUpdatedMsg, got %T", msg)
	}
	if len(updated.Entries) != 2 {
		t.Errorf("got %d entries, want 2", len(updated.Entries))
	}
}

func TestConversationWatcher_DetectsNewEntries(t *testing.T) {
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "subagents")
	writeAgentFile(t, subDir, "agent-test1.jsonl", testAgentLine1)

	collector := newMsgCollector()
	program := newTestProgram(collector)
	defer program.Quit()

	cw := NewConversationWatcher(subDir, "", program, "", nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go cw.Start(ctx)

	// Drain initial messages
	for i := 0; i < 2; i++ {
		_, ok := collector.waitForMsg(3 * time.Second)
		if !ok {
			t.Fatalf("timed out waiting for initial message %d", i)
		}
	}

	// Append a new line
	f, err := os.OpenFile(filepath.Join(subDir, "agent-test1.jsonl"), os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.WriteString(testAgentLine2)
	f.Close()
	if err != nil {
		t.Fatal(err)
	}

	// Should receive ConversationUpdatedMsg with all entries (snapshot)
	msg, ok := collector.waitForMsg(5 * time.Second)
	if !ok {
		t.Fatal("timed out waiting for ConversationUpdatedMsg after append")
	}
	updated, ok := msg.(ConversationUpdatedMsg)
	if !ok {
		t.Fatalf("expected ConversationUpdatedMsg, got %T", msg)
	}
	// Should be a full snapshot with both entries
	if len(updated.Entries) != 2 {
		t.Errorf("got %d entries, want 2 (full snapshot)", len(updated.Entries))
	}
}

func TestConversationWatcher_DirNotExist(t *testing.T) {
	collector := newMsgCollector()
	program := newTestProgram(collector)
	defer program.Quit()

	cw := NewConversationWatcher("/nonexistent/path/subagents", "", program, "", nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go cw.Start(ctx)

	// Should receive empty SubagentsDiscoveredMsg (no error)
	msg, ok := collector.waitForMsg(3 * time.Second)
	if !ok {
		t.Fatal("timed out waiting for SubagentsDiscoveredMsg")
	}
	discovered, ok := msg.(SubagentsDiscoveredMsg)
	if !ok {
		t.Fatalf("expected SubagentsDiscoveredMsg, got %T", msg)
	}
	if len(discovered.Agents) != 0 {
		t.Errorf("got %d agents, want 0", len(discovered.Agents))
	}
}

func TestConversationWatcher_DirAppearsLater(t *testing.T) {
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "subagents")
	// Don't create subDir yet

	collector := newMsgCollector()
	program := newTestProgram(collector)
	defer program.Quit()

	findDir := func(sessionID string) (string, error) {
		// Check if the dir exists now
		if _, err := os.Stat(subDir); err == nil {
			return subDir, nil
		}
		return "", os.ErrNotExist
	}

	cw := NewConversationWatcher("", "test-session", program, "", findDir)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go cw.Start(ctx)

	// Should receive empty SubagentsDiscoveredMsg initially
	msg, ok := collector.waitForMsg(3 * time.Second)
	if !ok {
		t.Fatal("timed out waiting for initial SubagentsDiscoveredMsg")
	}
	discovered, ok := msg.(SubagentsDiscoveredMsg)
	if !ok {
		t.Fatalf("expected SubagentsDiscoveredMsg, got %T", msg)
	}
	if len(discovered.Agents) != 0 {
		t.Errorf("initially got %d agents, want 0", len(discovered.Agents))
	}

	// Now create the directory with an agent file
	writeAgentFile(t, subDir, "agent-late.jsonl", testAgentLine1)

	// Wait for the watcher to discover the new directory and its files
	found := false
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		msg, ok := collector.waitForMsg(2 * time.Second)
		if !ok {
			continue
		}
		if disc, ok := msg.(SubagentsDiscoveredMsg); ok && len(disc.Agents) > 0 {
			found = true
			break
		}
		if _, ok := msg.(ConversationUpdatedMsg); ok {
			found = true
			break
		}
	}
	if !found {
		t.Error("watcher did not detect directory that appeared later")
	}
}

func TestConversationWatcher_ParentStatusUpdate(t *testing.T) {
	// Use t.TempDir() for full isolation (no writes to ~/.claude/).
	// Directory structure: tmpDir/{sessionID}/subagents/ and tmpDir/{sessionID}.jsonl
	tmpDir := t.TempDir()
	sessionID := "test-session"
	subDir := filepath.Join(tmpDir, sessionID, "subagents")
	parentPath := filepath.Join(tmpDir, sessionID+".jsonl")

	// Create subagents directory
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create a subagent conversation file.
	// The prompt must match the Agent tool_use input prompt in the parent conversation.
	agentPrompt := "test prompt for agent"
	agentLine := `{"type":"user","message":{"content":"` + agentPrompt + `"},"isSidechain":true,"agentId":"agent1","slug":"test-agent","sessionId":"` + sessionID + `"}` + "\n"
	agentLine2 := `{"type":"assistant","message":{"content":[{"type":"text","text":"Working on it."}]},"isSidechain":true,"agentId":"agent1","slug":"test-agent","sessionId":"` + sessionID + `"}` + "\n"
	writeAgentFile(t, subDir, "agent-test1.jsonl", agentLine+agentLine2)

	// Create parent conversation with Agent tool_use but NO tool_result (agent is Running).
	parentToolUse := `{"type":"assistant","message":{"content":[{"type":"tool_use","id":"toolu_test1","name":"Agent","input":{"description":"test agent","prompt":"` + agentPrompt + `","subagent_type":"general-task-executor"}}]}}` + "\n"
	if err := os.WriteFile(parentPath, []byte(parentToolUse), 0o644); err != nil {
		t.Fatal(err)
	}

	collector := newMsgCollector()
	program := newTestProgram(collector)
	defer program.Quit()

	// projectPath is empty so resolveParentPath uses the dir fallback
	// (dir = tmpDir/{sessionID}/subagents -> parent = tmpDir/{sessionID}.jsonl)
	cw := NewConversationWatcher(subDir, sessionID, program, "", nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go cw.Start(ctx)

	// Wait for initial SubagentsDiscoveredMsg and verify agent status is Running.
	var initialDiscovered SubagentsDiscoveredMsg
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		msg, ok := collector.waitForMsg(3 * time.Second)
		if !ok {
			break
		}
		if disc, ok := msg.(SubagentsDiscoveredMsg); ok && len(disc.Agents) > 0 {
			initialDiscovered = disc
			break
		}
	}
	if len(initialDiscovered.Agents) == 0 {
		t.Fatal("did not receive initial SubagentsDiscoveredMsg with agents")
	}
	if initialDiscovered.Agents[0].Status != claude.SubagentRunning {
		t.Errorf("initial status = %q, want %q", initialDiscovered.Agents[0].Status, claude.SubagentRunning)
	}

	// Drain any remaining initial messages (ConversationUpdatedMsg).
	drainDeadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(drainDeadline) {
		_, ok := collector.waitForMsg(500 * time.Millisecond)
		if !ok {
			break
		}
	}

	// Append a tool_result entry to the parent conversation file (agent is now Closed).
	parentToolResult := `{"type":"user","message":{"content":[{"type":"tool_result","tool_use_id":"toolu_test1","content":"agent completed"}]}}` + "\n"
	f, err := os.OpenFile(parentPath, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.WriteString(parentToolResult)
	f.Close()
	if err != nil {
		t.Fatal(err)
	}

	// Wait for SubagentsDiscoveredMsg with updated status.
	var updatedDiscovered SubagentsDiscoveredMsg
	deadline = time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		msg, ok := collector.waitForMsg(3 * time.Second)
		if !ok {
			break
		}
		if disc, ok := msg.(SubagentsDiscoveredMsg); ok && len(disc.Agents) > 0 {
			updatedDiscovered = disc
			break
		}
	}
	if len(updatedDiscovered.Agents) == 0 {
		t.Fatal("did not receive updated SubagentsDiscoveredMsg after parent change")
	}
	if updatedDiscovered.Agents[0].Status != claude.SubagentClosed {
		t.Errorf("updated status = %q, want %q", updatedDiscovered.Agents[0].Status, claude.SubagentClosed)
	}
}
