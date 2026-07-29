# Tasks: TUI Refactor

Ordered, dependency-sequenced. Each task is independently shippable: the build and tests
must be green at every task boundary (NFR-5). Phases group related tasks; the risky
decomposition (Phase 4) runs only after the golden-test safety net (Phase 3) exists.

Legend: `[ ]` pending · `[x]` complete. Each task lists the AC-IDs it satisfies and the
tests it must add/adapt.

---

## Phase 0: Baseline & Safety Net

### Task 0.1: Capture the coverage & behaviour baseline
- **Satisfies:** NFR-2.1 (establishes the 91.2% floor)
- **Files:** none (measurement only)
- **Action:** Record `go test ./internal/tui/ -cover` (baseline 91.2%). Save the current
  `moonbase` render of dashboard/pipeline/dossier for manual before/after comparison.
- **Test:** `go test ./internal/tui/ -cover` runs clean.
- **Done when:** baseline number is recorded in the PR description.

---

## Phase 1: Centralised KeyMap (D-1)

### Task 1.1: Define the `KeyMap`
- **Satisfies:** AC-1.1
- **Files:** `internal/tui/keys.go` (new)
- **Action:** Create `KeyMap` struct with one `key.Binding` per logical action, plus
  `DefaultKeyMap()`. Co-locate keys and help via `key.WithKeys` + `key.WithHelp`. Cover
  every key currently handled (audit the ~53 cases first).
- **Test:** `TestDefaultKeyMap_AllActionsHaveKeysAndHelp` — every binding has ≥1 key and
  non-empty help.
- **Done when:** builds; test passes.

### Task 1.2: Add `ShortHelp`/`FullHelp` to `KeyMap`
- **Satisfies:** AC-1.3 (enables generated help)
- **Files:** `internal/tui/keys.go`
- **Action:** Implement the `help.KeyMap` interface (`ShortHelp() []key.Binding`,
  `FullHelp() [][]key.Binding`), grouped by category (Navigation, Missions, Views,
  Tools, Comms, System) to mirror today's manual layout.
- **Test:** `TestKeyMap_FullHelpCoversAllBindings` — every binding appears in `FullHelp`.
- **Done when:** test passes.

### Task 1.3: Hold `KeyMap` on `App`; migrate dashboard handlers
- **Satisfies:** AC-1.2 (partial), NFR-1.1
- **Files:** `internal/tui/app.go`, `internal/tui/update_dashboard.go`
- **Action:** Add `keys KeyMap` to `App` (init in `NewApp`). Replace the 26 `case "x":`
  branches in `update_dashboard.go` with `key.Matches(msg, a.keys.X)`.
- **Test:** Adapt existing dashboard `Update` tests to send `tea.KeyMsg` and assert
  identical behaviour. All existing tests pass unchanged in intent.
- **Done when:** dashboard keys all route via `key.Matches`; tests green.

### Task 1.4: Migrate remaining handlers (common, comms, dossier, mission)
- **Satisfies:** AC-1.2 (complete), AC-1.4
- **Files:** `update_common.go`, `update_comms.go`, `update_dossier.go`, `update_mission.go`
- **Action:** Replace all remaining keypress `case "x":` branches with `key.Matches`.
  Explicitly cover every sub-mode handler, not just the top-level ones:
  - `update_common.go`: search handler, embedded-terminal handler, and file-browser
    handler (the file browser alone has ~8 key branches).
  - `update_comms.go`: comms handler plus its sub-modes (snippet picker, context-file
    input).
  - `update_dossier.go`: dossier, docs, and projects handlers.
  - `update_mission.go`: mission input handler.
- **Test:** Adapt existing tests; add `TestNoStringLiteralKeyCases` meta-test (or a
  documented, scoped grep gate) asserting zero keypress `case "…":` branches remain in
  key-handling switches (excluding any non-key string switches per AC-1.4).
- **Done when:** the scoped gate returns zero; tests green.

---

## Phase 2: Generated Help & Contextual Footer (D-1, D-2)

### Task 2.1: Regenerate the full help view from `KeyMap`
- **Satisfies:** AC-1.3, AC-4.2
- **Files:** `internal/tui/views_help.go`, `internal/tui/app.go`
- **Action:** Replace the hand-written help string with `help.Model` rendering
  `FullHelp()`. Keep the KND-flavoured header/footer prose (that's brand, not key data).
- **Test:** `TestHelpView_ContainsEveryBinding` — rendered help includes each binding's
  help text.
- **Done when:** help view is generated; manual visual check matches old layout closely.

### Task 2.2: Add per-view contextual footer
- **Satisfies:** AC-4.1, AC-4.3
- **Files:** `internal/tui/app.go`, view render funcs, `internal/tui/keys.go`
- **Action:** Add a `keysFor(view View) []key.Binding` helper. Render a `help.Model`
  short-help footer in the status bar for the active view.
- **Test:** `TestFooterShowsOnlyActiveViewKeys` — a key not valid in a view is absent
  from that view's short help.
- **Done when:** footer renders per view; test passes.

---

## Phase 3: Golden-File Safety Net (D-4)

### Task 3.1: Golden-test harness
- **Satisfies:** AC-5.2
- **Files:** `internal/tui/golden_test.go` (new), `internal/tui/testdata/` (new)
- **Action:** Add a helper that renders a view to a string at fixed 100×30 with a fixed
  colour profile, and compares to `testdata/<name>.golden`; support a `-update` flag.
  Pin the colour profile deterministically so goldens are stable across machines/CI —
  e.g. force a known `termenv`/`lipgloss` profile (TrueColor) and fixed
  dark-background assumption in test setup, rather than inheriting the ambient `TERM`.
- **Test:** the harness itself (self-check on a trivial view).
- **Done when:** `-update` writes goldens; re-run without `-update` passes.

### Task 3.2: Golden snapshots for primary views
- **Satisfies:** AC-5.1, AC-5.3, protects NFR-1.1
- **Files:** `internal/tui/golden_test.go`, `internal/tui/testdata/*.golden`
- **Action:** Golden tests for dashboard, pipeline, dossier with deterministic fixed
  state. Keep it to these three (bounded — Humble Object / Clean Coder).
- **Test:** the three golden tests pass; committed goldens reviewed by eye once.
- **Done when:** green; goldens committed.

---

## Phase 4: Theme System as Values (D-3, D-6)

### Task 4.1: Introduce `Theme`, `Styles`, `NewStyles`, registry
- **Satisfies:** AC-2.1, AC-2.3
- **Files:** `internal/tui/theme.go` (new)
- **Action:** Define immutable `Theme`, `Styles`, pure `NewStyles(Theme) Styles`, and
  `themeRegistry` containing the four existing themes (moonbase/treehouse/classified/nerv).
- **Test:** `TestNewStyles_Pure` (same theme → equal styles); `TestThemeRegistry_HasFour`.
- **Done when:** green; no render sites changed yet.

### Task 4.2: Carry theme/styles on `App`; migrate render sites
- **Satisfies:** AC-2.1, AC-2.2, AC-2.5
- **Files:** `internal/tui/app.go`, all `views_*.go` render funcs, `helpers_theme.go`
- **Action:** Add `theme Theme` + `styles Styles` to `App`. Replace reads of package-level
  `Style*`/`Color*` vars with `a.styles.*`. Rewrite `cycleTheme()` to advance the registry
  index and recompute `styles` — no global mutation. Ensure theme cycling is a pure value
  transformation on the model (closes the value-receiver mutation gap, AC-2.5): the
  handler should not rely on a `*App` side-effect to change shared state.
- **Test:** Adapt existing theme test(s); `TestCycleTheme_AdvancesAndWraps`;
  `TestCycleTheme_PureValueTransform` (given a model, cycling returns a model with a
  different theme and does not touch package-level state).
- **Done when:** rendering reads from `a.styles`; tests green.

### Task 4.3: Delete mutable globals; add colour-degradation guard
- **Satisfies:** AC-2.4, FR-2, D-6/FC-5
- **Files:** `internal/tui/styles.go`, `internal/tui/theme.go`
- **Action:** Remove the package-level mutable `Color*` vars once no references remain.
  Honour `NO_COLOR` in `NewStyles`. Keep `Badge*` consts.
- **Test:** `go test -race ./internal/tui/` over a theme-cycling render loop → no races
  (AC-2.4). `TestNoColor_DegradesPalette`.
- **Done when:** globals gone; `-race` clean.

---

## Phase 5: Decompose the God Struct (D-5)

Runs last. Each extraction is its own task, its own commit, its own green build. Golden
tests (Phase 3) must stay green after every extraction (proves no visual regression).

### Task 5.1: Define `AppContext` (read-only shared services)
- **Satisfies:** AC-3.2 (ISP)
- **Files:** `internal/tui/app.go` (or `context.go` new)
- **Action:** Define a small read-only struct passed to sub-model `Update`/`View` giving
  access to shared services (registry, backend, width/height, styles, keys) — not the
  whole `App`.
- **Test:** compile-level; used by later tasks.
- **Done when:** builds.

### Task 5.2: Extract `TerminalModel`
- **Satisfies:** AC-3.1, AC-3.2, NFR-2.1
- **Files:** `internal/tui/model_terminal.go` (new), `app.go`, `helpers_terminal.go`
- **Action:** Move `termInput`, `termOutput`, `termActive`, `cwd` and their handlers into
  `TerminalModel` with `Update`/`View`. `App` delegates.
- **Test:** move/author terminal `Update` tests against `TerminalModel` directly.
- **Done when:** golden tests still pass; coverage ≥ baseline.

### Task 5.3: Extract `DashboardModel`
- **Satisfies:** AC-3.1, AC-3.2
- **Files:** `internal/tui/model_dashboard.go` (new), `app.go`, `update_dashboard.go`
- **Action:** Move cursor/selection/roster-view state + handlers into `DashboardModel`.
- **Test:** dashboard `Update` tests target `DashboardModel`.
- **Done when:** golden tests pass; coverage ≥ baseline.

### Task 5.4: Extract `PipelineModel`
- **Satisfies:** AC-3.1, AC-3.2
- **Files:** `internal/tui/model_pipeline.go` (new), `app.go`, `update_pipeline.go`,
  `pipeline_exec.go`
- **Action:** Move `pipelineState`, `pipelineChat`, `pipelineOutput`, `pipelineRunning`,
  abort state, and handlers into `PipelineModel`.
- **Test:** pipeline `Update` tests target `PipelineModel`; risk-gate routing unchanged.
- **Done when:** golden tests pass; coverage ≥ baseline.

### Task 5.5: Fold existing sub-states in & slim `App`
- **Satisfies:** AC-3.3, AC-3.4
- **Files:** `internal/tui/app.go`
- **Action:** Route the already-extracted `CommsState`, `DocsState`, `ProjectsState`,
  `FileBrowser` through the same sub-model pattern. Confirm `App` holds ≤ ~15 fields
  (orchestration/shared only).
- **Test:** `TestApp_FieldCountBounded` (reflection or a documented count) asserts the
  top-level field budget; full suite green.
- **Done when:** `App` ≤ ~15 fields; coverage ≥ baseline.

---

## Phase 6: Verification & Close-out

### Task 6.1: Full gate + coverage guard
- **Satisfies:** NFR-2.1, NFR-3.1, NFR-3.2
- **Action:** Run `go build ./...`, `go vet ./...`, `go test -race ./...`,
  `go test ./internal/tui/ -cover` (≥ 91.2%), `moonbase lint`.
- **Done when:** all green; coverage ≥ baseline.

### Task 6.2: Behaviour parity check
- **Satisfies:** NFR-1.1
- **Action:** Manually exercise every key from the help view in a running TUI; confirm
  identical behaviour to the Phase-0 baseline. Diff Phase-0 vs current renders.
- **Done when:** no behavioural differences observed.

### Task 6.3: CHANGELOG + docs
- **Satisfies:** quality-gates docs requirement
- **Files:** `CHANGELOG.md`, `docs/architecture.md` (if it describes the TUI)
- **Action:** Record the refactor under `[Unreleased]` (internal refactor → note only if
  user-visible footer/help changes ship). Note the new component structure.
- **Done when:** CHANGELOG updated.

---

## Checkpoint

- [ ] Phase 1: all keys via `KeyMap`/`key.Matches`; zero string-literal key cases (AC-1.*)
- [ ] Phase 2: help + footer generated from `KeyMap` (AC-4.*)
- [ ] Phase 3: bounded golden tests for primary views (AC-5.*)
- [ ] Phase 4: themes are values; no mutable globals; `-race` clean (AC-2.*)
- [ ] Phase 5: `App` ≤ ~15 fields; sub-models own their state (AC-3.*)
- [ ] Phase 6: build + vet + `-race` + lint pass; coverage ≥ 91.2%; behaviour parity

## Sequencing Rationale

- **KeyMap first** (Phase 1) — lowest risk, mechanical, and unblocks Phases 2.
- **Help/footer** (Phase 2) — pure consumers of the KeyMap; cheap once it exists.
- **Golden net before decomposition** (Phase 3 before 4/5) — the snapshots are what let
  us prove Phases 4–5 changed no pixels.
- **Themes** (Phase 4) before decomposition so sub-models receive `Styles` by value from
  day one (they never see the old globals).
- **Decomposition last** (Phase 5) — highest blast radius; each extraction is isolated
  and guarded by golden + `Update` tests.
