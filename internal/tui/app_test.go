package tui

import (
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/yone/subagent-viewer/internal/claude"
	"github.com/yone/subagent-viewer/internal/watcher"
)

func TestAppModel_InitialState(t *testing.T) {
	m := NewAppModel(nil)
	if m.state != StateSelector {
		t.Errorf("initial state = %d, want StateSelector (%d)", m.state, StateSelector)
	}
}

func TestAppModel_SessionSelected(t *testing.T) {
	sessions := []claude.SessionInfo{
		{SessionID: "test-session", Project: "/test", HasTasks: true, HasDebugLog: true},
	}
	m := NewAppModel(sessions)
	m.width = 80
	m.height = 24

	newModel, cmd := m.Update(SessionSelectedMsg{Session: sessions[0]})
	mPtr := newModel.(*AppModel)

	if mPtr.state != StateViewer {
		t.Errorf("state = %d, want StateViewer (%d)", mPtr.state, StateViewer)
	}
	if mPtr.session.SessionID != "test-session" {
		t.Errorf("session ID = %q, want \"test-session\"", mPtr.session.SessionID)
	}
	if cmd == nil {
		t.Error("expected a command for starting watchers")
	}
}

func TestAppModel_TabSwitch(t *testing.T) {
	m := NewAppModel(nil)
	m.state = StateViewer
	m.width = 80
	m.height = 24

	// Switch to tab 2 (Agents)
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("2")})
	mPtr := newModel.(*AppModel)
	if mPtr.tabs.Active != 1 {
		t.Errorf("Active tab = %d, want 1", mPtr.tabs.Active)
	}

	// Switch to tab 3 (Logs)
	newModel, _ = mPtr.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("3")})
	mPtr = newModel.(*AppModel)
	if mPtr.tabs.Active != 2 {
		t.Errorf("Active tab = %d, want 2", mPtr.tabs.Active)
	}

	// Switch to tab 4 (Files)
	newModel, _ = mPtr.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("4")})
	mPtr = newModel.(*AppModel)
	if mPtr.tabs.Active != 3 {
		t.Errorf("Active tab = %d, want 3", mPtr.tabs.Active)
	}

	// Switch to tab 1 (Tasks)
	newModel, _ = mPtr.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("1")})
	mPtr = newModel.(*AppModel)
	if mPtr.tabs.Active != 0 {
		t.Errorf("Active tab = %d, want 0", mPtr.tabs.Active)
	}
}

func TestAppModel_TabCycle(t *testing.T) {
	m := NewAppModel(nil)
	m.state = StateViewer
	m.width = 80
	m.height = 24

	// Tab forward
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	mPtr := newModel.(*AppModel)
	if mPtr.tabs.Active != 1 {
		t.Errorf("Active tab after Tab = %d, want 1", mPtr.tabs.Active)
	}

	// Shift+Tab backward
	newModel, _ = mPtr.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	mPtr = newModel.(*AppModel)
	if mPtr.tabs.Active != 0 {
		t.Errorf("Active tab after Shift+Tab = %d, want 0", mPtr.tabs.Active)
	}
}

func TestAppModel_WindowResize(t *testing.T) {
	m := NewAppModel(nil)
	m.state = StateViewer

	newModel, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	mPtr := newModel.(*AppModel)

	if mPtr.width != 120 {
		t.Errorf("width = %d, want 120", mPtr.width)
	}
	if mPtr.height != 40 {
		t.Errorf("height = %d, want 40", mPtr.height)
	}
}

func TestAppModel_Quit(t *testing.T) {
	m := NewAppModel(nil)
	m.state = StateViewer

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if cmd == nil {
		t.Error("expected quit command")
	}
}

func TestAppModel_ActiveSessionIndicator(t *testing.T) {
	m := NewAppModel(nil)
	m.state = StateViewer
	m.session = claude.SessionInfo{SessionID: "test", Project: "/test"}
	m.sessionActive = true
	m.width = 80
	m.height = 24

	view := m.View()
	if len(view) == 0 {
		t.Error("view should not be empty")
	}
	if !strings.Contains(view, "セッションアクティブ") {
		t.Errorf("view should contain active session indicator, got %q", view)
	}
}

func TestAppModel_TaskChangedMsgUpdatesBadge(t *testing.T) {
	m := NewAppModel(nil)
	m.state = StateViewer
	m.width = 80
	m.height = 24

	// Send a TaskChangedMsg with a new task
	task := claude.Task{ID: "1", Subject: "Test task", Status: "in_progress"}
	newModel, _ := m.Update(watcher.TaskChangedMsg{Task: task})
	mPtr := newModel.(*AppModel)

	if mPtr.tabs.badges[0] != 1 {
		t.Errorf("badge count = %d, want 1", mPtr.tabs.badges[0])
	}

	// Send another TaskChangedMsg with a second task
	task2 := claude.Task{ID: "2", Subject: "Test task 2", Status: "pending"}
	newModel, _ = mPtr.Update(watcher.TaskChangedMsg{Task: task2})
	mPtr = newModel.(*AppModel)

	if mPtr.tabs.badges[0] != 2 {
		t.Errorf("badge count = %d, want 2", mPtr.tabs.badges[0])
	}
}

func TestAppModel_WatcherErrorMsg(t *testing.T) {
	m := NewAppModel(nil)
	m.state = StateViewer
	m.session = claude.SessionInfo{SessionID: "test", Project: "/test"}
	m.width = 80
	m.height = 24

	// Send a WatcherErrorMsg
	newModel, _ := m.Update(watcher.WatcherErrorMsg{Source: "tasks", Err: errors.New("permission denied")})
	mPtr := newModel.(*AppModel)

	if mPtr.lastError == "" {
		t.Error("lastError should be set after WatcherErrorMsg")
	}
	if !strings.Contains(mPtr.lastError, "tasks") {
		t.Errorf("lastError should contain source, got %q", mPtr.lastError)
	}
	if !strings.Contains(mPtr.lastError, "permission denied") {
		t.Errorf("lastError should contain error message, got %q", mPtr.lastError)
	}

	// View should contain the error
	view := mPtr.View()
	if !strings.Contains(view, "permission denied") {
		t.Errorf("view should contain error message, got %q", view)
	}
}

func TestAppModel_ArrowKeyTabSwitch(t *testing.T) {
	m := NewAppModel(nil)
	m.state = StateViewer
	m.width = 80
	m.height = 24

	// Right arrow should move to next tab
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRight})
	mPtr := newModel.(*AppModel)
	if mPtr.tabs.Active != 1 {
		t.Errorf("Active tab after Right = %d, want 1", mPtr.tabs.Active)
	}

	// Right arrow again
	newModel, _ = mPtr.Update(tea.KeyMsg{Type: tea.KeyRight})
	mPtr = newModel.(*AppModel)
	if mPtr.tabs.Active != 2 {
		t.Errorf("Active tab after Right = %d, want 2", mPtr.tabs.Active)
	}

	// Left arrow should move to previous tab
	newModel, _ = mPtr.Update(tea.KeyMsg{Type: tea.KeyLeft})
	mPtr = newModel.(*AppModel)
	if mPtr.tabs.Active != 1 {
		t.Errorf("Active tab after Left = %d, want 1", mPtr.tabs.Active)
	}
}

func TestAppModel_CleanupNilCancel(t *testing.T) {
	m := NewAppModel(nil)
	// cancelFunc is nil — Cleanup should not panic
	m.Cleanup()
}

func TestAppModel_CleanupCallsCancel(t *testing.T) {
	m := NewAppModel(nil)
	called := false
	m.cancelFunc = func() { called = true }

	m.Cleanup()

	if !called {
		t.Error("Cleanup should call cancelFunc")
	}
}

func TestAppModel_SubagentsDiscoveredMsg(t *testing.T) {
	m := NewAppModel(nil)
	m.state = StateViewer
	m.width = 80
	m.height = 24

	agents := []claude.SubagentInfo{
		{AgentID: "agent1", Slug: "test-agent", Prompt: "Hello", EntryCount: 5},
	}
	newModel, _ := m.Update(watcher.SubagentsDiscoveredMsg{Agents: agents})
	mPtr := newModel.(*AppModel)

	if len(mPtr.agentView.agents) != 1 {
		t.Errorf("agents count = %d, want 1", len(mPtr.agentView.agents))
	}
}

func TestAppModel_ConversationUpdatedMsg(t *testing.T) {
	m := NewAppModel(nil)
	m.state = StateViewer
	m.width = 80
	m.height = 24

	entries := []claude.ConversationEntry{
		{Type: claude.EntryTypeUser, Content: []claude.ContentBlock{{Type: "text", Text: "Hello"}}},
	}
	newModel, _ := m.Update(watcher.ConversationUpdatedMsg{
		AgentID: "agent1",
		Entries: entries,
		Info:    &claude.SubagentInfo{AgentID: "agent1", Slug: "test"},
	})
	mPtr := newModel.(*AppModel)

	if len(mPtr.agentView.conversations["agent1"]) != 1 {
		t.Errorf("conversation entries = %d, want 1", len(mPtr.agentView.conversations["agent1"]))
	}
}
