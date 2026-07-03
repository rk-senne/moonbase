# Design: Stability & Feature Upgrades

## Architecture Decision

This work is intentionally incremental — no architectural rewrites, no new packages beyond cobra. The principle is: fix what's broken, remove what's dead, then add what's cool. Each change should be independently mergeable.

**Key decisions:**
1. **Cobra replaces manual `os.Args` parsing** — proper CLI framework, tab completion, version injection. Single biggest reliability improvement.
2. **Glamour for markdown rendering** — already in go.mod (pulled as transitive dep). Zero new dependency cost.
3. **slog for structured logging** — stdlib, no external deps. Debug logs go to file, not stdout.
4. **File splits stay in `internal/tui/` package** — no sub-packages (avoids export hell). Split by responsibility, not by abstraction.
5. **`context.Context` propagation for shutdown** — added to `executePhase`, stored on `App`, cancelled on abort.

---

## Files Affected

| File | Change Type | Purpose |
|------|-------------|---------|
| `internal/tui/pipeline_exec.go` | modify | Move `pipelineRunning` into App; add context param |
| `internal/tui/app.go` | major modify | Move `streamCh` to struct; extract Update handlers; fix `NewApp()` path |
| `internal/tui/update_dashboard.go` | new | Dashboard view key handling |
| `internal/tui/update_comms.go` | new | COMMS view key handling + streaming logic |
| `internal/tui/update_pipeline.go` | new | Pipeline view key handling + phase management |
| `internal/tui/update_mission.go` | new | Mission input handling |
| `internal/tui/update_common.go` | new | Shared update helpers (boot, resize, tick) |
| `internal/tui/render.go` | new | Glamour markdown rendering wrapper |
| `.github/workflows/ci.yml` | modify | Remove lint fallback |
| `internal/pipeline/pipeline.go` | modify | Remove dead `Backend` interface |
| `cmd/moonbase/main.go` | rewrite | Cobra-based CLI with subcommands |
| `cmd/moonbase/deploy.go` | new | Deploy subcommand (extracted from main) |
| `cmd/moonbase/mission.go` | modify | Add `--dry-run` flag |
| `cmd/moonbase/init.go` | modify | Fix `writeTemplate` error handling |
| `cmd/moonbase/version.go` | new | Version subcommand with ldflags |
| `cmd/moonbase/list.go` | new | Dynamic list from registry |
| `internal/backend/backends.go` | modify | Remove `envExists` (use shared or inline) |
| `internal/logging/logger.go` | new | slog setup with file handler |
| `docs/design.md` | rewrite | Match actual file structure |
| `Makefile` | modify | Add `lint`, `coverage` targets |
| `.goreleaser.yml` | modify | Add ldflags for version injection |

---

## Component Designs

### 1. Global State → App Fields (P0)

Current (bad):
```go
// internal/tui/pipeline_exec.go:32
var pipelineRunning bool

// internal/tui/app.go:1281
var streamCh <-chan chat.StreamChunk
```

After:
```go
// Fields added to App struct in app.go
type App struct {
    // ... existing fields ...
    pipelineRunning bool
    streamCh        <-chan chat.StreamChunk
    cancelPipeline  context.CancelFunc  // for graceful shutdown
}
```

All reads/writes go through `a.pipelineRunning` and `a.streamCh`. The `executePhase` function becomes a method on `App` (it already accesses App state via closure — this makes it explicit).

---

### 2. CI Lint Fix (P0)

Before:
```yaml
- name: Agent lint
  run: go run ./cmd/moonbase lint 2>/dev/null || echo "lint command not yet wired"
```

After:
```yaml
- name: Agent lint
  run: go run ./cmd/moonbase lint
```

One line change. The `runLint()` function already works and returns non-zero on failure.

---

### 3. TUI File Split (P1)

The `Update()` method is currently a ~600-line switch statement. Split by extracting each case into a function:

```go
// app.go — Update remains here but delegates:
func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        return a.handleKeyMsg(msg)
    case tea.WindowSizeMsg:
        return a.handleResize(msg)
    case PhaseResultMsg:
        return a.handlePhaseResult(msg)
    case streamChunkMsg:
        return a.handleStreamChunk(msg)
    // ... other msg types
    }
    // spinner, blink, tick updates
    return a.handleTick(msg)
}

// update_dashboard.go
func (a App) handleDashboardKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) { ... }

// update_comms.go
func (a App) handleCommsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) { ... }
func (a App) handleStreamChunk(msg streamChunkMsg) (tea.Model, tea.Cmd) { ... }

// update_pipeline.go
func (a App) handlePipelineKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) { ... }
func (a App) handlePhaseResult(msg PhaseResultMsg) (tea.Model, tea.Cmd) { ... }

// update_mission.go
func (a App) handleMissionKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) { ... }
```

**Rules:**
- `app.go` keeps: struct definition, `NewApp()`, `Init()`, top-level `Update()` dispatch, `View()`
- Each `update_*.go` handles one view's keys and related messages
- No new types needed — same `App` struct, same package

Target: `app.go` goes from 1,481 lines to ~350 lines.

---

### 4. Agent Path Resolution (P1)

Extract `agentsDir()` from `cmd/moonbase/main.go` into a shared location accessible by the TUI:

```go
// internal/agents/resolve.go
package agents

// FindAgentsDir searches for the agents directory in priority order:
// 1. Explicit path from config
// 2. Relative to the executable binary
// 3. Relative to CWD
// 4. ~/.moonbase/agents
// 5. ~/.config/moonbase/agents
// 6. CWD/.kiro/agents
func FindAgentsDir(configPath string) (string, error) {
    if configPath != "" {
        if fi, err := os.Stat(configPath); err == nil && fi.IsDir() {
            return configPath, nil
        }
    }
    // ... same logic as current agentsDir() in main.go ...
}
```

Then `NewApp()` calls:
```go
dir, err := agents.FindAgentsDir(cfg.AgentsDir)
if err != nil {
    // Show helpful error in TUI boot screen instead of panic
}
reg := agents.NewRegistry(dir)
```

---

### 5. Dead Code Removal (P1)

**Pipeline Backend interface** — delete lines 53-55 of `internal/pipeline/pipeline.go`:
```go
// DELETE THIS:
type Backend interface {
    Deploy(agentName string, prompt string, input string) (string, error)
}
```
No code references it. The TUI uses `backend.Backend` directly.

**Duplicate `envExists`** — inline as `os.Getenv(key) != ""` at both call sites, or keep one in `backend` package and import in main. Inlining is simpler since it's a one-liner.

---

### 6. Glamour Markdown Rendering (Feature)

```go
// internal/tui/render.go
package tui

import (
    "github.com/charmbracelet/glamour"
)

var mdRenderer *glamour.TermRenderer

func init() {
    // Use dark theme; width will be set on resize
    r, err := glamour.NewTermRenderer(
        glamour.WithAutoStyle(),
        glamour.WithWordWrap(80),
    )
    if err != nil {
        // Fallback: raw text (no-op renderer)
        return
    }
    mdRenderer = r
}

// RenderMarkdown renders markdown to styled terminal output.
// Falls back to raw text if renderer is unavailable.
func RenderMarkdown(md string, width int) string {
    if mdRenderer == nil {
        return md
    }
    rendered, err := mdRenderer.Render(md)
    if err != nil {
        return md
    }
    return rendered
}
```

Usage in COMMS view:
```go
// When displaying agent response
rendered := RenderMarkdown(msg.Content, a.width - sidebarWidth - 4)
```

Pipeline chat: only render `PipelineMsg` where `msg.Agent != ""` (skip system messages).

---

### 7. Cobra CLI Migration (Feature)

```go
// cmd/moonbase/root.go
package main

import (
    "github.com/spf13/cobra"
)

var (
    version = "dev"
    commit  = "none"
    date    = "unknown"
    debug   bool
)

var rootCmd = &cobra.Command{
    Use:   "moonbase",
    Short: "KND Tactical Operations Terminal",
    Long:  "14-agent AI development pipeline with spec-driven methodology",
    Run: func(cmd *cobra.Command, args []string) {
        // No subcommand = launch TUI (or pipe mode)
        if !isTerminal() {
            runPipeMode()
            return
        }
        runTUI()
    },
}

func init() {
    rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "Enable debug logging to ~/.config/moonbase/debug.log")
    
    rootCmd.AddCommand(deployCmd)
    rootCmd.AddCommand(missionCmd)
    rootCmd.AddCommand(initCmd)
    rootCmd.AddCommand(installCmd)
    rootCmd.AddCommand(statusCmd)
    rootCmd.AddCommand(lintCmd)
    rootCmd.AddCommand(listCmd)
    rootCmd.AddCommand(configCmd)
    rootCmd.AddCommand(versionCmd)
    rootCmd.AddCommand(snippetCmd)
}

func main() {
    rootCmd.Execute()
}
```

```go
// cmd/moonbase/version.go
var versionCmd = &cobra.Command{
    Use:   "version",
    Short: "Print moonbase version",
    Run: func(cmd *cobra.Command, args []string) {
        fmt.Printf("moonbase %s (commit: %s, built: %s)\n", version, commit, date)
    },
}
```

```go
// cmd/moonbase/mission.go
var dryRun bool

var missionCmd = &cobra.Command{
    Use:   "mission [task]",
    Short: "Run full KND Council pipeline on a task",
    Args:  cobra.MinimumNArgs(1),
    Run: func(cmd *cobra.Command, args []string) {
        task := strings.Join(args, " ")
        if dryRun {
            runMissionDryRun(task)
            return
        }
        runMission(task)
    },
}

func init() {
    missionCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview pipeline plan without executing")
}
```

---

### 8. Dry-Run Mode (Feature)

```go
// cmd/moonbase/mission.go
func runMissionDryRun(task string) {
    fmt.Println("🌙 MISSION DRY-RUN")
    fmt.Printf("   Task: %s\n\n", task)
    
    p := pipeline.New(task)
    
    // Discover project context
    cwd, _ := os.Getwd()
    ctx, _ := discovery.Discover(cwd)
    
    // Show backend
    be := backend.Preferred()
    fmt.Printf("   Backend: %s\n", be.Name())
    fmt.Println()
    
    // Show phases
    fmt.Println("   PHASES:")
    for _, phase := range p.Phases {
        if phase.Conditional {
            trigger := p.ShouldInvokeConditional(&phase)
            if trigger.Invoke {
                fmt.Printf("   ⚡ [%d] %s — %s (TRIGGERED: %s)\n", 
                    phase.Number, phase.Name, phase.Operative, trigger.Reason)
            } else {
                fmt.Printf("   ⊘  [%d] %s — %s (skip: %s)\n", 
                    phase.Number, phase.Name, phase.Operative, trigger.Reason)
            }
        } else {
            fmt.Printf("   →  [%d] %s — %s\n", phase.Number, phase.Name, phase.Operative)
        }
    }
    
    fmt.Println()
    if ctx != nil {
        fmt.Printf("   Context: %s\n", ctx.Summary())
    }
    fmt.Println("   (No agents invoked. Use without --dry-run to execute.)")
}
```

---

### 9. Structured Logging (Feature)

```go
// internal/logging/logger.go
package logging

import (
    "log/slog"
    "os"
    "path/filepath"
)

var Logger *slog.Logger

// Init sets up the logger. If debug is true, writes JSON to debug.log.
// Otherwise, logs only WARN+ to stderr.
func Init(debug bool) {
    if debug {
        home, _ := os.UserHomeDir()
        logPath := filepath.Join(home, ".config", "moonbase", "debug.log")
        os.MkdirAll(filepath.Dir(logPath), 0o700)
        f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
        if err != nil {
            // Fall back to discard
            Logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
            return
        }
        Logger = slog.New(slog.NewJSONHandler(f, &slog.HandlerOptions{Level: slog.LevelDebug}))
    } else {
        Logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
    }
}
```

Usage:
```go
logging.Logger.Info("phase started", "phase", phase.Number, "agent", phase.AgentName)
logging.Logger.Error("backend failed", "error", err, "backend", be.Name())
```

---

### 10. Graceful Shutdown (Feature)

```go
// In App struct:
type App struct {
    // ...
    pipelineCtx    context.Context
    cancelPipeline context.CancelFunc
}

// When starting a mission:
func (a *App) startMission(task string) tea.Cmd {
    ctx, cancel := context.WithCancel(context.Background())
    a.pipelineCtx = ctx
    a.cancelPipeline = cancel
    // ...
    return a.startNextPhase()
}

// When user presses Esc during pipeline:
func (a *App) abortMission() {
    if a.cancelPipeline != nil {
        a.cancelPipeline()
    }
    a.pipelineRunning = false
    a.pipelineState.Stop("Aborted by human")
}

// In executePhase — respect context:
func executePhase(ctx context.Context, ...) tea.Cmd {
    return func() tea.Msg {
        select {
        case <-ctx.Done():
            return PhaseResultMsg{Err: ctx.Err()}
        case r := <-ch:
            return PhaseResultMsg{Output: r.output, Err: r.err}
        }
    }
}
```

---

### 11. Dynamic runList (P2)

```go
// cmd/moonbase/list.go
func runList() {
    dir := agentsDir()
    reg := agents.NewRegistry(dir)
    all := reg.All()
    
    fmt.Println("🌙 KND MOONBASE — OPERATIVE ROSTER")
    fmt.Println("═══════════════════════════════════════")
    fmt.Println()
    
    // Group by pipeline position
    var core, specialists []agents.Agent
    for _, a := range all {
        if a.PipelinePosition > 0 && a.PipelinePosition <= 5 {
            core = append(core, a)
        } else {
            specialists = append(specialists, a)
        }
    }
    
    fmt.Println("  SECTOR V")
    for _, a := range core {
        fmt.Printf("  [%d] %-18s %s\n", a.PipelinePosition, a.Designation, a.Role)
    }
    
    fmt.Println()
    fmt.Println("  SPECIALISTS")
    for _, a := range specialists {
        id := strings.TrimPrefix(a.Name, "numbuh-")
        if a.Name == "sector-z" { id = "Z" }
        fmt.Printf("  [%s] %-18s %s\n", id, a.Designation, a.Role)
    }
    // ... backends section unchanged
}
```

---

### 12. Makefile Additions (P3)

```makefile
lint:
	go vet ./...
	go run ./cmd/moonbase lint

coverage:
	go test ./... -coverprofile=coverage.out -timeout 60s
	go tool cover -func=coverage.out | tail -1
	go tool cover -html=coverage.out -o coverage.html
	@echo "→ Open coverage.html for details"

.PHONY: lint coverage
```

---

### 13. Version Injection via goreleaser

```yaml
# .goreleaser.yml additions
builds:
  - ldflags:
      - -s -w
      - -X main.version={{.Version}}
      - -X main.commit={{.ShortCommit}}
      - -X main.date={{.Date}}
```

And in Makefile for dev builds:
```makefile
VERSION ?= dev
build:
	go build -ldflags "-X main.version=$(VERSION) -X main.commit=$(shell git rev-parse --short HEAD) -X main.date=$(shell date -u +%Y-%m-%d)" -o $(BUILD_DIR)/$(APP_NAME) $(MAIN)
```
