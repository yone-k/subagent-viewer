package watcher

import (
	"context"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fsnotify/fsnotify"
	"github.com/yone/cc-subagent-viewer/internal/claude"
)

// TaskWatcher watches a tasks directory for changes.
type TaskWatcher struct {
	dir     string
	program *tea.Program
}

// NewTaskWatcher creates a new TaskWatcher.
func NewTaskWatcher(dir string, program *tea.Program) *TaskWatcher {
	return &TaskWatcher{dir: dir, program: program}
}

// Start begins watching for task changes. It first loads all existing tasks,
// then watches for file system events.
func (tw *TaskWatcher) Start(ctx context.Context) {
	// Initial load
	tasks, err := claude.LoadTasks(tw.dir)
	if err != nil {
		tw.program.Send(WatcherErrorMsg{Source: "tasks", Err: err})
		return
	}
	tw.program.Send(TasksUpdatedMsg{Tasks: tasks})

	// Set up fsnotify watcher
	w, err := fsnotify.NewWatcher()
	if err != nil {
		tw.program.Send(WatcherErrorMsg{Source: "tasks", Err: err})
		return
	}
	if err := w.Add(tw.dir); err != nil {
		tw.program.Send(WatcherErrorMsg{Source: "tasks", Err: err})
		w.Close()
		return
	}

	go func() {
		defer w.Close()
		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-w.Events:
				if !ok {
					return
				}
				// Only process .json files, ignore .lock and .highwatermark
				if !strings.HasSuffix(event.Name, ".json") {
					continue
				}
				if event.Op&(fsnotify.Create|fsnotify.Write|fsnotify.Rename) != 0 {
					task, err := claude.LoadTask(event.Name)
					if err != nil {
						continue
					}
					tw.program.Send(TaskChangedMsg{Task: task})
				}
			case err, ok := <-w.Errors:
				if !ok {
					return
				}
				tw.program.Send(WatcherErrorMsg{Source: "tasks", Err: err})
			}
		}
	}()
}
