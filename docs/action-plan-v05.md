# Moonbase v0.5 — Action Plan

## Session Goal
Transform Moonbase from a launcher into a full AI command center with embedded chat, visual flair, and developer tools.

---

## Phase 1: COMMS Panel (Embedded AI Chat)
**Priority: HIGHEST — this is the killer feature**

- [ ] Create `internal/chat/chat.go` — message history, streaming state
- [ ] Anthropic API streaming integration (Claude as primary)
- [ ] OpenAI API streaming as fallback
- [ ] Ollama local as offline fallback
- [ ] Chat viewport with scrolling (bubbles/viewport)
- [ ] Text input at bottom of COMMS panel
- [ ] Agent system prompt injected automatically
- [ ] Token-by-token streaming render
- [ ] Press `C` from dossier → opens COMMS with that agent
- [ ] Chat history persists per-agent in `~/.config/moonbase/chats/`

## Phase 2: Agent Portraits (ANSI Art)
**Priority: HIGH — visual personality**

- [ ] Create `internal/tui/portraits.go`
- [ ] Design 14 unique ASCII/ANSI portraits (8-10 lines each)
- [ ] Show portrait in dossier view (right side)
- [ ] Show portrait in COMMS panel header
- [ ] Portraits colored with agent's theme color

## Phase 3: Full-Width Layout Redesign
**Priority: HIGH — fill the screen**

- [ ] Detect terminal size on launch (`tea.WindowSizeMsg`)
- [ ] Responsive panels: sidebar (25%), main (50%), right panel (25%)
- [ ] Dashboard quadrant layout:
  - Top-left: Agent roster (sidebar)
  - Top-right: Intel feed + system status
  - Bottom: COMMS panel or mission view
- [ ] Dossier layout:
  - Left: Info + capabilities
  - Right: Portrait + personality stats
- [ ] Minimum width fallback (collapse right panel if < 120 cols)

## Phase 4: Clock + Stats
**Priority: MEDIUM — atmosphere + utility**

- [ ] Live clock in header (updates every second via tea.Tick)
- [ ] Mission elapsed timer (during pipeline)
- [ ] Agent deployment stats (track in `~/.config/moonbase/stats.json`)
- [ ] Stats view: bar chart of agent usage
- [ ] "Threat level" gauge based on `git diff --stat` (files changed → LOW/MED/HIGH)

## Phase 5: Tool Launcher
**Priority: MEDIUM — convenience**

- [ ] Press `L` → launch lazygit in alt screen
- [ ] Press `D` → launch lazydocker in alt screen
- [ ] Press `B` → launch btop in alt screen
- [ ] Return to Moonbase after tool exits
- [ ] Detect which tools are installed, gray out unavailable

## Phase 6: File Watcher + History
**Priority: LOWER — nice-to-have**

- [ ] File watcher panel: show last 10 modified files (fsnotify)
- [ ] Mission history: log completed missions to JSON
- [ ] History view: browse past missions with results
- [ ] Search history with `/` in history view

---

## Technical Notes

### API Keys (for COMMS)
- `ANTHROPIC_API_KEY` — already detected ✓
- `OPENAI_API_KEY` — already detected ✓
- Ollama — no key needed, just needs `ollama serve` running

### New Dependencies Needed
```
go get github.com/charmbracelet/bubbles/viewport
go get github.com/fsnotify/fsnotify  (for file watcher)
```

### File Structure Additions
```
internal/
├── chat/
│   ├── chat.go           ← message model, history
│   ├── anthropic.go      ← Claude streaming
│   ├── openai.go         ← GPT streaming
│   └── ollama.go         ← local model streaming
├── tui/
│   ├── portraits.go      ← ANSI art for all agents
│   ├── comms.go          ← chat panel view
│   ├── stats.go          ← agent usage stats view
│   └── tools.go          ← tool launcher
└── watcher/
    └── watcher.go        ← file change detection
```

---

## How to Start Tomorrow

```bash
cd ~/Workspace/Personal/moonbase
# Start with: "Continue building Moonbase v0.5 — start with the COMMS panel (embedded AI chat)"
```

Give me this plan and I'll pick up exactly where we left off.

---

## Current State (as of tonight)
- 2 commits, clean build
- 14 agents, all working
- TUI boots with logo, sidebar, dossier, pipeline view
- Deploy works (kiro-cli + clipboard)
- Search, git commands, spawn hooks all wired
- 3 themes
- `moonbase deploy <n>` CLI command
- Config at `~/.config/moonbase/config.json`
