package watcher

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
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
