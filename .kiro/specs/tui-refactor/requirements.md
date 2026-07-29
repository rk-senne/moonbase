# Requirements: TUI Refactor

## Overview

The moonbase TUI (`internal/tui/`) works and is well-tested (91.2% coverage), but it
carries structural debt that will slow every future change: a 57-field `App` god
struct, ~53 string-literal key cases scattered across 5 files, a theme system that
mutates package-level globals, and a hand-maintained help view that can silently drift
from the real key bindings. This spec defines a refactor that pays down that debt and
future-proofs the TUI for new views, themes, and contributors — without changing what
the user sees or regressing test coverage.

This is a **structural refactor**, not a feature change. The guiding constraint: the
observable behaviour of the TUI stays the same. Every keystroke that worked before
works after.

## Motivation (Evidence)

Measured against the current codebase on 2026-07-29:

- `App` struct in `internal/tui/app.go` has **57 fields** — a single type holding
  dashboard, dossier, pipeline, comms, mission, docs, projects, terminal, file-browser,
  and system state. It changes for many unrelated reasons (a god object).
- **~53 string-literal key cases** (`case "n":`, `case "r":` …) across
  `update_dashboard.go` (26), `update_common.go` (9), `update_comms.go` (9),
  `update_dossier.go` (7), `update_mission.go` (2). Keys and their help text live in
  different files with no link between them.
- `internal/tui/helpers_theme.go` `cycleTheme()` **mutates package-level `var`
  colours** in `styles.go`. This is global mutable state shared across every render.
- `internal/tui/views_help.go` is a **hand-written string** that must be manually kept
  in sync with the key handlers. Nothing enforces that it matches reality.
- Stack: Bubbletea **v1.3.10** (v1 API), Bubbles v1.0.0, Lipgloss v1.1.1.
  `bubbles/key` and `bubbles/help` are available but unused; `bubbles/viewport` is
  already used in comms/docview.

## Functional Requirements

### FR-1: Centralised key bindings (single source of truth)

The TUI SHALL define all key bindings in one place using `bubbles/key.Binding`, and all
key handling and help text SHALL derive from that definition.

- **AC-1.1**: WHEN a key binding is defined THEN its keys AND its help text SHALL be
  declared together in a single `KeyMap` structure.
- **AC-1.2**: WHEN a key handler matches a keypress THEN it SHALL use
  `key.Matches(msg, keymap.X)` rather than a string literal (`case "x":`).
- **AC-1.3**: WHEN the help view is rendered THEN its content SHALL be generated from the
  `KeyMap` so that it cannot diverge from the active bindings.
- **AC-1.4**: WHEN the refactor is complete THEN no keypress `case "<literal>":` branch
  SHALL remain in any key-handling switch in the `update_*.go` files — including the
  sub-mode handlers (search, terminal, file browser, comms sub-modes, snippet/context
  input). Verified by a scoped grep/meta-test over key-handling switches (non-key
  string switches, if any, are out of scope for this AC and must be excluded from the
  gate pattern).

### FR-2: Theme system without global mutable state

Themes SHALL be represented as an immutable `Styles`/`Theme` value carried by the model,
not as mutated package-level globals.

- **AC-2.1**: WHEN a theme is active THEN its colours SHALL be read from a `Theme` value
  reachable through the model, not from package-level `var` colours.
- **AC-2.2**: WHEN the user cycles the theme (`T`) THEN the model SHALL swap to a
  different `Theme` value and re-render, with no mutation of shared global state.
- **AC-2.3**: WHEN a new theme is added THEN it SHALL be added by registering a new
  `Theme` value (extension), WITHOUT modifying a `switch` over theme names
  (Open-Closed Principle).
- **AC-2.4**: WHEN themes are exercised concurrently in tests (`-race`) THEN no data
  race SHALL be reported.
- **AC-2.5**: WHEN the theme refactor is complete THEN theme cycling SHALL be
  expressible as a pure value transformation on the model, and `cycleTheme` (or its
  replacement) SHALL NOT depend on mutating shared state via a pointer receiver.
  *(This closes the original "value-receiver mutation risk" gap: today `cycleTheme` is a
  `*App` method invoked from a value-receiver key handler, which works only by accident
  of Go taking the address of a local copy.)*

### FR-3: Decompose the `App` god struct into cohesive sub-models

The `App` struct SHALL be decomposed so that state which changes for the same reason is
grouped together, and each view's state is owned by a focused sub-model.

- **AC-3.1**: WHEN the refactor is complete THEN `App` SHALL delegate view state to
  sub-models (e.g. Dashboard, Comms, FileBrowser, Terminal, Pipeline) rather than
  holding all fields directly.
- **AC-3.2**: WHEN a sub-model handles a message THEN it SHALL expose an
  `Update`-style method and own its own fields (information hiding); `App` SHALL NOT
  reach into sub-model internals from key handlers.
- **AC-3.3**: WHEN the decomposition is complete THEN the top-level `App` SHALL hold
  no more than ~15 fields (orchestration/shared state only); each sub-model SHALL be
  independently constructible.
- **AC-3.4**: WHEN a new view is added in future THEN it SHALL be addable by writing a
  new sub-model and registering it, WITHOUT growing the `App` field list for that
  view's private state.

### FR-4: Contextual help footer

The TUI SHALL show, per view, a short footer of the keys relevant to the current
context, generated from the `KeyMap` via `bubbles/help`.

- **AC-4.1**: WHEN a view is active THEN a footer SHALL show the short help for the keys
  valid in that view.
- **AC-4.2**: WHEN `?` is pressed THEN the full help view SHALL be shown (long help),
  also generated from the `KeyMap`.
- **AC-4.3**: WHEN a key is not valid in the current view THEN it SHALL NOT appear in
  that view's footer.

### FR-5: Golden-file render tests (bounded)

The TUI SHALL have a small, focused set of golden-file tests for the primary views to
catch gross visual regressions, while behavioural testing remains on `Update`.

- **AC-5.1**: WHEN a primary view (dashboard, pipeline, dossier) is rendered at a fixed
  width/height with fixed state THEN its output SHALL match a committed golden file.
- **AC-5.2**: WHEN golden files are generated THEN they SHALL be regenerable via a
  documented `-update` flag on the test.
- **AC-5.3**: The golden-test suite SHALL remain small (primary views only) — GUI output
  is volatile, so behavioural coverage stays on `Update` per the Humble Object pattern.

## Non-Functional Requirements

### NFR-1: No observable behaviour change

- **AC-NFR-1.1**: WHEN any key that worked before the refactor is pressed THEN it SHALL
  produce the same result after the refactor.
- **AC-NFR-1.2**: WHEN the full test suite runs THEN all previously passing tests SHALL
  still pass (adapted only where they asserted on internal structure, not behaviour).

### NFR-2: No test-coverage regression

- **AC-NFR-2.1**: WHEN `go test ./internal/tui/ -cover` runs after the refactor THEN
  coverage SHALL be ≥ 91.2% (the pre-refactor baseline).

### NFR-3: Quality gates hold

- **AC-NFR-3.1**: `go build ./...`, `go vet ./...`, and `go test -race ./...` SHALL all
  pass.
- **AC-NFR-3.2**: `moonbase lint` SHALL continue to report all 14 agents valid
  (unaffected, but verified as a guard against collateral damage).

### NFR-4: Framework scope

- **AC-NFR-4.1**: The refactor SHALL remain on Bubbletea **v1** (`View() string`,
  `tea.KeyMsg`). Migration to Bubbletea v2 is explicitly OUT OF SCOPE and deferred to a
  separate spec.

### NFR-5: Incremental, reversible delivery

- **AC-NFR-5.1**: WHEN each phase (see tasks.md) is complete THEN the build and tests
  SHALL pass, so the refactor can be merged and paused at any phase boundary.

## Scope

### In Scope

- Centralised `KeyMap` with `bubbles/key`; migrate all key handlers to `key.Matches`.
- Help view + contextual footer generated from the `KeyMap` (`bubbles/help`).
- Theme system as immutable values carried by the model; theme registry for extension.
- Decomposition of `App` into cohesive sub-models (behaviour-preserving).
- A bounded set of golden-file render tests for primary views.
- Colour-profile degradation guard (graceful behaviour on 8/256-colour terminals).

### Out of Scope

- Bubbletea v2 migration (separate future spec — see NFR-4).
- New user-facing features, new views, or new agent behaviour.
- Mouse support (tracked as a future consideration, not built here).
- Persisting the selected theme to config (candidate follow-up, not required here).
- Internationalisation of labels.

## Future Considerations (anticipated gaps, not built here)

These are recorded so the design leaves the door open (Open-Closed) but they are not
delivered by this spec:

- **FC-1: Bubbletea v2 migration** — `View() tea.View`, `tea.KeyPressMsg`, mouse modes.
  The `KeyMap` + sub-model decomposition in this spec makes v2 migration far cheaper.
- **FC-2: Theme persistence** — save the chosen theme to `internal/config` so it
  survives restarts. The `Theme` value + registry makes this a small addition.
- **FC-3: Mouse support** — Bubbletea supports it; sub-models give a clean place to
  handle mouse messages per-view.
- **FC-4: Very-small-terminal layout** — explicit minimum-size handling and graceful
  degradation below (e.g.) 80×24.
- **FC-5: Accessibility** — honour `NO_COLOR`, provide a high-contrast theme, avoid
  colour as the only signal (already partly satisfied by the weight-first palette).
- **FC-6: Render performance** — viewport virtualisation for very large pipeline
  outputs / intel logs if they grow unbounded.

## Traceability

Each requirement maps to a design decision in `design.md` and one or more tasks in
`tasks.md`. Every task references the AC-IDs it satisfies.
