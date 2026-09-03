# 🌙 Moonbase

**K.N.D. Tactical Operations Terminal**

A 14-agent AI development pipeline with spec-driven methodology, human interaction protocols, and project-aware execution. Built for Kiro CLI, compatible with any AI backend.

```bash
moonbase init                    # make any project agent-ready
moonbase deploy 4 "check auth"  # deploy Numbuh 4 with a task
moonbase mission "add pagination to /users API"  # run the pipeline (adaptive)
```

---

## What It Does

- **14 AI operatives** with distinct roles, personalities, and scoped permissions
- **Spec-driven development** — agents discover `.kiro/specs/` and steering rules automatically
- **Human interaction protocol** — agents ask focused questions when uncertain, proceed when confident
- **Risk-gated pipeline** — QA classifies risk, work loops back until it holds
- **Project-aware** — discovers your stack, conventions, and specs before doing work
- **Self-auditing** — agents can scan their own codebase for bugs and tech debt

---

## Documentation

Full guides live in [`wiki/`](wiki/) — staged for the project **[GitHub Wiki](https://github.com/rk-senne/moonbase/wiki)**: Installation, CLI Reference, The Pipeline, Agents, Skills & Prompts, Flywheel & Observability, Configuration, Architecture, Kiro Native Interop, and Contributing. Design docs and feature specs are in [`docs/`](docs/) and [`.kiro/specs/`](.kiro/specs/).

---

## Installation

moonbase is a single self-contained binary (agents are embedded) for macOS and Linux.

### Install script (recommended)

```bash
curl -fsSL https://raw.githubusercontent.com/rk-senne/moonbase/main/install.sh | sh
```

Downloads the latest release for your platform, verifies its checksum, installs to
`~/.local/bin`, and sets up the agents. Override with `MOONBASE_INSTALL_DIR` or pin a
version with `MOONBASE_VERSION=v1.6.0`.

### With Go

```bash
go install github.com/rk-senne/moonbase/cmd/moonbase@latest
moonbase setup   # install the embedded agents to ~/.moonbase/agents
```

### Download a release

Grab `moonbase_<os>_<arch>.tar.gz` from the
[Releases page](https://github.com/rk-senne/moonbase/releases), extract it, put
`moonbase` on your `PATH`, then run `moonbase setup`.

### Build from source

```bash
git clone https://github.com/rk-senne/moonbase.git
cd moonbase && make build && cp bin/moonbase ~/.local/bin/
```

> Agents are baked into the binary, so moonbase works in **any project** with no repo
> checkout — `moonbase init` / `moonbase setup` just work from anywhere.

### Private / internal use (current)

The repository is currently private, so the install script and a plain `go install`
need access. Until it's made public, install within the org by either:

```bash
# Build from source (you have repo access)
git clone git@github.com:rk-senne/moonbase.git
cd moonbase && make build && cp bin/moonbase ~/.local/bin/

# …or go install with private-module + git auth configured
GOPRIVATE=github.com/rk-senne/* go install github.com/rk-senne/moonbase/cmd/moonbase@latest
```

No engineering change is needed to go public later: the release owner is derived from
the binary's module path, goreleaser auto-detects the repo from the git remote, and the
install script targets public release assets — so flipping the repo to public makes the
one-line installer and `go install` work with **no code edits**.

---

## Quick Start

```bash
# Build from source
make build

# Initialize a project for agent-driven development
cd my-project
moonbase init

# Deploy an agent
moonbase deploy 1 "analyze the user authentication flow"

# Run the full KND Council pipeline
moonbase mission "add rate limiting to the API"

# Check environment
moonbase status
```

---

## CLI Commands

| Command | Description |
|---------|-------------|
| `moonbase` | Launch the TUI dashboard |
| `moonbase init` | Scaffold `.kiro/` in any project (specs, steering, agents) |
| `moonbase deploy <n> [task]` | Deploy operative by numbuh (interactive kiro-cli session) |
| `moonbase mission <task>` | Run the KND Council pipeline (auto-classifies depth, escalates on risk) |
| `moonbase mission --fast <task>` | Skip analysis/architecture, run implementation + QA only (use for trivial/well-specified tasks) |
| `moonbase mission --full <task>` | Force all 5 mandatory phases regardless of task complexity |
| `moonbase mission --depth <level> <task>` | Override auto-classification (`trivial`, `simple`, `complex`) |
| `moonbase mission --sequential <task>` | Disable parallel specialist fan-out for this mission |
| `moonbase install [--all] [--global]` | Install agents to project's `.kiro/agents/` (or `~/.kiro/agents/` with `--global`) |
| `moonbase compile [--out] [--validate] [--agent]` | Compile agents to Kiro-native JSON |
| `moonbase setup` | Install agents globally to `~/.moonbase/agents/` |
| `moonbase status` | Show environment health check |
| `moonbase lint` | Validate all agent `.md` files |
| `moonbase config` | Show current YAML configuration |
| `moonbase list` | Show operative roster |
| `moonbase guide [numbuh]` | Show usage guide for agents (aliases: `man`, `howto`) |
| `moonbase history` | Show mission history (`--json`, `--all`, `--limit N`) |
| `moonbase replay <id>` | Replay a previous mission (`--dry-run`) |
| `moonbase export <id>` | Export mission details |
| `moonbase snippet save <name>` | Save a prompt snippet from stdin |
| `moonbase snippet list` | List saved snippets |
| `moonbase flywheel` | Show pipeline learning insights (incl. token/cost) |
| `moonbase version` | Print version information |

**Pipe mode:**
```bash
echo "fix the auth bug" | moonbase              # pipe to KND Council
echo "check security" | moonbase deploy 274     # pipe to specific agent
```

---

## The Pipeline

```
Human Request
    ↓
Numbuh 1  → Requirements (ACs, scope, risks)
    ↓
Numbuh 2  → Design (blueprint, trade-offs, file impact)
    ↓
Numbuh 3  → Implementation (code, tests, build)
    ↓
Numbuh 4  → QA (verify, risk gate)
    ↓                    ↑
    ├── MEDIUM → fix ────┘ (max 2 rework loops)
    ├── HIGH → redesign (back to Numbuh 2)
    ├── CRITICAL → STOP (escalate to human)
    └── LOW ↓
Numbuh 5  → Review (final gate, PR package)
    ↓
Human Approval
```

**Conditional specialists** deploy based on content signals:

| Operative | Triggers When |
|-----------|--------------|
| Numbuh 0 | >5 files changed, core logic modified, new patterns |
| Numbuh 274 | Auth/secrets touched, new endpoints, dependency CVEs |
| Numbuh 362 | CI/CD, Docker, env vars, deployment config changed |
| Numbuh 9 | Version upgrades, breaking changes, migrations |
| Numbuh 13 | Edge case coverage needed, fragile flows, parsers |
| Numbuh 86 | Dead code discovered, unused deps, zombie features |
| Numbuh 999 | README needed, ADRs, changelogs |
| Sector Z | Old/mysterious code touched, legacy context needed |

**Parallel fan-out:** Independent specialists (read-only, no write tools) execute
concurrently after QA returns LOW risk. Bounded to `max_specialist_concurrency`
(default 4). One failure does not cancel siblings. Disable with
`parallel_specialists: false` in config or `--sequential` on the CLI. Set
`specialist_panes: true` (and run moonbase inside a tmux/cmux session) to deploy
each triggered specialist into its own **split pane** — live and interactive —
instead of running them headless.

---

## Agent Format

Each operative is a single self-contained `.md` file:

```yaml
---
name: numbuh-4
designation: Wallabee Beatles
role: QA / Verification
tools: [read, shell, grep, glob, code, knowledge, subagent]
shell:
  allowed_commands: ["go test ./...", "npm test", "mvn test"]
  read_only: true
routing:
  available: [numbuh-2, numbuh-3, numbuh-5, numbuh-274, numbuh-362, numbuh-0]
  trusted: [numbuh-3, numbuh-5]
pipeline_position: 4
triggers: null
mcp_servers: []          # optional MCP servers — compiled to Kiro's mcpServers
---

# Numbuh 4 — QA / Verification

## Identity
Australian. Blunt. Brave. Evidence-driven...

## Purpose
Core question: "Does it hold when I hit it?"

## Output Formats
...

## Operating Protocol
(Evidence standard, human interaction, spec awareness, handoff protocol)
```

One file = one complete agent. Copy to any project's `.kiro/agents/` and it works.

---

## Project Structure

```
moonbase/
├── agents/              ← 14 agent .md files (source of truth)
├── doctrine/            ← 8 operating doctrine documents
├── .kiro/
│   ├── specs/           ← Feature specs (requirements, design, tasks)
│   ├── steering/        ← Project-wide dev rules
│   └── agents/          ← Installed agents for kiro-cli
├── internal/            ← Go source (17 packages)
│   ├── agents/          ← YAML frontmatter parser + registry
│   ├── pipeline/        ← Orchestrator, risk gate, triggers
│   ├── discovery/       ← Project context (.kiro/specs, stack detection)
│   ├── compile/         ← Kiro-native JSON compiler (types, compile, write, staleness)
│   ├── backend/         ← AI backend integrations (kiro-cli, openai, anthropic, kimi, ollama, clipboard)
│   ├── clipboard/       ← Cross-platform clipboard (macOS/Linux/Windows)
│   ├── config/          ← YAML config (no secrets)
│   ├── tui/             ← Bubbletea TUI (dashboard, pipeline, dossier)
│   └── ...              ← chat, history, watcher, docs, snippets
├── cmd/moonbase/        ← CLI entry point + subcommands
├── .github/workflows/   ← CI + release automation
├── .goreleaser.yml      ← Binary release config
├── Makefile
└── go.mod
```

---

## Key Capabilities

### Adaptive Pipeline Depth

The pipeline automatically classifies task complexity and selects the minimum
viable depth, escalating mid-run if the risk gate signals the shallow path was
insufficient.

| Depth | Phases | When |
|-------|--------|------|
| `trivial` | 3 → 4 (Implement + QA) | Short tasks with trivial indicators (typo, rename, unused import) |
| `simple` | 1 → 3 → 4 (Analysis + Implement + QA) | Moderate tasks with some complexity |
| `complex` | 1 → 2 → 3 → 4 → 5 (full pipeline) | Long tasks, many signals, multi-scope |

**Core invariant:** QA (Phase 4) always runs at every depth. If QA returns
MEDIUM or HIGH risk on a shallow pipeline, it automatically escalates to a
deeper depth and re-runs with the added context.

```bash
moonbase mission "fix typo in README"           # → trivial (2 phases)
moonbase mission "add rate limiting to the API"  # → complex (5 phases)
moonbase mission --depth simple "any task"       # override classification
moonbase mission --full "any task"               # force full pipeline
```

### Human Interaction Protocol

```
CERTAIN  → proceed silently
LIKELY   → proceed, label assumption
UNCERTAIN → ask the human (focused question + options + default)
UNKNOWN  → stop, ask, do not guess
```

### Project Discovery

When deployed, agents automatically find:
- `.kiro/specs/` — requirements, design, tasks (references AC-IDs)
- `.kiro/steering/` — project conventions and rules
- Build configs — detects Go/Java/Node/Python/Rust
- README — project overview

### Kiro Native Interop

```bash
moonbase compile --validate    # Emit Kiro-native JSON for all 14 agents
moonbase deploy 3 --native     # Deploy via kiro-cli chat --agent numbuh-3
```

Compiles moonbase's `.md` agent definitions into Kiro-native agent JSON that
passes `kiro-cli agent validate`. When deployed natively, agents inherit Kiro's
permission engine (shell allowlists, write paths), hook system (`agentSpawn`,
`preToolUse`, `postToolUse`, `stop`), and MCP server infrastructure — without
moonbase reimplementing enforcement. See `docs/MIGRATION-NATIVE.md` for the
safety delegation table and opt-in guide.

### Security

- **SafeEnv** — child processes get allowlisted env vars only (no secret leakage)
- **Hook guard** — blocks dangerous commands (curl, rm, python, eval, shell pipes)
- **Input validation** — agent IDs validated against `[a-zA-Z0-9-]`
- **File permissions** — all user data written with `0600`/`0700`
- **No secrets in config** — API keys come from environment variables only

### Flywheel (Self-Improvement)

Moonbase logs pipeline execution data to `~/.moonbase/flywheel.jsonl`.
Over time, patterns emerge that reveal which agents struggle, which phases
get reworked most, and where the pipeline bottlenecks.

Token consumption and estimated cost are captured per phase when using
OpenAI, Anthropic, or Kimi API backends. The flywheel shows total cost,
per-agent breakdowns, and identifies cost-heavy phases.

```bash
moonbase flywheel          # Show learning insights (incl. token/cost)
```

Optional token budget enforcement stops runaway missions:

```yaml
# ~/.config/moonbase/config.yaml
token_budget:
  max_tokens_per_mission: 500000   # hard cap (0 = unlimited)
  warn_threshold_pct: 80           # warn at 80% of budget

model_pricing:
  gpt-4o:
    prompt: 2.50       # USD per 1M prompt tokens (override defaults)
    completion: 10.00
```

### Skills & Stored Prompts

When `moonbase init` scaffolds a project, it creates:
- `.kiro/skills/` — Domain-specific knowledge agents reference progressively
- `.kiro/prompts/` — Reusable named workflows

**Curated Skills Library:** `moonbase init` installs 16 production-ready skills:

| Skill | Focus |
|-------|-------|
| `testing-discipline` | Table-driven Go tests, race detection, regression-first fixes |
| `security-review` | OWASP-aligned checklist: auth, injection, secrets, CVEs |
| `git-workflow` | Conventional commits, atomic changes, PR hygiene |
| `api-design` | REST resource modeling, pagination, versioning, error shapes |
| `error-handling` | Wrap with context, sentinels, fail fast, log at boundaries |
| `docker-build` | Multi-stage builds, layer caching, non-root, .dockerignore |
| `concurrency-patterns` | Goroutine lifecycle, context, mutex vs channel, races |
| `observability` | Structured logging, metrics, traces, never log secrets |
| `refactoring-safely` | Characterization tests first, small steps, green builds |
| `code-review` | Four lenses + constructive etiquette |
| `architecture-boundaries` | Dependency Rule, REP/CCP/CRP, ADP/SDP/SAP, the Main Sequence, boundary cost |
| `architecture-diagrams` | C4 levels, notation that stands alone, deployment-per-environment |
| `design-patterns` | Principles behind the patterns, choosing between them, overuse caution |
| `incremental-migration` | Strangler fig, branch by abstraction, parallel run, tracer write |
| `distributed-data` | Shared-database hazards, sagas vs 2PC, splitting tables safely |
| `algorithmic-complexity` | Complexity bounds, silent preconditions, degenerate inputs |

The last six are derived from the research corpus in [`research/`](research/) and are
attributed to their sources; see [`doctrine/ArchitectureDoctrine.md`](doctrine/ArchitectureDoctrine.md)
for the Reference Canon mapping rules to books.

**Progressive Skill Loading:** Skills use YAML frontmatter (`name`, `description`) so
agents see a lightweight catalog (~100 tokens per skill) instead of full content. Agents
request specific skills with `@skill(name)` to load content on demand — saving context
window tokens for what matters.

```markdown
---
name: docker-build
description: Docker multi-stage build patterns and CI integration.
---

# Docker Build Patterns
...
```

Agents automatically discover skills and display the catalog. Legacy skills without
frontmatter are still loaded eagerly for backward compatibility.

### Terminal Tools Arsenal

The TUI ships a **Tools** view — press `i` from any stage to browse a curated
catalog of critical and cool terminal tools:

| Critical | Cool & Stable |
|----------|---------------|
| git, ripgrep, fzf, jq, tmux, neovim, lazygit, GitHub CLI | btop, bat, eza, zoxide, git-delta, fish, starship, **oh-my-posh**, lazydocker, fd, tree, tig, direnv, duf, yq, just, hyperfine, dust, procs, tealdeer, glow, yazi, zellij, atuin, difftastic, k9s, cmux — plus macOS-only: **mas**, **trash**, **terminal-notifier** |

Each tool shows a live install status (✓/✗). Select one and moonbase installs it
via your platform's package manager — **Homebrew on macOS; the native manager
(apt / dnf / pacman) on Linux**, with Linuxbrew used only as a fallback — but
**only after an explicit `y/n` confirmation that shows the exact command it will
run**. Tools with no package-manager path (or that are macOS-only, like cmux)
show manual guidance and are never auto-run. Install commands are assembled
solely from an allowlisted catalog, never from user input.

The terminal multiplexer launcher (`M`) is **OS-aware: tmux on Linux, cmux on
macOS** (falling back to tmux when cmux isn't installed). moonbase integrates
with **both tmux and cmux** through one layer (`internal/mux`): pipeline
notifications (phase complete, CRITICAL risk, mission complete) fire through
whichever is active, `moonbase status` reports the detected multiplexer and
whether you're in a session, and `moonbase deploy <n> --pane` deploys an
operative into a **split pane** of the active multiplexer.

The **Settings** view (`S`) is **OS-aware**: it shows a **macOS** section and a
**Linux** section, with the running OS highlighted (`✓ · this machine`) and the
other grayed out. Each section has an **Install all** action that installs every
missing recommended tool for that OS in one package-manager command (after a
`y/n` confirmation showing the exact command).

### Live Mission Visibility

- Press `m` from **any** view to brief a new mission; `enter` reliably deploys the
  council (an empty briefing simply asks for an objective — it never navigates away).
- A **mission-in-progress indicator** (`⚡ <phase> P<done>/<total>`) stays in the
  header from every view, so you never lose sight of a running mission.
- Each operative's pipeline feedback is labelled with its **KND persona and voice**
  (e.g. `▸ Numbuh 4 · Wallabee Beatles`) in the operative's colour, and the live
  agent output streams into the pipeline view (markdown rendering is cached so the
  view stays smooth).

---

## Build & Install

```bash
# Development
make run              # go run ./cmd/moonbase
make build            # go build -o bin/moonbase
make test             # go test ./...

# Install locally
cp bin/moonbase /usr/local/bin/

# Release (produces cross-platform binaries)
git tag v1.4.0 && git push --tags
# GitHub Actions builds: darwin/linux × amd64/arm64
```

---

## Quality

| Metric | Value |
|--------|-------|
| Tests | 1,494 (1,242 test functions incl. subtests) |
| Packages tested | 18/19 (the root package is `go:embed`-only) |
| Go LOC | ~15,300 |
| Direct dependencies | 8 |
| CI | GitHub Actions (vet + build + `govulncheck` + `-race` test on every push) |
| Releases | goreleaser (4 platform binaries on tag) |
| Agent lint | `moonbase lint` validates all 14 agents |

---

## Stack

| Layer | Tool | Why |
|-------|------|-----|
| Language | Go 1.26 | Single binary, fast, cross-platform |
| TUI | Bubble Tea v2 + Lip Gloss v2 + Bubbles v2 | Elm architecture, terminal styling, components |
| Agents | Markdown + YAML frontmatter | Portable, readable, versionable |
| Backend | Kiro CLI (primary), OpenAI, Anthropic, Kimi, Ollama | Tool execution, API streaming, multi-turn |
| Clipboard | pbcopy / xclip / xsel / wl-copy / clip | Cross-platform fallback |
| CI | GitHub Actions | Automated quality gates |
| Releases | goreleaser | Cross-platform binaries |

---

## Philosophy

1. **Pure markdown agents** — one file per operative, no assembly required
2. **Spec before code** — understand before building, spec the non-trivial
3. **Ask don't guess** — focused questions beat wrong assumptions
4. **Evidence over claims** — prove it or label it an assumption
5. **Self-auditing** — agents can scan their own system
6. **Tool-agnostic** — works with Kiro, Codex, OpenAI, Anthropic, Ollama
7. **Single binary** — no runtime deps, no install complexity
8. **Security by default** — env isolation, input validation, permission hardening

---

## License

MIT — see [LICENSE](LICENSE). Copyright (c) 2026 Senne.

---

*"Kids Next Door... battle stations."*
