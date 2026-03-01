package claude

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"time"
)

// seekChunkSize is the byte size of each chunk when seeking backwards through a file.
const seekChunkSize = 8192

// LogLevel represents the severity level of a debug log entry.
type LogLevel string

const (
	LevelDEBUG      LogLevel = "DEBUG"
	LevelERROR      LogLevel = "ERROR"
	LevelWARN       LogLevel = "WARN"
	LevelMCP        LogLevel = "MCP"
	LevelSTARTUP    LogLevel = "STARTUP"
	LevelMETA       LogLevel = "META"
	LevelATTACHMENT LogLevel = "ATTACHMENT"
)

// LogEntry represents a single parsed log entry, potentially spanning multiple lines.
type LogEntry struct {
	Timestamp time.Time
	Level     LogLevel
	Message   string
	Raw       string
}

// logLineRe matches a log line in the format: 2026-03-01T00:39:12.103Z [LEVEL] message
var logLineRe = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}Z) \[(\w+)\] (.*)$`)

const timestampLayout = "2006-01-02T15:04:05.000Z"

// ParseLogLine parses a single log line into a LogEntry.
// Returns an error if the line does not match the expected format.
func ParseLogLine(line string) (LogEntry, error) {
	matches := logLineRe.FindStringSubmatch(line)
	if matches == nil {
		return LogEntry{}, fmt.Errorf("invalid log line format: %q", line)
	}

	ts, err := time.Parse(timestampLayout, matches[1])
	if err != nil {
		return LogEntry{}, fmt.Errorf("invalid timestamp %q: %w", matches[1], err)
	}

	return LogEntry{
		Timestamp: ts,
		Level:     LogLevel(matches[2]),
		Message:   matches[3],
		Raw:       line,
	}, nil
}

// isLogLine reports whether a line starts with a timestamp (i.e., is not a continuation line).
func isLogLine(line string) bool {
	return logLineRe.MatchString(line)
}

// parseLines parses raw lines into LogEntry slices, merging continuation lines
// into the preceding entry's Message and Raw fields.
// Lines that appear before the first valid log-start line are discarded.
func parseLines(lines []string) []LogEntry {
	var entries []LogEntry
	for _, line := range lines {
		if isLogLine(line) {
			entry, err := ParseLogLine(line)
			if err != nil {
				continue
			}
			entries = append(entries, entry)
		} else {
			// Continuation line: append to the last entry
			if len(entries) > 0 {
				last := &entries[len(entries)-1]
				last.Message += "\n" + line
				last.Raw += "\n" + line
			}
		}
	}
	return entries
}

// ReadLogTail reads the last maxLines log entries from the file at path.
// It seeks backwards through the file in seekChunkSize-byte chunks.
// Returns the parsed entries, the byte offset at the end of the file, and any error.
func ReadLogTail(path string, maxLines int) ([]LogEntry, int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, 0, err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return nil, 0, err
	}
	fileSize := info.Size()
	if fileSize == 0 {
		return nil, 0, nil
	}

	// Read the file from the end in chunks, collecting enough log-start lines.
	// Chunks are appended in reverse file order (last chunk first), then
	// reversed before joining to restore the original byte order.
	// This avoids O(n^2) copying from repeated slice prepend operations.
	var chunks [][]byte
	offset := fileSize
	logStartCount := 0

	for offset > 0 {
		chunkSize := int64(seekChunkSize)
		if chunkSize > offset {
			chunkSize = offset
		}
		offset -= chunkSize

		buf := make([]byte, chunkSize)
		_, err := f.ReadAt(buf, offset)
		if err != nil && err != io.EOF {
			return nil, 0, err
		}
		// Append in reverse file order (newest chunk at index 0 after reversal).
		chunks = append(chunks, buf)

		// Count log-start lines in the new chunk to decide whether we have enough.
		// We need more than maxLines because we want exactly maxLines entries
		// (continuation lines do not count).
		// Note: chunk boundaries may split a line, but since we collect
		// maxLines+alpha log-start lines, the final tail of N entries is unaffected.
		chunkLines := splitLines(string(buf))
		for _, l := range chunkLines {
			if isLogLine(l) {
				logStartCount++
			}
		}
		if logStartCount > maxLines {
			break
		}
		if offset == 0 {
			break
		}
	}

	// Reverse chunks to restore original file order, then join.
	for i, j := 0, len(chunks)-1; i < j; i, j = i+1, j-1 {
		chunks[i], chunks[j] = chunks[j], chunks[i]
	}
	collected := bytes.Join(chunks, nil)
	lines := splitLines(string(collected))
	entries := parseLines(lines)

	if len(entries) > maxLines {
		entries = entries[len(entries)-maxLines:]
	}

	return entries, fileSize, nil
}

// ReadLogFrom reads log entries from the file at path starting at the given byte offset.
// If the offset exceeds the current file size (e.g., after truncation), it resets to 0.
// Returns the parsed entries, the new byte offset (end of file), and any error.
func ReadLogFrom(path string, offset int64) ([]LogEntry, int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, 0, err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return nil, 0, err
	}
	fileSize := info.Size()

	// If offset exceeds file size, file was likely truncated; reset to beginning.
	if offset > fileSize {
		offset = 0
	}

	if offset == fileSize {
		// No new data.
		return nil, fileSize, nil
	}

	_, err = f.Seek(offset, io.SeekStart)
	if err != nil {
		return nil, 0, err
	}

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, 0, err
	}

	entries := parseLines(lines)
	return entries, fileSize, nil
}

// splitLines splits text into lines, removing the trailing empty line if the text
// ends with a newline.
func splitLines(text string) []string {
	lines := strings.Split(text, "\n")
	// Remove trailing empty element caused by a final newline.
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}
