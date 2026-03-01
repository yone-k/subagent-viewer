package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/yone/cc-subagent-viewer/internal/claude"
	"github.com/yone/cc-subagent-viewer/internal/watcher"
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
	entries       []claude.LogEntry
	filterLevels  map[claude.LogLevel]bool
	searchQuery   string
	searchInput   textinput.Model
	searching     bool
	filterCursor  int
	scrollOffset  int
	width         int
	height        int
	filteredCache []claude.LogEntry
	filteredDirty bool
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
	m.width = width
	m.height = height
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
				filtered := m.filteredEntries()
				if m.scrollOffset > len(filtered) {
					m.scrollOffset = len(filtered)
				}
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
			filtered := m.filteredEntries()
			if m.scrollOffset > len(filtered) {
				m.scrollOffset = len(filtered)
			}
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
				filtered := m.filteredEntries()
				if m.scrollOffset >= len(filtered) {
					m.scrollOffset = len(filtered)
				}
			}
		}
	}
	return m, nil
}

func (m LogViewModel) isAtBottom() bool {
	filtered := m.filteredEntries()
	viewHeight := m.height - 4
	if viewHeight < 1 {
		viewHeight = 10
	}
	if len(filtered) <= viewHeight {
		return true
	}
	return m.scrollOffset >= len(filtered)
}

func (m *LogViewModel) scrollToBottom() {
	filtered := m.filteredEntries()
	m.scrollOffset = len(filtered)
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

	// Filtered entries
	filtered := m.filteredEntries()
	viewHeight := m.height - 4 // Reserve space for filter bar and status
	if viewHeight < 1 {
		viewHeight = 10
	}

	// Calculate visible range
	start := 0
	if len(filtered) > viewHeight {
		if m.scrollOffset > viewHeight {
			start = m.scrollOffset - viewHeight
		}
	}
	end := start + viewHeight
	if end > len(filtered) {
		end = len(filtered)
	}

	for i := start; i < end; i++ {
		entry := filtered[i]
		levelStyle := logLevelStyle(entry.Level)
		ts := entry.Timestamp.Format("15:04:05.000")
		levelStr := levelStyle.Render(fmt.Sprintf("[%s]", entry.Level))
		// Flatten multi-line messages (continuation lines) to prevent exceeding view height
		msg := strings.ReplaceAll(entry.Message, "\n", " ")
		b.WriteString(fmt.Sprintf("%s %s %s\n", DimStyle.Render(ts), levelStr, msg))
	}

	b.WriteString(HelpStyle.Render(fmt.Sprintf("\n%d entries", len(filtered))))

	return b.String()
}

func (m LogViewModel) renderFilterBar() string {
	items := make([]FilterItem, len(logFilterDefs))
	for i, f := range logFilterDefs {
		items[i] = FilterItem{
			Label:  f.label,
			Active: m.filterLevels[f.level],
		}
	}

	filterBar := RenderFilterBar(items, m.filterCursor)

	if m.searching {
		filterBar += "  Search: " + m.searchInput.View()
	} else if m.searchQuery != "" {
		filterBar += "  Search: " + m.searchQuery
	}

	return filterBar
}

func logLevelStyle(level claude.LogLevel) lipgloss.Style {
	switch level {
	case claude.LevelDEBUG:
		return LogLevelDEBUG
	case claude.LevelERROR:
		return LogLevelERROR
	case claude.LevelWARN:
		return LogLevelWARN
	case claude.LevelMCP:
		return LogLevelMCP
	case claude.LevelSTARTUP:
		return LogLevelSTARTUP
	case claude.LevelMETA:
		return LogLevelMETA
	case claude.LevelATTACHMENT:
		return LogLevelATTACHMENT
	default:
		return LogLevelDEBUG
	}
}
