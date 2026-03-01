package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/yone/subagent-viewer/internal/claude"
)

func TestSelectorModel_Init(t *testing.T) {
	sessions := []claude.SessionInfo{
		{SessionID: "s1", Project: "/test/proj", Timestamp: 1000, FirstInput: "hello"},
		{SessionID: "s2", Project: "/test/proj2", Timestamp: 2000, FirstInput: "world"},
	}
	m := NewSelectorModel(sessions)
	if len(m.sessions) != 2 {
		t.Errorf("expected 2 sessions loaded, got %d", len(m.sessions))
	}
}

func TestSelectorModel_SelectSession(t *testing.T) {
	sessions := []claude.SessionInfo{
		{SessionID: "s1", Project: "/test/proj", Timestamp: 1000, FirstInput: "hello"},
	}
	m := NewSelectorModel(sessions)
	m.SetSize(80, 24)

	// Simulate Enter key
	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(SelectorModel)

	if cmd == nil {
		t.Fatal("expected a command from Enter press")
	}
	// Execute the command to get the message
	msg := cmd()
	if _, ok := msg.(SessionSelectedMsg); !ok {
		t.Errorf("expected SessionSelectedMsg, got %T", msg)
	}
}

func TestSelectorModel_EmptyList(t *testing.T) {
	m := NewSelectorModel(nil)
	m.SetSize(80, 24)
	view := m.View()
	if !strings.Contains(view, "セッションが見つかりません") {
		t.Errorf("empty state message not found in view: %s", view)
	}
}

func TestSessionItem_LongFirstInput(t *testing.T) {
	longInput := strings.Repeat("a", 61)
	item := sessionItem{session: claude.SessionInfo{
		SessionID:  "s1",
		Project:    "/test",
		Timestamp:  1000,
		FirstInput: longInput,
	}}
	// Description() now returns the raw string; truncation is done at render time via truncateText
	desc := item.Description()
	if desc != longInput {
		t.Errorf("Description() should return raw input, got %q", desc)
	}

	// Verify truncateText handles long strings correctly
	truncated := truncateText(longInput, 60)
	if !strings.Contains(truncated, "...") {
		t.Errorf("truncateText should append '...' for long input, got %q", truncated)
	}
	if len(truncated) > 60 {
		t.Errorf("truncateText result should not exceed maxWidth, got len=%d", len(truncated))
	}
}

func TestTruncateText(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxWidth int
		wantFit  bool // true if result should equal input (fits within maxWidth)
	}{
		{"short ASCII", "hello", 10, true},
		{"exact fit", "hello", 5, true},
		{"needs truncation", "hello world", 8, false},
		{"newlines removed", "line1\nline2\nline3", 50, true},
		{"Japanese full-width", "日本語テスト", 6, false}, // 6 chars = 12 columns, maxWidth=6 triggers truncation
		{"Japanese fits", "日本語", 10, true},              // 3 chars = 6 columns, fits in 10
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateText(tt.input, tt.maxWidth)
			if strings.Contains(result, "\n") {
				t.Errorf("truncateText should not contain newlines, got %q", result)
			}
			if tt.wantFit && strings.Contains(result, "...") {
				t.Errorf("expected no truncation, got %q", result)
			}
			if !tt.wantFit && !strings.Contains(result, "...") {
				t.Errorf("expected truncation with '...', got %q", result)
			}
		})
	}
}

func TestSelectorModel_DisplayFormat(t *testing.T) {
	sessions := []claude.SessionInfo{
		{
			SessionID:      "s1",
			Project:        "/test/project",
			Timestamp:      1772326237190,
			FirstInput:     "プロジェクトを分析して",
			HasTasks:       true,
			HasDebugLog:    true,
			HasFileHistory: false,
		},
	}
	m := NewSelectorModel(sessions)
	m.SetSize(120, 40)

	// Get the item description
	items := m.list.Items()
	if len(items) == 0 {
		t.Fatal("no items in list")
	}
	item := items[0].(sessionItem)
	title := item.Title()
	desc := item.Description()

	// Should contain project path
	if !strings.Contains(title, "/test/project") {
		t.Errorf("title should contain project path, got %q", title)
	}

	// Should contain first input
	if !strings.Contains(desc, "プロジェクトを分析して") {
		t.Errorf("description should contain first input, got %q", desc)
	}
}
