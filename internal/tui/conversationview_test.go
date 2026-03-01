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
	escKey = tea.KeyMsg{Type: tea.KeyEsc}
	jKey   = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")}
	kKey   = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")}
	xKey   = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("X")}
	uKey   = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("U")}
	rKey   = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("R")}
	hKey   = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("H")}
)

func TestConversationView_EmptyState(t *testing.T) {
	m := newTestConversationView(nil)
	view := m.View()

	if !strings.Contains(view, "エントリなし") {
		t.Errorf("empty state should contain 'エントリなし', got:\n%s", view)
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
}

func TestConversationView_DefaultFilterShowsTextOnly(t *testing.T) {
	m := newTestConversationView(testEntries())
	view := m.View()

	// Default filter: text=true, others=false
	// Should show text entries
	if !strings.Contains(view, "Hello") {
		t.Error("default filter should show text content 'Hello'")
	}
	if !strings.Contains(view, "Hi there") {
		t.Error("default filter should show text content 'Hi there'")
	}

	// Should NOT show tool_use, tool_result, thinking entries
	if strings.Contains(view, "[TOOL]") {
		t.Error("default filter should hide tool_use blocks")
	}
	if strings.Contains(view, "[TOOL_RESULT]") {
		t.Error("default filter should hide tool_result blocks")
	}
	if strings.Contains(view, "[THINKING]") {
		t.Error("default filter should hide thinking blocks")
	}
}

func TestConversationView_FilterToggle(t *testing.T) {
	m := newTestConversationView(testEntries())

	// Initially text=true. Toggle text off with X.
	m, handled := m.Update(xKey)
	if !handled {
		t.Error("X key should be handled")
	}
	if m.filterTypes["text"] != false {
		t.Error("after X, text filter should be false")
	}

	// Toggle tool_use on with U
	m, handled = m.Update(uKey)
	if !handled {
		t.Error("U key should be handled")
	}
	if m.filterTypes["tool_use"] != true {
		t.Error("after U, tool_use filter should be true")
	}

	// Toggle tool_result on with R
	m, handled = m.Update(rKey)
	if !handled {
		t.Error("R key should be handled")
	}
	if m.filterTypes["tool_result"] != true {
		t.Error("after R, tool_result filter should be true")
	}

	// Toggle thinking on with H
	m, handled = m.Update(hKey)
	if !handled {
		t.Error("H key should be handled")
	}
	if m.filterTypes["thinking"] != true {
		t.Error("after H, thinking filter should be true")
	}

	// Verify the view now shows tool/result/thinking but not text
	view := m.View()
	if strings.Contains(view, "Hello") || strings.Contains(view, "Hi there") {
		t.Error("with text filter off, text content should not appear")
	}
	if !strings.Contains(view, "[TOOL]") {
		t.Error("with tool_use filter on, [TOOL] should appear")
	}
	if !strings.Contains(view, "[TOOL_RESULT]") {
		t.Error("with tool_result filter on, [TOOL_RESULT] should appear")
	}
	if !strings.Contains(view, "[THINKING]") {
		t.Error("with thinking filter on, [THINKING] should appear")
	}
}

func TestConversationView_AllFiltersOffShowsMessage(t *testing.T) {
	m := newTestConversationView(testEntries())

	// Turn off text (the only default-on filter)
	m, _ = m.Update(xKey)

	view := m.View()
	if !strings.Contains(view, "フィルタ条件に一致するエントリなし") {
		t.Errorf("all filters off should show 'フィルタ条件に一致するエントリなし', got:\n%s", view)
	}
}

func TestConversationView_Scroll(t *testing.T) {
	// Create entries with enough text lines to require scrolling
	longText := strings.Repeat("This is a line of text.\n", 50)
	entries := []claude.ConversationEntry{
		{Type: claude.EntryTypeAssistant, Content: []claude.ContentBlock{{Type: "text", Text: longText}}},
	}
	m := newTestConversationView(entries)

	// Initially scrollOffset=0
	if m.scrollOffset != 0 {
		t.Fatalf("initial scrollOffset = %d, want 0", m.scrollOffset)
	}

	// Scroll down with j
	m, handled := m.Update(jKey)
	if !handled {
		t.Error("j key should be handled")
	}
	if m.scrollOffset != 1 {
		t.Errorf("after j, scrollOffset = %d, want 1", m.scrollOffset)
	}

	// Scroll down more
	m, _ = m.Update(jKey)
	if m.scrollOffset != 2 {
		t.Errorf("after second j, scrollOffset = %d, want 2", m.scrollOffset)
	}

	// Scroll up with k
	m, _ = m.Update(kKey)
	if m.scrollOffset != 1 {
		t.Errorf("after k, scrollOffset = %d, want 1", m.scrollOffset)
	}

	// At 0, k should not go negative
	m.scrollOffset = 0
	m, _ = m.Update(kKey)
	if m.scrollOffset != 0 {
		t.Errorf("at top after k, scrollOffset = %d, want 0", m.scrollOffset)
	}
}

func TestConversationView_EscapeReturns(t *testing.T) {
	m := newTestConversationView(testEntries())

	_, handled := m.Update(escKey)
	if handled {
		t.Error("escape should return handled=false so caller navigates back")
	}
}

func TestConversationView_FilterBar(t *testing.T) {
	m := newTestConversationView(testEntries())
	view := m.View()

	// Filter bar should be rendered
	if !strings.Contains(view, "Filter:") {
		t.Error("view should contain filter bar with 'Filter:'")
	}
}

func TestConversationView_SetDataResetsState(t *testing.T) {
	m := newTestConversationView(testEntries())

	// Simulate scrolling
	m.scrollOffset = 5

	// SetData should reset scroll state
	m.SetData("new-agent", []claude.ConversationEntry{
		{Type: claude.EntryTypeUser, Content: []claude.ContentBlock{{Type: "text", Text: "New"}}},
	}, nil)

	if m.scrollOffset != 0 {
		t.Errorf("after SetData, scrollOffset = %d, want 0", m.scrollOffset)
	}
	if m.agentID != "new-agent" {
		t.Errorf("after SetData, agentID = %q, want %q", m.agentID, "new-agent")
	}
	if !m.filteredDirty {
		t.Error("after SetData, filteredDirty should be true")
	}
}

func TestConversationView_UpdateEntriesMarksDirty(t *testing.T) {
	entries := testEntries()
	m := newTestConversationView(entries)

	// Build cache through Update() round-trip (value receiver + clampScroll triggers filteredBlocks)
	m, _ = m.Update(uKey) // toggle filter on
	m, _ = m.Update(uKey) // toggle back to original
	if m.filteredDirty {
		t.Fatal("after filter round-trip, filteredDirty should be false")
	}

	// UpdateEntries should set dirty
	m.UpdateEntries(entries, nil)
	if !m.filteredDirty {
		t.Error("after UpdateEntries, filteredDirty should be true")
	}
}

func TestConversationView_ContentBlockRendering(t *testing.T) {
	entries := []claude.ConversationEntry{
		{
			Type: claude.EntryTypeAssistant,
			Content: []claude.ContentBlock{
				{Type: "text", Text: "Here is the result"},
				{Type: "tool_use", ToolName: "Bash", ToolInput: `{"command":"ls -la"}`},
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

	// Enable all filters
	m, _ = m.Update(uKey) // tool_use on
	m, _ = m.Update(rKey) // tool_result on
	m, _ = m.Update(hKey) // thinking on

	view := m.View()

	if !strings.Contains(view, "Here is the result") {
		t.Error("view should show text content")
	}
	if !strings.Contains(view, "[TOOL]") {
		t.Error("view should show [TOOL] tag for tool_use block")
	}
	if !strings.Contains(view, "Bash") {
		t.Error("view should show tool name 'Bash'")
	}
	if !strings.Contains(view, "command") {
		t.Error("view should show tool input content")
	}
	if !strings.Contains(view, "[TOOL_RESULT]") {
		t.Error("view should show [TOOL_RESULT] tag for tool_result block")
	}
	if !strings.Contains(view, "total 42") {
		t.Error("view should show tool_result text")
	}
	if !strings.Contains(view, "[THINKING]") {
		t.Error("view should show [THINKING] tag for thinking block")
	}
	if !strings.Contains(view, "The user wants to see directory listing") {
		t.Error("view should show thinking text")
	}
}

func TestConversationView_SeparatorSkippedWhenAllBlocksFiltered(t *testing.T) {
	// Entry with only tool_use content
	entries := []claude.ConversationEntry{
		{Type: claude.EntryTypeAssistant, Content: []claude.ContentBlock{{Type: "tool_use", ToolName: "Read", ToolInput: `{}`}}},
	}
	m := newTestConversationView(entries)

	// Default: tool_use is off, so this entry's separator should also be hidden
	view := m.View()
	if strings.Contains(view, "[A]") {
		t.Error("separator should be hidden when all blocks in entry are filtered out")
	}
}

func TestConversationView_SeparatorShowsCorrectTag(t *testing.T) {
	entries := []claude.ConversationEntry{
		{Type: claude.EntryTypeUser, Content: []claude.ContentBlock{{Type: "text", Text: "User msg"}}},
		{Type: claude.EntryTypeAssistant, Content: []claude.ContentBlock{{Type: "text", Text: "Assistant msg"}}},
	}
	m := newTestConversationView(entries)

	view := m.View()
	if !strings.Contains(view, "[U]") {
		t.Error("user entry should have [U] separator tag")
	}
	if !strings.Contains(view, "[A]") {
		t.Error("assistant entry should have [A] separator tag")
	}
}

func TestRenderContentBlock(t *testing.T) {
	tests := []struct {
		name     string
		block    claude.ContentBlock
		contains []string
	}{
		{
			name:     "text block",
			block:    claude.ContentBlock{Type: "text", Text: "Hello world"},
			contains: []string{"Hello world"},
		},
		{
			name:     "tool_use block",
			block:    claude.ContentBlock{Type: "tool_use", ToolName: "Read", ToolInput: `{"path":"/tmp"}`},
			contains: []string{"[TOOL]", "Read", "path"},
		},
		{
			name:     "tool_result block",
			block:    claude.ContentBlock{Type: "tool_result", Text: "result data"},
			contains: []string{"[TOOL_RESULT]", "result data"},
		},
		{
			name:     "thinking block",
			block:    claude.ContentBlock{Type: "thinking", Text: "deep thought"},
			contains: []string{"[THINKING]", "deep thought"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lines := renderContentBlock(tt.block, 80)
			joined := strings.Join(lines, "\n")
			for _, want := range tt.contains {
				if !strings.Contains(joined, want) {
					t.Errorf("renderContentBlock output should contain %q, got:\n%s", want, joined)
				}
			}
		})
	}
}
