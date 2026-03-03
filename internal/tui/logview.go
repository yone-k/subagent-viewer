package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/yone-k/cc-subagent-viewer/internal/claude"
	"github.com/yone-k/cc-subagent-viewer/internal/watcher"
)

const maxLogEntries = 10000

type logFilterDef struct {
	label string
	level claude.LogLevel
}

var logFilterDefs = []logFilterDef{
	{"Debug", claude.LevelDEBUG},
	{"Error", claude.LevelERROR},
	{"Warn", claude.LevelWARN},
	{"MCP", claude.LevelMCP},
	{"Startup", claude.LevelSTARTUP},
	{"Meta", claude.LevelMETA},
	{"Attach", claude.LevelATTACHMENT},
}

// LogViewModel manages the Logs tab view.
type LogViewModel struct {
	entries            []claude.LogEntry
	filterLevels       map[claude.LogLevel]bool
	searchQuery        string
	searchInput        textinput.Model
	searching          bool
	filterCursor       int
	scrollOffset       int
	width              int
	height             int
	filteredCache      []claude.LogEntry
	filteredDirty      bool
	renderedLinesCache []string
}

// NewLogViewModel creates a new LogViewModel.
func NewLogViewModel() LogViewModel {
	ti := textinput.New()
	ti.Placeholder = "検索..."
	ti.CharLimit = 100

	return LogViewModel{
		filterLevels: map[claude.LogLevel]bool{
			claude.LevelDEBUG:      false,
			claude.LevelERROR:      true,
			claude.LevelWARN:       true,
			claude.LevelMCP:        true,
			claude.LevelSTARTUP:    true,
			claude.LevelMETA:       true,
			claude.LevelATTACHMENT: true,
		},
		searchInput:   ti,
		filteredDirty: true,
	}
}

// SetSize updates the view dimensions.
func (m *LogViewModel) SetSize(width, height int) {
	if m.width == width && m.height == height {
		return
	}
	m.width = width
	m.height = height
	m.filteredDirty = true
	m.clampScroll()
}

// EntryCount returns the number of stored entries.
func (m LogViewModel) EntryCount() int {
	return len(m.entries)
}

// Init initializes the model.
func (m LogViewModel) Init() tea.Cmd {
	return nil
}

// Update handles messages.
func (m LogViewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case watcher.LogEntriesMsg:
		wasAtBottom := m.isAtBottom()
		if msg.Initial {
			m.entries = msg.Entries
		} else {
			m.entries = append(m.entries, msg.Entries...)
		}
		// Ring buffer: trim oldest entries
		if len(m.entries) > maxLogEntries {
			m.entries = m.entries[len(m.entries)-maxLogEntries:]
		}
		m.filteredDirty = true
		if wasAtBottom {
			m.scrollToBottom()
		}
		return m, nil

	case tea.KeyMsg:
		if m.searching {
			switch msg.Type {
			case tea.KeyEscape:
				m.searching = false
				m.searchInput.Blur()
				return m, nil
			case tea.KeyEnter:
				m.searchQuery = m.searchInput.Value()
				m.searching = false
				m.searchInput.Blur()
				m.filteredDirty = true
				m.clampScroll()
				return m, nil
			default:
				var cmd tea.Cmd
				m.searchInput, cmd = m.searchInput.Update(msg)
				m.searchQuery = m.searchInput.Value()
				m.filteredDirty = true
				return m, cmd
			}
		}

		switch {
		case key.Matches(msg, LogKeys.FilterLeft):
			if m.filterCursor > 0 {
				m.filterCursor--
			}
		case key.Matches(msg, LogKeys.FilterRight):
			if m.filterCursor < len(logFilterDefs)-1 {
				m.filterCursor++
			}
		case key.Matches(msg, LogKeys.FilterToggle):
			level := logFilterDefs[m.filterCursor].level
			m.filterLevels[level] = !m.filterLevels[level]
			m.filteredDirty = true
			m.clampScroll()
		case key.Matches(msg, LogKeys.Search):
			m.searching = true
			m.searchInput.Focus()
			return m, m.searchInput.Cursor.BlinkCmd()
		default:
			switch msg.String() {
			case "up", "k":
				if m.scrollOffset > 0 {
					m.scrollOffset--
				}
			case "down", "j":
				m.scrollOffset++
				m.clampScroll()
			}
		}
	}
	return m, nil
}

func (m LogViewModel) isAtBottom() bool {
	return m.scrollOffset >= m.maxScroll()
}

func (m *LogViewModel) scrollToBottom() {
	m.scrollOffset = m.maxScroll()
}

func (m LogViewModel) viewHeight() int {
	h := m.height - 4
	if h < 1 {
		h = 10
	}
	return h
}

func (m *LogViewModel) maxScroll() int {
	total := len(m.renderedLines())
	viewH := m.viewHeight()
	ms := total - viewH
	if ms < 0 {
		ms = 0
	}
	return ms
}

func (m *LogViewModel) clampScroll() {
	ms := m.maxScroll()
	if m.scrollOffset > ms {
		m.scrollOffset = ms
	}
}

func (m *LogViewModel) filteredEntries() []claude.LogEntry {
	if !m.filteredDirty && m.filteredCache != nil {
		return m.filteredCache
	}
	var filtered []claude.LogEntry
	for _, entry := range m.entries {
		if !m.filterLevels[entry.Level] {
			continue
		}
		if m.searchQuery != "" && !strings.Contains(entry.Message, m.searchQuery) && !strings.Contains(entry.Raw, m.searchQuery) {
			continue
		}
		filtered = append(filtered, entry)
	}
	m.filteredCache = filtered
	m.filteredDirty = false
	return filtered
}

// View renders the log viewer.
func (m LogViewModel) View() string {
	if len(m.entries) == 0 {
		return EmptyStateStyle.Render("デバッグログなし")
	}

	var b strings.Builder

	// Filter bar
	b.WriteString(m.renderFilterBar())
	b.WriteString("\n\n")

	lines := m.renderedLines()
	viewH := m.viewHeight()

	start := m.scrollOffset
	if start >= len(lines) {
		start = len(lines) - 1
	}
	if start < 0 {
		start = 0
	}
	end := start + viewH
	if end > len(lines) {
		end = len(lines)
	}

	b.WriteString(strings.Join(lines[start:end], "\n"))
	b.WriteString(HelpStyle.Render(fmt.Sprintf("\n%d entries", len(m.filteredEntries()))))

	return b.String()
}
