package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/yone/subagent-viewer/internal/claude"
	"github.com/yone/subagent-viewer/internal/watcher"
)

// TestAgentView_EmptyState verifies that View() contains "サブエージェントなし" when no agents exist.
func TestAgentView_EmptyState(t *testing.T) {
	m := NewAgentViewModel()
	m.SetSize(80, 24)

	got := m.View()
	if !strings.Contains(got, "サブエージェントなし") {
		t.Errorf("expected View() to contain 'サブエージェントなし', got:\n%s", got)
	}
}

// TestAgentView_ModeTransition verifies mode transitions between list and conversation.
func TestAgentView_ModeTransition(t *testing.T) {
	agents := []claude.SubagentInfo{
		{AgentID: "a1", Slug: "test-agent", Prompt: "test prompt", EntryCount: 3},
	}

	t.Run("enter_key_transitions_to_conversation", func(t *testing.T) {
		m := NewAgentViewModel()
		m.SetSize(80, 24)

		// Inject agents via SubagentsDiscoveredMsg
		updated, _ := m.Update(watcher.SubagentsDiscoveredMsg{Agents: agents})
		m = updated.(AgentViewModel)

		if m.Mode() != AgentViewModeList {
			t.Fatalf("expected initial mode to be AgentViewModeList, got %d", m.Mode())
		}

		// Press enter to select first agent
		enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
		updated, _ = m.Update(enterMsg)
		m = updated.(AgentViewModel)

		if m.Mode() != AgentViewModeConversation {
			t.Errorf("expected mode to be AgentViewModeConversation after enter, got %d", m.Mode())
		}
	})

	t.Run("esc_key_returns_to_list", func(t *testing.T) {
		m := NewAgentViewModel()
		m.SetSize(80, 24)

		// Inject agents and transition to conversation mode
		updated, _ := m.Update(watcher.SubagentsDiscoveredMsg{Agents: agents})
		m = updated.(AgentViewModel)

		enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
		updated, _ = m.Update(enterMsg)
		m = updated.(AgentViewModel)

		if m.Mode() != AgentViewModeConversation {
			t.Fatalf("expected mode to be AgentViewModeConversation, got %d", m.Mode())
		}

		// Press esc to return to list
		escMsg := tea.KeyMsg{Type: tea.KeyEscape}
		updated, _ = m.Update(escMsg)
		m = updated.(AgentViewModel)

		if m.Mode() != AgentViewModeList {
			t.Errorf("expected mode to be AgentViewModeList after esc, got %d", m.Mode())
		}
	})
}

// TestAgentView_ConversationUpdatedMsg verifies that ConversationUpdatedMsg updates stored data.
func TestAgentView_ConversationUpdatedMsg(t *testing.T) {
	m := NewAgentViewModel()
	m.SetSize(80, 24)

	agents := []claude.SubagentInfo{
		{AgentID: "a1", Slug: "test-agent", Prompt: "test prompt", EntryCount: 3},
	}

	entries := []claude.ConversationEntry{
		{Type: claude.EntryTypeUser, Content: []claude.ContentBlock{{Type: "text", Text: "Hello"}}},
	}

	info := &claude.SubagentInfo{AgentID: "a1", Slug: "test-agent", EntryCount: 1}

	// Inject agents first
	updated, _ := m.Update(watcher.SubagentsDiscoveredMsg{Agents: agents})
	m = updated.(AgentViewModel)

	// Send ConversationUpdatedMsg
	updated, _ = m.Update(watcher.ConversationUpdatedMsg{
		AgentID: "a1",
		Entries: entries,
		Info:    info,
	})
	m = updated.(AgentViewModel)

	// Verify data is stored by entering conversation mode and checking the view
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	updated, _ = m.Update(enterMsg)
	m = updated.(AgentViewModel)

	if m.Mode() != AgentViewModeConversation {
		t.Fatalf("expected AgentViewModeConversation, got %d", m.Mode())
	}

	got := m.View()
	if !strings.Contains(got, "Hello") {
		t.Errorf("expected conversation view to contain 'Hello', got:\n%s", got)
	}
}
