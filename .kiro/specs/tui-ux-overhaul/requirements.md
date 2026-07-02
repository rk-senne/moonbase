# Requirements: Moonbase TUI UX Overhaul

## Overview

Apply real UI/UX principles to the Moonbase TUI — reducing visual noise, improving information hierarchy, and making the pipeline feel responsive and parseable. Focused on high-impact changes that make the TUI genuinely useful for daily development, not just visually impressive.

**Design principles applied:** Nielsen's heuristics, Fitts's Law, Gestalt grouping, progressive disclosure, information scent.

---

## User Stories

### US-1: At-a-Glance Status
As a developer, I want to see the project context and backend status immediately so I know moonbase is connected and aware of my environment.

### US-2: Agent Recognition
As a developer, I want to identify agents by role (not just number) so I can quickly pick the right operative without memorizing codenames.

### US-3: Pipeline Readability
As a developer watching the pipeline execute, I want to clearly distinguish phase boundaries, agent output, system status, and risk signals so I can follow what's happening without parsing walls of text.

### US-4: Responsive Feedback
As a developer, I want instant visual acknowledgment when I trigger an action so the system never feels frozen or unresponsive.

### US-5: Reduced Visual Noise
As a developer using moonbase for hours, I want a calmer colour palette and less border clutter so the interface doesn't cause eye strain.

---

## Acceptance Criteria

### AC-1: Context Status Bar

#### AC-1.1: Backend + Stack Indicator
- **WHEN** the TUI is displaying any view
- **THEN** the header shows detected backend and project stack on the right side
- **SHALL** format as `backend │ language/tool` (e.g., `kiro-cli │ go/go`)
- **SHALL** show `no backend` in warning colour if only clipboard available

#### AC-1.2: Project Context Signal
- **WHEN** project discovery found specs or steering rules
- **THEN** a subtle indicator appears (e.g., `◆ spec` in the header)
- **SHALL** show nothing extra if no project context found (don't clutter)

---

### AC-2: Sidebar Agent Roles

#### AC-2.1: Role Label Next to Name
- **WHEN** the sidebar renders the agent roster
- **THEN** each agent shows its short role (max 12 chars) right-aligned
- **SHALL** format as: `◉ Numbuh 4   QA` (name left, role right, dim colour)

#### AC-2.2: Section Headers
- **WHEN** the sidebar renders
- **THEN** agents are grouped under `◆ SECTOR V` and `◆ SPECIALISTS` headers
- **SHALL** use dim colour for headers, normal for agent names, bright for selected agent

#### AC-2.3: Number Key Hints
- **WHEN** the sidebar is visible
- **THEN** each agent shows its quick-select key: `[4]` before the name
- **SHALL** use dim colour for the bracket hint, only on dashboard view

---

### AC-3: Pipeline Chat Hierarchy

#### AC-3.1: Phase Section Headers
- **WHEN** a new pipeline phase starts in the chat view
- **THEN** a visual section header appears: `──── Phase N: Name (Operative) ────`
- **SHALL** use brand colour for the header, dim for the divider lines

#### AC-3.2: Agent Output Formatting
- **WHEN** agent output is displayed in pipeline chat
- **THEN** content is indented with `│ ` prefix (pipe character) for visual containment
- **SHALL** truncate to first 8 meaningful lines with `[+N more lines]` indicator

#### AC-3.3: Phase Completion Summary
- **WHEN** a phase completes
- **THEN** a summary line shows: `└── ✅ Complete (Xs)` or `└── ❌ Failed: reason`
- **SHALL** include elapsed time in seconds

#### AC-3.4: Risk Gate Visual
- **WHEN** the QA risk gate fires
- **THEN** a prominent styled line shows the verdict with colour coding
- **SHALL** use: green for LOW, amber for MEDIUM, red for HIGH/CRITICAL
- **SHALL** show routing action: `→ Proceed to Review` or `→ Back to Implementation (rework 1/2)`

---

### AC-4: Responsive Feedback

#### AC-4.1: Immediate Dispatch Acknowledgment
- **WHEN** user starts a mission (presses enter)
- **THEN** immediately (0ms) show: mission title + "Dispatching to Numbuh 1..."
- **SHALL** appear before any backend response arrives

#### AC-4.2: Phase Elapsed Timer
- **WHEN** a pipeline phase is running
- **THEN** the sidebar shows elapsed time next to the running phase
- **SHALL** update every second: `🔄 1. Analysis  12s`
- **SHALL** show a "Still working..." message in chat after 30s of silence

#### AC-4.3: Deploy Acknowledgment
- **WHEN** user presses enter on dossier to deploy an agent
- **THEN** immediately show "Deploying {name}..." in the intel feed
- **SHALL** appear before any backend handoff occurs

---

### AC-5: Colour & Visual Refinement

#### AC-5.1: Reduced Palette
- **WHEN** the TUI renders
- **THEN** use a maximum of 4 functional colours + 1 brand accent + dim
- **SHALL** replace current high-saturation colours with softer alternatives:
  - Active/success: `#5AF78E` (was `#00FF88`)
  - Warning: `#F3C14B` (was `#FFAA00`)
  - Error: `#FF6B6B` (was `#FF4444`)
  - Info: `#7EC8E3` (was `#00BBFF`)
  - Brand (headings only): `#FFD700` (keep)
  - Dim: `#6B7280` (was `#555555`)

#### AC-5.2: Border Reduction
- **WHEN** panels render
- **THEN** use minimal borders: sidebar gets left-border only, main panel gets no border, pipeline sidebar gets no border
- **SHALL** only use full borders for: modal overlays (help, mission input) and the dossier detail panel

#### AC-5.3: Weight Over Colour
- **WHEN** creating information hierarchy
- **THEN** use bold/dim/normal weight as the PRIMARY differentiator
- **SHALL** reserve colour for status indicators (pass/fail/warning) only, not for general text categorisation

---

### AC-6: Dossier Progressive Disclosure

#### AC-6.1: Essential Info First
- **WHEN** the dossier view opens for an agent
- **THEN** show only: name, designation, role, core question (one sentence), quick stats (tool count, access level, routes), shortcut
- **SHALL** fit in 8-10 lines maximum

#### AC-6.2: Details on Demand
- **WHEN** user presses `p` in dossier view
- **THEN** show the full agent prompt in a scrollable viewport
- **SHALL** replace the current dossier content (not overlay)

#### AC-6.3: Handoff Chain Visualisation
- **WHEN** user presses `h` in dossier view
- **THEN** show the agent's routing chain as a visual flow: `Numbuh 4 → Numbuh 3 (fix) / Numbuh 2 (redesign) / Numbuh 5 (approve)`
- **SHALL** show both available and trusted agents with different styling

---

### AC-7: Navigation & Wayfinding

#### AC-7.1: Breadcrumb Header
- **WHEN** any view is active
- **THEN** the header shows a breadcrumb: `🌙 MOONBASE › {View} › {Detail}`
- **SHALL** examples: `› Dashboard`, `› Dossier › Numbuh 4`, `› Pipeline › Phase 3`, `› Help`

#### AC-7.2: Abort Confirmation
- **WHEN** user presses `esc` during active pipeline execution
- **THEN** show "Press [esc] again to abort mission" in the status bar
- **SHALL** require double-esc to actually abort (single esc = show warning, second esc = confirm)
- **SHALL** timeout the warning after 3 seconds (revert to normal status bar)

---

## Scope

### In Scope
- Colour palette update (styles.go)
- Sidebar layout with roles + grouping (views.go sidebar renderer)
- Pipeline chat formatting (views.go pipeline renderer)
- Feedback timing (app.go mission entry + pipeline dispatch)
- Header breadcrumb (views.go header renderer)
- Phase elapsed timer (app.go + views.go)
- Dossier reorganisation (views.go dossier renderer)
- Border reduction (styles.go)
- Abort confirmation (app.go key handler)

### Out of Scope
- New views or screens
- New key bindings beyond what exists
- Animation changes (boot sequence stays as-is)
- Backend/pipeline logic changes
- Agent content changes

---

## Dependencies
- No new Go dependencies required
- All changes are in `internal/tui/` (styles.go, views.go, app.go)
- No changes to agent files, doctrine, or Go packages outside TUI

---

## Risks

| Risk | Mitigation |
|------|-----------|
| Colour changes look bad on some terminals | Use ANSI 256-colour codes that degrade gracefully; test on iTerm2 + default Terminal.app |
| Border removal makes panels hard to distinguish | Rely on whitespace + indentation + alignment instead; test at various terminal widths |
| Elapsed timer adds noise | Only show seconds when > 5s elapsed; use dim colour |
| Sidebar roles truncate at narrow widths | Fall back to just names (no roles) below 30-char sidebar width |

---

## Rollback Note
All changes are purely visual (styles + rendering). No logic changes. Revert any file to restore previous appearance. The `theme` field in config could later support "classic" vs "refined" toggle.
