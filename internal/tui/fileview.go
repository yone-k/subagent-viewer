package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/yone/subagent-viewer/internal/claude"
	"github.com/yone/subagent-viewer/internal/watcher"
)

// FileViewModel manages the Files tab view.
type FileViewModel struct {
	groups       []claude.FileGroup
	expanded     map[string]bool
	selected     int
	scrollOffset int
	width        int
	height       int
}

// NewFileViewModel creates a new FileViewModel.
func NewFileViewModel() FileViewModel {
	return FileViewModel{
		expanded: make(map[string]bool),
	}
}

// SetSize updates the view dimensions.
func (m *FileViewModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// Init initializes the model.
func (m FileViewModel) Init() tea.Cmd {
	return nil
}

// groupLines returns how many lines group at index takes in the view.
func (m FileViewModel) groupLines(index int) int {
	if index < 0 || index >= len(m.groups) {
		return 0
	}
	if m.expanded[m.groups[index].Hash] {
		return 1 + len(m.groups[index].Versions)
	}
	return 1
}

func (m FileViewModel) viewHeight() int {
	h := m.height
	if h < 1 {
		h = 1
	}
	return h
}

// clampScroll adjusts scrollOffset so the selected group is visible.
func (m FileViewModel) clampScroll() FileViewModel {
	viewHeight := m.viewHeight()
	// Upper bound check: scroll up to show selected
	if m.selected < m.scrollOffset {
		m.scrollOffset = m.selected
	}
	// Lower bound check: ensure selected is visible
	// If the single group at selected exceeds viewHeight, just show it from top
	if m.groupLines(m.selected) >= viewHeight {
		m.scrollOffset = m.selected
	} else {
		// Accumulate lines from scrollOffset to selected; if it exceeds viewHeight, advance scrollOffset
		totalLines := 0
		for i := m.scrollOffset; i <= m.selected; i++ {
			totalLines += m.groupLines(i)
		}
		for m.scrollOffset < m.selected && totalLines > viewHeight {
			totalLines -= m.groupLines(m.scrollOffset)
			m.scrollOffset++
		}
	}
	// Clamp scrollOffset
	maxOffset := max(0, len(m.groups)-1)
	if m.scrollOffset > maxOffset {
		m.scrollOffset = maxOffset
	}
	if m.scrollOffset < 0 {
		m.scrollOffset = 0
	}
	return m
}

// Update handles messages.
func (m FileViewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case watcher.FileHistoryUpdatedMsg:
		m.groups = msg.Groups
		if m.selected >= len(m.groups) && len(m.groups) > 0 {
			m.selected = len(m.groups) - 1
		} else if len(m.groups) == 0 {
			m.selected = 0
		}
		m = m.clampScroll()
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, FileKeys.Enter):
			if m.selected < len(m.groups) {
				hash := m.groups[m.selected].Hash
				m.expanded[hash] = !m.expanded[hash]
			}
			m = m.clampScroll()
		case key.Matches(msg, FileKeys.Escape):
			// Collapse all
			m.expanded = make(map[string]bool)
			m = m.clampScroll()
		default:
			switch msg.String() {
			case "up", "k":
				if m.selected > 0 {
					m.selected--
				}
			case "down", "j":
				if m.selected < len(m.groups)-1 {
					m.selected++
				}
			}
			m = m.clampScroll()
		}
	}
	return m, nil
}

// View renders the file history view.
func (m FileViewModel) View() string {
	if len(m.groups) == 0 {
		return EmptyStateStyle.Render("ファイル変更なし")
	}

	viewHeight := m.viewHeight()

	var b strings.Builder
	linesRendered := 0
	for i := m.scrollOffset; i < len(m.groups) && linesRendered < viewHeight; i++ {
		group := m.groups[i]
		prefix := "  "
		if i == m.selected {
			prefix = "> "
		}

		arrow := "▶"
		if m.expanded[group.Hash] {
			arrow = "▼"
		}

		if i == m.selected {
			b.WriteString(fmt.Sprintf("%s%s %s %s\n", prefix, arrow, SelectedLabelStyle.Render(group.Hash), SelectedDetailStyle.Render(fmt.Sprintf("(%d versions)", len(group.Versions)))))
		} else {
			b.WriteString(fmt.Sprintf("%s%s %s %s\n", prefix, arrow, group.Hash, DimStyle.Render(fmt.Sprintf("(%d versions)", len(group.Versions)))))
		}
		linesRendered++

		if m.expanded[group.Hash] {
			for _, v := range group.Versions {
				if linesRendered >= viewHeight {
					break
				}
				sizeStr := formatSize(v.Size)
				b.WriteString(fmt.Sprintf("    v%d  %s\n", v.Version, DimStyle.Render(sizeStr)))
				linesRendered++
			}
		}
	}

	return b.String()
}

func formatSize(bytes int64) string {
	switch {
	case bytes >= 1024*1024:
		return fmt.Sprintf("%.1f MB", float64(bytes)/(1024*1024))
	case bytes >= 1024:
		return fmt.Sprintf("%.1f KB", float64(bytes)/1024)
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}
