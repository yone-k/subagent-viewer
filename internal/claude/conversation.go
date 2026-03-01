package claude

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"
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
	Prompt       string // first user message, truncated
	Description  string // from parent conversation Agent tool_use
	SubagentType string // from parent conversation Agent tool_use (e.g. "Explore", "general-task-executor")
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
	Type     string          `json:"type"`
	Text     string          `json:"text"`
	Thinking string          `json:"thinking"`
	Name     string          `json:"name"`
	Input    json.RawMessage `json:"input"`
	Content  json.RawMessage `json:"content"`
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
		info.Prompt = truncateString(firstUserPrompt, 60)
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
	Description  string `json:"description"`
	Prompt       string `json:"prompt"`
	SubagentType string `json:"subagent_type"`
}

// AgentDescription holds description and subagent type extracted from parent conversation.
type AgentDescription struct {
	Description  string
	SubagentType string
}

// ExtractAgentDescriptions parses a parent conversation JSONL file and extracts
// Agent tool_use descriptions. Returns a map from the full prompt string to AgentDescription.
func ExtractAgentDescriptions(parentPath string) (map[string]AgentDescription, error) {
	f, err := os.Open(parentPath)
	if err != nil {
		return nil, fmt.Errorf("opening parent conversation: %w", err)
	}
	defer f.Close()

	descriptions := make(map[string]AgentDescription)

	scanner := bufio.NewScanner(f)
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

		if rl.Type != "assistant" {
			continue
		}

		var msg rawMessage
		if err := json.Unmarshal(rl.Message, &msg); err != nil {
			continue
		}

		// Parse content blocks to find Agent tool_use
		var rawBlocks []rawContentBlock
		if err := json.Unmarshal(msg.Content, &rawBlocks); err != nil {
			continue
		}

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
			key := input.Prompt
			descriptions[key] = AgentDescription{
				Description:  input.Description,
				SubagentType: input.SubagentType,
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading parent conversation: %w", err)
	}

	return descriptions, nil
}

// EnrichSubagentsWithDescriptions matches subagents with descriptions extracted
// from the parent conversation. Matching is done by comparing the agent's Prompt
// (truncated to 60 chars) as a prefix of the full prompt keys. When multiple keys
// share the same prefix, the shortest (most specific) key is chosen; ties in length
// are broken by lexicographic order (smallest key wins) for determinism.
func EnrichSubagentsWithDescriptions(agents []SubagentInfo, descriptions map[string]AgentDescription) {
	for i := range agents {
		prompt := agents[i].Prompt
		if prompt == "" {
			continue
		}
		promptRunes := []rune(prompt)
		bestKey := ""
		for key := range descriptions {
			keyRunes := []rune(key)
			if len(keyRunes) >= len(promptRunes) && string(keyRunes[:len(promptRunes)]) == prompt {
				keyLen := len([]rune(key))
				bestLen := len([]rune(bestKey))
				if bestKey == "" || keyLen < bestLen || (keyLen == bestLen && key < bestKey) {
					bestKey = key
				}
			}
		}
		if bestKey != "" {
			agents[i].Description = descriptions[bestKey].Description
			agents[i].SubagentType = descriptions[bestKey].SubagentType
		}
	}
}

// DiscoverSubagents scans a directory for agent-*.jsonl files and returns info about each.
// Results are sorted by file modification time descending (newest first).
func DiscoverSubagents(subagentsDir string) ([]SubagentInfo, error) {
	matches, err := filepath.Glob(filepath.Join(subagentsDir, "agent-*.jsonl"))
	if err != nil {
		return nil, fmt.Errorf("globbing subagent files: %w", err)
	}

	var agents []SubagentInfo
	for _, path := range matches {
		_, info, err := ParseConversationFile(path)
		if err != nil {
			continue // skip broken files
		}
		if info != nil {
			fi, err := os.Stat(path)
			if err == nil {
				info.CreatedAt = fi.ModTime()
			}
			agents = append(agents, *info)
		}
	}

	// Sort by CreatedAt descending (newest first)
	sort.Slice(agents, func(i, j int) bool {
		return agents[i].CreatedAt.After(agents[j].CreatedAt)
	})

	return agents, nil
}
