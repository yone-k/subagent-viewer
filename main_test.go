package main

import (
	"testing"
)

func TestParseArgs(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantMode runMode
		wantID   string
	}{
		{"no args", []string{"cmd"}, modeSelector, ""},
		{"valid lowercase UUID", []string{"cmd", "550e8400-e29b-41d4-a716-446655440000"}, modeViewer, "550e8400-e29b-41d4-a716-446655440000"},
		{"valid uppercase UUID", []string{"cmd", "550E8400-E29B-41D4-A716-446655440000"}, modeViewer, "550e8400-e29b-41d4-a716-446655440000"},
		{"invalid session ID", []string{"cmd", "not-a-uuid"}, modeError, "not-a-uuid"},
		{"extra args ignored", []string{"cmd", "550e8400-e29b-41d4-a716-446655440000", "extra"}, modeViewer, "550e8400-e29b-41d4-a716-446655440000"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotMode, gotID := parseArgs(tt.args)
			if gotMode != tt.wantMode {
				t.Errorf("parseArgs() mode = %d, want %d", gotMode, tt.wantMode)
			}
			if gotID != tt.wantID {
				t.Errorf("parseArgs() sessionID = %q, want %q", gotID, tt.wantID)
			}
		})
	}
}
