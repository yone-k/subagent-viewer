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
	escKey   = tea.KeyMsg{Type: tea.KeyEsc}
	jKey     = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")}
	kKey     = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")}
	enterKey      = tea.KeyMsg{Type: tea.KeyEnter}
	shiftLeftKey  = tea.KeyMsg{Type: tea.KeyShiftLeft}
	shiftRightKey = tea.KeyMsg{Type: tea.KeyShiftRight}
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

	// filterCursor=0 (Text)。TextはデフォルトON。Enterでトグルオフ。
	m, handled := m.Update(enterKey)
	if !handled {
		t.Error("enter key should be handled")
	}
	if m.filterTypes["text"] != false {
		t.Error("after enter at cursor 0, text filter should be false")
	}

	// カーソルを右に移動してTool(index=1)に移動し、Enterでトグルオン
	m, _ = m.Update(shiftRightKey)
	m, _ = m.Update(enterKey)
	if m.filterTypes["tool_use"] != true {
		t.Error("after enter at cursor 1, tool_use filter should be true")
	}

	// カーソルを右に移動してResult(index=2)に移動し、Enterでトグルオン
	m, _ = m.Update(shiftRightKey)
	m, _ = m.Update(enterKey)
	if m.filterTypes["tool_result"] != true {
		t.Error("after enter at cursor 2, tool_result filter should be true")
	}

	// カーソルを右に移動してThinking(index=3)に移動し、Enterでトグルオン
	m, _ = m.Update(shiftRightKey)
	m, _ = m.Update(enterKey)
	if m.filterTypes["thinking"] != true {
		t.Error("after enter at cursor 3, thinking filter should be true")
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

	// Turn off text (the only default-on filter) - cursor is at 0 (Text)
	m, _ = m.Update(enterKey)

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

	// SetData now scrolls to bottom, so isAtBottom should be true
	if !m.isAtBottom() {
		t.Fatal("after SetData, isAtBottom should be true")
	}

	// Scroll up with k
	m, handled := m.Update(kKey)
	if !handled {
		t.Error("k key should be handled")
	}
	if m.isAtBottom() {
		t.Error("after k, should not be at bottom")
	}

	// Scroll down with j should increase scrollOffset
	prevOffset := m.scrollOffset
	m, _ = m.Update(jKey)
	if m.scrollOffset != prevOffset+1 {
		t.Errorf("after j, scrollOffset = %d, want %d", m.scrollOffset, prevOffset+1)
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

	// SetData should scroll to bottom
	m.SetData("new-agent", []claude.ConversationEntry{
		{Type: claude.EntryTypeUser, Content: []claude.ContentBlock{{Type: "text", Text: "New"}}},
	}, nil)

	if !m.isAtBottom() {
		t.Error("after SetData, isAtBottom should be true")
	}
	if m.agentID != "new-agent" {
		t.Errorf("after SetData, agentID = %q, want %q", m.agentID, "new-agent")
	}
	// filteredDirty is set to true by SetData, but maxScroll() rebuilds the cache,
	// so by the time SetData returns, filteredDirty is false again.
}

func TestConversationView_UpdateEntriesMarksDirty(t *testing.T) {
	// Use long text entries so maxScroll > 0 (enables scroll-up to leave bottom)
	longText := strings.Repeat("Line of text.\n", 50)
	entries := []claude.ConversationEntry{
		{Type: claude.EntryTypeAssistant, Content: []claude.ContentBlock{{Type: "text", Text: longText}}},
	}
	m := newTestConversationView(entries)

	// Build cache through Update() round-trip (value receiver + clampScroll triggers filteredBlocks)
	// カーソルを1(Tool)に移動してenterでtool_useをON
	m, _ = m.Update(shiftRightKey)
	m, _ = m.Update(enterKey)
	// もう一度enterでtool_useをOFF（元に戻す）
	m, _ = m.Update(enterKey)
	if m.filteredDirty {
		t.Fatal("after filter round-trip, filteredDirty should be false")
	}

	// Scroll up so we are not at bottom (UpdateEntries only preserves filteredDirty
	// when not at bottom, since maxScroll() rebuilds cache)
	m.scrollOffset = 0

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

	// Enable all filters using cursor navigation
	// cursor=0 (Text already on), move to 1 (Tool) and enable
	m, _ = m.Update(shiftRightKey)
	m, _ = m.Update(enterKey)
	// move to 2 (Result) and enable
	m, _ = m.Update(shiftRightKey)
	m, _ = m.Update(enterKey)
	// move to 3 (Thinking) and enable
	m, _ = m.Update(shiftRightKey)
	m, _ = m.Update(enterKey)

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

func TestConversationView_AutoScrollOnUpdateEntries(t *testing.T) {
	longText := strings.Repeat("This is a line.\n", 50)
	entries := []claude.ConversationEntry{
		{Type: claude.EntryTypeAssistant, Content: []claude.ContentBlock{{Type: "text", Text: longText}}},
	}
	m := newTestConversationView(entries)

	// Should be at bottom after SetData
	if !m.isAtBottom() {
		t.Fatal("should be at bottom after SetData")
	}

	// UpdateEntries while at bottom should keep at bottom
	moreEntries := append(entries, claude.ConversationEntry{
		Type: claude.EntryTypeUser, Content: []claude.ContentBlock{{Type: "text", Text: "new message"}},
	})
	m.UpdateEntries(moreEntries, nil)
	if !m.isAtBottom() {
		t.Error("should still be at bottom after UpdateEntries when was at bottom")
	}

	// Scroll up, then UpdateEntries should NOT scroll to bottom
	m.scrollOffset = 0
	m.UpdateEntries(moreEntries, nil)
	if m.isAtBottom() {
		t.Error("should not be at bottom after UpdateEntries when was not at bottom")
	}
}

func TestConversationView_FilterCursorNavigation(t *testing.T) {
	m := newTestConversationView(testEntries())

	// Initial cursor at 0
	if m.filterCursor != 0 {
		t.Errorf("initial filterCursor = %d, want 0", m.filterCursor)
	}

	// Move right to the last position (3)
	for i := 0; i < len(convFilterDefs)-1; i++ {
		m, _ = m.Update(shiftRightKey)
	}
	if m.filterCursor != len(convFilterDefs)-1 {
		t.Errorf("filterCursor = %d, want %d", m.filterCursor, len(convFilterDefs)-1)
	}

	// Right at the end should not go further
	m, _ = m.Update(shiftRightKey)
	if m.filterCursor != len(convFilterDefs)-1 {
		t.Errorf("filterCursor should stay at %d, got %d", len(convFilterDefs)-1, m.filterCursor)
	}

	// Move left back to 0
	for i := 0; i < len(convFilterDefs)-1; i++ {
		m, _ = m.Update(shiftLeftKey)
	}
	if m.filterCursor != 0 {
		t.Errorf("filterCursor = %d, want 0", m.filterCursor)
	}

	// Left at 0 should not go negative
	m, _ = m.Update(shiftLeftKey)
	if m.filterCursor != 0 {
		t.Errorf("filterCursor should stay at 0, got %d", m.filterCursor)
	}
}

func TestWordWrap(t *testing.T) {
	tests := []struct {
		name  string
		input string
		width int
		want  string
	}{
		{
			name:  "simple text no wrap needed",
			input: "hello world",
			width: 80,
			want:  "hello world",
		},
		{
			name:  "wraps long line",
			input: "aaa bbb ccc",
			width: 7,
			want:  "aaa bbb\nccc",
		},
		{
			name:  "preserves leading indentation",
			input: "    indented text",
			width: 80,
			want:  "    indented text",
		},
		{
			name:  "preserves consecutive spaces",
			input: "key:  value",
			width: 80,
			want:  "key:  value",
		},
		{
			name:  "preserves multi-level JSON-like indentation",
			input: "{\n  \"key\": {\n    \"nested\": true\n  }\n}",
			width: 80,
			want:  "{\n  \"key\": {\n    \"nested\": true\n  }\n}",
		},
		{
			name:  "preserves empty lines in multiline text",
			input: "line1\n\nline3",
			width: 80,
			want:  "line1\n\nline3",
		},
		{
			name:  "width zero returns input unchanged",
			input: "hello world",
			width: 0,
			want:  "hello world",
		},
		{
			name:  "preserves 4-space indentation across lines",
			input: "    line1\n    line2\n        deep",
			width: 80,
			want:  "    line1\n    line2\n        deep",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := wordWrap(tt.input, tt.width)
			if got != tt.want {
				t.Errorf("wordWrap(%q, %d)\ngot:  %q\nwant: %q", tt.input, tt.width, got, tt.want)
			}
		})
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
