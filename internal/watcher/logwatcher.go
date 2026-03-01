package watcher

import (
	"context"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/yone-k/cc-subagent-viewer/internal/claude"
)

const logPollInterval = 500 * time.Millisecond

// LogWatcher polls a debug log file for new entries.
// Start must be called from a single goroutine.
type LogWatcher struct {
	path    string
	program *tea.Program
	offset  int64
}

// NewLogWatcher creates a new LogWatcher.
func NewLogWatcher(path string, program *tea.Program) *LogWatcher {
	return &LogWatcher{path: path, program: program}
}

// Start begins polling the log file. It first reads the tail,
// then polls for new entries at regular intervals.
func (lw *LogWatcher) Start(ctx context.Context) {
	// Initial tail read
	entries, offset, err := claude.ReadLogTail(lw.path, 1000)
	if err == nil {
		lw.offset = offset
		if len(entries) > 0 {
			lw.program.Send(LogEntriesMsg{Entries: entries, Initial: true})
		}
	} else if !os.IsNotExist(err) {
		// File doesn't exist yet is OK - we'll keep polling.
		// Any other error is reported.
		lw.program.Send(WatcherErrorMsg{Source: "log", Err: err})
	}

	ticker := time.NewTicker(logPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			entries, newOffset, err := claude.ReadLogFrom(lw.path, lw.offset)
			if err != nil {
				if !os.IsNotExist(err) {
					lw.program.Send(WatcherErrorMsg{Source: "log", Err: err})
				}
				continue
			}
			lw.offset = newOffset
			if len(entries) > 0 {
				lw.program.Send(LogEntriesMsg{Entries: entries, Initial: false})
			}
		}
	}
}
