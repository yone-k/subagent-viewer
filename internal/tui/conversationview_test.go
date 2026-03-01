package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/yone/subagent-viewer/internal/claude"
)

// helper: create a ConversationViewModel with reasonable size and test entries.
func newTestConversationView(entries []claude.ConversationEntry) ConversationViewModel {
	m := NewConversationViewModel()
	m.SetSize(120, 30)
	if entries != nil {
		m.SetData("test-agent", entries, &claude.SubagentInfo{
			AgentID: "test-agent",
			Slug:    "test-slug",
		})
	}
	return m
}

func testEntries() []claude.ConversationEntry {
	return []claude.ConversationEntry{
		{Type: claude.EntryTypeUser, Content: []claude.ContentBlock{{Type: "text", Text: "Hello"}}},
		{Type: claude.EntryTypeAssistant, Content: []claude.ContentBlock{{Type: "text", Text: "Hi there"}}},
		{Type: claude.EntryTypeAssistant, Content: []claude.ContentBlock{{Type: "tool_use", ToolName: "Read", ToolInput: `{"path":"/tmp/file.txt"}`}}},
		{Type: claude.EntryTypeUser, Content: []claude.ContentBlock{{Type: "tool_result", Text: "file contents here"}}},
		{Type: claude.EntryTypeAssistant, Content: []claude.ContentBlock{{Type: "thinking", Text: "Let me think about this carefully"}}},
	}
}

var (
	tabKey = tea.KeyMsg{Type: tea.KeyTab}
	escKey = tea.KeyMsg{Type: tea.KeyEsc}
	jKey   = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")}
	kKey   = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")}
)

func TestConversationView_EmptyState(t *testing.T) {
	m := newTestConversationView(nil)
	view := m.View()

	if !strings.Contains(view, "エントリなし") && !strings.Contains(view, "エントリを選択してください") {
		t.Errorf("empty state should contain 'エントリなし' or 'エントリを選択してください', got:\n%s", view)
	}
}

func TestConversationView_EmptyState_SetDataEmpty(t *testing.T) {
	m := NewConversationViewModel()
	m.SetSize(120, 30)
	m.SetData("agent", []claude.ConversationEntry{}, nil)

	view := m.View()

	if !strings.Contains(view, "エントリなし") {
		t.Errorf("empty entries should show 'エントリなし', got:\n%s", view)
	}
	if !strings.Contains(view, "エントリを選択してください") {
		t.Errorf("empty entries should show 'エントリを選択してください' in detail pane, got:\n%s", view)
	}
}

func TestConversationView_PaneFocusSwitch(t *testing.T) {
	m := newTestConversationView(testEntries())

	// Initially should be PaneEntryList
	if m.focusPane != PaneEntryList {
		t.Fatalf("initial focusPane = %d, want PaneEntryList (%d)", m.focusPane, PaneEntryList)
	}

	// Press tab -> PaneDetail
	m, handled := m.Update(tabKey)
	if !handled {
		t.Error("tab should be handled")
	}
	if m.focusPane != PaneDetail {
		t.Errorf("after tab, focusPane = %d, want PaneDetail (%d)", m.focusPane, PaneDetail)
	}

	// Press tab again -> PaneEntryList
	m, handled = m.Update(tabKey)
	if !handled {
		t.Error("tab should be handled")
	}
	if m.focusPane != PaneEntryList {
		t.Errorf("after second tab, focusPane = %d, want PaneEntryList (%d)", m.focusPane, PaneEntryList)
	}
}

func TestConversationView_LeftPaneNavigation(t *testing.T) {
	m := newTestConversationView(testEntries())

	// Initially entrySelected=0
	if m.entrySelected != 0 {
		t.Fatalf("initial entrySelected = %d, want 0", m.entrySelected)
	}

	// Move down with j
	m, handled := m.Update(jKey)
	if !handled {
		t.Error("j key should be handled")
	}
	if m.entrySelected != 1 {
		t.Errorf("after j, entrySelected = %d, want 1", m.entrySelected)
	}

	// Move down again
	m, _ = m.Update(jKey)
	if m.entrySelected != 2 {
		t.Errorf("after second j, entrySelected = %d, want 2", m.entrySelected)
	}

	// Move up with k
	m, _ = m.Update(kKey)
	if m.entrySelected != 1 {
		t.Errorf("after k, entrySelected = %d, want 1", m.entrySelected)
	}

	// Verify detailScroll resets on navigation: first set a nonzero detailScroll
	m.detailScroll = 5
	m, _ = m.Update(jKey)
	if m.detailScroll != 0 {
		t.Errorf("after entry navigation, detailScroll = %d, want 0 (should reset)", m.detailScroll)
	}

	// At the beginning, k should not go below 0
	m.entrySelected = 0
	m, _ = m.Update(kKey)
	if m.entrySelected != 0 {
		t.Errorf("at start after k, entrySelected = %d, want 0", m.entrySelected)
	}

	// At the end, j should not exceed len-1
	m.entrySelected = len(m.entries) - 1
	m, _ = m.Update(jKey)
	if m.entrySelected != len(m.entries)-1 {
		t.Errorf("at end after j, entrySelected = %d, want %d", m.entrySelected, len(m.entries)-1)
	}
}

func TestConversationView_RightPaneScroll(t *testing.T) {
	// Create an entry with many lines of text to enable scrolling
	longText := strings.Repeat("This is a line of text.\n", 50)
	entries := []claude.ConversationEntry{
		{Type: claude.EntryTypeAssistant, Content: []claude.ContentBlock{{Type: "text", Text: longText}}},
	}
	m := newTestConversationView(entries)

	// Switch to detail pane
	m, _ = m.Update(tabKey)
	if m.focusPane != PaneDetail {
		t.Fatalf("focusPane = %d, want PaneDetail", m.focusPane)
	}

	// Render once to populate detailLines
	_ = m.View()

	// Initially detailScroll=0
	if m.detailScroll != 0 {
		t.Fatalf("initial detailScroll = %d, want 0", m.detailScroll)
	}

	// Scroll down with j
	m, handled := m.Update(jKey)
	if !handled {
		t.Error("j key should be handled in detail pane")
	}
	if m.detailScroll != 1 {
		t.Errorf("after j in detail pane, detailScroll = %d, want 1", m.detailScroll)
	}

	// Scroll down more
	m, _ = m.Update(jKey)
	if m.detailScroll != 2 {
		t.Errorf("after second j in detail pane, detailScroll = %d, want 2", m.detailScroll)
	}

	// Scroll up with k
	m, _ = m.Update(kKey)
	if m.detailScroll != 1 {
		t.Errorf("after k in detail pane, detailScroll = %d, want 1", m.detailScroll)
	}

	// At 0, k should not go negative
	m.detailScroll = 0
	m, _ = m.Update(kKey)
	if m.detailScroll != 0 {
		t.Errorf("at top after k, detailScroll = %d, want 0", m.detailScroll)
	}
}

func TestConversationView_EscapeReturns(t *testing.T) {
	m := newTestConversationView(testEntries())

	_, handled := m.Update(escKey)
	if handled {
		t.Error("escape should return handled=false so caller navigates back")
	}
}

func TestConversationView_DetailShowsFullContent(t *testing.T) {
	toolInput := `{"command":"ls -la","path":"/home/user"}`
	entries := []claude.ConversationEntry{
		{
			Type: claude.EntryTypeAssistant,
			Content: []claude.ContentBlock{
				{Type: "text", Text: "Here is the result"},
				{Type: "tool_use", ToolName: "Bash", ToolInput: toolInput},
			},
		},
		{
			Type: claude.EntryTypeUser,
			Content: []claude.ContentBlock{
				{Type: "tool_result", Text: "total 42\ndrwxr-xr-x 2 user user 4096 file.txt"},
			},
		},
		{
			Type: claude.EntryTypeAssistant,
			Content: []claude.ContentBlock{
				{Type: "thinking", Text: "The user wants to see directory listing"},
			},
		},
	}

	m := newTestConversationView(entries)

	// Select entry 0 (tool_use)
	m.entrySelected = 0
	view := m.View()

	if !strings.Contains(view, "[TOOL]") {
		t.Error("detail should show [TOOL] tag for tool_use block")
	}
	if !strings.Contains(view, "Bash") {
		t.Error("detail should show tool name 'Bash'")
	}
	// Tool input should be shown (formatted JSON includes keys)
	if !strings.Contains(view, "command") {
		t.Error("detail should show tool input content (key 'command')")
	}
	if !strings.Contains(view, "Here is the result") {
		t.Error("detail should show text block content")
	}

	// Select entry 1 (tool_result)
	m.entrySelected = 1
	view = m.View()

	if !strings.Contains(view, "total 42") {
		t.Error("detail should show full tool_result text without truncation")
	}
	if !strings.Contains(view, "file.txt") {
		t.Error("detail should show the complete tool_result content")
	}

	// Select entry 2 (thinking)
	m.entrySelected = 2
	view = m.View()

	if !strings.Contains(view, "[thinking]") {
		t.Error("detail should show [thinking] tag")
	}
	if !strings.Contains(view, "The user wants to see directory listing") {
		t.Error("detail should show full thinking text without truncation")
	}
}

func TestConversationView_SetDataResetsState(t *testing.T) {
	m := newTestConversationView(testEntries())

	// Simulate user navigation to put the model in a non-zero state
	m.entrySelected = 3
	m.entryScroll = 2
	m.detailScroll = 5
	m.focusPane = PaneDetail

	// SetData should reset scroll state
	m.SetData("new-agent", []claude.ConversationEntry{
		{Type: claude.EntryTypeUser, Content: []claude.ContentBlock{{Type: "text", Text: "New"}}},
	}, nil)

	if m.entrySelected != 0 {
		t.Errorf("after SetData, entrySelected = %d, want 0", m.entrySelected)
	}
	if m.entryScroll != 0 {
		t.Errorf("after SetData, entryScroll = %d, want 0", m.entryScroll)
	}
	if m.detailScroll != 0 {
		t.Errorf("after SetData, detailScroll = %d, want 0", m.detailScroll)
	}
	if m.agentID != "new-agent" {
		t.Errorf("after SetData, agentID = %q, want %q", m.agentID, "new-agent")
	}
}

func TestConversationView_UpdateEntriesPreservesSelection(t *testing.T) {
	entries := testEntries() // 5 entries
	m := newTestConversationView(entries)

	// Navigate to entry 3
	m.entrySelected = 3

	// Update with same number of entries: selection preserved
	m.UpdateEntries(entries, nil)
	if m.entrySelected != 3 {
		t.Errorf("after UpdateEntries with same count, entrySelected = %d, want 3", m.entrySelected)
	}

	// Update with fewer entries (2): selection should be clamped to len-1
	shortEntries := entries[:2]
	m.UpdateEntries(shortEntries, nil)
	if m.entrySelected != 1 {
		t.Errorf("after UpdateEntries with 2 entries, entrySelected = %d, want 1 (clamped to len-1)", m.entrySelected)
	}

	// Update with 1 entry: selection should be clamped to 0
	m.entrySelected = 1
	singleEntry := entries[:1]
	m.UpdateEntries(singleEntry, nil)
	if m.entrySelected != 0 {
		t.Errorf("after UpdateEntries with 1 entry, entrySelected = %d, want 0", m.entrySelected)
	}

	// Update with empty entries: selection stays at 0 (no crash)
	m.UpdateEntries([]claude.ConversationEntry{}, nil)
	// No assertion on entrySelected for empty - just verify no panic

	// Update with more entries: selection preserved at current value
	m.entrySelected = 0
	m.UpdateEntries(entries, nil)
	if m.entrySelected != 0 {
		t.Errorf("after UpdateEntries with more entries, entrySelected = %d, want 0", m.entrySelected)
	}
}
