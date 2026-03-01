package watcher

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/yone-k/cc-subagent-viewer/internal/claude"
)

const conversationPollInterval = 1 * time.Second

// ConversationWatcher polls subagent conversation files for changes.
type ConversationWatcher struct {
	dir          string
	program      *tea.Program
	sessionID    string
	projectPath  string
	findDirFunc  func(string) (string, error)
	offsets      map[string]int64
	entries      map[string][]claude.ConversationEntry
	infos        map[string]*claude.SubagentInfo
	parentPath   string // cached parent conversation path (empty until resolved)
	parentOffset int64
	parentCache  *claude.AgentDescriptionCache
	parentMtime  time.Time
}

// NewConversationWatcher creates a new ConversationWatcher.
// dir may be empty; in that case, findDirFunc is called each poll to discover it.
func NewConversationWatcher(dir string, sessionID string, program *tea.Program, projectPath string, findDirFunc func(string) (string, error)) *ConversationWatcher {
	return &ConversationWatcher{
		dir:         dir,
		program:     program,
		sessionID:   sessionID,
		projectPath: projectPath,
		findDirFunc: findDirFunc,
		offsets:     make(map[string]int64),
		entries:     make(map[string][]claude.ConversationEntry),
		infos:       make(map[string]*claude.SubagentInfo),
	}
}

// Start begins polling for conversation file changes.
func (cw *ConversationWatcher) Start(ctx context.Context) {
	// Try to discover dir if empty
	cw.tryDiscoverDir()

	// Initial scan
	cw.scan()

	ticker := time.NewTicker(conversationPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// If dir is still empty, try to discover it
			if cw.dir == "" {
				cw.tryDiscoverDir()
			}
			cw.poll()
		}
	}
}

func (cw *ConversationWatcher) tryDiscoverDir() {
	if cw.dir != "" || cw.findDirFunc == nil || cw.sessionID == "" {
		return
	}
	dir, err := cw.findDirFunc(cw.sessionID)
	if err == nil && dir != "" {
		cw.dir = dir
	}
}

// resolveParentPath computes and caches the parent conversation JSONL path.
// It uses projectPath if available, otherwise derives the path from
// the subagents directory structure. Returns "" if neither is available yet.
func (cw *ConversationWatcher) resolveParentPath() string {
	if cw.parentPath != "" {
		return cw.parentPath
	}
	if cw.projectPath != "" {
		cw.parentPath = claude.ParentConversationPath(cw.projectPath, cw.sessionID)
		return cw.parentPath
	}
	if cw.dir != "" {
		// dir = .../projects/{encoded}/{sessionID}/subagents
		// parent = .../projects/{encoded}/{sessionID}.jsonl
		sessionDir := filepath.Dir(cw.dir)
		cw.parentPath = sessionDir + ".jsonl"
		return cw.parentPath
	}
	return ""
}

// enrichAndSendAgents enriches agents with descriptions from the parent
// conversation, updates parentOffset, and sends a SubagentsDiscoveredMsg.
func (cw *ConversationWatcher) enrichAndSendAgents(agents []claude.SubagentInfo) {
	parentPath := cw.resolveParentPath()
	if parentPath != "" {
		result, err := claude.ExtractAgentDescriptionsIncremental(
			parentPath, cw.parentOffset, cw.parentCache,
		)
		cw.parentCache = result.Cache
		cw.parentOffset = result.Offset
		cw.parentMtime = result.ModTime
		if err == nil {
			claude.EnrichSubagentsWithDescriptions(agents, cw.parentCache.Descriptions)
			// Only default unset status to Running when enrichment succeeded;
			// if the parent file could not be read, status remains empty (unknown).
			for i := range agents {
				if agents[i].Status == "" {
					agents[i].Status = claude.SubagentRunning
				}
			}
		}
	}
	// Sort descending (newest first) for display
	sort.Slice(agents, func(i, j int) bool {
		return agents[i].CreatedAt.After(agents[j].CreatedAt)
	})
	cw.program.Send(SubagentsDiscoveredMsg{Agents: agents})
}

// buildAgentsList constructs a []SubagentInfo from the cached cw.infos map.
func (cw *ConversationWatcher) buildAgentsList() []claude.SubagentInfo {
	agents := make([]claude.SubagentInfo, 0, len(cw.infos))
	for _, info := range cw.infos {
		if info != nil {
			agents = append(agents, *info)
		}
	}
	return agents
}

// scan does the initial full scan of all agent files.
func (cw *ConversationWatcher) scan() {
	if cw.dir == "" {
		// Send empty discovery message
		cw.program.Send(SubagentsDiscoveredMsg{})
		return
	}

	files, err := claude.DiscoverSubagentFiles(cw.dir)
	if err != nil {
		cw.program.Send(SubagentsDiscoveredMsg{})
		return
	}

	// Parse each file once
	for _, f := range files {
		entries, info, err := claude.ParseConversationFile(f.Path)
		if err != nil {
			continue
		}

		cw.offsets[f.Path] = f.Size
		cw.entries[f.Path] = entries
		if info != nil {
			info.CreatedAt = f.CreatedAt // Propagate CreatedAt from file stat
			cw.infos[f.Path] = info
		}

		if len(entries) > 0 && info != nil {
			cw.program.Send(ConversationUpdatedMsg{
				AgentID: info.AgentID,
				Entries: entries,
				Info:    info,
			})
		}
	}

	// Build agents list from cache and enrich
	agents := cw.buildAgentsList()
	cw.enrichAndSendAgents(agents)
}

// poll checks for new files and new entries in existing files.
func (cw *ConversationWatcher) poll() {
	if cw.dir == "" {
		return
	}

	// Check if parent conversation file has changed
	parentChanged := false
	if parentPath := cw.resolveParentPath(); parentPath != "" {
		if info, err := os.Stat(parentPath); err == nil {
			sizeChanged := info.Size() != cw.parentOffset
			mtimeChanged := !info.ModTime().Equal(cw.parentMtime)
			if sizeChanged || mtimeChanged {
				parentChanged = true
				// Same size but different mtime means file was replaced (e.g. /compact)
				if !sizeChanged && mtimeChanged {
					cw.parentOffset = 0
					cw.parentCache = nil
				}
			}
		}
	}

	// Check for new agent files (use DiscoverSubagentFiles to consistently filter compact files)
	discoveredFiles, err := claude.DiscoverSubagentFiles(cw.dir)
	if err != nil {
		return
	}

	newFileFound := false
	for _, df := range discoveredFiles {
		if _, exists := cw.offsets[df.Path]; !exists {
			// New file found
			newFileFound = true
			entries, info, err := claude.ParseConversationFile(df.Path)
			if err != nil {
				continue
			}

			cw.offsets[df.Path] = df.Size
			cw.entries[df.Path] = entries
			if info != nil {
				info.CreatedAt = df.CreatedAt
				cw.infos[df.Path] = info
			}

			if info != nil && len(entries) > 0 {
				cw.program.Send(ConversationUpdatedMsg{
					AgentID: info.AgentID,
					Entries: entries,
					Info:    info,
				})
			}
		}
	}

	// Send updated agents list if parent changed or new files were found
	if parentChanged || newFileFound {
		agents := cw.buildAgentsList()
		cw.enrichAndSendAgents(agents)
	}

	// Check existing files for new content
	for path, prevOffset := range cw.offsets {
		fi, err := os.Stat(path)
		if err != nil {
			continue
		}
		currentSize := fi.Size()
		if currentSize <= prevOffset {
			continue
		}

		// Read new lines from offset
		newEntries, info, readErr := cw.readNewEntries(path, prevOffset)
		if readErr != nil {
			// Scanner error (e.g. token too long, I/O error):
			// do NOT advance offset to avoid permanently skipping unread data.
			continue
		}
		if len(newEntries) == 0 {
			// No valid entries but no error — safe to advance offset
			cw.offsets[path] = currentSize
			continue
		}

		cw.offsets[path] = currentSize

		// Append to accumulated entries
		cw.entries[path] = append(cw.entries[path], newEntries...)
		if info != nil {
			if existing := cw.infos[path]; existing != nil {
				existing.EntryCount = len(cw.entries[path])
			} else {
				cw.infos[path] = info
			}
		}

		agentID := ""
		agentInfo := cw.infos[path]
		if agentInfo != nil {
			agentID = agentInfo.AgentID
			agentInfo.EntryCount = len(cw.entries[path])
		}

		// Send full snapshot
		cw.program.Send(ConversationUpdatedMsg{
			AgentID: agentID,
			Entries: cw.entries[path],
			Info:    agentInfo,
		})
	}
}

// readNewEntries reads new JSONL lines from the given offset.
// Returns an error if the scanner encountered a read error (e.g. token too long),
// so the caller can avoid advancing the offset and losing unread data.
func (cw *ConversationWatcher) readNewEntries(path string, offset int64) ([]claude.ConversationEntry, *claude.SubagentInfo, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()

	if _, err := f.Seek(offset, 0); err != nil {
		return nil, nil, err
	}

	return claude.ParseConversationEntries(f)
}


