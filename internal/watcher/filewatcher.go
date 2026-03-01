package watcher

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fsnotify/fsnotify"
	"github.com/yone/subagent-viewer/internal/claude"
)

const debounceInterval = 200 * time.Millisecond

// FileWatcher watches a file-history directory for changes.
type FileWatcher struct {
	dir     string
	program *tea.Program
}

// NewFileWatcher creates a new FileWatcher.
func NewFileWatcher(dir string, program *tea.Program) *FileWatcher {
	return &FileWatcher{dir: dir, program: program}
}

// Start begins watching for file history changes.
func (fw *FileWatcher) Start(ctx context.Context) {
	// Initial load
	groups, err := claude.LoadFileHistory(fw.dir)
	if err != nil {
		fw.program.Send(WatcherErrorMsg{Source: "file-history", Err: err})
		return
	}
	fw.program.Send(FileHistoryUpdatedMsg{Groups: groups})

	// Set up fsnotify watcher
	w, err := fsnotify.NewWatcher()
	if err != nil {
		fw.program.Send(WatcherErrorMsg{Source: "file-history", Err: err})
		return
	}

	if err := w.Add(fw.dir); err != nil {
		fw.program.Send(WatcherErrorMsg{Source: "file-history", Err: err})
		w.Close()
		return
	}

	go func() {
		defer w.Close()
		// debounceTimer is only accessed within the select loop goroutine;
		// AfterFunc callback accesses fw fields but does not race because
		// it only calls program.Send (thread-safe).
		var debounceTimer *time.Timer
		for {
			select {
			case <-ctx.Done():
				if debounceTimer != nil {
					debounceTimer.Stop()
				}
				return
			case event, ok := <-w.Events:
				if !ok {
					return
				}
				// File history versions are immutable (create-only, never updated)
				if event.Op&fsnotify.Create != 0 {
					if debounceTimer != nil {
						debounceTimer.Stop()
					}
					debounceTimer = time.AfterFunc(debounceInterval, func() {
						groups, err := claude.LoadFileHistory(fw.dir)
						if err != nil {
							fw.program.Send(WatcherErrorMsg{Source: "file-history", Err: err})
							return
						}
						fw.program.Send(FileHistoryUpdatedMsg{Groups: groups})
					})
				}
			case err, ok := <-w.Errors:
				if !ok {
					return
				}
				fw.program.Send(WatcherErrorMsg{Source: "file-history", Err: err})
			}
		}
	}()
}
