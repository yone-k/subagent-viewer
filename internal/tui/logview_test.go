package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/yone-k/cc-subagent-viewer/internal/claude"
	"github.com/yone-k/cc-subagent-viewer/internal/watcher"
)

func makeLogEntry(level claude.LogLevel, message string) claude.LogEntry {
	return claude.LogEntry{
		Timestamp: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		Level:     level,
		Message:   message,
		Raw:       "2026-03-01T00:00:00.000Z [" + string(level) + "] " + message,
	}
}

// enterKey, shiftLeftKey, shiftRightKey are declared in conversationview_test.go (same package)

func TestLogView_UpdateWithEntries(t *testing.T) {
	m := NewLogViewModel()
	m.SetSize(80, 24)

	entries := []claude.LogEntry{
		makeLogEntry(claude.LevelDEBUG, "debug msg"),
		makeLogEntry(claude.LevelERROR, "error msg"),
	}
	newModel, _ := m.Update(watcher.LogEntriesMsg{Entries: entries, Initial: true})
	m = newModel.(LogViewModel)

	if m.EntryCount() != 2 {
		t.Errorf("expected 2 entries, got %d", m.EntryCount())
	}
}

func TestLogView_RingBuffer(t *testing.T) {
	m := NewLogViewModel()
	m.SetSize(80, 24)

	// Add more than maxEntries (10000)
	entries := make([]claude.LogEntry, 10500)
	for i := range entries {
		entries[i] = makeLogEntry(claude.LevelDEBUG, "msg")
	}
	newModel, _ := m.Update(watcher.LogEntriesMsg{Entries: entries, Initial: true})
	m = newModel.(LogViewModel)

	if m.EntryCount() > 10000 {
		t.Errorf("ring buffer should cap at 10000, got %d", m.EntryCount())
	}
}

func TestLogView_FilterByLevel(t *testing.T) {
	m := NewLogViewModel()
	m.SetSize(80, 24)

	entries := []claude.LogEntry{
		makeLogEntry(claude.LevelDEBUG, "debug msg"),
		makeLogEntry(claude.LevelERROR, "error msg"),
		makeLogEntry(claude.LevelWARN, "warn msg"),
	}
	newModel, _ := m.Update(watcher.LogEntriesMsg{Entries: entries, Initial: true})
	m = newModel.(LogViewModel)

	// DEBUG is disabled by default, so only ERROR and WARN should be visible
	view := m.View()
	if strings.Contains(view, "debug msg") {
		t.Error("filtered out DEBUG should not appear in view")
	}
	if !strings.Contains(view, "error msg") {
		t.Error("ERROR should appear in view")
	}
}

func TestLogView_FilterToggle_AllLevels(t *testing.T) {
	m := NewLogViewModel()
	m.SetSize(80, 24)

	// DEBUGはデフォルトオフ、他は全てオン
	if m.filterLevels[claude.LevelDEBUG] != false {
		t.Error("DEBUG should be disabled by default")
	}
	for _, def := range logFilterDefs[1:] {
		if !m.filterLevels[def.level] {
			t.Errorf("level %s should be enabled by default", def.level)
		}
	}

	// filterCursor=0 (Debug) の状態でenterを押すとDEBUGがオンになる
	newModel, _ := m.Update(enterKey)
	m = newModel.(LogViewModel)
	if !m.filterLevels[claude.LevelDEBUG] {
		t.Error("DEBUG should be enabled after enter at cursor 0")
	}

	// カーソルを右に移動してError(index=1)をトグル
	newModel, _ = m.Update(shiftRightKey)
	m = newModel.(LogViewModel)
	if m.filterCursor != 1 {
		t.Errorf("filterCursor = %d, want 1", m.filterCursor)
	}
	newModel, _ = m.Update(enterKey)
	m = newModel.(LogViewModel)
	if m.filterLevels[claude.LevelERROR] {
		t.Error("ERROR should be disabled after toggle")
	}

	// もう一度enterで戻す
	newModel, _ = m.Update(enterKey)
	m = newModel.(LogViewModel)
	if !m.filterLevels[claude.LevelERROR] {
		t.Error("ERROR should be re-enabled after second toggle")
	}
}

func TestLogView_Search(t *testing.T) {
	m := NewLogViewModel()
	m.SetSize(80, 24)

	entries := []claude.LogEntry{
		makeLogEntry(claude.LevelERROR, "hello world"),
		makeLogEntry(claude.LevelERROR, "foo bar"),
		makeLogEntry(claude.LevelERROR, "hello again"),
	}
	newModel, _ := m.Update(watcher.LogEntriesMsg{Entries: entries, Initial: true})
	m = newModel.(LogViewModel)

	// Set search query directly
	m.searchQuery = "hello"
	m.filteredDirty = true

	view := m.View()
	if !strings.Contains(view, "hello world") {
		t.Error("matching entry should appear")
	}
	if strings.Contains(view, "foo bar") {
		t.Error("non-matching entry should not appear")
	}
}

func TestLogView_SearchMode(t *testing.T) {
	m := NewLogViewModel()
	m.SetSize(80, 24)

	// Enter search mode with /
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	m = newModel.(LogViewModel)
	if !m.searching {
		t.Error("should be in search mode after /")
	}

	// Exit search mode with Esc
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	m = newModel.(LogViewModel)
	if m.searching {
		t.Error("should exit search mode after Esc")
	}
}

func TestLogView_AutoScroll(t *testing.T) {
	m := NewLogViewModel()
	m.SetSize(80, 24)

	// With no entries, isAtBottom should be true
	if !m.isAtBottom() {
		t.Error("isAtBottom should be true with no entries")
	}

	// Add many entries to exceed viewport
	entries := make([]claude.LogEntry, 50)
	for i := range entries {
		entries[i] = makeLogEntry(claude.LevelERROR, "line")
	}
	newModel, _ := m.Update(watcher.LogEntriesMsg{Entries: entries, Initial: true})
	m = newModel.(LogViewModel)

	// After initial load, should be at bottom
	if !m.isAtBottom() {
		t.Error("isAtBottom should be true after initial load")
	}

	// Scroll up - should no longer be at bottom
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	m = newModel.(LogViewModel)
	if m.isAtBottom() {
		t.Error("isAtBottom should be false after scrolling up")
	}
}

func TestLogView_AutoScroll_NewEntries(t *testing.T) {
	m := NewLogViewModel()
	m.SetSize(80, 24)

	// Add entries to fill viewport (use ERROR since DEBUG is off by default)
	entries := make([]claude.LogEntry, 50)
	for i := range entries {
		entries[i] = makeLogEntry(claude.LevelERROR, "line")
	}
	newModel, _ := m.Update(watcher.LogEntriesMsg{Entries: entries, Initial: true})
	m = newModel.(LogViewModel)

	// At bottom: new entries should auto-scroll
	newEntries := []claude.LogEntry{makeLogEntry(claude.LevelERROR, "new entry")}
	newModel, _ = m.Update(watcher.LogEntriesMsg{Entries: newEntries, Initial: false})
	m = newModel.(LogViewModel)

	view := m.View()
	if !strings.Contains(view, "new entry") {
		t.Error("auto-scroll should show new entry at bottom")
	}

	// Scroll up to leave bottom
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	m = newModel.(LogViewModel)
	if m.isAtBottom() {
		t.Error("should not be at bottom after scrolling up")
	}

	// Add another entry - should NOT auto-scroll since not at bottom
	prevOffset := m.scrollOffset
	newEntries2 := []claude.LogEntry{makeLogEntry(claude.LevelERROR, "another new")}
	newModel, _ = m.Update(watcher.LogEntriesMsg{Entries: newEntries2, Initial: false})
	m = newModel.(LogViewModel)

	if m.scrollOffset != prevOffset {
		t.Errorf("scrollOffset should not change when not at bottom, was %d, got %d", prevOffset, m.scrollOffset)
	}
}

func TestLogView_EmptyState(t *testing.T) {
	m := NewLogViewModel()
	m.SetSize(80, 24)

	view := m.View()
	if !strings.Contains(view, "デバッグログなし") {
		t.Errorf("empty state message not found in view: %s", view)
	}
}

func TestLogView_ScrollOffset_ClampOnFilterChange(t *testing.T) {
	m := NewLogViewModel()
	m.SetSize(80, 24)

	entries := []claude.LogEntry{
		makeLogEntry(claude.LevelDEBUG, "debug1"),
		makeLogEntry(claude.LevelDEBUG, "debug2"),
		makeLogEntry(claude.LevelDEBUG, "debug3"),
		makeLogEntry(claude.LevelERROR, "error1"),
	}
	newModel, _ := m.Update(watcher.LogEntriesMsg{Entries: entries, Initial: true})
	m = newModel.(LogViewModel)

	// Set scrollOffset to a large value
	m.scrollOffset = 100

	// DEBUGはデフォルトオフなので、現在はERRORのみ(1件)が表示
	// filterCursorは0(Debug)の状態。Enterを押してDEBUGをオンにする
	newModel, _ = m.Update(enterKey)
	m = newModel.(LogViewModel)

	// After enabling DEBUG: 4 short entries = 4 rendered lines
	// viewHeight = 24 - 4 = 20, maxScroll = max(4 - 20, 0) = 0
	// scrollOffset should be clamped to 0
	if m.scrollOffset != 0 {
		t.Errorf("scrollOffset = %d, want 0 (totalLines 4 < viewHeight 20)", m.scrollOffset)
	}
}

func TestLogView_FilterCursorNavigation(t *testing.T) {
	m := NewLogViewModel()
	m.SetSize(80, 24)

	// Initial cursor should be at 0
	if m.filterCursor != 0 {
		t.Errorf("initial filterCursor = %d, want 0", m.filterCursor)
	}

	// Move right to the last position
	for i := 0; i < len(logFilterDefs)-1; i++ {
		newModel, _ := m.Update(shiftRightKey)
		m = newModel.(LogViewModel)
	}
	if m.filterCursor != len(logFilterDefs)-1 {
		t.Errorf("filterCursor = %d, want %d", m.filterCursor, len(logFilterDefs)-1)
	}

	// Right at the end should not go further
	newModel, _ := m.Update(shiftRightKey)
	m = newModel.(LogViewModel)
	if m.filterCursor != len(logFilterDefs)-1 {
		t.Errorf("filterCursor should stay at %d, got %d", len(logFilterDefs)-1, m.filterCursor)
	}

	// Move left back to 0
	for i := 0; i < len(logFilterDefs)-1; i++ {
		newModel, _ := m.Update(shiftLeftKey)
		m = newModel.(LogViewModel)
	}
	if m.filterCursor != 0 {
		t.Errorf("filterCursor = %d, want 0", m.filterCursor)
	}

	// Left at 0 should not go negative
	newModel, _ = m.Update(shiftLeftKey)
	m = newModel.(LogViewModel)
	if m.filterCursor != 0 {
		t.Errorf("filterCursor should stay at 0, got %d", m.filterCursor)
	}
}

func TestLogView_WordWrap(t *testing.T) {
	m := NewLogViewModel()
	m.SetSize(60, 24)

	// Message much longer than available width after prefix (~21 chars for ERROR)
	longMsg := strings.TrimSpace(strings.Repeat("word ", 20))
	entries := []claude.LogEntry{
		makeLogEntry(claude.LevelERROR, longMsg),
	}
	newModel, _ := m.Update(watcher.LogEntriesMsg{Entries: entries, Initial: true})
	m = newModel.(LogViewModel)

	view := m.View()
	lines := strings.Split(view, "\n")

	// Count lines containing "word" (the message content)
	wordLines := 0
	for _, line := range lines {
		if strings.Contains(line, "word") {
			wordLines++
		}
	}
	if wordLines < 2 {
		t.Errorf("expected long message to wrap into at least 2 lines, got %d lines containing 'word'", wordLines)
	}
}

func TestLogView_ContinuationLineIndent(t *testing.T) {
	m := NewLogViewModel()
	m.SetSize(60, 24)

	longMsg := strings.TrimSpace(strings.Repeat("word ", 20))
	entries := []claude.LogEntry{
		makeLogEntry(claude.LevelERROR, longMsg),
		makeLogEntry(claude.LevelMCP, longMsg),
	}
	newModel, _ := m.Update(watcher.LogEntriesMsg{Entries: entries, Initial: true})
	m = newModel.(LogViewModel)

	view := m.View()
	lines := strings.Split(view, "\n")

	// Collect indent widths from continuation lines (start with spaces, contain "word")
	indentWidths := make(map[int]bool)
	for _, line := range lines {
		if strings.HasPrefix(line, " ") && strings.Contains(line, "word") {
			trimmed := strings.TrimLeft(line, " ")
			indent := len(line) - len(trimmed)
			if indent > 0 {
				indentWidths[indent] = true
			}
		}
	}

	// There should be continuation lines with leading spaces
	if len(indentWidths) == 0 {
		t.Fatal("expected continuation lines with leading spaces, found none")
	}

	// ERROR ([ERROR]=7) and MCP ([MCP]=5) have different prefix widths,
	// so their continuation lines should have different indent widths
	if len(indentWidths) < 2 {
		t.Errorf("expected different indent widths for ERROR and MCP levels, got single width: %v", indentWidths)
	}
}

func TestLogView_ShortMessageNoWrap(t *testing.T) {
	m := NewLogViewModel()
	m.SetSize(80, 24)

	entries := []claude.LogEntry{
		makeLogEntry(claude.LevelERROR, "short msg"),
	}
	newModel, _ := m.Update(watcher.LogEntriesMsg{Entries: entries, Initial: true})
	m = newModel.(LogViewModel)

	view := m.View()
	lines := strings.Split(view, "\n")

	// Short message should appear on exactly 1 line
	msgLines := 0
	for _, line := range lines {
		if strings.Contains(line, "short msg") {
			msgLines++
		}
	}
	if msgLines != 1 {
		t.Errorf("short message should appear on exactly 1 line, got %d", msgLines)
	}
}

func TestLogView_LineScroll(t *testing.T) {
	m := NewLogViewModel()
	m.SetSize(40, 10) // viewHeight = 10 - 4 = 6

	// Create entries with long messages that wrap to multiple lines at width 40
	longMsg := strings.TrimSpace(strings.Repeat("word ", 15))
	entries := make([]claude.LogEntry, 5)
	for i := range entries {
		entries[i] = makeLogEntry(claude.LevelERROR, longMsg)
	}
	newModel, _ := m.Update(watcher.LogEntriesMsg{Entries: entries, Initial: true})
	m = newModel.(LogViewModel)

	if !m.isAtBottom() {
		t.Fatal("should be at bottom after initial load")
	}

	// With wrapping, scrollOffset at bottom should be greater than entry count
	// (line-based scrolling produces more scroll positions than entry-based)
	bottomOffset := m.scrollOffset
	if bottomOffset <= len(entries) {
		t.Errorf("scrollOffset at bottom (%d) should be > entry count (%d) when messages wrap to multiple lines",
			bottomOffset, len(entries))
	}

	// Scroll up with k: scrollOffset should decrease by 1
	prevOffset := m.scrollOffset
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	m = newModel.(LogViewModel)
	if m.scrollOffset != prevOffset-1 {
		t.Errorf("after k, scrollOffset should decrease by 1: was %d, got %d", prevOffset, m.scrollOffset)
	}

	// Scroll down with j: scrollOffset should increase by 1
	prevOffset = m.scrollOffset
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	m = newModel.(LogViewModel)
	if m.scrollOffset != prevOffset+1 {
		t.Errorf("after j, scrollOffset should increase by 1: was %d, got %d", prevOffset, m.scrollOffset)
	}
}

func TestLogView_ResizeClampScroll(t *testing.T) {
	m := NewLogViewModel()
	m.SetSize(40, 10) // narrow width

	// Create entries with long messages that wrap at narrow width
	longMsg := strings.TrimSpace(strings.Repeat("word ", 15))
	entries := make([]claude.LogEntry, 5)
	for i := range entries {
		entries[i] = makeLogEntry(claude.LevelERROR, longMsg)
	}
	newModel, _ := m.Update(watcher.LogEntriesMsg{Entries: entries, Initial: true})
	m = newModel.(LogViewModel)

	if !m.isAtBottom() {
		t.Fatal("should be at bottom after initial load")
	}
	narrowBottomOffset := m.scrollOffset

	// Widen to 200: messages no longer wrap → fewer total lines → lower maxScroll
	m.SetSize(200, 10)

	// Should not panic
	view := m.View()
	if view == "" {
		t.Fatal("View should not return empty string after resize")
	}

	// After widening, scrollOffset should be clamped to the new (lower) maxScroll.
	// At width 200, all messages fit on one line → 5 total lines.
	// viewHeight = 6, maxScroll = max(5 - 6, 0) = 0.
	// So scrollOffset should be less than the narrow bottom offset.
	if m.scrollOffset >= narrowBottomOffset {
		t.Errorf("scrollOffset (%d) should be less than narrow bottom offset (%d) after widening",
			m.scrollOffset, narrowBottomOffset)
	}
}
