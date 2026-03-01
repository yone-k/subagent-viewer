package tui

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/yone/subagent-viewer/internal/claude"
)

// ConversationPane represents which pane is focused in the split view.
type ConversationPane int

const (
	PaneEntryList ConversationPane = iota
	PaneDetail
)

// ConversationViewModel manages the split-pane conversation view.
type ConversationViewModel struct {
	entries       []claude.ConversationEntry
	info          *claude.SubagentInfo
	agentID       string
	entrySelected int
	entryScroll   int
	detailScroll int
	focusPane    ConversationPane
	width, height int
}

// NewConversationViewModel creates a new ConversationViewModel.
func NewConversationViewModel() ConversationViewModel {
	return ConversationViewModel{}
}

// SetSize updates the view dimensions.
func (m *ConversationViewModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// SetData sets the conversation data and resets all scroll state.
func (m *ConversationViewModel) SetData(agentID string, entries []claude.ConversationEntry, info *claude.SubagentInfo) {
	m.agentID = agentID
	m.entries = entries
	m.info = info
	m.entrySelected = 0
	m.entryScroll = 0
	m.detailScroll = 0
}

// UpdateEntries updates entries and info, preserving scroll state where possible.
func (m *ConversationViewModel) UpdateEntries(entries []claude.ConversationEntry, info *claude.SubagentInfo) {
	m.entries = entries
	m.info = info
	if len(entries) > 0 && m.entrySelected >= len(entries) {
		m.entrySelected = len(entries) - 1
	}
	// detailScroll is preserved
}

// Update handles key messages. Returns the updated model and whether the key was handled.
// If handled is false (on Esc), the caller should navigate back.
func (m ConversationViewModel) Update(msg tea.KeyMsg) (ConversationViewModel, bool) {
	switch {
	case key.Matches(msg, ConversationKeys.SwitchPane):
		if m.focusPane == PaneEntryList {
			m.focusPane = PaneDetail
		} else {
			m.focusPane = PaneEntryList
		}
		return m, true

	case key.Matches(msg, ConversationKeys.Escape):
		return m, false
	}

	switch m.focusPane {
	case PaneEntryList:
		switch msg.String() {
		case "up", "k":
			if m.entrySelected > 0 {
				m.entrySelected--
				m.detailScroll = 0
				m.ensureEntryVisible()
			}
		case "down", "j":
			if m.entrySelected < len(m.entries)-1 {
				m.entrySelected++
				m.detailScroll = 0
				m.ensureEntryVisible()
			}
		}

	case PaneDetail:
		switch msg.String() {
		case "up", "k":
			if m.detailScroll > 0 {
				m.detailScroll--
			}
		case "down", "j":
			maxScroll := m.currentDetailLineCount() - m.detailVisibleHeight()
			if maxScroll < 0 {
				maxScroll = 0
			}
			if m.detailScroll < maxScroll {
				m.detailScroll++
			}
		}
	}

	return m, true
}

// View renders the split-pane conversation view.
func (m ConversationViewModel) View() string {
	header := m.renderHeader()

	leftWidth := m.width * 35 / 100
	if leftWidth < 20 {
		leftWidth = 20
	}
	rightWidth := m.width - leftWidth - 4 // account for borders

	// Render pane contents
	entryListContent := m.renderEntryList()
	detailContent := m.renderDetail()

	// Apply border styles based on focus
	var leftStyle, rightStyle lipgloss.Style
	if m.focusPane == PaneEntryList {
		leftStyle = PaneFocusedBorder.Width(leftWidth)
		rightStyle = PaneUnfocusedBorder.Width(rightWidth)
	} else {
		leftStyle = PaneUnfocusedBorder.Width(leftWidth)
		rightStyle = PaneFocusedBorder.Width(rightWidth)
	}

	leftPane := leftStyle.Render(entryListContent)
	rightPane := rightStyle.Render(detailContent)

	return header + "\n" + lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)
}

// renderHeader renders the header line with agent name and entry count.
func (m ConversationViewModel) renderHeader() string {
	name := m.agentID
	if m.info != nil {
		if m.info.Slug != "" {
			name = m.info.Slug
		} else if m.info.AgentID != "" {
			name = m.info.AgentID
		}
	}
	header := TitleStyle.Render(fmt.Sprintf("会話: %s", name))
	header += HelpStyle.Render(fmt.Sprintf("  %d エントリ", len(m.entries)))
	return header
}

// renderEntryList renders the left pane entry list.
func (m ConversationViewModel) renderEntryList() string {
	if len(m.entries) == 0 {
		return EmptyStateStyle.Render("エントリなし")
	}

	visibleLines := m.entryListVisibleLines()
	leftContentWidth := m.leftContentWidth()

	var b strings.Builder
	for i := m.entryScroll; i < len(m.entries) && i < m.entryScroll+visibleLines; i++ {
		entry := m.entries[i]
		prefix := "  "
		if i == m.entrySelected {
			prefix = "> "
		}

		var tag string
		if entry.Type == claude.EntryTypeUser {
			tag = ConversationUserStyle.Render("[U]")
		} else {
			tag = ConversationAssistantStyle.Render("[A]")
		}

		summary := m.entrySummary(entry, leftContentWidth-8)
		b.WriteString(fmt.Sprintf("%s%s %s\n", prefix, tag, summary))
	}

	return b.String()
}

// currentDetailLineCount returns the number of rendered lines for the currently selected entry.
func (m ConversationViewModel) currentDetailLineCount() int {
	if len(m.entries) == 0 || m.entrySelected >= len(m.entries) {
		return 0
	}
	entry := m.entries[m.entrySelected]
	rightContentWidth := m.rightContentWidth()

	var b strings.Builder
	for _, block := range entry.Content {
		switch block.Type {
		case "text":
			b.WriteString(wordWrap(block.Text, rightContentWidth))
			b.WriteString("\n")
		case "tool_use":
			b.WriteString("[TOOL] " + block.ToolName + "\n")
			b.WriteString(formatJSON(block.ToolInput, rightContentWidth))
			b.WriteString("\n")
		case "tool_result":
			b.WriteString(wordWrap(block.Text, rightContentWidth))
			b.WriteString("\n")
		case "thinking":
			b.WriteString("[thinking]\n")
			b.WriteString(wordWrap(block.Text, rightContentWidth))
			b.WriteString("\n")
		}
	}
	return len(strings.Split(b.String(), "\n"))
}

// renderDetail renders the right pane with full content of the selected entry.
func (m ConversationViewModel) renderDetail() string {
	if len(m.entries) == 0 || m.entrySelected >= len(m.entries) {
		return EmptyStateStyle.Render("エントリを選択してください")
	}

	entry := m.entries[m.entrySelected]
	rightContentWidth := m.rightContentWidth()

	var b strings.Builder
	for _, block := range entry.Content {
		switch block.Type {
		case "text":
			b.WriteString(wordWrap(block.Text, rightContentWidth))
			b.WriteString("\n")
		case "tool_use":
			b.WriteString(DetailToolNameStyle.Render("[TOOL] "+block.ToolName) + "\n")
			b.WriteString(formatJSON(block.ToolInput, rightContentWidth))
			b.WriteString("\n")
		case "tool_result":
			b.WriteString(wordWrap(block.Text, rightContentWidth))
			b.WriteString("\n")
		case "thinking":
			b.WriteString(ConversationThinkingStyle.Render("[thinking]") + "\n")
			b.WriteString(wordWrap(block.Text, rightContentWidth))
			b.WriteString("\n")
		}
	}

	rendered := b.String()

	lines := strings.Split(rendered, "\n")

	// Apply scroll
	visibleHeight := m.detailVisibleHeight()
	if m.detailScroll >= len(lines) {
		m.detailScroll = len(lines) - 1
	}
	if m.detailScroll < 0 {
		m.detailScroll = 0
	}

	end := m.detailScroll + visibleHeight
	if end > len(lines) {
		end = len(lines)
	}

	visible := lines[m.detailScroll:end]
	return strings.Join(visible, "\n")
}

// entrySummary returns a one-line summary of an entry.
func (m ConversationViewModel) entrySummary(entry claude.ConversationEntry, maxWidth int) string {
	if maxWidth <= 0 {
		maxWidth = 40
	}

	for _, block := range entry.Content {
		switch block.Type {
		case "text":
			text := strings.ReplaceAll(block.Text, "\n", " ")
			runes := []rune(text)
			if len(runes) > maxWidth {
				return string(runes[:maxWidth]) + "..."
			}
			return text
		case "tool_use":
			return ConversationToolStyle.Render("[TOOL] " + block.ToolName)
		case "thinking":
			return ConversationThinkingStyle.Render("[thinking] ...")
		case "tool_result":
			text := strings.ReplaceAll(block.Text, "\n", " ")
			runes := []rune(text)
			if len(runes) > maxWidth {
				return string(runes[:maxWidth]) + "..."
			}
			return text
		}
	}
	return ""
}

// ensureEntryVisible adjusts entryScroll so that entrySelected is visible.
func (m *ConversationViewModel) ensureEntryVisible() {
	visibleLines := m.entryListVisibleLines()
	if visibleLines <= 0 {
		return
	}
	if m.entrySelected < m.entryScroll {
		m.entryScroll = m.entrySelected
	}
	if m.entrySelected >= m.entryScroll+visibleLines {
		m.entryScroll = m.entrySelected - visibleLines + 1
	}
}

// entryListVisibleLines returns the number of visible lines in the entry list pane.
func (m ConversationViewModel) entryListVisibleLines() int {
	lines := m.height - 4 // header and borders
	if lines < 1 {
		lines = 10
	}
	return lines
}

// detailVisibleHeight returns the number of visible lines in the detail pane.
func (m ConversationViewModel) detailVisibleHeight() int {
	lines := m.height - 4
	if lines < 1 {
		lines = 10
	}
	return lines
}

// leftContentWidth returns the usable width inside the left pane.
func (m ConversationViewModel) leftContentWidth() int {
	w := m.width * 35 / 100
	if w < 20 {
		w = 20
	}
	return w
}

// rightContentWidth returns the usable width inside the right pane.
func (m ConversationViewModel) rightContentWidth() int {
	leftWidth := m.width * 35 / 100
	if leftWidth < 20 {
		leftWidth = 20
	}
	w := m.width - leftWidth - 4
	if w < 10 {
		w = 10
	}
	return w
}

// wordWrap wraps text at word boundaries to fit within the given width.
func wordWrap(text string, width int) string {
	if width <= 0 {
		return text
	}

	var result strings.Builder
	for _, line := range strings.Split(text, "\n") {
		if result.Len() > 0 {
			result.WriteString("\n")
		}
		words := strings.Fields(line)
		if len(words) == 0 {
			continue
		}

		currentLineLen := 0
		for i, word := range words {
			wordLen := len([]rune(word))
			if i == 0 {
				result.WriteString(word)
				currentLineLen = wordLen
			} else if currentLineLen+1+wordLen > width {
				result.WriteString("\n")
				result.WriteString(word)
				currentLineLen = wordLen
			} else {
				result.WriteString(" ")
				result.WriteString(word)
				currentLineLen += 1 + wordLen
			}
		}
	}
	return result.String()
}

// formatJSON attempts to pretty-print a JSON string. Falls back to raw text on parse failure.
func formatJSON(input string, width int) string {
	var parsed interface{}
	if err := json.Unmarshal([]byte(input), &parsed); err != nil {
		return wordWrap(input, width)
	}
	formatted, err := json.MarshalIndent(parsed, "", "  ")
	if err != nil {
		return wordWrap(input, width)
	}
	return wordWrap(string(formatted), width)
}
