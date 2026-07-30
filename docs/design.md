# Moonbase

> K.N.D. Tactical Operations Terminal — Command center for AI agent orchestration.

## What Is This?

Moonbase is a terminal UI (TUI) built in Go with Bubbletea that acts as a command center for managing and deploying AI agents. The agents follow the Kids Next Door (KND) operative structure — a council of specialized AI personas that handle different phases of software development.

Moonbase is **AI-tool agnostic**. It doesn't call AI APIs directly. It orchestrates whichever AI backend you have available (kiro-cli, codex, openai, anthropic, ollama, cursor) — making your agent system portable across any tool.

---

## Architecture

```
moonbase/
├── cmd/
│   └── moonbase/
│       ├── main.go                 ← entry point, CLI routing, deploy, list, snippet
│       ├── init.go                 ← `moonbase init` scaffolding
│       ├── install.go              ← `moonbase install` agent installation
│       ├── mission.go              ← `moonbase mission` pipeline execution
│       ├── status.go              ← `moonbase status` health checks
│       └── main_test.go            ← CLI integration tests
├── internal/
│   ├── tui/
│   │   ├── app.go                  ← root bubbletea model (Update, View, Init)
│   │   ├── views.go                ← all view rendering (dashboard, dossier, pipeline, help)
│   │   ├── styles.go               ← lipgloss theme (colors, borders, badges)
│   │   ├── boot.go                 ← boot sequence animation
│   │   ├── animations.go           ← transition and loading animations
│   │   ├── comms.go                ← COMMS panel (chat input/output)
│   │   ├── docview.go              ← document viewer (scrollable viewport)
│   │   ├── filebrowser.go          ← file browser panel
│   │   ├── pipeline_exec.go        ← pipeline execution view integration
│   │   ├── portraits.go            ← ASCII art operative portraits
│   │   ├── projectnav.go           ← project navigator panel
│   │   ├── protocol.go             ← human interaction protocol UI
│   │   └── tui_test.go             ← TUI unit tests
│   ├── agents/
│   │   ├── agent.go                ← Agent struct/schema definitions
│   │   ├── parser.go               ← YAML frontmatter + markdown body parser
│   │   ├── registry.go             ← agent lookup, filtering, loading from disk
│   │   ├── resolve.go              ← agent directory resolution logic
│   │   └── parser_test.go          ← parser unit tests
│   ├── backend/
│   │   ├── backend.go              ← Backend interface + SafeEnv
│   │   ├── backends.go             ← backend detection, dispatch, available check
│   │   └── backend_test.go         ← backend tests
│   ├── pipeline/
│   │   ├── pipeline.go             ← phase state machine, orchestrator
│   │   ├── context.go              ← pipeline context (files touched, output)
│   │   ├── riskgate.go             ← risk assessment logic (LOW/MEDIUM/HIGH/CRITICAL)
│   │   ├── triggers.go             ← conditional specialist trigger evaluation
│   │   ├── pipeline_test.go        ← pipeline unit tests
│   │   └── integration_test.go     ← pipeline integration tests
│   ├── discovery/
│   │   ├── discovery.go            ← project context discovery (.kiro/specs, stack)
│   │   ├── compose.go              ← prompt composition (agent + context + task)
│   │   ├── steering.go             ← steering rules loader
│   │   └── discovery_test.go       ← discovery tests
│   ├── config/
│   │   ├── config.go               ← YAML app config (Load, Show, Path)
│   │   ├── migrate.go              ← config migration (JSON → YAML)
│   │   └── config_test.go          ← config tests
│   ├── chat/
│   │   ├── chat.go                 ← chat message types
│   │   ├── stream.go               ← streaming output handler
│   │   ├── persist.go              ← chat history persistence
│   │   └── persist_test.go         ← persistence tests
│   ├── clipboard/
│   │   ├── clipboard.go            ← cross-platform clipboard (pbcopy/xclip/wl-copy)
│   │   └── clipboard_test.go       ← clipboard tests
│   ├── history/
│   │   ├── history.go              ← mission history storage and export
│   │   └── history_test.go         ← history tests
│   ├── docs/
│   │   ├── docs.go                 ← embedded documentation loader
│   │   └── docs_test.go            ← docs tests
│   ├── snippets/
│   │   ├── snippets.go             ← prompt snippet management
│   │   └── snippets_test.go        ← snippets tests
│   ├── watcher/
│   │   ├── watcher.go              ← file system watcher for live reload
│   │   └── watcher_test.go         ← watcher tests
│   ├── platform/
│   │   ├── platform.go             ← OS detection utilities
│   │   └── platform_test.go        ← platform tests
│   └── projects/
│       ├── projects.go             ← multi-project management
│       └── projects_test.go        ← projects tests
├── agents/                          ← the actual agent .md files (source of truth)
│   ├── numbuh-0.md through numbuh-999.md
│   ├── sector-z.md
│   └── knd-council.md
├── doctrine/                        ← operating doctrine documents (reference only)
├── docs/
│   └── design.md                    ← this file (architecture + vision)
├── .kiro/
│   ├── specs/                       ← feature specifications
│   └── steering/                    ← project development rules
├── .github/workflows/               ← CI + release automation
├── .goreleaser.yml                  ← cross-platform binary release config
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

---

## Tech Stack

| Layer | Tool | Why |
|-------|------|-----|
| Language | Go 1.26 | Single binary, fast, cross-platform |
| TUI Framework | Bubbletea | Elm architecture, composable, active |
| Styling | Lipgloss | CSS-like terminal styling, colors, borders |
| Components | Bubbles | Pre-built spinners, tables, viewports, inputs |
| Config | YAML | App config in `~/.config/moonbase/config.yaml` |
| Agent Format | Markdown + YAML frontmatter | Human and machine readable, portable |
| Build | Make + goreleaser | Simple dev builds, cross-platform releases |

---

## TUI Architecture

The TUI uses a flat file structure (no subdirectories). All views are rendered from `views.go` based on the current app state in `app.go`.

### Key Files

| File | Responsibility |
|------|---------------|
| `app.go` | Root Bubbletea model — state, Init, Update (key handling, messages), View dispatch |
| `views.go` | All view rendering: dashboard, dossier, pipeline, help overlay, mission input |
| `styles.go` | Lipgloss theme: colors, borders, badges, layout styles |
| `boot.go` | Boot sequence animation (startup splash) |
| `animations.go` | View transition animations |
| `comms.go` | COMMS panel — chat interface with agent relay, file attach, snippets |
| `docview.go` | Scrollable document viewer (viewport wrapper) |
| `filebrowser.go` | In-TUI file browser for project navigation |
| `pipeline_exec.go` | Pipeline execution integration (phase tracking, live output) |
| `portraits.go` | ASCII art operative portraits for dossier view |
| `projectnav.go` | Project navigator (multi-project switching) |
| `protocol.go` | Human interaction protocol UI (certainty levels, questions) |

### State Management

The app model in `app.go` holds all state — current view, selected agent, pipeline state, COMMS history. Views are pure render functions that read state and return strings.

---

## Pipeline Architecture

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

The pipeline state machine lives in `internal/pipeline/pipeline.go`. Risk assessment in `riskgate.go` determines whether work loops back or proceeds. Conditional specialists (`triggers.go`) deploy based on content signals (files changed, patterns detected).

---

## AI Backend Integration

Each backend implements detection and deployment. The system uses `SafeEnv()` to prevent leaking sensitive environment variables to subprocesses.

| Backend | How it deploys |
|---------|---------------|
| **kiro-cli** | `syscall.Exec` replaces process — full TTY control |
| **codex** | CLI invocation with agent prompt |
| **clipboard** | Copies full composed prompt (fallback for any tool) |

Backend detection happens at runtime via `exec.LookPath` and environment variable checks.

---

## CLI Commands

```bash
moonbase                    # launch TUI dashboard
moonbase init               # scaffold .kiro/ in any project
moonbase deploy <numbuh>    # deploy specific operative (interactive session)
moonbase mission <task>     # run full council pipeline
moonbase install [--all]    # install agents to .kiro/agents/
moonbase status             # environment health check
moonbase lint               # validate all agent .md files
moonbase config             # show current configuration
moonbase list               # print operative roster (dynamic from registry)
moonbase snippet save/list  # manage prompt snippets
moonbase export <id>        # export mission history
moonbase help               # operations manual
```

---

## Security

- **SafeEnv** — child processes get allowlisted env vars only (no secret leakage)
- **Hook guard** — `isValidAgentID()` restricts to `[a-zA-Z0-9-]`, prevents path traversal
- **Input validation** — snippet names validated (length, no path separators, no control chars)
- **File permissions** — user data written with `0600`/`0700`
- **Pipe input limit** — stdin capped at 1MB to prevent OOM
- **No secrets in config** — API keys come from environment variables only

---

## Build & Install

```bash
# Development
make run              # go run ./cmd/moonbase
make build            # go build -o bin/moonbase ./cmd/moonbase
make test             # go test ./...

# Release (via goreleaser on tag)
git tag v1.0.1 && git push --tags
# GitHub Actions produces: darwin/linux × amd64/arm64 binaries
```

---

## Philosophy

1. **Tool-agnostic** — works with any AI backend, locks into none
2. **Portable agents** — agent files are plain markdown with YAML frontmatter, usable anywhere
3. **Command-center UX** — tactical operations energy, not a toy
4. **Useful first, pretty second** — every element serves a purpose
5. **Single binary** — no runtime deps, no install complexity
6. **Spec-driven** — understand before building, spec the non-trivial
7. **Security by default** — env isolation, input validation, permission hardening
