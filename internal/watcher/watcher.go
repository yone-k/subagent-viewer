package watcher

import (
	"github.com/yone-k/cc-subagent-viewer/internal/claude"
)

// TasksUpdatedMsg is sent when all tasks are loaded/reloaded.
type TasksUpdatedMsg struct {
	Tasks []claude.Task
}

// TaskChangedMsg is sent when a single task file changes.
type TaskChangedMsg struct {
	Task claude.Task
}

// LogEntriesMsg is sent when new log entries are available.
type LogEntriesMsg struct {
	Entries []claude.LogEntry
	Initial bool // true for initial tail load
}

// WatcherErrorMsg is sent when a watcher encounters an error.
type WatcherErrorMsg struct {
	Source string
	Err    error
}

// SubagentsDiscoveredMsg is sent when subagent files are discovered or updated.
type SubagentsDiscoveredMsg struct {
	Agents []claude.SubagentInfo
}

// ConversationUpdatedMsg is sent when a subagent's conversation is updated.
type ConversationUpdatedMsg struct {
	AgentID string
	Entries []claude.ConversationEntry
	Info    *claude.SubagentInfo
}
