# Design: Moonbase TUI UX Overhaul

## Architecture Decision

All changes are rendering-only. No new packages, no logic changes. The TUI remains a Bubbletea Elm-architecture app. We modify: colour constants, sidebar renderer, header renderer, pipeline chat renderer, dossier renderer, and key-handling state for abort confirmation.

---

## Files Affected

| File | Change Type | Purpose |
|------|------------|---------|
| `internal/tui/styles.go` | modify | New colour palette, reduced borders, weight-first hierarchy |
| `internal/tui/views.go` | modify | Sidebar roles/grouping, pipeline chat sections, breadcrumb header, dossier layout |
| `internal/tui/app.go` | modify | Abort confirmation state, elapsed timer tick, immediate feedback |

---

## Visual Design Specification

### Colour System

```go
// Functional colours (status indicators only)
ColorSuccess  = "#5AF78E"   // green  — pass, complete, active
ColorWarning  = "#F3C14B"   // amber  — in progress, caution
ColorError    = "#FF6B6B"   // red    — fail, critical, danger
ColorInfo     = "#7EC8E3"   // cyan   — links, interactive elements

// Structural colours (text hierarchy)
ColorBrand    = "#FFD700"   // gold   — headings, brand marks only
ColorText     = "#E4E4E7"   // light  — primary text (not white — less harsh)
ColorMuted    = "#9CA3AF"   // gray   — secondary text, labels
ColorDim      = "#6B7280"   // dark   — disabled, hints, dividers
ColorBg       = "#1a1a2e"   // navy   — header/statusbar background

// Usage rules:
// - Max 2 colours per panel at any time
// - Hierarchy: Bold text > Normal text > Muted text > Dim text
// - Colour ONLY for status (pass/fail/running/warning)
// - Everything else: white/gray weight hierarchy
```

### Border System

```go
// Before: Every panel had full box borders
// After: Minimal dividers

StyleSidebar = lipgloss.NewStyle().
    BorderRight(true).
    BorderStyle(lipgloss.NormalBorder()).
    BorderForeground(ColorDim).
    Padding(1, 1, 1, 0)

StylePanel = lipgloss.NewStyle().
    Padding(0, 1)  // No border — content breathes

StyleModal = lipgloss.NewStyle().
    Border(lipgloss.RoundedBorder()).
    BorderForeground(ColorInfo).
    Padding(1, 2)  // Full border only for modals
```

---

## Component Specifications

### 1. Header with Breadcrumb + Context

```
┌─────────────────────────────────────────────────────────────────────┐
│ 🌙 MOONBASE › Dashboard                       kiro-cli │ go/go │ ◆ │
└─────────────────────────────────────────────────────────────────────┘
```

**Layout:** Left = brand + breadcrumb. Right = backend + stack + spec indicator.

```go
func (a App) renderHeader(view string) string {
    left := fmt.Sprintf("🌙 MOONBASE › %s", view)
    
    right := a.activeBackend.Name()
    if a.projectCtx != nil && a.projectCtx.Stack.Language != "" {
        right += " │ " + a.projectCtx.Stack.Language + "/" + a.projectCtx.Stack.BuildTool
    }
    if a.projectCtx != nil && a.projectCtx.HasSpecs() {
        right += " │ ◆"
    }
    
    // Pad and join
    gap := a.width - len(left) - len(right) - 4
    header := left + strings.Repeat(" ", max(1, gap)) + right
    
    return StyleHeader.Width(a.width).Render(header)
}
```

**Breadcrumb values:**
- Dashboard: `› Dashboard`
- Dossier: `› Dossier › {agent.Designation}`
- Pipeline: `› Pipeline › Phase {N}: {Name}`
- Mission: `› New Mission`
- Help: `› Operations Manual`
- Docs: `› Documentation`
- Projects: `› Projects`

### 2. Sidebar with Roles + Grouping

```
◆ SECTOR V
 [0] Numbuh 0    Overseer
 [1] Numbuh 1    Analyst
 [2] Numbuh 2    Architect
▸[3] Numbuh 3    Implement
 [4] Numbuh 4    QA
 [5] Numbuh 5    Reviewer

◆ SPECIALISTS
 [6] Numbuh 362  DevOps
 [7] Numbuh 274  Security
 [8] Numbuh 86   Cleanup
 [9] Numbuh 999  Docs
 [F] Numbuh 13   Chaos

◆ META
 [K] KND Council Pipeline
 [Z] Sector Z    Legacy
```

**Rendering rules:**
- Section headers: brand colour, bold
- Selected agent: `▸` prefix + active colour + bold
- Unselected: space prefix + normal text
- Role: right-aligned, muted colour, truncated to 10 chars
- Key hints: `[N]` in dim colour, only on dashboard view
- Fall back to no-roles layout if sidebar width < 28

```go
type sidebarGroup struct {
    title  string
    agents []sidebarEntry
}

type sidebarEntry struct {
    key   string  // "4", "K", "Z"
    name  string  // "Numbuh 4"
    role  string  // "QA"
    index int     // registry index
}

var sidebarLayout = []sidebarGroup{
    {"SECTOR V", []sidebarEntry{
        {"0", "Numbuh 0", "Overseer", 0},
        {"1", "Numbuh 1", "Analyst", 1},
        {"2", "Numbuh 2", "Architect", 2},
        {"3", "Numbuh 3", "Implement", 3},
        {"4", "Numbuh 4", "QA", 4},
        {"5", "Numbuh 5", "Reviewer", 5},
    }},
    {"SPECIALISTS", []sidebarEntry{
        {"6", "Numbuh 362", "DevOps", 6},
        {"7", "Numbuh 274", "Security", 7},
        {"8", "Numbuh 86", "Cleanup", 8},
        {"9", "Numbuh 999", "Docs", 9},
        {"F", "Numbuh 13", "Chaos", 10},
    }},
    {"META", []sidebarEntry{
        {"K", "Council", "Pipeline", 11},
        {"Z", "Sector Z", "Legacy", 12},
    }},
}
```

### 3. Pipeline Chat with Phase Sections

```
━━━ MISSION: Add pagination to /users ━━━

──── Phase 1: Requirements (Numbuh 1) ─────────────
│ Mission Objective: Add pagination support
│ AC-1.1: WHEN page/pageSize params THEN limit results
│ AC-1.2: WHEN no params THEN default 20 results
│ Risks: None identified
│ Handoff: → Numbuh 2
└── ✅ Complete (3.2s)

──── Phase 2: Design (Numbuh 2) ───────────────────
│ Route: limit/offset in repository, handler parses params
│ Files: internal/users/repo.go, internal/users/handler.go
│ Pattern: Follow existing FindAll + add Paginated variant
└── ✅ Complete (5.1s)

──── Phase 3: Implementation (Numbuh 3) ───────────
│ Added PaginatedFindAll with limit/offset
│ Updated handler to parse page/pageSize query params
│ Added TestFindAll_WithPagination
│ go test ./... — PASS
│ [+12 more lines]
└── ✅ Complete (8.4s)

──── Phase 4: QA (Numbuh 4) ───────────────────────
│ AC-1.1: ✅ PASS — page=2&pageSize=10 returns 10 items
│ AC-1.2: ✅ PASS — no params returns 20
│ Verdict: LOW
└── ✅ Complete (4.7s)

  🎯 Risk Gate: LOW → Proceed to Review

──── Phase 5: Review (Numbuh 5) ───────────────────
│ 🔄 Running (7s)...
```

**Rendering logic:**

```go
func (a App) formatPhaseChat(phase int, operative string, output string, status string, elapsed time.Duration) string {
    var b strings.Builder
    
    // Phase header
    header := fmt.Sprintf("──── Phase %d: %s (%s) ", phase, phaseName(phase), operative)
    header += strings.Repeat("─", max(1, chatWidth - len(header)))
    b.WriteString(lipgloss.NewStyle().Foreground(ColorBrand).Render(header) + "\n")
    
    // Content (indented with pipe)
    lines := strings.Split(strings.TrimSpace(output), "\n")
    maxLines := 8
    for i, line := range lines {
        if i >= maxLines {
            b.WriteString(lipgloss.NewStyle().Foreground(ColorDim).Render(
                fmt.Sprintf("│ [+%d more lines]", len(lines)-maxLines)) + "\n")
            break
        }
        b.WriteString("│ " + line + "\n")
    }
    
    // Footer
    switch status {
    case "complete":
        b.WriteString(lipgloss.NewStyle().Foreground(ColorSuccess).Render(
            fmt.Sprintf("└── ✅ Complete (%s)", elapsed.Round(100*time.Millisecond))) + "\n")
    case "failed":
        b.WriteString(lipgloss.NewStyle().Foreground(ColorError).Render(
            "└── ❌ Failed") + "\n")
    case "running":
        b.WriteString(lipgloss.NewStyle().Foreground(ColorWarning).Render(
            fmt.Sprintf("│ 🔄 Running (%ds)...", int(elapsed.Seconds()))) + "\n")
    }
    
    return b.String()
}
```

### 4. Risk Gate Visual

```
  🎯 Risk Gate: LOW → Proceed to Review
  🎯 Risk Gate: MEDIUM → Back to Implementation (rework 1/2)
  🎯 Risk Gate: HIGH → Back to Design (rework 1/2)
  🛑 Risk Gate: CRITICAL — Pipeline stopped. Escalate to human.
```

**Colour coding:**
- LOW: success (green)
- MEDIUM: warning (amber)
- HIGH: error (red)
- CRITICAL: error (red) + bold

### 5. Dossier (Progressive Disclosure)

**Default view (essential info):**

```
  NUMBUH 4 — Wallabee Beatles
  QA / Verification Operative

  "Does it hold when I hit it?"

  ━━ Quick Stats ━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  Tools     7: read, shell, grep, glob, code, knowledge, subagent
  Access    Read-only shell (test commands)
  Routes    → 3 (fix) │ → 2 (redesign) │ → 5 (approve) │ → 274 (security)
  Position  Pipeline #4
  Shortcut  ctrl+shift+4

  [enter] DEPLOY  [p] FULL PROMPT  [h] ROUTES  [t] HOOK  [esc] BACK
```

**Key `p` → scrollable full prompt:**
```
  ━━ Full Agent Prompt (9,476 chars) ━━━━━━━━

  # Numbuh 4 — QA / Verification Operative
  
  You are Numbuh 4 (Wallabee Beetles)...
  [scrollable viewport with ↑↓/pgup/pgdn]

  [esc] BACK TO DOSSIER
```

### 6. Elapsed Timer in Pipeline Sidebar

```
◆ PIPELINE
──────────────

✅ 1. Requirements    3s
✅ 2. Design          5s
🔄 3. Implementation 12s  ← updates every tick
⏳ 4. QA
⏳ 5. Review
──────────────
⚡ 6. Oversight
⚡ 7. Security
⚡ 8. Deploy Prep
```

**Implementation:** Store `phaseStartTime time.Time` in App. On each clock tick (every second), if pipeline running, render elapsed. On phase complete, store final duration in phase struct.

### 7. Abort Confirmation

**State machine:**

```
Normal state:
  [esc] → if pipeline running: set abortPending=true, show warning
        → if pipeline idle: navigate back

Abort pending state (3s timeout):
  [esc] → actually abort pipeline
  [any other key] → cancel pending, back to normal
  [3s elapsed] → cancel pending automatically
```

**Status bar during pending:**
```
⚠️ Press [esc] again to abort mission. Any other key to cancel.
```

---

## Data Flow

### Elapsed Timer

```
animTick (every 150ms) or clockTick (every 1s)
    ↓
If pipelineRunning && currentPhase.Status == Running:
    elapsed = time.Since(phaseStartTime)
    ↓
renderPipeline() reads elapsed, formats "12s"
    ↓
If elapsed > 30s && no chat update in 30s:
    append "Still working..." to pipelineChat
```

### Abort Confirmation

```
User presses [esc]
    ↓
If pipelineRunning:
    set abortPending = true
    set abortPendingAt = time.Now()
    return (don't abort yet)
    ↓
On next [esc] within 3s:
    abort pipeline
    set abortPending = false
    ↓
On any other key OR 3s timeout:
    set abortPending = false
    (cancelled)
```

---

## Edge Cases

| Scenario | Handling |
|----------|----------|
| Terminal width < 60 | Fall back to 1-column layout, no sidebar roles |
| Terminal width < 80 | Fall back to 2-column, shorter roles |
| Agent name very long | Truncate at sidebar width - role width - 4 |
| Phase output is empty | Show "│ (no output)" in muted |
| Phase output > 50 lines | Show first 8 + "[+N more]" |
| Clock tick while pipeline idle | Don't show elapsed (only when running) |
| Breadcrumb exceeds header width | Truncate middle with "..." |

---

## Migration Notes

- Old `ColorActive`/`ColorWarning`/etc. variables are replaced in-place — all views automatically pick up new values
- `StylePanel` border removal affects all panels at once — test all views
- Sidebar grouping requires the registry's sort order to match the new `sidebarLayout` structure
- The `renderPipeline()` function needs the most changes (phase sections + elapsed)
- Dossier changes are isolated to `renderDossier()`
