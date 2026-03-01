package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/yone-k/cc-subagent-viewer/internal/claude"
)

// StatsUpdatedMsg is sent when stats data is available.
type StatsUpdatedMsg struct {
	Stats *claude.ProjectStats
}

// StatsViewModel manages the Stats tab view.
type StatsViewModel struct {
	stats  *claude.ProjectStats
	width  int
	height int
}

// NewStatsViewModel creates a new StatsViewModel.
func NewStatsViewModel() StatsViewModel {
	return StatsViewModel{}
}

// SetSize updates the view dimensions.
func (m *StatsViewModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// Init initializes the model.
func (m StatsViewModel) Init() tea.Cmd {
	return nil
}

// Update handles messages.
func (m StatsViewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case StatsUpdatedMsg:
		m.stats = msg.Stats
	}
	return m, nil
}

// View renders the stats view.
func (m StatsViewModel) View() string {
	if m.stats == nil {
		return EmptyStateStyle.Render("統計情報なし")
	}

	var b strings.Builder

	// Cost
	b.WriteString(StatsLabelStyle.Render("コスト"))
	b.WriteString(StatsValueStyle.Render(fmt.Sprintf("$%.3f", m.stats.LastCost)))
	b.WriteString("\n")

	// Duration
	b.WriteString(StatsLabelStyle.Render("所要時間"))
	b.WriteString(StatsValueStyle.Render(formatDuration(m.stats.LastDuration)))
	b.WriteString("\n")

	// Tokens
	b.WriteString(StatsLabelStyle.Render("入力トークン"))
	b.WriteString(StatsValueStyle.Render(formatTokens(m.stats.LastTotalInputTokens)))
	b.WriteString("\n")

	b.WriteString(StatsLabelStyle.Render("出力トークン"))
	b.WriteString(StatsValueStyle.Render(formatTokens(m.stats.LastTotalOutputTokens)))
	b.WriteString("\n")

	// Model breakdown
	if len(m.stats.LastModelUsage) > 0 {
		b.WriteString("\n")
		b.WriteString(StatsLabelStyle.Render("モデル別使用量"))
		b.WriteString("\n")
		for model, usage := range m.stats.LastModelUsage {
			b.WriteString(fmt.Sprintf("  %s\n", model))
			b.WriteString(fmt.Sprintf("    入力: %s  出力: %s\n",
				formatTokens(usage.InputTokens),
				formatTokens(usage.OutputTokens)))
			if usage.CacheCreationInputTokens > 0 || usage.CacheReadInputTokens > 0 {
				b.WriteString(fmt.Sprintf("    キャッシュ作成: %s  キャッシュ読取: %s\n",
					formatTokens(usage.CacheCreationInputTokens),
					formatTokens(usage.CacheReadInputTokens)))
			}
		}
	}

	return b.String()
}

func formatDuration(ms int64) string {
	totalSeconds := ms / 1000
	minutes := totalSeconds / 60
	seconds := totalSeconds % 60
	if minutes > 0 {
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	}
	return fmt.Sprintf("%ds", seconds)
}

func formatTokens(n int64) string {
	if n == 0 {
		return "0"
	}
	prefix := ""
	if n < 0 {
		prefix = "-"
		n = -n
	}
	s := fmt.Sprintf("%d", n)
	// Add comma separators
	var result []byte
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result = append(result, ',')
		}
		result = append(result, byte(c))
	}
	return prefix + string(result)
}
