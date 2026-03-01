package claude

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// SubagentStatus represents whether a subagent is still running or has completed.
type SubagentStatus string

const (
	SubagentRunning SubagentStatus = "running"
	SubagentClosed  SubagentStatus = "closed"
)

// ConversationEntryType represents the type of a conversation entry.
type ConversationEntryType string

const (
	EntryTypeUser      ConversationEntryType = "user"
	EntryTypeAssistant ConversationEntryType = "assistant"
)

// ContentBlock represents a single block within a conversation message.
type ContentBlock struct {
	Type      string // "text", "tool_use", "tool_result", "thinking"
	Text      string
	ToolName  string
	ToolInput string
}

// ConversationEntry represents a single parsed conversation entry.
type ConversationEntry struct {
	Type    ConversationEntryType
	Content []ContentBlock
}

// SubagentInfo holds metadata about a discovered subagent.
type SubagentInfo struct {
	AgentID      string
	Slug         string
	Prompt       string         // first user message, truncated
	Description  string         // from parent conversation Agent tool_use
	SubagentType string         // from parent conversation Agent tool_use (e.g. "Explore", "general-task-executor")
	Status       SubagentStatus // running or closed, determined from parent conversation
	EntryCount   int
	FilePath     string
	CreatedAt    time.Time // file modification time, used for sorting
}

// rawLine represents the top-level JSON structure of a conversation JSONL line.
type rawLine struct {
	Type    string          `json:"type"`
	Message json.RawMessage `json:"message"`
	AgentID string          `json:"agentId"`
	Slug    string          `json:"slug"`
}

// rawMessage represents the message field within a conversation line.
type rawMessage struct {
	Content json.RawMessage `json:"content"`
}

// rawContentBlock represents a single content block within a message content array.
type rawContentBlock struct {
	Type      string          `json:"type"`
	ID        string          `json:"id"`
	Text      string          `json:"text"`
	Thinking  string          `json:"thinking"`
	Name      string          `json:"name"`
	Input     json.RawMessage `json:"input"`
	Content   json.RawMessage `json:"content"`
	ToolUseID string          `json:"tool_use_id"`
}

// ParseConversationFile parses a JSONL conversation file and returns entries and subagent info.
func ParseConversationFile(path string) ([]ConversationEntry, *SubagentInfo, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("opening conversation file: %w", err)
	}
	defer f.Close()

	entries, info, err := ParseConversationEntries(f)
	if err != nil {
		return nil, nil, err
	}

	// Set FilePath on info (only available when reading from a file)
	if info != nil {
		info.FilePath = path
	}

	return entries, info, nil
}

// ParseConversationEntries parses conversation entries from a reader.
// This is the core parsing logic shared by ParseConversationFile and incremental readers.
func ParseConversationEntries(r io.Reader) ([]ConversationEntry, *SubagentInfo, error) {
	var entries []ConversationEntry
	var info *SubagentInfo
	firstUserPrompt := ""
	firstLineParsed := false

	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var rl rawLine
		if err := json.Unmarshal([]byte(line), &rl); err != nil {
			continue
		}

		// Skip non-user/assistant types (e.g. "progress")
		if rl.Type != "user" && rl.Type != "assistant" {
			continue
		}

		// Extract agentId and slug from first valid line
		if !firstLineParsed {
			info = &SubagentInfo{
				AgentID: rl.AgentID,
				Slug:    rl.Slug,
			}
			firstLineParsed = true
		}

		// Parse the message
		var msg rawMessage
		if err := json.Unmarshal(rl.Message, &msg); err != nil {
			continue
		}

		// Parse content blocks (polymorphic: string or array)
		blocks := ParseContentBlocks(msg.Content)

		// Capture first user prompt
		if firstUserPrompt == "" && rl.Type == "user" && len(blocks) > 0 {
			for _, b := range blocks {
				if b.Text != "" {
					firstUserPrompt = b.Text
					break
				}
			}
		}

		entry := ConversationEntry{
			Type:    ConversationEntryType(rl.Type),
			Content: blocks,
		}
		entries = append(entries, entry)
	}
	if err := scanner.Err(); err != nil {
		return nil, nil, err
	}

	// Finalize SubagentInfo
	if info != nil {
		info.EntryCount = len(entries)
		info.Prompt = truncateString(firstUserPrompt, 120)
	}

	return entries, info, nil
}

// ParseContentBlocks parses polymorphic content: either a JSON string or an array of content blocks.
func ParseContentBlocks(raw json.RawMessage) []ContentBlock {
	// Try string first
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return []ContentBlock{{Type: "text", Text: s}}
	}

	// Try array of content blocks
	var rawBlocks []rawContentBlock
	if err := json.Unmarshal(raw, &rawBlocks); err != nil {
		return nil
	}

	var blocks []ContentBlock
	for _, rb := range rawBlocks {
		switch rb.Type {
		case "text":
			blocks = append(blocks, ContentBlock{Type: "text", Text: rb.Text})
		case "tool_use":
			toolInput := ""
			if rb.Input != nil {
				inputBytes, err := json.Marshal(json.RawMessage(rb.Input))
				if err == nil {
					toolInput = string(inputBytes)
				}
			}
			blocks = append(blocks, ContentBlock{
				Type:      "tool_use",
				ToolName:  rb.Name,
				ToolInput: toolInput,
			})
		case "tool_result":
			text := parseToolResultContent(rb.Content)
			blocks = append(blocks, ContentBlock{Type: "tool_result", Text: text})
		case "thinking":
			blocks = append(blocks, ContentBlock{Type: "thinking", Text: rb.Thinking})
		}
	}
	return blocks
}

// parseToolResultContent converts tool_result content to a string.
// Content can be a plain string or an array of objects.
func parseToolResultContent(raw json.RawMessage) string {
	if raw == nil {
		return ""
	}
	// Try string
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	// Fallback: stringify the raw JSON
	return string(raw)
}

// truncateString truncates a string to maxLen characters.
func truncateString(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen])
}

// agentToolInput represents the parsed input of an Agent tool_use block.
type agentToolInput struct {
	Description     string `json:"description"`
	Prompt          string `json:"prompt"`
	SubagentType    string `json:"subagent_type"`
	RunInBackground bool   `json:"run_in_background"`
}

// AgentDescription holds description and subagent type extracted from parent conversation.
type AgentDescription struct {
	Prompt       string // full prompt from Agent tool_use input
	Description  string
	SubagentType string
	ToolUseID    string
	Status       SubagentStatus
}

// isAsyncLaunchResult checks if a tool_result content indicates an async agent
// launch confirmation rather than actual completion. This serves as a fallback
// when the run_in_background flag is not available.
func isAsyncLaunchResult(content json.RawMessage) bool {
	if len(content) == 0 {
		return false
	}
	return strings.Contains(string(content), "Async agent launched")
}

// extractCompletedToolUseIDs scans raw content for task-notification tags
// indicating background agent completion. Returns tool_use IDs that have
// completed based on <tool-use-id> and <status>completed</status> pairs.
func extractCompletedToolUseIDs(content string) []string {
	var ids []string
	remaining := content
	for {
		// Find next <tool-use-id>...</tool-use-id>
		start := strings.Index(remaining, "<tool-use-id>")
		if start == -1 {
			break
		}
		start += len("<tool-use-id>")
		end := strings.Index(remaining[start:], "</tool-use-id>")
		if end == -1 {
			break
		}
		id := remaining[start : start+end]
		// Check if <status>completed</status> follows within this notification
		afterTag := remaining[start+end:]
		if strings.Contains(afterTag, "<status>completed</status>") {
			ids = append(ids, id)
		}
		remaining = remaining[start+end:]
	}
	return ids
}

// ExtractAgentDescriptions parses a parent conversation JSONL file and extracts
// Agent tool_use descriptions. Returns a slice of AgentDescription in the order they appear,
// with Status set to SubagentRunning or SubagentClosed based on whether a corresponding
// tool_result entry exists. Background agents (run_in_background=true) are always
// treated as running since their immediate tool_result is just a launch confirmation.
func ExtractAgentDescriptions(parentPath string) ([]AgentDescription, error) {
	result, err := ExtractAgentDescriptionsIncremental(parentPath, 0, nil)
	if err != nil {
		return nil, err
	}
	return result.Cache.Descriptions, nil
}

// AgentDescriptionCache holds accumulated state for incremental parsing.
// Callers should only read Descriptions; the other fields are internal bookkeeping.
type AgentDescriptionCache struct {
	Descriptions  []AgentDescription
	completedIDs  map[string]bool
	backgroundIDs map[string]bool
}

// IncrementalParseResult holds the return values from ExtractAgentDescriptionsIncremental.
type IncrementalParseResult struct {
	Cache   *AgentDescriptionCache
	Offset  int64
	ModTime time.Time // file modification time at the time of parsing
}

// ExtractAgentDescriptionsIncremental extracts agent descriptions incrementally
// starting from the given offset. Returns updated cache, new offset, file mod time, and error.
func ExtractAgentDescriptionsIncremental(parentPath string, offset int64, cache *AgentDescriptionCache) (IncrementalParseResult, error) {
	f, err := os.Open(parentPath)
	if err != nil {
		return IncrementalParseResult{Cache: cache, Offset: offset}, fmt.Errorf("opening parent conversation: %w", err)
	}
	defer f.Close()

	// Get file info (size and mtime in a single stat)
	fi, err := f.Stat()
	if err != nil {
		return IncrementalParseResult{Cache: cache, Offset: offset}, fmt.Errorf("stat parent conversation: %w", err)
	}
	fileSize := fi.Size()
	fileMtime := fi.ModTime()

	// Truncate detection: if file shrank, reset and re-parse from beginning
	if fileSize < offset {
		cache = nil
		offset = 0
	}

	// Seek to offset if resuming
	if offset > 0 {
		if _, err := f.Seek(offset, 0); err != nil {
			return IncrementalParseResult{Cache: cache, Offset: offset, ModTime: fileMtime}, fmt.Errorf("seeking parent conversation: %w", err)
		}
	}

	// Initialize cache if nil
	if cache == nil {
		cache = &AgentDescriptionCache{
			completedIDs:  make(map[string]bool),
			backgroundIDs: make(map[string]bool),
		}
	}

	// Scan lines from current position
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	var parsedBytes int64
	for scanner.Scan() {
		lineBytes := scanner.Bytes()
		line := string(lineBytes)
		if line == "" {
			// Empty lines still consume bytes (just the newline)
			parsedBytes += int64(len(lineBytes)) + 1
			continue
		}

		var rl rawLine
		if err := json.Unmarshal([]byte(line), &rl); err != nil {
			// JSON parse failed — do NOT count these bytes
			// so next call re-reads from this line's start
			continue
		}

		// Successfully parsed line — count its bytes
		parsedBytes += int64(len(lineBytes)) + 1

		var msg rawMessage
		if err := json.Unmarshal(rl.Message, &msg); err != nil {
			continue
		}

		var rawBlocks []rawContentBlock
		if err := json.Unmarshal(msg.Content, &rawBlocks); err != nil {
			continue
		}

		// Check all messages for task-notification completion signals.
		// Background agents emit <task-notification> with <tool-use-id> and
		// <status>completed</status> when they finish.
		for _, completedID := range extractCompletedToolUseIDs(string(msg.Content)) {
			cache.completedIDs[completedID] = true
		}

		switch rl.Type {
		case "assistant":
			for _, rb := range rawBlocks {
				if rb.Type != "tool_use" || rb.Name != "Agent" {
					continue
				}
				if rb.Input == nil {
					continue
				}
				var input agentToolInput
				if err := json.Unmarshal(rb.Input, &input); err != nil {
					continue
				}
				if input.Prompt == "" || input.Description == "" {
					continue
				}
				if input.RunInBackground {
					cache.backgroundIDs[rb.ID] = true
				}
				cache.Descriptions = append(cache.Descriptions, AgentDescription{
					Prompt:       input.Prompt,
					Description:  input.Description,
					SubagentType: input.SubagentType,
					ToolUseID:    rb.ID,
				})
			}
		case "user":
			for _, rb := range rawBlocks {
				if rb.Type != "tool_result" || rb.ToolUseID == "" {
					continue
				}
				if cache.backgroundIDs[rb.ToolUseID] {
					continue
				}
				if isAsyncLaunchResult(rb.Content) {
					continue
				}
				cache.completedIDs[rb.ToolUseID] = true
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return IncrementalParseResult{Cache: cache, Offset: offset, ModTime: fileMtime}, fmt.Errorf("reading parent conversation: %w", err)
	}

	// Clamp offset to file size: Scanner strips the trailing newline from each
	// token, so our "+1" per line may overshoot by 1 byte when the last scanned
	// line has no trailing newline (e.g. partial write in progress).
	newOffset := offset + parsedBytes
	if newOffset > fileSize {
		newOffset = fileSize
	}
	if parsedBytes == 0 {
		return IncrementalParseResult{Cache: cache, Offset: newOffset, ModTime: fileMtime}, nil
	}

	// Re-determine status for ALL descriptions based on current completedIDs
	for i := range cache.Descriptions {
		if cache.completedIDs[cache.Descriptions[i].ToolUseID] {
			cache.Descriptions[i].Status = SubagentClosed
		} else {
			cache.Descriptions[i].Status = SubagentRunning
		}
	}

	return IncrementalParseResult{Cache: cache, Offset: newOffset, ModTime: fileMtime}, nil
}

// EnrichSubagentsWithDescriptions matches subagents with descriptions extracted
// from the parent conversation. Agents are sorted by CreatedAt ascending internally
// for deterministic matching; the caller is responsible for any display-order sorting.
// Matching strategy (each description is consumed once matched to handle duplicates):
//  1. Exact match: the agent's Prompt exactly matches the description's full prompt.
//  2. Prefix match (fallback): the agent's Prompt (truncated to 120 chars) is a prefix
//     of the description's full prompt.
func EnrichSubagentsWithDescriptions(agents []SubagentInfo, descriptions []AgentDescription) {
	// Sort agents by CreatedAt ascending for deterministic matching
	sort.Slice(agents, func(i, j int) bool {
		return agents[i].CreatedAt.Before(agents[j].CreatedAt)
	})

	// Track which descriptions have been consumed
	consumed := make([]bool, len(descriptions))

	for i := range agents {
		prompt := agents[i].Prompt
		if prompt == "" {
			continue
		}

		matchIdx := -1

		// 1. Try exact match first (agent prompt == description's full prompt)
		for j, desc := range descriptions {
			if consumed[j] {
				continue
			}
			if desc.Prompt == prompt {
				matchIdx = j
				break
			}
		}

		// 2. Fallback to prefix match (agent prompt is a prefix of description's full prompt)
		if matchIdx == -1 {
			for j, desc := range descriptions {
				if consumed[j] {
					continue
				}
				if strings.HasPrefix(desc.Prompt, prompt) {
					matchIdx = j
					break
				}
			}
		}

		// Apply matched description
		if matchIdx >= 0 {
			agents[i].Description = descriptions[matchIdx].Description
			agents[i].SubagentType = descriptions[matchIdx].SubagentType
			agents[i].Status = descriptions[matchIdx].Status
			consumed[matchIdx] = true
		}
	}
}

// SubagentFile represents a discovered subagent conversation file without parsing its content.
type SubagentFile struct {
	Path      string
	CreatedAt time.Time // file modification time (used as creation time proxy)
	Size      int64
}

// DiscoverSubagentFiles discovers subagent conversation files in the given directory
// using only glob + stat (no content parsing). Files are returned sorted by CreatedAt descending.
func DiscoverSubagentFiles(subagentsDir string) ([]SubagentFile, error) {
	matches, err := filepath.Glob(filepath.Join(subagentsDir, "agent-*.jsonl"))
	if err != nil {
		return nil, fmt.Errorf("globbing subagent files: %w", err)
	}

	var files []SubagentFile
	for _, path := range matches {
		// Skip internal compact agents created by /compact command
		base := filepath.Base(path)
		if strings.Contains(base, "compact") {
			continue
		}

		fi, err := os.Stat(path)
		if err != nil {
			continue // skip files we cannot stat
		}

		files = append(files, SubagentFile{
			Path:      path,
			CreatedAt: fi.ModTime(),
			Size:      fi.Size(),
		})
	}

	// Sort by CreatedAt descending (newest first)
	sort.Slice(files, func(i, j int) bool {
		return files[i].CreatedAt.After(files[j].CreatedAt)
	})

	return files, nil
}

// DiscoverSubagents scans a directory for agent-*.jsonl files and returns info about each.
// Results are sorted by file modification time descending (newest first).
func DiscoverSubagents(subagentsDir string) ([]SubagentInfo, error) {
	files, err := DiscoverSubagentFiles(subagentsDir)
	if err != nil {
		return nil, err
	}

	var agents []SubagentInfo
	for _, f := range files {
		_, info, err := ParseConversationFile(f.Path)
		if err != nil || info == nil {
			continue
		}
		info.CreatedAt = f.CreatedAt
		agents = append(agents, *info)
	}
	return agents, nil // already sorted by DiscoverSubagentFiles
}
