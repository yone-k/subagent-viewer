package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/yone-k/cc-subagent-viewer/internal/claude"
)

func (m LogViewModel) renderLogEntry(entry claude.LogEntry) []string {
	ts := entry.Timestamp.Format("15:04:05.000")
	levelStr := fmt.Sprintf("[%s]", entry.Level)

	styledTs := DimStyle.Render(ts)
	styledLevel := logLevelStyle(entry.Level).Render(levelStr)
	prefix := styledTs + " " + styledLevel + " "
	prefixWidth := lipgloss.Width(prefix)

	wrapWidth := m.width - prefixWidth
	if wrapWidth < 10 {
		wrapWidth = 10
	}

	msg := strings.ReplaceAll(entry.Message, "\n", " ")
	wrapped := wordWrap(msg, wrapWidth)
	msgLines := strings.Split(wrapped, "\n")

	result := make([]string, len(msgLines))
	result[0] = prefix + msgLines[0]
	indent := strings.Repeat(" ", prefixWidth)
	for i := 1; i < len(msgLines); i++ {
		result[i] = indent + msgLines[i]
	}
	return result
}

func (m *LogViewModel) renderedLines() []string {
	needsRebuild := m.filteredDirty || m.renderedLinesCache == nil
	filtered := m.filteredEntries()
	if !needsRebuild {
		return m.renderedLinesCache
	}
	var lines []string
	for _, entry := range filtered {
		lines = append(lines, m.renderLogEntry(entry)...)
	}
	m.renderedLinesCache = lines
	return lines
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
