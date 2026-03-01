package tui

import (
	"strings"
	"testing"

	"github.com/yone/cc-subagent-viewer/internal/claude"
)

// StatsUpdatedMsg is sent when stats are available.
// (Defined in statsview.go)

func TestStatsView_UpdateWithStats(t *testing.T) {
	m := NewStatsViewModel()
	m.SetSize(80, 24)

	stats := &claude.ProjectStats{
		LastCost:              1.178,
		LastDuration:          1212000,
		LastTotalInputTokens:  150000,
		LastTotalOutputTokens: 25000,
	}
	newModel, _ := m.Update(StatsUpdatedMsg{Stats: stats})
	m = newModel.(StatsViewModel)

	if m.stats == nil {
		t.Error("stats should not be nil after update")
	}
}

func TestStatsView_ShowsStats(t *testing.T) {
	m := NewStatsViewModel()
	m.SetSize(80, 24)

	stats := &claude.ProjectStats{
		LastCost:              1.178,
		LastDuration:          1212000,
		LastTotalInputTokens:  150000,
		LastTotalOutputTokens: 25000,
	}
	newModel, _ := m.Update(StatsUpdatedMsg{Stats: stats})
	m = newModel.(StatsViewModel)

	view := m.View()
	if strings.Contains(view, "統計情報なし") {
		t.Error("should not show empty state when stats are available")
	}
	if !strings.Contains(view, "$1.178") {
		t.Errorf("should show cost, got:\n%s", view)
	}
}

func TestStatsView_DurationFormat(t *testing.T) {
	m := NewStatsViewModel()
	m.SetSize(80, 24)

	stats := &claude.ProjectStats{
		LastDuration: 1212000, // 20m 12s in milliseconds
	}
	newModel, _ := m.Update(StatsUpdatedMsg{Stats: stats})
	m = newModel.(StatsViewModel)

	view := m.View()
	if !strings.Contains(view, "20m 12s") {
		t.Errorf("expected '20m 12s' in view, got:\n%s", view)
	}
}

func TestStatsView_CostFormat(t *testing.T) {
	m := NewStatsViewModel()
	m.SetSize(80, 24)

	stats := &claude.ProjectStats{
		LastCost: 1.178,
	}
	newModel, _ := m.Update(StatsUpdatedMsg{Stats: stats})
	m = newModel.(StatsViewModel)

	view := m.View()
	if !strings.Contains(view, "$1.178") {
		t.Errorf("expected '$1.178' in view, got:\n%s", view)
	}
}

func TestStatsView_TokenFormat(t *testing.T) {
	m := NewStatsViewModel()
	m.SetSize(80, 24)

	stats := &claude.ProjectStats{
		LastTotalInputTokens:  150000,
		LastTotalOutputTokens: 25000,
	}
	newModel, _ := m.Update(StatsUpdatedMsg{Stats: stats})
	m = newModel.(StatsViewModel)

	view := m.View()
	if !strings.Contains(view, "150,000") {
		t.Errorf("expected '150,000' in view, got:\n%s", view)
	}
	if !strings.Contains(view, "25,000") {
		t.Errorf("expected '25,000' in view, got:\n%s", view)
	}
}

func TestStatsView_ModelBreakdown(t *testing.T) {
	m := NewStatsViewModel()
	m.SetSize(80, 24)

	stats := &claude.ProjectStats{
		LastModelUsage: map[string]claude.ModelUsage{
			"claude-sonnet-4-20250514": {
				InputTokens:  120000,
				OutputTokens: 20000,
			},
		},
	}
	newModel, _ := m.Update(StatsUpdatedMsg{Stats: stats})
	m = newModel.(StatsViewModel)

	view := m.View()
	if !strings.Contains(view, "claude-sonnet-4-20250514") {
		t.Errorf("expected model name in view, got:\n%s", view)
	}
}

func TestStatsView_EmptyState(t *testing.T) {
	m := NewStatsViewModel()
	m.SetSize(80, 24)

	view := m.View()
	if !strings.Contains(view, "統計情報なし") {
		t.Errorf("empty state not shown: %s", view)
	}
}

func TestFormatTokens_EdgeCases(t *testing.T) {
	tests := []struct {
		name string
		n    int64
		want string
	}{
		{"zero", 0, "0"},
		{"one", 1, "1"},
		{"999", 999, "999"},
		{"1000", 1000, "1,000"},
		{"1234567", 1234567, "1,234,567"},
		{"negative", -1234, "-1,234"},
		{"negative_large", -1234567, "-1,234,567"},
		{"negative_small", -1, "-1"},
		{"negative_999", -999, "-999"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatTokens(tt.n)
			if got != tt.want {
				t.Errorf("formatTokens(%d) = %q, want %q", tt.n, got, tt.want)
			}
		})
	}
}
