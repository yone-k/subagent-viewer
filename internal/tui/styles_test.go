package tui

import (
	"strings"
	"testing"
)

func TestRenderFilterBar_CursorBracket(t *testing.T) {
	items := []FilterItem{
		{Label: "A", Active: true},
		{Label: "B", Active: false},
		{Label: "C", Active: true},
	}

	// Cursor at index 1 — "B" should be bracketed
	result := RenderFilterBar(items, 1)

	if !strings.Contains(result, "[B]") {
		t.Errorf("cursor item should be bracketed with [B], got: %s", result)
	}
	// "A" and "C" should NOT be bracketed
	if strings.Contains(result, "[A]") {
		t.Error("non-cursor item A should not be bracketed")
	}
	if strings.Contains(result, "[C]") {
		t.Error("non-cursor item C should not be bracketed")
	}
}

func TestRenderFilterBar_ActiveInactiveStyles(t *testing.T) {
	items := []FilterItem{
		{Label: "On", Active: true},
		{Label: "Off", Active: false},
	}

	result := RenderFilterBar(items, -1) // no cursor

	// Active item should use FilterActiveStyle (contains the label text)
	activeRendered := FilterActiveStyle.Render("On")
	inactiveRendered := FilterInactiveStyle.Render("Off")

	if !strings.Contains(result, activeRendered) {
		t.Errorf("active item should use FilterActiveStyle.\nwant substring: %q\ngot: %q", activeRendered, result)
	}
	if !strings.Contains(result, inactiveRendered) {
		t.Errorf("inactive item should use FilterInactiveStyle.\nwant substring: %q\ngot: %q", inactiveRendered, result)
	}
}

func TestRenderFilterBar_Prefix(t *testing.T) {
	items := []FilterItem{
		{Label: "X", Active: true},
	}

	result := RenderFilterBar(items, 0)

	if !strings.HasPrefix(result, "Filter: ") {
		t.Errorf("result should start with 'Filter: ', got: %q", result)
	}
}

func TestRenderFilterBar_EmptySlice(t *testing.T) {
	result := RenderFilterBar(nil, 0)

	if result != "Filter: " {
		t.Errorf("empty items should return 'Filter: ', got: %q", result)
	}

	result2 := RenderFilterBar([]FilterItem{}, 0)
	if result2 != "Filter: " {
		t.Errorf("empty slice should return 'Filter: ', got: %q", result2)
	}
}

func TestRenderFilterBar_CursorAtFirstAndLast(t *testing.T) {
	items := []FilterItem{
		{Label: "First", Active: true},
		{Label: "Middle", Active: false},
		{Label: "Last", Active: true},
	}

	// Cursor at first item
	result := RenderFilterBar(items, 0)
	if !strings.Contains(result, "[First]") {
		t.Errorf("first item should be bracketed, got: %s", result)
	}

	// Cursor at last item
	result = RenderFilterBar(items, 2)
	if !strings.Contains(result, "[Last]") {
		t.Errorf("last item should be bracketed, got: %s", result)
	}
}

func TestWordWrap(t *testing.T) {
	tests := []struct {
		name  string
		input string
		width int
		want  string
	}{
		{
			name:  "simple text no wrap needed",
			input: "hello world",
			width: 80,
			want:  "hello world",
		},
		{
			name:  "wraps long line",
			input: "aaa bbb ccc",
			width: 7,
			want:  "aaa bbb\nccc",
		},
		{
			name:  "preserves leading indentation",
			input: "    indented text",
			width: 80,
			want:  "    indented text",
		},
		{
			name:  "preserves consecutive spaces",
			input: "key:  value",
			width: 80,
			want:  "key:  value",
		},
		{
			name:  "preserves multi-level JSON-like indentation",
			input: "{\n  \"key\": {\n    \"nested\": true\n  }\n}",
			width: 80,
			want:  "{\n  \"key\": {\n    \"nested\": true\n  }\n}",
		},
		{
			name:  "preserves empty lines in multiline text",
			input: "line1\n\nline3",
			width: 80,
			want:  "line1\n\nline3",
		},
		{
			name:  "width zero returns input unchanged",
			input: "hello world",
			width: 0,
			want:  "hello world",
		},
		{
			name:  "preserves 4-space indentation across lines",
			input: "    line1\n    line2\n        deep",
			width: 80,
			want:  "    line1\n    line2\n        deep",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := wordWrap(tt.input, tt.width)
			if got != tt.want {
				t.Errorf("wordWrap(%q, %d)\ngot:  %q\nwant: %q", tt.input, tt.width, got, tt.want)
			}
		})
	}
}
