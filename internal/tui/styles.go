package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

// Color palette - ANSI 16 colors for terminal theme integration.
// Colors follow the user's terminal color scheme (Solarized, Dracula, Catppuccin, etc.).
var (
	colorPrimary = lipgloss.Color("13") // Bright Magenta
	colorSuccess = lipgloss.Color("2")  // Green
	colorWarning = lipgloss.Color("3")  // Yellow
	colorMuted = lipgloss.Color("7") // White + Faint - dim/secondary elements
	colorDanger  = lipgloss.Color("1")  // Red
	colorCyan    = lipgloss.Color("6")  // Cyan
	colorMagenta = lipgloss.Color("5")  // Magenta
	colorBlue        = lipgloss.Color("4")  // Blue
	colorBrightWhite = lipgloss.Color("15") // Bright White - high-contrast text on colored backgrounds
)

// Tab styles
var (
	ActiveTabStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(colorPrimary).
		Padding(0, 2)

	InactiveTabStyle = lipgloss.NewStyle().
		Foreground(colorMuted).
		Faint(true).
		Padding(0, 2)

	TabGapStyle = lipgloss.NewStyle().
		Padding(0, 1)
)

// Status icon styles
var (
	StatusCompleted = lipgloss.NewStyle().
		Foreground(colorSuccess).
		SetString("✓")

	StatusInProgress = lipgloss.NewStyle().
		Foreground(colorWarning).
		SetString("●")

	StatusPending = lipgloss.NewStyle().
		Foreground(colorMuted).
		Faint(true).
		SetString("○")

	StatusBlocked = lipgloss.NewStyle().
		Foreground(colorDanger).
		SetString("✗")
)

// Log level styles
var (
	LogLevelDEBUG = lipgloss.NewStyle().
		Foreground(colorMuted).
		Faint(true)

	LogLevelERROR = lipgloss.NewStyle().
		Foreground(colorDanger).
		Bold(true)

	LogLevelWARN = lipgloss.NewStyle().
		Foreground(colorWarning)

	LogLevelMCP = lipgloss.NewStyle().
		Foreground(colorCyan)

	LogLevelSTARTUP = lipgloss.NewStyle().
		Foreground(colorSuccess)

	LogLevelMETA = lipgloss.NewStyle().
		Foreground(colorMagenta)

	LogLevelATTACHMENT = lipgloss.NewStyle().
		Foreground(colorBlue)
)

// General UI styles
var (
	TitleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(colorPrimary).
		Padding(0, 1)

	BorderStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorPrimary).
		Padding(0, 1)

	HelpStyle = lipgloss.NewStyle().
		Foreground(colorMuted).
		Faint(true)

	EmptyStateStyle = lipgloss.NewStyle().
		Foreground(colorMuted).
		Faint(true).
		Italic(true).
		Padding(2, 4)

	WarningStyle = lipgloss.NewStyle().
		Foreground(colorWarning).
		Bold(true)

	ActiveSessionStyle = lipgloss.NewStyle().
		Foreground(colorSuccess).
		Bold(true)

	FilterActiveStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(colorBrightWhite).
		Background(colorBlue).
		Padding(0, 1)

	FilterInactiveStyle = lipgloss.NewStyle().
		Foreground(colorMuted).
		Faint(true).
		Padding(0, 1)

	ProgressBarFilled = lipgloss.NewStyle().
		Foreground(colorSuccess)

	ProgressBarEmpty = lipgloss.NewStyle().
		Foreground(colorMuted).
		Faint(true)

	DimStyle = lipgloss.NewStyle().
		Foreground(colorMuted).
		Faint(true)

	StatsLabelStyle = lipgloss.NewStyle().
		Foreground(colorMuted).
		Faint(true).
		Width(20)

	StatsValueStyle = lipgloss.NewStyle().
		Bold(true)

	// Selected item styles for list views
	SelectedLabelStyle = lipgloss.NewStyle().
		Foreground(colorPrimary).
		Bold(true)

	SelectedDetailStyle = lipgloss.NewStyle().
		Foreground(colorPrimary).
		Faint(true)
)

// renderListItem renders a list item with selection-aware styling.
// When selected, the label uses SelectedLabelStyle and details use SelectedDetailStyle.
// When not selected, the label is rendered as plain text and details use DimStyle.
// The prefix is "> " for selected items and "  " for unselected items.
// details are appended after the label in the appropriate style.
func renderListItem(selected bool, label string, details ...string) string {
	return renderListItemWithIcon(selected, "", label, details...)
}

// renderListItemWithIcon renders a list item with an optional icon prefix.
// The icon is placed between the cursor prefix ("> " / "  ") and the label,
// outside SelectedLabelStyle to avoid ANSI escape sequence conflicts.
func renderListItemWithIcon(selected bool, icon string, label string, details ...string) string {
	prefix := "  "
	if selected {
		prefix = "> "
	}

	var line string
	if selected {
		line = prefix + icon + SelectedLabelStyle.Render(label)
		for _, d := range details {
			line += SelectedDetailStyle.Render(d)
		}
	} else {
		line = prefix + icon + label
		for _, d := range details {
			line += DimStyle.Render(d)
		}
	}
	return line
}

// truncateText replaces newlines with spaces and truncates to maxWidth display columns, appending "..." if needed.
// Uses display width (East Asian full-width characters count as 2 columns) for accurate TUI rendering.
func truncateText(s string, maxWidth int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.Join(strings.Fields(s), " ")
	if runewidth.StringWidth(s) <= maxWidth {
		return s
	}
	return runewidth.Truncate(s, maxWidth, "...")
}

// Conversation styles
var (
	ConversationUserStyle = lipgloss.NewStyle().
		Foreground(colorSuccess).
		Bold(true)

	ConversationAssistantStyle = lipgloss.NewStyle().
		Foreground(colorPrimary)

	ConversationToolStyle = lipgloss.NewStyle().
		Foreground(colorCyan).
		Italic(true)

	ConversationThinkingStyle = lipgloss.NewStyle().
		Foreground(colorMuted).
		Faint(true).
		Italic(true)

	ConversationSeparatorStyle = lipgloss.NewStyle().
		Foreground(colorMuted).
		Faint(true)

	ConversationToolResultStyle = lipgloss.NewStyle().
		Foreground(colorCyan)
)
