# Requirements: Stability & Feature Upgrades

## Overview

Moonbase has reached functional parity — the CLI, TUI, and pipeline all work. This spec addresses accumulated tech debt (P0-P3 priorities), structural issues that hurt maintainability, and new features that make moonbase significantly more polished and useful.

The work divides into three tracks:
1. **Stability** — fix bugs, eliminate global state, handle errors properly
2. **Code Health** — remove dead code, split oversized files, DRY up duplication
3. **Feature Upgrades** — glamour rendering, cobra CLI, dry-run mode, version command, structured logging

---

## User Stories

### US-1: Stable Pipeline Execution
As a developer running a mission, I want the TUI pipeline to be free of race conditions and global mutable state so that execution is predictable regardless of timing or concurrency.

### US-2: Reliable CI
As a contributor, I want CI to actually catch agent lint failures and code issues so that I get fast feedback on broken changes.

### US-3: Maintainable Codebase
As a contributor, I want files under 300 lines, no dead code, no duplicated functions, and accurate documentation so that I can navigate and modify the codebase confidently.

### US-4: Works From Any Directory
As a user who installed moonbase globally, I want the TUI and CLI to resolve agent paths correctly regardless of where I launch from.

### US-5: Beautiful Terminal Output
As a user, I want agent outputs rendered as formatted markdown in my terminal so that I can read responses without raw markdown noise.

### US-6: Professional CLI
As a user, I want proper subcommand parsing with help text, tab completion, and a version command so that moonbase feels like a polished CLI tool.

### US-7: Pipeline Preview
As a developer, I want to preview which agents would trigger for a mission without actually executing it, so that I can verify my pipeline configuration.

### US-8: Debuggable Operations
As a developer debugging pipeline issues, I want structured logs with timestamps and context so that I can trace what happened without inserting print statements.

---

## Acceptance Criteria

### AC-1: Global State Elimination (P0)

#### AC-1.1: pipelineRunning Moved to App Struct
- **WHEN** the TUI pipeline executes phases
- **THEN** the `pipelineRunning` flag is a field on the `App` struct, not a package-level `var`
- **SHALL** compile with `-race` flag without warnings during concurrent pipeline tests

#### AC-1.2: streamCh Moved to App Struct
- **WHEN** COMMS view streams responses from an AI backend
- **THEN** the `streamCh` channel is a field on the `App` struct, not a package-level `var`
- **SHALL** allow multiple App instances in tests without shared state

---

### AC-2: CI Lint Enforcement (P0)

#### AC-2.1: Lint Step Fails on Agent Issues
- **WHEN** CI runs the lint job
- **THEN** `go run ./cmd/moonbase lint` runs without `2>/dev/null` or `|| echo` fallback
- **SHALL** fail the CI job if any agent `.md` file fails parsing or validation

---

### AC-3: TUI File Splitting (P1)

#### AC-3.1: app.go Under 300 Lines
- **WHEN** the `internal/tui/` package is inspected
- **THEN** no single file exceeds 400 lines (soft limit, allowing some flexibility)
- **SHALL** split Update logic into per-view handler files

#### AC-3.2: Logical Separation
- **WHEN** looking for TUI update handlers
- **THEN** each view's key handling and state transitions live in a dedicated file
- **SHALL** maintain a single `App` struct (no fragmentation of state)

---

### AC-4: Agent Path Resolution (P1)

#### AC-4.1: TUI Uses agentsDir Logic
- **WHEN** `NewApp()` initializes the agent registry
- **THEN** it uses the same multi-path resolution as the CLI's `agentsDir()` function
- **SHALL NOT** hardcode `"./agents"` or any single relative path

#### AC-4.2: Works From Any CWD
- **WHEN** moonbase is launched from `/tmp` or any non-project directory
- **THEN** agents are found via executable-relative, home dir, or config-specified path
- **SHALL** show a helpful error if no agents directory can be located

---

### AC-5: Dead Code Removal (P1)

#### AC-5.1: Pipeline Backend Interface Removed
- **WHEN** `internal/pipeline/pipeline.go` is inspected
- **THEN** the unused `Backend` interface (lines 53-55) does not exist
- **SHALL** not break any existing code (confirmed by `go build ./...`)

#### AC-5.2: Duplicate envExists Consolidated
- **WHEN** searching for `envExists` across the codebase
- **THEN** exactly one definition exists (in a shared location or inlined as `os.Getenv`)
- **SHALL** not duplicate logic between `cmd/moonbase/main.go` and `internal/backend/backends.go`

---

### AC-6: Error Handling (P2)

#### AC-6.1: writeTemplate Returns Errors
- **WHEN** `moonbase init` writes template files
- **THEN** `writeTemplate()` propagates `os.WriteFile` errors to the caller
- **SHALL** display which file failed and why (permission denied, disk full, etc.)

#### AC-6.2: Snippet Name Validated
- **WHEN** `moonbase snippet save <name>` is called
- **THEN** the name is validated for length (≤100 chars) and safe characters
- **SHALL** reject names with path separators or control characters

---

### AC-7: Dynamic Agent List (P2)

#### AC-7.1: runList Reads from Registry
- **WHEN** `moonbase list` is executed
- **THEN** agent data comes from parsing the actual agent `.md` files via the registry
- **SHALL** always reflect the current state of the `agents/` directory

---

### AC-8: Documentation Accuracy (P2)

#### AC-8.1: docs/design.md Matches Reality
- **WHEN** `docs/design.md` describes the file structure
- **THEN** every path referenced actually exists in the codebase
- **SHALL** reflect the flat `internal/tui/` structure (not the old views/components plan)

---

### AC-9: Glamour Markdown Rendering (Feature)

#### AC-9.1: COMMS View Renders Markdown
- **WHEN** agent output is displayed in the COMMS view
- **THEN** it is rendered through `glamour` with terminal-appropriate styling
- **SHALL** handle code blocks, headers, lists, bold/italic, and links
- **SHALL** fall back to raw text if glamour rendering fails

#### AC-9.2: Pipeline Chat Renders Markdown
- **WHEN** phase results are displayed in the pipeline chat view
- **THEN** markdown formatting is applied to agent output sections
- **SHALL NOT** render system messages (✅, ❌, ⏭️ lines) through glamour

---

### AC-10: Cobra CLI Migration (Feature)

#### AC-10.1: Subcommand Parsing via Cobra
- **WHEN** any moonbase command is executed
- **THEN** parsing is handled by cobra with proper flag/arg validation
- **SHALL** provide `--help` on every subcommand automatically

#### AC-10.2: Shell Completion
- **WHEN** `moonbase completion bash/zsh/fish` is run
- **THEN** a working completion script is generated
- **SHALL** complete subcommands, agent numbuhs, and flags

#### AC-10.3: Version Command
- **WHEN** `moonbase version` or `moonbase --version` is run
- **THEN** displays version, commit hash, and build date
- **SHALL** be injected at build time via `-ldflags`

---

### AC-11: Dry-Run Mode (Feature)

#### AC-11.1: Mission Preview
- **WHEN** `moonbase mission --dry-run "add pagination"` is executed
- **THEN** output shows: which mandatory phases would run, which conditional phases would trigger (and why), estimated pipeline duration, and backend that would be used
- **SHALL NOT** invoke any AI backend or modify any state

---

### AC-12: Structured Logging (Feature)

#### AC-12.1: slog Integration
- **WHEN** moonbase performs significant operations (backend calls, phase transitions, errors)
- **THEN** structured log entries are written with timestamp, level, component, and context
- **SHALL** use Go 1.21+ `log/slog` with JSON handler when `--debug` flag is set

#### AC-12.2: Debug Flag
- **WHEN** `moonbase --debug <command>` is run
- **THEN** debug-level logs are written to `~/.config/moonbase/debug.log`
- **SHALL NOT** output debug logs to stdout (they'd interfere with TUI)

---

### AC-13: Makefile DX Improvements (P3)

#### AC-13.1: Lint Target
- **WHEN** `make lint` is run
- **THEN** it executes `go vet ./...` and `go run ./cmd/moonbase lint`
- **SHALL** fail on any lint error

#### AC-13.2: Coverage Target
- **WHEN** `make coverage` is run
- **THEN** it generates an HTML coverage report and opens it
- **SHALL** display the total coverage percentage in the terminal

---

### AC-14: Graceful Shutdown (Feature)

#### AC-14.1: Context Propagation
- **WHEN** the TUI is running a pipeline and the user presses Ctrl+C
- **THEN** a context cancellation propagates to the active backend call
- **SHALL** terminate the subprocess or HTTP request within 5 seconds

#### AC-14.2: No Orphaned Goroutines
- **WHEN** the TUI exits (normally or via abort)
- **THEN** all goroutines spawned by pipeline execution are cleaned up
- **SHALL** not leak file handles or temp files

---

## Scope

### In Scope
- P0: Global state fix, CI lint fix
- P1: File splitting, path resolution, dead code removal
- P2: Error handling, dynamic list, docs update
- P3: Makefile targets
- Features: Glamour rendering, cobra migration, dry-run, slog, graceful shutdown

### Out of Scope
- New agents or agent content changes
- Anthropic/OpenAI backend implementation (remains stub — separate spec)
- Multi-project workspace support
- Agent marketplace or sharing
- Recursive watcher (low priority, separate concern)
- CONTRIBUTING.md and CHANGELOG.md (documentation, not code)

---

## Dependencies

| Dependency | Version | Purpose | Status |
|------------|---------|---------|--------|
| `github.com/charmbracelet/glamour` | v1.0.0 | Markdown rendering | Already in go.mod |
| `github.com/spf13/cobra` | v1.8+ | CLI framework | New dependency |
| `log/slog` | stdlib | Structured logging | Go 1.21+ (we use 1.24) |
| `github.com/charmbracelet/bubbletea` | v1.3.10 | TUI (existing) | Already in go.mod |

---

## Risks

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Cobra migration breaks pipe mode | Medium | High | Test pipe mode explicitly in migration; cobra supports custom stdin handling |
| app.go split introduces import cycles | Low | Medium | Keep single package `tui/`; split by file, not by package |
| Glamour rendering adds latency | Low | Low | Render in goroutine; cache rendered output per message |
| agentsDir() resolution finds wrong directory | Medium | Medium | Priority order is: config > executable-relative > cwd > home; log which was selected |
| Cobra adds binary size | Low | Low | ~2MB increase; acceptable for the functionality gained |
