# Tasks: Moonbase TUI UX Overhaul

## Milestone 1: Colour & Borders (Foundation)

### Task 1: Update colour palette
- **Requirements:** AC-5.1
- **Files:** `internal/tui/styles.go`
- **Action:** Replace all 7 colour constants with the new 6-colour system. Active→#5AF78E, Warning→#F3C14B, Error→#FF6B6B, Info→#7EC8E3, Brand→#FFD700 (keep), Dim→#6B7280. Add ColorText=#E4E4E7 and ColorMuted=#9CA3AF.
- **Test:** `go build ./...` passes. Visual: TUI renders without jarring colours.
- **Status:** pending

### Task 2: Reduce borders
- **Requirements:** AC-5.2
- **Files:** `internal/tui/styles.go`
- **Action:** StyleSidebar→right-border only. StylePanel→no border (padding only). Add StyleModal with full rounded border for help/mission overlays. Keep StatusBar background.
- **Test:** Visual: panels are separated by whitespace not boxes. Modals still have borders.
- **Status:** pending

### Checkpoint: Colour & Borders
- [ ] Palette has 8 named colours (6 functional + brand + dim)
- [ ] Max 2 colours per panel enforced by style usage
- [ ] Sidebar has right-border only
- [ ] Main panel has no border
- [ ] Help/mission overlays have rounded border
- [ ] `go build ./...` passes

---

## Milestone 2: Sidebar Enhancement

### Task 3: Add role labels
- **Requirements:** AC-2.1
- **Files:** `internal/tui/views.go` (sidebar renderer)
- **Action:** After rendering agent name, right-pad and append role string in muted colour. Truncate role to 10 chars. Skip roles if sidebar width < 28.
- **Test:** Sidebar shows `Numbuh 4   QA` alignment.
- **Status:** pending

### Task 4: Add section headers
- **Requirements:** AC-2.2
- **Files:** `internal/tui/views.go` (sidebar renderer)
- **Action:** Group agents into SECTOR V (0-5), SPECIALISTS (362, 274, 86, 999, 13), META (council, sector-z, 9). Render `◆ SECTOR V` etc. as brand-coloured headers between groups.
- **Test:** Sidebar shows three labelled sections.
- **Status:** pending

### Task 5: Add key hints
- **Requirements:** AC-2.3
- **Files:** `internal/tui/views.go` (sidebar renderer)
- **Action:** Prefix each agent with `[N]` in dim colour where N is the quick-select key (0-9, K, Z, F). Only show hints in dashboard view (not pipeline/dossier).
- **Test:** Dashboard shows `[4] Numbuh 4   QA`, pipeline sidebar shows just `◉ Numbuh 4`.
- **Status:** pending

### Checkpoint: Sidebar
- [ ] Roles visible next to each agent name
- [ ] Three section headers separate groups
- [ ] Key hints show on dashboard only
- [ ] Narrow terminals fall back gracefully

---

## Milestone 3: Header & Navigation

### Task 6: Breadcrumb header
- **Requirements:** AC-7.1
- **Files:** `internal/tui/views.go` (header renderer)
- **Action:** Change `renderHeader(title)` to `renderHeader(breadcrumb)`. Pass view-specific breadcrumb from each render function: "Dashboard", "Dossier › Numbuh 4", "Pipeline › Phase 3", etc.
- **Test:** Header shows `🌙 MOONBASE › Dashboard` on main view.
- **Status:** pending

### Task 7: Context status (right side of header)
- **Requirements:** AC-1.1, AC-1.2
- **Files:** `internal/tui/views.go` (header renderer)
- **Action:** Right-align backend name + stack + spec indicator in the header. Use activeBackend.Name() and projectCtx.Stack fields. Show `◆` if specs exist.
- **Test:** Header shows `kiro-cli │ go/go │ ◆` on right side.
- **Status:** pending

### Checkpoint: Header
- [ ] Breadcrumb shows current view name
- [ ] Backend + stack visible on the right
- [ ] Spec indicator appears when .kiro/specs/ exists
- [ ] Header doesn't overflow at narrow widths (truncate middle)

---

## Milestone 4: Pipeline Chat Overhaul

### Task 8: Phase section headers in chat
- **Requirements:** AC-3.1
- **Files:** `internal/tui/views.go` (pipeline renderer)
- **Action:** When rendering pipeline chat, replace the current flat message list with structured phase sections. Each phase gets a `──── Phase N: Name (Operative) ────` header in brand colour.
- **Test:** Pipeline chat shows clear phase boundaries.
- **Status:** pending

### Task 9: Indented agent output with pipe prefix
- **Requirements:** AC-3.2
- **Files:** `internal/tui/views.go` (pipeline renderer) or `internal/tui/pipeline_exec.go` (when recording output)
- **Action:** Prefix each line of agent output with `│ `. Truncate to 8 lines max with `[+N more]` indicator.
- **Test:** Agent text is visually contained within the phase section.
- **Status:** pending

### Task 10: Phase completion footers
- **Requirements:** AC-3.3
- **Files:** `internal/tui/pipeline_exec.go` (handlePhaseResult) + `internal/tui/views.go`
- **Action:** After each phase completes, add a `└── ✅ Complete (Xs)` line in success colour. For failures: `└── ❌ Failed: reason`. Store elapsed per phase.
- **Test:** Each completed phase shows timing.
- **Status:** pending

### Task 11: Risk gate visual
- **Requirements:** AC-3.4
- **Files:** `internal/tui/pipeline_exec.go` (handlePhaseResult)
- **Action:** After risk gate fires, format a prominent line: `🎯 Risk Gate: {LEVEL} → {action}`. Use colour coding per level (green/amber/red).
- **Test:** Risk gate result is visually distinct from regular chat.
- **Status:** pending

### Task 12: Phase elapsed timer in sidebar
- **Requirements:** AC-4.2
- **Files:** `internal/tui/app.go` (add phaseStartTime field), `internal/tui/views.go` (pipeline sidebar)
- **Action:** Store `phaseStartTime` when a phase starts. On each tick, if running, show elapsed seconds next to the running phase badge. After 30s silence, add "Still working..." to chat.
- **Test:** Running phase shows `🔄 3. Implement  12s` updating every second.
- **Status:** pending

### Checkpoint: Pipeline Chat
- [ ] Phase sections visually separated
- [ ] Agent output indented and truncated
- [ ] Completion time shown per phase
- [ ] Risk gate is colour-coded and prominent
- [ ] Elapsed timer updates in sidebar
- [ ] "Still working..." appears after 30s

---

## Milestone 5: Feedback & Dossier

### Task 13: Immediate dispatch feedback
- **Requirements:** AC-4.1, AC-4.3
- **Files:** `internal/tui/app.go` (mission entry handler, deploy handler)
- **Action:** On mission enter: immediately show mission title + "Dispatching to Numbuh 1..." in chat before returning the tea.Cmd. On deploy: immediately add intel entry "Deploying {name}..." before backend invocation.
- **Test:** No blank screen between action and response.
- **Status:** pending

### Task 14: Dossier progressive disclosure
- **Requirements:** AC-6.1, AC-6.2, AC-6.3
- **Files:** `internal/tui/views.go` (dossier renderer)
- **Action:** Restructure dossier to show: name, designation, role, core question, quick stats (tool count, access, routes, position, shortcut) in 10 lines. Full prompt on `p` key. Route chain on `h` key.
- **Test:** Dossier is scannable at a glance. Details available on demand.
- **Status:** pending

### Task 15: Abort confirmation
- **Requirements:** AC-7.2
- **Files:** `internal/tui/app.go` (key handler)
- **Action:** Add `abortPending bool` and `abortPendingAt time.Time` to App. On esc during running pipeline: set pending + show warning in status bar. On second esc within 3s: abort. On other key or timeout: cancel pending.
- **Test:** Single esc shows warning. Double esc aborts. Wait 3s = cancelled.
- **Status:** pending

### Checkpoint: Feedback & Dossier
- [ ] Mission entry shows immediate acknowledgment
- [ ] Dossier shows essential info in ~10 lines
- [ ] `p` shows full scrollable prompt
- [ ] `h` shows routing chain
- [ ] Double-esc required to abort running pipeline
- [ ] Abort warning appears in status bar

---

## Final Verification

- [ ] All milestones complete
- [ ] `go build ./...` passes
- [ ] `go vet ./...` clean
- [ ] TUI launches without panic at 80x24 minimum
- [ ] TUI renders correctly at 120x40 (comfortable)
- [ ] TUI renders correctly at 200x50 (large)
- [ ] All 5 main views render: Dashboard, Dossier, Pipeline, Mission, Help
- [ ] Colour palette is calm and professional
- [ ] Borders are minimal — content breathes
- [ ] Phase sections are clearly parseable
- [ ] Risk gate is visually prominent
- [ ] Sidebar shows roles and grouping
- [ ] Header shows context (breadcrumb + backend + stack)
- [ ] No view feels "wall of text"
