package tui

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/yone/subagent-viewer/internal/claude"
	"github.com/yone/subagent-viewer/internal/watcher"
)

func TestFileView_UpdateWithGroups(t *testing.T) {
	m := NewFileViewModel()
	m.SetSize(80, 24)

	groups := []claude.FileGroup{
		{Hash: "abcd1234", Versions: []claude.FileVersion{
			{Hash: "abcd1234", Version: 1, Path: "/tmp/abcd1234@v1", Size: 100},
			{Hash: "abcd1234", Version: 2, Path: "/tmp/abcd1234@v2", Size: 150},
		}},
	}
	newModel, _ := m.Update(watcher.FileHistoryUpdatedMsg{Groups: groups})
	m = newModel.(FileViewModel)

	if len(m.groups) != 1 {
		t.Errorf("expected 1 group, got %d", len(m.groups))
	}
}

func TestFileView_ExpandCollapse(t *testing.T) {
	m := NewFileViewModel()
	m.SetSize(80, 24)

	groups := []claude.FileGroup{
		{Hash: "abcd1234", Versions: []claude.FileVersion{
			{Hash: "abcd1234", Version: 1, Path: "/tmp/abcd1234@v1", Size: 100},
		}},
	}
	newModel, _ := m.Update(watcher.FileHistoryUpdatedMsg{Groups: groups})
	m = newModel.(FileViewModel)

	// Initially collapsed
	if m.expanded["abcd1234"] {
		t.Error("groups should be collapsed initially")
	}

	// Toggle expand with Enter
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(FileViewModel)

	if !m.expanded["abcd1234"] {
		t.Error("group should be expanded after Enter")
	}

	// Toggle collapse with Enter
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(FileViewModel)

	if m.expanded["abcd1234"] {
		t.Error("group should be collapsed after second Enter")
	}
}

func TestFileView_VersionDisplay(t *testing.T) {
	m := NewFileViewModel()
	m.SetSize(80, 24)

	groups := []claude.FileGroup{
		{Hash: "abcd1234", Versions: []claude.FileVersion{
			{Hash: "abcd1234", Version: 1, Path: "/tmp/abcd1234@v1", Size: 1024},
			{Hash: "abcd1234", Version: 2, Path: "/tmp/abcd1234@v2", Size: 2048},
		}},
	}
	newModel, _ := m.Update(watcher.FileHistoryUpdatedMsg{Groups: groups})
	m = newModel.(FileViewModel)

	// Expand group
	m.expanded["abcd1234"] = true

	view := m.View()
	// Should show version info with size
	if !strings.Contains(view, "v1") {
		t.Error("view should contain version v1")
	}
	if !strings.Contains(view, "v2") {
		t.Error("view should contain version v2")
	}
}

func TestFileView_EmptyState(t *testing.T) {
	m := NewFileViewModel()
	m.SetSize(80, 24)

	view := m.View()
	if !strings.Contains(view, "ファイル変更なし") {
		t.Errorf("empty state not shown: %s", view)
	}
}

func TestFileView_SelectedClampOnGroupsReduced(t *testing.T) {
	m := NewFileViewModel()
	m.SetSize(80, 24)

	// Set up 3 groups
	groups := []claude.FileGroup{
		{Hash: "aaa", Versions: []claude.FileVersion{{Hash: "aaa", Version: 1, Path: "/tmp/aaa@v1", Size: 10}}},
		{Hash: "bbb", Versions: []claude.FileVersion{{Hash: "bbb", Version: 1, Path: "/tmp/bbb@v1", Size: 20}}},
		{Hash: "ccc", Versions: []claude.FileVersion{{Hash: "ccc", Version: 1, Path: "/tmp/ccc@v1", Size: 30}}},
	}
	newModel, _ := m.Update(watcher.FileHistoryUpdatedMsg{Groups: groups})
	m = newModel.(FileViewModel)

	// Move selected to the last item (index 2)
	m.selected = 2

	// Now reduce groups to only 1 item
	reducedGroups := []claude.FileGroup{
		{Hash: "aaa", Versions: []claude.FileVersion{{Hash: "aaa", Version: 1, Path: "/tmp/aaa@v1", Size: 10}}},
	}
	newModel, _ = m.Update(watcher.FileHistoryUpdatedMsg{Groups: reducedGroups})
	m = newModel.(FileViewModel)

	if m.selected >= len(m.groups) {
		t.Errorf("selected = %d, but only %d groups exist", m.selected, len(m.groups))
	}
	if m.selected != 0 {
		t.Errorf("selected = %d, want 0", m.selected)
	}
}

func TestFileView_ScrollFollowsSelection(t *testing.T) {
	m := NewFileViewModel()
	m.SetSize(80, 5) // height=5: viewHeight = 5

	// Create 10 groups, each with 2 versions
	groups := make([]claude.FileGroup, 10)
	for i := range groups {
		hash := fmt.Sprintf("hash%04d", i)
		groups[i] = claude.FileGroup{
			Hash: hash,
			Versions: []claude.FileVersion{
				{Hash: hash, Version: 1, Path: fmt.Sprintf("/tmp/%s@v1", hash), Size: 100},
				{Hash: hash, Version: 2, Path: fmt.Sprintf("/tmp/%s@v2", hash), Size: 200},
			},
		}
	}
	newModel, _ := m.Update(watcher.FileHistoryUpdatedMsg{Groups: groups})
	m = newModel.(FileViewModel)

	if m.scrollOffset != 0 {
		t.Errorf("initial scrollOffset = %d, want 0", m.scrollOffset)
	}

	// Move down 4 times to item index 4 — items 0-4 fill 5 lines (1 each collapsed)
	for i := 0; i < 4; i++ {
		newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
		m = newModel.(FileViewModel)
	}
	if m.selected != 4 {
		t.Errorf("selected = %d, want 4", m.selected)
	}
	if m.scrollOffset != 0 {
		t.Errorf("scrollOffset = %d, want 0 (all collapsed, 5 fit in viewHeight 5)", m.scrollOffset)
	}

	// Move down to index 5 — should scroll
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	m = newModel.(FileViewModel)
	if m.selected != 5 {
		t.Errorf("selected = %d, want 5", m.selected)
	}
	if m.scrollOffset != 1 {
		t.Errorf("scrollOffset = %d, want 1", m.scrollOffset)
	}

	// Now expand group 5 (selected) — it becomes 3 lines (1 header + 2 versions)
	// viewHeight = 5, so scrollOffset must adjust
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(FileViewModel)
	if !m.expanded[groups[5].Hash] {
		t.Error("group 5 should be expanded")
	}
	// After expansion: group 5 = 3 lines, groups 1-4 = 1 line each
	// scrollOffset needs to ensure group 5 (3 lines) fits in viewHeight (5)
	// groups from scrollOffset to 5 must fit in 5 lines
	// If scrollOffset=1: groups 1,2,3,4 (4 lines) + group 5 (3 lines) = 7 > 5, so need to scroll more
	// scrollOffset should adjust so that accumulated lines from scrollOffset to 5 <= 5
	// scrollOffset=3: groups 3,4 (2 lines) + group 5 (3 lines) = 5 <= 5 ✓
	if m.scrollOffset < 1 {
		t.Errorf("scrollOffset = %d, should be >= 1 after expansion", m.scrollOffset)
	}

	// Move back up — scroll should follow
	for i := 0; i < 5; i++ {
		newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
		m = newModel.(FileViewModel)
	}
	if m.selected != 0 {
		t.Errorf("selected = %d, want 0", m.selected)
	}
	if m.scrollOffset != 0 {
		t.Errorf("scrollOffset = %d, want 0 after moving to top", m.scrollOffset)
	}
}
