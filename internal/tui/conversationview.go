package tui

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/yone/subagent-viewer/internal/claude"
)

// renderedBlock is a pre-rendered block for display in the conversation view.
type renderedBlock struct {
	lines []string
}

// ConversationViewModel manages the single-column conversation view with filtering.
type ConversationViewModel struct {
	entries       []claude.ConversationEntry
	info          *claude.SubagentInfo
	agentID       string
	filterTypes   map[string]bool
	filteredDirty bool
	filteredCache []renderedBlock
	scrollOffset  int
	width, height int
}

// NewConversationViewModel creates a new ConversationViewModel.
func NewConversationViewModel() ConversationViewModel {
	return ConversationViewModel{
		filterTypes: map[string]bool{
			"text":        true,
			"tool_use":    false,
			"tool_result": false,
			"thinking":    false,
		},
		filteredDirty: true,
	}
}

// SetSize updates the view dimensions.
func (m *ConversationViewModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.filteredDirty = true
}

// SetData sets the conversation data and resets all scroll state.
func (m *ConversationViewModel) SetData(agentID string, entries []claude.ConversationEntry, info *claude.SubagentInfo) {
	m.agentID = agentID
	m.entries = entries
	m.info = info
	m.scrollOffset = 0
	m.filteredDirty = true
}

// UpdateEntries updates entries and info, preserving scroll state where possible.
func (m *ConversationViewModel) UpdateEntries(entries []claude.ConversationEntry, info *claude.SubagentInfo) {
	m.entries = entries
	m.info = info
	m.filteredDirty = true
}

// Update handles key messages. Returns the updated model and whether the key was handled.
// If handled is false (on Esc), the caller should navigate back.
func (m ConversationViewModel) Update(msg tea.KeyMsg) (ConversationViewModel, bool) {
	switch {
	case key.Matches(msg, ConversationKeys.FilterText):
		m.filterTypes["text"] = !m.filterTypes["text"]
		m.filteredDirty = true
		m.clampScroll()
		return m, true

	case key.Matches(msg, ConversationKeys.FilterToolUse):
		m.filterTypes["tool_use"] = !m.filterTypes["tool_use"]
		m.filteredDirty = true
		m.clampScroll()
		return m, true

	case key.Matches(msg, ConversationKeys.FilterToolResult):
		m.filterTypes["tool_result"] = !m.filterTypes["tool_result"]
		m.filteredDirty = true
		m.clampScroll()
		return m, true

	case key.Matches(msg, ConversationKeys.FilterThinking):
		m.filterTypes["thinking"] = !m.filterTypes["thinking"]
		m.filteredDirty = true
		m.clampScroll()
		return m, true

	case key.Matches(msg, ConversationKeys.Escape):
		return m, false
	}

	switch msg.String() {
	case "up", "k":
		if m.scrollOffset > 0 {
			m.scrollOffset--
		}
	case "down", "j":
		m.scrollOffset++
		m.clampScroll()
	}

	return m, true
}

// clampScroll ensures scrollOffset does not exceed the total number of lines.
func (m *ConversationViewModel) clampScroll() {
	total := m.totalLines()
	viewH := m.viewHeight()
	maxScroll := total - viewH
	if maxScroll < 0 {
		maxScroll = 0
	}
	if m.scrollOffset > maxScroll {
		m.scrollOffset = maxScroll
	}
}

// totalLines returns the total number of rendered lines across all filtered blocks.
func (m *ConversationViewModel) totalLines() int {
	blocks := m.filteredBlocks()
	total := 0
	for _, b := range blocks {
		total += len(b.lines)
	}
	return total
}

// viewHeight returns the number of lines available for content display.
func (m ConversationViewModel) viewHeight() int {
	// header (1) + filter bar (1) + blank line (1) = 3 lines reserved
	h := m.height - 3
	if h < 1 {
		h = 10
	}
	return h
}

// View renders the single-column conversation view.
func (m ConversationViewModel) View() string {
	header := m.renderHeader()
	filterBar := m.renderConversationFilterBar()

	if len(m.entries) == 0 {
		return header + "\n" + filterBar + "\n" + EmptyStateStyle.Render("エントリなし")
	}

	blocks := m.filteredBlocks()

	// Collect all lines
	var allLines []string
	for _, b := range blocks {
		allLines = append(allLines, b.lines...)
	}

	if len(allLines) == 0 {
		return header + "\n" + filterBar + "\n" + EmptyStateStyle.Render("フィルタ条件に一致するエントリなし")
	}

	// Apply scroll
	viewH := m.viewHeight()
	start := m.scrollOffset
	if start >= len(allLines) {
		start = len(allLines) - 1
	}
	if start < 0 {
		start = 0
	}
	end := start + viewH
	if end > len(allLines) {
		end = len(allLines)
	}

	visible := allLines[start:end]
	return header + "\n" + filterBar + "\n" + strings.Join(visible, "\n")
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

// renderConversationFilterBar renders the filter bar for content type filtering.
func (m ConversationViewModel) renderConversationFilterBar() string {
	type filterDef struct {
		key       string
		label     string
		blockType string
	}
	filters := []filterDef{
		{"X", "teXt", "text"},
		{"U", "tool_Use", "tool_use"},
		{"R", "tool_Result", "tool_result"},
		{"H", "tHinking", "thinking"},
	}

	var parts []string
	for _, f := range filters {
		label := formatFilterLabel(f.key, f.label)
		if m.filterTypes[f.blockType] {
			parts = append(parts, FilterActiveStyle.Render(label))
		} else {
			parts = append(parts, FilterInactiveStyle.Render(label))
		}
	}

	return "Filter: " + strings.Join(parts, " ")
}

// filteredBlocks returns the filtered and rendered blocks for display.
func (m *ConversationViewModel) filteredBlocks() []renderedBlock {
	if !m.filteredDirty && m.filteredCache != nil {
		return m.filteredCache
	}

	contentWidth := m.width - 2
	if contentWidth < 10 {
		contentWidth = 10
	}

	var blocks []renderedBlock
	for _, entry := range m.entries {
		// Collect rendered blocks for this entry
		var entryBlocks []renderedBlock
		for _, block := range entry.Content {
			if !m.filterTypes[block.Type] {
				continue
			}
			lines := renderContentBlock(block, contentWidth)
			entryBlocks = append(entryBlocks, renderedBlock{
				lines: lines,
			})
		}

		// Skip entry entirely if all blocks filtered out
		if len(entryBlocks) == 0 {
			continue
		}

		// Add separator
		var tag string
		if entry.Type == claude.EntryTypeUser {
			tag = ConversationUserStyle.Render("[U]")
		} else {
			tag = ConversationAssistantStyle.Render("[A]")
		}
		sepLine := ConversationSeparatorStyle.Render(strings.Repeat("─", contentWidth/3)) + " " + tag
		blocks = append(blocks, renderedBlock{
			lines: []string{sepLine},
		})

		blocks = append(blocks, entryBlocks...)
	}

	m.filteredCache = blocks
	m.filteredDirty = false
	return blocks
}

// renderContentBlock renders a single ContentBlock into display lines.
func renderContentBlock(block claude.ContentBlock, width int) []string {
	switch block.Type {
	case "text":
		wrapped := wordWrap(block.Text, width)
		return strings.Split(wrapped, "\n")
	case "tool_use":
		header := ConversationToolStyle.Render("[TOOL] " + block.ToolName)
		body := formatJSON(block.ToolInput, width)
		lines := []string{header}
		lines = append(lines, strings.Split(body, "\n")...)
		return lines
	case "tool_result":
		header := ConversationToolResultStyle.Render("[TOOL_RESULT]")
		wrapped := wordWrap(block.Text, width)
		lines := []string{header}
		lines = append(lines, strings.Split(wrapped, "\n")...)
		return lines
	case "thinking":
		header := ConversationThinkingStyle.Render("[THINKING]")
		wrapped := wordWrap(block.Text, width)
		lines := []string{header}
		lines = append(lines, strings.Split(wrapped, "\n")...)
		return lines
	}
	return nil
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
