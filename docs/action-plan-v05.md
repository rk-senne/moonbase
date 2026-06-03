# Moonbase v1.0 — Full Action Plan

## Vision
A sci-fi lofi terminal command center. Think: Ghost in the Shell interfaces, Cowboy Bebop ship consoles, Evangelion NERV command screens — but usable. Every panel has purpose. Every animation is subtle. The tool looks like it belongs in a cyberpunk anime but functions like a senior developer's daily driver.

---

## Visual Direction: Sci-Fi Lofi

### Aesthetic References
- NERV HQ (Evangelion) — status panels, phase countdowns, threat levels
- Bebop's ship computer — warm colors on dark, retro scan lines
- Ghost in the Shell terminals — data cascading, matrix-style feeds
- TRON Legacy — clean lines, cyan glow, minimal but dense
- KND Moonbase — the literal space station command room

### Design Rules
1. **Dark background, glowing elements** — text appears to emit light
2. **Scanline headers** — ░░░ bars across the top for that CRT feel
3. **Subtle animation** — blinking cursors, pulsing dots, spinning spinners
4. **Dense information** — every panel has live data, no dead space
5. **Color language** — cyan=info, green=active, amber=warning, red=error, gold=KND brand
6. **Box drawing characters** — ┌─┐│└─┘╔═╗║╚═╝ for structure
7. **Monospace typography** — everything aligned to grid
8. **Ambient motion** — clock ticks, spinners rotate, status dots pulse

### Color Themes

**Moonbase (default):**
```
Background: #0a0a1a (near-black with blue tint)
Primary:    #00FFAA (cyberpunk green)
Accent:     #FFD700 (KND gold)
Info:       #00BBFF (cool cyan)
Warning:    #FFAA00 (warm amber)
Error:      #FF4444 (alert red)
Dim:        #333355 (faded purple-gray)
```

**Treehouse:**
```
Background: #0d1a0d (deep forest)
Primary:    #33FF33 (matrix green)
Accent:     #8B4513 (wood brown)
Info:       #66CC66 (leaf green)
```

**Classified:**
```
Background: #1a0000 (blood dark)
Primary:    #FF3333 (alert red)
Accent:     #FF6600 (warning orange)
Info:       #CC0000 (deep red)
```

**NERV:**
```
Background: #0a0014 (purple-black)
Primary:    #FF6600 (evangelion orange)
Accent:     #9900CC (purple)
Info:       #FF3399 (magenta)
```

---

## Phase 1: COMMS Panel (Embedded AI Chat)
**The killer feature — talk to agents without leaving Moonbase**

- [ ] Create `internal/chat/` package
- [ ] Message model: role (user/assistant), content, timestamp
- [ ] Anthropic Claude streaming (SSE chunked response)
- [ ] OpenAI streaming fallback
- [ ] Ollama local streaming fallback
- [ ] Chat viewport with auto-scroll (bubbles/viewport)
- [ ] Text input bar at bottom with blinking cursor
- [ ] Agent system prompt auto-injected
- [ ] Token-by-token render (characters appear as they stream)
- [ ] Press `C` from dossier → opens COMMS with that agent
- [ ] Agent portrait shown in COMMS header
- [ ] Color-coded messages (user=dim, agent=themed)
- [ ] Chat history per-agent saved to `~/.config/moonbase/chats/<agent>.json`
- [ ] Multi-agent mode: type `@numbuh-4` to hot-switch agents mid-chat
- [ ] Context injection: press `ctrl+f` to attach a file to the message
- [ ] Code blocks syntax-highlighted in responses
- [ ] Copy last response with `ctrl+c` (not kill signal — detect context)

## Phase 2: Agent Portraits (ANSI Art)
**Visual identity for each operative**

- [ ] Create `internal/tui/portraits.go`
- [ ] 14 unique portraits (10 lines × 15 cols each):
  - Numbuh 0: Elder silhouette, long coat
  - Numbuh 1: Bald, sunglasses, serious
  - Numbuh 2: Goggles on forehead, grinning
  - Numbuh 3: Long hair, huge smile, heart eyes
  - Numbuh 4: Messy bowl cut, fists up
  - Numbuh 5: Cap, cool expression, arms crossed
  - Numbuh 13: Nervous, sweat drops
  - Numbuh 86: Military beret, pointing forward
  - Numbuh 274: Smirk, shadow over eyes
  - Numbuh 362: Crown/tiara, commanding pose
  - Numbuh 999: Silhouette with stars, mysterious
  - Numbuh 9: Half-shadow (between worlds)
  - Sector Z: Glitched/static effect
  - KND Council: Treehouse silhouette
- [ ] Portraits colored with agent-specific accent
- [ ] Show in dossier (right panel)
- [ ] Show in COMMS header (small version)
- [ ] Animated "speaking" effect when agent is streaming response

## Phase 3: Full-Width Responsive Layout
**Fill every pixel with useful information**

### Dashboard (default view):
```
┌─── HEADER (scanline + clock + status) ────────────────────────────────┐
├────────────┬─────────────────────────────┬────────────────────────────┤
│ SIDEBAR    │  MAIN PANEL                 │  RIGHT PANEL               │
│            │                             │                            │
│ Operatives │  Intel Feed                 │  System Status             │
│ (roster)   │  (scrolling activity log)   │  • Git branch/status       │
│            │                             │  • Docker containers       │
│ ────────── │                             │  • CPU/MEM gauges          │
│ AI Backend │                             │  ──────────────────        │
│ (selector) │                             │  Recent Files              │
│            │                             │  (last modified)           │
│ ────────── │  ──────────────────────     │  ──────────────────        │
│ Tools      │  COMMS (mini preview)       │  Threat Level              │
│ • lazygit  │  "Last msg from Numbuh 2"   │  ██████░░░░ MEDIUM         │
│ • docker   │                             │                            │
│ • btop     │                             │  Mission History           │
│            │                             │  #12 auth-fix ✅            │
│            │                             │  #11 pagination ✅          │
├────────────┴─────────────────────────────┴────────────────────────────┤
│ STATUS BAR (keybinds + clock + active agent)                          │
└───────────────────────────────────────────────────────────────────────┘
```

### COMMS view (full chat):
```
┌─── HEADER ────────────────────────────────────────────────────────────┐
├────────────┬──────────────────────────────────────────────────────────┤
│ SIDEBAR    │  ┌─ COMMS: Numbuh 4 ─────────────── 👊 ──────────────┐  │
│ (roster)   │  │                                                     │  │
│            │  │  [4]: Show me what you built.                       │  │
│            │  │                                                     │  │
│  ◉ Wally   │  │  > Here's the auth controller...                   │  │
│            │  │                                                     │  │
│            │  │  [4]: Line 23. No validation. You crash on null.    │  │
│            │  │  [4]: Fix it. I'm not passing this.                 │  │
│            │  │                                                     │  │
│            │  │  > _                                                │  │
│            │  │                                                     │  │
│            │  └─────────────────────────────────────────────────────┘  │
├────────────┴──────────────────────────────────────────────────────────┤
│ [enter] SEND  [ctrl+f] ATTACH FILE  [@] SWITCH AGENT  [esc] BACK     │
└───────────────────────────────────────────────────────────────────────┘
```

### Dossier view (with portrait):
```
┌─── HEADER ────────────────────────────────────────────────────────────┐
├────────────┬──────────────────────────────┬───────────────────────────┤
│ SIDEBAR    │  OPERATIVE DOSSIER           │   ╭─────────────╮         │
│ (roster)   │                              │   │   ≋≋≋≋≋≋≋   │         │
│            │  NUMBUH 4 — WALLABEE         │   │   ò_ó  👊   │         │
│  ◉ Wally   │  Role: QA / Verification     │   │   /███\     │         │
│            │  Type: ESTP — The Warrior    │   │   ╱    ╲    │         │
│            │                              │   ╰─────────────╯         │
│            │  ─── CAPABILITIES ───────    │                           │
│            │  Tools: read, shell, grep    │   PERSONALITY STATS       │
│            │  Shell: mvn test, go test    │   ═══════════════════     │
│            │  Spawn: git diff --stat      │   Aggression ████████░ 9  │
│            │                              │   Precision  ██████░░░ 7  │
│            │  ─── PERSONALITY ────────    │   Patience   ██░░░░░░░ 2  │
│            │  Courageous, competitive,    │   Loyalty    █████████░ 9 │
│            │  impulsive but protective.   │   Humor      ███░░░░░░ 3  │
│            │                              │                           │
├────────────┴──────────────────────────────┴───────────────────────────┤
│ [enter] DEPLOY  [C] COMMS  [c] COPY  [t] SPAWN HOOK  [esc] BACK      │
└───────────────────────────────────────────────────────────────────────┘
```

- [ ] Detect terminal dimensions, adapt layout
- [ ] 3-column layout for ≥140 cols
- [ ] 2-column layout for 80-139 cols
- [ ] 1-column layout for < 80 cols (mobile/small)
- [ ] Panels resize proportionally

## Phase 4: Clock, Stats, Gauges
**Ambient data — the sci-fi background hum**

- [ ] Live clock in header (HH:MM:SS, updates every second)
- [ ] Mission elapsed timer during pipeline
- [ ] Threat level gauge: maps `git diff --stat` line count to LOW/MED/HIGH/CRITICAL
- [ ] Agent deployment stats stored in `~/.config/moonbase/stats.json`
- [ ] Stats panel: horizontal bar chart of agent usage
- [ ] Session uptime counter in status bar
- [ ] Animated progress bars (phase completion percentage)

## Phase 5: Tool Launcher
**Never leave Moonbase**

- [ ] Press `L` → launch lazygit (suspends TUI, resumes on exit)
- [ ] Press `D` → launch lazydocker
- [ ] Press `B` → launch btop
- [ ] Press `V` → launch nvim/vim on last modified file
- [ ] Detect installed tools, show in sidebar with ✓/✗
- [ ] Return to Moonbase cleanly after tool exits (tea.ExecProcess)
- [ ] Custom tool launcher config in config.json

## Phase 6: File Watcher
**Live filesystem awareness**

- [ ] fsnotify watches project directory
- [ ] "Recent Files" panel shows last 10 modified files with timestamps
- [ ] File changes trigger intel feed entry
- [ ] Modified files highlighted in pipeline view (which files each phase touched)
- [ ] Press `w` to toggle watcher on/off

## Phase 7: Mission History + Export
**Persistent memory**

- [ ] Log every mission to `~/.config/moonbase/history.json`
- [ ] Each entry: task, timestamp, phases completed, outcome, duration
- [ ] History view: scrollable list of past missions
- [ ] Press `H` to open history
- [ ] `moonbase export <mission-id>` → generates markdown report
- [ ] Report includes: requirements (Phase 1), design (Phase 2), files changed, QA results, PR summary
- [ ] Mission replay: view what each phase produced

## Phase 8: Power User Features
**The pro stuff**

- [ ] **Snippet library**: save reusable prompts per agent
  - `moonbase snippet save "audit auth flow"` 
  - Press `S` in COMMS to pick from saved snippets
- [ ] **Pipe mode**: `echo "fix bug" | moonbase deploy 3` — scriptable
- [ ] **Macro recording**: record sequence of operations, replay as custom pipeline
- [ ] **Agent hot-reload**: fsnotify on `agents/` dir, live reload on change
- [ ] **Handoff visualization**: when agent says "HANDOFF TO", auto-pulse that agent in sidebar
- [ ] **Notification bell**: terminal bell + status flash when pipeline phase completes

## Phase 9: Integrations
**Connect to the outside world**

- [ ] **GitHub PR**: press `P` after Numbuh 5's review → creates PR via `gh`
- [ ] **Clipboard watch**: detect copied text, toast "Send to [active agent]?"
- [ ] **Jira/Linear pull**: `moonbase mission --from-ticket JIRA-123` pulls description as input
- [ ] **Webhook**: `moonbase serve` → HTTP endpoint that accepts mission briefs (for CI/Slack bots)

## Phase 10: Boot Sequence + Ambient Polish
**The vibes**

- [ ] Randomized boot messages (occasional jokes, KND references)
  - "Father detected on network... just kidding. 😏"
  - "Numbuh 13 tripped over a cable. Systems rebooting..."
  - "Supreme Leader 362 authorized your access. Welcome aboard."
- [ ] Ambient blinking dot next to active backend ("● connected")
- [ ] Scan-line animation on header (subtle horizontal sweep every 30s)
- [ ] "Data cascade" effect on boot (matrix-style falling characters)
- [ ] Panel borders glow brighter when that panel has focus
- [ ] NERV-style "ALERT" flash when threat level goes HIGH
- [ ] Typing indicator (⋯) when AI is generating response in COMMS

---

## Implementation Priority

| Order | Phase | Effort | Impact |
|-------|-------|--------|--------|
| 1 | COMMS (chat) | High | 🔥🔥🔥 Game changer |
| 2 | Full-width layout | Medium | 🔥🔥🔥 Fills the screen |
| 3 | Portraits | Medium | 🔥🔥 Visual identity |
| 4 | Clock/stats/gauges | Low | 🔥🔥 Sci-fi atmosphere |
| 5 | Tool launcher | Low | 🔥🔥 Workflow speed |
| 6 | Boot polish | Low | 🔥 Vibes |
| 7 | File watcher | Medium | 🔥 Awareness |
| 8 | Mission history | Medium | 🔥 Memory |
| 9 | Power features | High | 🔥 Pro tools |
| 10 | Integrations | High | 🔥 Ecosystem |

---

## New Dependencies

```bash
go get github.com/charmbracelet/bubbles/viewport   # scrollable panels
go get github.com/fsnotify/fsnotify                 # file watcher
go get github.com/shirou/gopsutil/v3                # CPU/MEM stats
```

---

## How to Start Tomorrow

```bash
cd ~/Workspace/Personal/moonbase
```

Say:
> "Let's build Moonbase v0.5. Start with the COMMS panel — I want embedded AI chat with Anthropic streaming. Then do the full-width layout redesign."

---

## Current State
```
Commits:  3 (88d6ecf)
Binary:   4.8MB
Agents:   14
Views:    Boot, Dashboard, Dossier, Pipeline, Mission, Help
Features: Deploy, search, git commands, spawn hooks, themes
Config:   ~/.config/moonbase/config.json
```
