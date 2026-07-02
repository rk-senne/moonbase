# Design: Moonbase v3 — Production-Ready

## Architecture Decision

The TUI pipeline becomes a real execution engine that spawns backend calls as Bubbletea commands (async, non-blocking). The CLI deploy uses `os.Exec` to fully replace the process with kiro-cli (interactive mode). Config moves to YAML with no secrets. Dead packages are consolidated or removed.

**Key ADR:** The TUI pipeline runs agents asynchronously using `tea.Cmd`. Each phase starts a goroutine that invokes the backend, and the result comes back as a `tea.Msg`. This keeps the TUI responsive (spinner works, user can abort) while agents run.

---

## Files Affected

| File | Change Type | Purpose |
|------|------------|---------|
| `internal/tui/app.go` | modify | Wire pipeline to real backend via tea.Cmd |
| `internal/tui/pipeline_exec.go` | new | Pipeline execution commands (tea.Cmd wrappers) |
| `cmd/moonbase/main.go` | modify | Fix deploy to use exec.Syscall for interactive mode |
| `cmd/moonbase/mission.go` | modify | Minor fixes from deploy learnings |
| `cmd/moonbase/main_test.go` | new | CLI command tests |
| `internal/backend/backend_test.go` | new | Backend detection and selection tests |
| `internal/config/config.go` | rewrite | YAML-based config, no secrets |
| `internal/config/config_test.go` | new | Config load/save/default tests |
| `internal/config/migrate.go` | new | JSON → YAML migration |
| Doctrine files | modify/delete | Remove stale references |
| `internal/docs/docs.go` | audit | Keep if TUI uses, remove if dead |
| `internal/history/history.go` | audit | Keep if TUI uses, remove if dead |
| `internal/platform/platform.go` | audit | Keep if TUI uses, remove if dead |
| `internal/projects/projects.go` | audit | Keep if TUI uses, remove if dead |

---

## Component Designs

### 1. TUI Pipeline Execution

```go
// internal/tui/pipeline_exec.go

// PhaseResultMsg is sent when a pipeline phase completes.
type PhaseResultMsg struct {
    Phase  int
    Output string
    Err    error
}

// executePhase returns a tea.Cmd that runs an agent via the backend.
func (a App) executePhase(phase pipeline.Phase) tea.Cmd {
    return func() tea.Msg {
        agent := a.registry.GetByName(phase.AgentName)
        if agent == nil {
            return PhaseResultMsg{Phase: phase.Number, Err: fmt.Errorf("agent %s not found", phase.AgentName)}
        }

        // Discover project context (cached from pipeline start)
        composed := discovery.ComposePrompt(agent.Prompt, a.projectCtx, a.pipelineState.Context.ForPhase(phase.Number))

        // Deploy to preferred backend
        output, err := a.backend.Deploy(*agent, a.projectCtx, composed)
        return PhaseResultMsg{Phase: phase.Number, Output: output, Err: err}
    }
}
```

**State machine in Update():**
```
Mission entered → executePhase(1) → PhaseResultMsg{1} 
    → record output → check risk gate if phase 4
    → executePhase(next) → PhaseResultMsg{next} → ...
    → complete or abort
```

### 2. CLI Deploy (Interactive)

```go
// cmd/moonbase/main.go — runDeploy

func runDeploy(numbuh string) {
    // ... resolve agent, discover context, compose prompt ...

    // For interactive mode: exec replaces process with kiro-cli
    // User gets full terminal interaction with the agent
    if kiro, err := exec.LookPath("kiro-cli"); err == nil {
        tmpFile := writeTempPrompt(composed)
        defer os.Remove(tmpFile)

        // syscall.Exec replaces the process — no output capture
        syscall.Exec(kiro, []string{"kiro-cli", "chat", "--context", tmpFile}, os.Environ())
    }

    // Fallback: clipboard
}
```

The key insight: `syscall.Exec` replaces the moonbase process with kiro-cli. The user then interacts directly with kiro-cli as if they'd launched it themselves. No output capture, no subprocess management. When kiro-cli exits, the user is back at their shell.

### 3. Config (YAML)

```yaml
# ~/.config/moonbase/config.yaml
default_backend: kiro-cli
theme: moonbase
agents_dir: ""  # empty = auto-detect

# Agent display order (optional override)
agent_order:
  - numbuh-0
  - numbuh-1
  - numbuh-2
  - numbuh-3
  - numbuh-4
  - numbuh-5
  - numbuh-362
  - numbuh-274
  - numbuh-86
  - numbuh-999
  - numbuh-13
  - knd-council
  - sector-z
  - numbuh-9
```

```go
// internal/config/config.go

type Config struct {
    DefaultBackend string   `yaml:"default_backend"`
    Theme          string   `yaml:"theme"`
    AgentsDir      string   `yaml:"agents_dir"`
    AgentOrder     []string `yaml:"agent_order,omitempty"`
}

func Load() Config                      // reads YAML, falls back to defaults
func Save(cfg Config) error             // writes YAML
func Path() string                      // ~/.config/moonbase/config.yaml
func MigrateFromJSON() (bool, error)    // detect old JSON, convert, delete
```

**No API keys in config.** Backends read from environment (`OPENAI_API_KEY`, `ANTHROPIC_API_KEY`). Config stores only preferences.

### 4. Test Architecture

```
cmd/moonbase/
├── main_test.go          ← Test CLI routing, install, deploy resolution
│                           Uses: temp dirs, mock findAgentsSource override

internal/backend/
├── backend_test.go       ← Test DetectAll, DetectAvailable, Preferred
│                           Uses: mock exec.LookPath via test helper

internal/config/
├── config_test.go        ← Test Load, Save, defaults, YAML format
├── migrate_test.go       ← Test JSON → YAML migration
│                           Uses: temp dirs, real file I/O
```

### 5. Dead Code Audit

Based on grep analysis of what the TUI imports:

| Package | Used by TUI? | Action |
|---------|-------------|--------|
| `internal/docs/` | Yes (docview) | **Keep** — renders documentation |
| `internal/history/` | Yes (mission history) | **Keep** — stores mission records |
| `internal/platform/` | Yes (OS detection) | **Keep** — used for platform-specific paths |
| `internal/projects/` | Yes (project nav) | **Keep** — project browser |
| `internal/snippets/` | Yes (snippet picker) | **Keep** — but audit for stale JSON pattern |
| `internal/watcher/` | Yes (file watcher) | **Keep** — live file monitoring |
| `internal/chat/` | Yes (COMMS) | **Keep** — chat state management |

No packages need removal. They're all used by the TUI. The cleanup is:
- Remove any JSON references to old agent format within them
- Ensure they don't import removed packages

---

## Data Flow

### TUI Pipeline (new)

```
User enters mission
    ↓
App stores task, creates Pipeline, discovers ProjectContext
    ↓
App returns executePhase(1) as tea.Cmd
    ↓
[goroutine: compose prompt → backend.Deploy → return output]
    ↓
PhaseResultMsg{1, output} arrives in Update()
    ↓
App records output in PipelineContext, advances pipeline
    ↓
If phase == 4: apply risk gate
    ├── LOW → executePhase(5)
    ├── MEDIUM → executePhase(3) with rework
    ├── HIGH → executePhase(2) with rework
    └── CRITICAL → stop, show error
    ↓
If conditional: evaluate trigger → skip or executePhase(N)
    ↓
Phase 5 complete → show final summary
```

### CLI Deploy (new)

```
moonbase deploy 4 "check auth"
    ↓
Resolve agent: agents/numbuh-4.md → ParseAgentFile → Agent struct
    ↓
Discover: cwd → ProjectContext (specs, steering, stack)
    ↓
Compose: steering + agent.Prompt + specs + task → full prompt string
    ↓
Write to temp file: /tmp/moonbase-XXXX.md
    ↓
syscall.Exec("kiro-cli", ["kiro-cli", "chat", "--context", tmpFile])
    ↓
[moonbase process replaced — user now in kiro-cli session]
```

---

## Edge Cases

| Scenario | Handling |
|----------|----------|
| Backend returns error mid-pipeline | Mark phase failed, show error in chat, offer retry |
| User aborts during backend call | Cancel context (if backend supports), mark aborted |
| No backend available for TUI pipeline | Fall back to simulated pipeline with message "No backend — showing simulation" |
| Config YAML is malformed | Log warning, use defaults, don't crash |
| JSON config exists at old path | Auto-migrate to YAML, delete JSON, warn user once |
| kiro-cli doesn't support `--context` flag | Try `--system-prompt`, then fall back to piping |
| Agent frontmatter parsing regression | Parser tests catch this — all 14 agents are tested every run |

---

## Error Handling

| Error | Response | Requirement |
|-------|----------|-------------|
| Backend not available | Show warning, offer clipboard fallback | AC-2.2 |
| Phase execution timeout | Mark failed after 120s, offer retry/skip | AC-1.1 |
| Risk gate parse failure | Default to MEDIUM, warn in chat | AC-1.2 |
| Config file permission error | Use defaults, warn on stdout | AC-3.1 |
| Agent file not found for deploy | Print available agents with numbers | AC-2.1 |
| kiro-cli exec fails | Fall through to clipboard | AC-2.1 |

---

## Security Considerations
- Config file MUST NOT contain API keys or secrets
- Temp files containing prompts are deleted after use (defer cleanup)
- `syscall.Exec` replaces process — no zombie processes
- Clipboard content is user-controlled — no secret injection risk

---

## Breaking Changes
- **Breaking:** Config file moves from `config.json` to `config.yaml`. Migration auto-runs once.
- **Non-breaking:** CLI commands unchanged. TUI keybindings unchanged.
- **Non-breaking:** Agent format unchanged.
