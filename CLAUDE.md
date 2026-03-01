# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Test Commands

```bash
go build ./...                    # Build all packages
go vet ./...                      # Static analysis
go test ./...                     # Run all tests
go test ./internal/claude/...     # Run tests for a single package
go test -run TestLoadTasks ./internal/claude/  # Run a single test
go test -count=1 ./...            # Run all tests without cache
```

## Architecture

Go TUI tool for monitoring Claude Code subagent tasks, debug logs, file changes, and session stats in real-time. Built with Bubble Tea (Elm architecture).

### Three-layer design (strict dependency direction: tui → watcher → claude)

- **`internal/claude/`** — Pure domain layer. Parses Claude Code data files (`~/.claude/`). Zero framework imports — stdlib only. All parser functions accept file paths as parameters (not hardcoded `~/.claude`) for testability with `testdata/` fixtures.

- **`internal/watcher/`** — Filesystem observation layer. Three watchers run as goroutines, dispatching `tea.Msg` via `program.Send()`:
  - `TaskWatcher`: fsnotify on `tasks/{UUID}/*.json`
  - `LogWatcher`: 500ms polling on `debug/{UUID}.txt` (polling avoids kqueue event flood on macOS)
  - `FileWatcher`: fsnotify + 200ms debounce on `file-history/{UUID}/`

- **`internal/tui/`** — Bubble Tea presentation layer. `AppModel` is root, holds sub-models by value (standard Bubble Tea pattern). Two states: `StateSelector` (session picker) → `StateViewer` (4-tab viewer: Tasks/Logs/Files/Stats).

### Key patterns

- **Two-phase watcher startup**: `startWatchersCmd()` returns a `tea.Cmd` that sends `startWatchersMsg` back to the model, ensuring `m.program` is set before goroutines call `program.Send()`.
- **Log tail**: `ReadLogTail` seeks backwards in 8KB chunks — never reads entire file (can be 10MB+).
- **Ring buffer**: `LogViewModel` caps entries at 10,000 with oldest eviction.
- **Filtered entries cache**: `filteredDirty` flag avoids recomputing filters on every render cycle.
- **Context-based lifecycle**: `context.WithCancel` stops all watcher goroutines on quit, preventing leaks.

### Data sources (all under `~/.claude/`)

| Path | Format | Notes |
|------|--------|-------|
| `tasks/{UUID}/*.json` | Individual JSON files | `.lock` = session active, `.highwatermark` = max task ID |
| `debug/{UUID}.txt` | Timestamped lines: `ISO8601 [LEVEL] msg` | 7 levels: DEBUG/ERROR/WARN/MCP/STARTUP/META/ATTACHMENT. Continuation lines (no timestamp) merge into previous entry |
| `file-history/{UUID}/` | `{16-hex-hash}@v{N}` | Same hash = same source file, versions are immutable (write-once) |
| `history.jsonl` | One JSON object per line | Some old entries lack `sessionId` — filter them out |
| `.claude.json` | Global config JSON | `projects[path].last*` stats only for latest session per project |

## Conventions

- UI strings are in Japanese.
- TDD: tests use `testdata/` fixtures and `t.TempDir()` for isolation.
- Watcher tests use `helpers_test.go` shared infrastructure (`msgCollector`, `newTestProgram`).
