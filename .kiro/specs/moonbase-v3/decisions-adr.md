# Architecture Decisions: Moonbase v3

## ADR-006: syscall.Exec for Interactive Deploy

### Status
Proposed

### Context
`moonbase deploy 4` needs to give the user an interactive AI session. Using `exec.Command` with stdin/stdout piping creates a subprocess — this often breaks terminal features (colours, readline, ctrl+C handling). The alternative is `syscall.Exec` which replaces the current process entirely.

### Decision
Use `syscall.Exec` to replace the moonbase process with kiro-cli for interactive deploy. The user's terminal is handed directly to kiro-cli with full TTY support.

### Consequences
- User gets native kiro-cli experience (full terminal features work)
- Moonbase process ceases to exist after exec (no cleanup possible after)
- Temp file for prompt must be cleaned up BEFORE exec (or use a path kiro-cli knows to clean)
- On failure, exec returns an error and moonbase can fall back to clipboard
- This approach doesn't work for pipeline (which needs to capture output) — pipeline uses subprocess

### Alternatives Considered
- **exec.Command with Stdin/Stdout:** Works but breaks terminal features; user experience is degraded
- **Write prompt to known location, tell user to run kiro-cli:** Too manual, bad UX
- **OS-specific PTY forwarding:** Complex, non-portable, overkill

### Reversibility
HIGH — can switch to exec.Command if syscall.Exec causes issues on any platform.

---

## ADR-007: tea.Cmd for Async Pipeline in TUI

### Status
Proposed

### Context
The TUI needs to run AI backend calls that take 10-60 seconds each. If run synchronously, the TUI freezes (no spinner, no key handling, no abort). Bubbletea's solution is `tea.Cmd` — a function that runs in a goroutine and returns a message when done.

### Decision
Each pipeline phase is a `tea.Cmd`. The TUI dispatches `executePhase(N)`, which runs the backend in a goroutine. When done, a `PhaseResultMsg` arrives in `Update()`. The TUI remains responsive throughout.

### Consequences
- Spinner continues while backend runs
- User can press `esc` to abort
- Only one phase runs at a time (sequential pipeline)
- Error handling happens in Update() when the message arrives
- Need to track "pipeline running" state to prevent double-dispatch

### Alternatives Considered
- **Synchronous execution:** Rejected — freezes TUI, terrible UX
- **Channel-based manual goroutine:** More complex than tea.Cmd, re-invents what Bubbletea provides
- **External process manager:** Over-engineered for sequential pipeline

### Reversibility
HIGH — the pipeline package already works standalone. TUI execution is just the delivery mechanism.

---

## ADR-008: YAML Config, No Secrets

### Status
Proposed

### Context
The current config stores API keys in plaintext JSON (`"apiKey": "$OPENAI_API_KEY"`). This is a security anti-pattern — config files get committed to dotfile repos, backed up to cloud, or exposed via screenshots. Additionally, JSON is inconsistent with the rest of moonbase (agents use YAML frontmatter).

### Decision
Config uses YAML. API keys are NEVER stored in config — backends read directly from environment variables. Config stores only preferences: default_backend, theme, agents_dir, agent_order.

### Consequences
- Users who set keys in old config.json must use environment variables instead
- Migration detects old file, converts preferences, drops API keys, warns user
- Config file is safe to commit to dotfiles
- Simpler schema (no nested backends map)
- Consistent with agent format (YAML)

### Alternatives Considered
- **Keep JSON with secrets:** Rejected — security risk
- **Encrypted config:** Over-engineered — env vars are the standard
- **TOML:** Rejected — YAML is already in the project, no need for a third format

### Reversibility
MEDIUM — migration is one-way (JSON → YAML). Old file is preserved as .bak.
