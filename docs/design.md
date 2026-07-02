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
│       └── main.go                 ← entry point
├── internal/
│   ├── tui/
│   │   ├── app.go                  ← root bubbletea model
│   │   ├── styles.go               ← lipgloss theme (colors, borders, badges)
│   │   ├── keys.go                 ← key bindings
│   │   ├── views/
│   │   │   ├── dashboard.go        ← main command center view
│   │   │   ├── dossier.go          ← operative detail view
│   │   │   ├── pipeline.go         ← mission/pipeline running view
│   │   │   ├── help.go             ← operations manual overlay
│   │   │   ├── backend.go          ← AI backend selector
│   │   │   └── mission.go          ← mission briefing input
│   │   └── components/
│   │       ├── sidebar.go          ← operative roster sidebar
│   │       ├── header.go           ← scanline header bar
│   │       ├── statusbar.go        ← bottom key hints
│   │       ├── intelfeed.go        ← activity log panel
│   │       ├── phaselist.go        ← pipeline phase tracker
│   │       └── systemstatus.go     ← CPU/MEM/GIT/DOCKER panel
│   ├── agents/
│   │   ├── loader.go               ← reads agent .md files (YAML frontmatter + body)
│   │   ├── agent.go                ← agent struct/schema
│   │   ├── registry.go             ← agent lookup, filtering
│   │   └── validator.go            ← config validation
│   ├── backend/
│   │   ├── backend.go              ← interface for AI backends
│   │   ├── detect.go               ← auto-detect available backends
│   │   ├── kiro.go                 ← kiro-cli integration
│   │   ├── codex.go                ← codex CLI integration
│   │   ├── openai.go               ← openai API integration
│   │   ├── anthropic.go            ← anthropic API integration
│   │   ├── ollama.go               ← ollama local integration
│   │   └── clipboard.go            ← copy-prompt-to-clipboard fallback
│   ├── pipeline/
│   │   ├── pipeline.go             ← phase state machine
│   │   ├── phase.go                ← individual phase execution
│   │   ├── riskgate.go             ← risk assessment logic
│   │   └── history.go              ← mission log/history
│   ├── system/
│   │   ├── git.go                  ← git status, branch, diff
│   │   ├── docker.go               ← docker ps, container status
│   │   └── stats.go                ← CPU, memory, disk
│   └── config/
│       ├── config.go               ← app config (paths, defaults)
│       └── paths.go                ← agent dir, steering dir resolution
├── agents/                          ← the actual agent .md files (YAML frontmatter + markdown body)
│   ├── numbuh-0.md
│   ├── numbuh-1.md
│   ├── numbuh-2.md
│   ├── numbuh-3.md
│   ├── numbuh-4.md
│   ├── numbuh-5.md
│   ├── numbuh-9.md
│   ├── numbuh-13.md
│   ├── numbuh-86.md
│   ├── numbuh-274.md
│   ├── numbuh-362.md
│   ├── numbuh-999.md
│   ├── sector-z.md
│   └── knd-council.md
├── docs/
│   ├── design.md                    ← this file (architecture + vision)
│   ├── agent-format.md              ← agent .md file format specification
│   └── action-plan-v05.md           ← historical action plan
├── go.mod
├── go.sum
├── Makefile
├── .gitignore
└── README.md
```

---

## Tech Stack

| Layer | Tool | Why |
|-------|------|-----|
| Language | Go 1.22+ | Single binary, fast, cross-platform |
| TUI Framework | Bubbletea | Elm architecture, composable, active |
| Styling | Lipgloss | CSS-like terminal styling, colors, borders |
| Components | Bubbles | Pre-built spinners, tables, viewports, inputs |
| Config | YAML frontmatter in .md | Agent configs are markdown with YAML metadata, human and machine readable |
| Build | Make | Simple, universal |

---

## Views

### 1. Dashboard (default)
- Header: scanline "K.N.D. MOONBASE" bar
- Left sidebar: operative roster (Sector V + Specialists) with status indicators
- Right main panel: mission briefing / intel feed / system status
- Bottom: key hints contextual to current focus

### 2. Dossier (operative selected)
- Full operative profile: name, codename, role, archetype, MBTI
- Capabilities: tools, shell commands, write permissions, spawn hook
- Personality summary
- Escalation chain (who they hand off to)
- Actions: deploy, copy prompt, preview, run spawn hook

### 3. Pipeline (mission active)
- Phase list with status (✅ 🔄 ⏳), timing, summary
- Live output viewport (scrollable)
- Risk gate status
- Files touched tracker
- Controls: next, retry, skip, abort

### 4. Help (overlay)
- Full operations manual
- Key bindings organized by context
- "The KND Way" — brief explanation of the system

### 5. Mission Briefing (input)
- Text input for task description
- Backend selector
- Pipeline mode selector (full council, single operative, custom)

### 6. Backend Selector
- List of detected AI backends with availability status
- Switch active backend
- Configure API keys/paths

---

## AI Backend Integration

Each backend implements:

```go
type Backend interface {
    Name() string
    Available() bool
    Deploy(agent Agent, task string) error
    Stream() <-chan string  // live output (optional)
}
```

| Backend | How it deploys |
|---------|---------------|
| **kiro-cli** | `kiro-cli chat --agent <name>` with piped task |
| **codex** | `codex --prompt <system_prompt> <task>` |
| **openai** | API call with agent prompt as system message |
| **anthropic** | API call with agent prompt as system message |
| **ollama** | `ollama run <model>` with system prompt |
| **clipboard** | Copies full prompt to clipboard (fallback for any tool) |

---

## CLI Commands (non-TUI mode)

```bash
moonbase                    # launch TUI dashboard
moonbase list               # print operative roster
moonbase deploy <numbuh>    # deploy specific operative (opens AI backend)
moonbase mission <task>     # run full council pipeline
moonbase status             # show current mission status
moonbase backends           # list available AI backends
moonbase install            # symlink agents to ~/.kiro/agents/ for Kiro compat
moonbase help               # operations manual
```

---

## Key Bindings

### Global
| Key | Action |
|-----|--------|
| `?` | Toggle help overlay |
| `q` | Quit (with confirmation if mission active) |
| `tab` | Cycle focus between panels |
| `esc` | Back / close overlay |
| `/` | Search operatives |

### Dashboard
| Key | Action |
|-----|--------|
| `0-9`, `F1` | Select operative by numbuh |
| `k` | Select KND Council |
| `m` | New mission briefing |
| `d` | Quick git diff |
| `g` | Quick git status |
| `l` | View activity log |

### Dossier (operative selected)
| Key | Action |
|-----|--------|
| `enter` | Deploy operative with current backend |
| `c` | Copy full system prompt to clipboard |
| `p` | Preview full prompt in viewport |
| `t` | Run spawn hook and show output |
| `h` | Show handoff chain |
| `i` | Toggle full info |

### Pipeline (mission active)
| Key | Action |
|-----|--------|
| `n` | Advance to next phase |
| `r` | Retry current phase |
| `s` | Skip phase |
| `l` | View full log |
| `esc` | Abort mission (with confirmation) |

---

## Configuration

`~/.config/moonbase/config.json`:

```json
{
  "agentsDir": "./agents",
  "steeringDir": "./.kiro/steering",
  "defaultBackend": "kiro-cli",
  "theme": "moonbase",
  "backends": {
    "openai": {
      "apiKey": "$OPENAI_API_KEY",
      "model": "gpt-4o"
    },
    "anthropic": {
      "apiKey": "$ANTHROPIC_API_KEY",
      "model": "claude-sonnet-4-20250514"
    },
    "ollama": {
      "model": "llama3.1"
    },
    "kiro": {
      "binary": "kiro-cli"
    }
  }
}
```

---

## Design Tokens (Lipgloss Theme)

```go
// Colors
ColorActive    = lipgloss.Color("#00FF88")  // green — active/success
ColorWarning   = lipgloss.Color("#FFAA00")  // amber — in progress
ColorError     = lipgloss.Color("#FF4444")  // red — error/critical
ColorInfo      = lipgloss.Color("#00BBFF")  // cyan — info/highlight
ColorDim       = lipgloss.Color("#555555")  // gray — inactive
ColorHeader    = lipgloss.Color("#FF6600")  // orange — KND brand

// Borders
BorderActive   = lipgloss.NormalBorder()
BorderPanel    = lipgloss.RoundedBorder()

// Badges
BadgeActive    = "◉"
BadgeInactive  = "○"
BadgePass      = "✅"
BadgeRunning   = "🔄"
BadgeWaiting   = "⏳"
BadgeFail      = "❌"
```

---

## Build & Install

```bash
# Development
make run              # go run cmd/moonbase/main.go
make build            # go build -o bin/moonbase cmd/moonbase/main.go
make test             # go test ./...

# Install globally
make install          # copies binary to /usr/local/bin/moonbase

# Cross-compile
make release          # builds for darwin-arm64, darwin-amd64, linux-amd64
```

---

## Milestones

### v0.1 — Foundation
- [ ] Project scaffold (go mod, directory structure)
- [ ] Agent JSON loader and registry
- [ ] Basic TUI with sidebar + main panel
- [ ] Lipgloss theme and styling
- [ ] Dashboard view with operative roster
- [ ] Key navigation (select operative, tab between panels)

### v0.2 — Dossier & Deploy
- [ ] Dossier view (full operative profile)
- [ ] Backend detection (which AI tools are available)
- [ ] Deploy operative (open AI backend with prompt loaded)
- [ ] Copy prompt to clipboard
- [ ] Spawn hook execution + display

### v0.3 — Pipeline
- [ ] Pipeline state machine
- [ ] Mission briefing input
- [ ] Phase tracking view with timing
- [ ] Risk gate logic
- [ ] Live output viewport

### v0.4 — Backend Integrations
- [ ] kiro-cli integration
- [ ] codex integration
- [ ] openai API direct
- [ ] anthropic API direct
- [ ] ollama integration

### v0.5 — Polish
- [ ] Help overlay
- [ ] System status panel (git, docker, resources)
- [ ] Intel feed (activity log with timestamps)
- [ ] Mission history
- [ ] `moonbase install` (symlink agents to .kiro/agents)
- [ ] CLI subcommands (list, deploy, mission, status)

### v1.0 — Release
- [ ] Cross-platform builds
- [ ] README with screenshots
- [ ] Homebrew formula
- [ ] Config file support
- [ ] Theme customization

---

## Philosophy

1. **Tool-agnostic** — works with any AI backend, locks into none
2. **Portable agents** — agent files are plain markdown with YAML frontmatter, usable anywhere
3. **Command-center UX** — Pentagon/Moonbase energy, not a toy
4. **Useful first, pretty second** — every pixel serves a purpose
5. **Single binary** — no runtime deps, no install complexity
6. **KND-flavored** — the lore isn't decoration, it's the UX language
