# Requirements — Bubble Tea v2 Migration

## Overview

Migrate the moonbase TUI from the v1 Charm stack to v2:

| Library | From (current) | To |
|---|---|---|
| Bubble Tea | `github.com/charmbracelet/bubbletea` v1.3.10 | `charm.land/bubbletea/v2` |
| Lip Gloss | `github.com/charmbracelet/lipgloss` v1 | `charm.land/lipgloss/v2` |
| Bubbles | `github.com/charmbracelet/bubbles` v1 | `charm.land/bubbles/v2` |
| termenv | used for golden color-profile pinning | replaced by `tea.WithColorProfile` |

This is a **breaking, mechanical-but-wide** migration. It touches nearly every file under `internal/tui/` plus `cmd/moonbase/root.go`. It delivers **no user-visible feature change** — the win is staying on the supported major version (better key/mouse handling, deterministic styles, the "cursed" renderer) and unblocking future v2-only components.

## Goals

- G-1: Compile and run on the v2 stack with identical behavior.
- G-2: Preserve the golden-tested render output (see AC-8 for the regeneration policy).
- G-3: Keep the App aggregate-root architecture, KeyMap single-source-of-truth, and sub-model decomposition intact.
- G-4: Keep every quality gate green (build, vet, `govulncheck`, `-race`, `moonbase lint`, coverage ≥ ~91%).

## Non-Goals

- No new features, no view redesigns, no keybinding changes.
- No adoption of new v2-only widgets beyond what a like-for-like migration needs.
- Not switching the golden-test philosophy — only adapting how color is pinned.

## Requirements

### R-1 — Dependencies & import paths
- **1.1** The module SHALL depend on `charm.land/bubbletea/v2`, `charm.land/lipgloss/v2`, and `charm.land/bubbles/v2`, with the v1 modules removed from `go.mod`/`go.sum`.
- **1.2** All import statements SHALL use the v2 vanity paths (alias `tea` preserved for bubbletea).
- **1.3** `go mod tidy` SHALL leave a clean module graph; `govulncheck ./...` SHALL report 0 affecting vulnerabilities.

### R-2 — Model interface (declarative View)
- **2.1** `App.View()` SHALL return `tea.View` (not `string`).
- **2.2** Alt-screen and mouse enablement SHALL be expressed as `tea.View` fields (`AltScreen = true`, `MouseMode = tea.MouseModeCellMotion`) rather than `NewProgram` options.
- **2.3** `cmd/moonbase/root.go` SHALL construct the program as `tea.NewProgram(tui.NewApp())` with the former `WithAltScreen`/`WithMouseCellMotion` options removed.
- **2.4** All internal render helpers that currently build the frame as a `string` MAY keep returning `string`; only the top-level `View()` returns `tea.View` (wrapping the composed string).

### R-3 — Keyboard events
- **3.1** Every `case tea.KeyMsg:` SHALL become `case tea.KeyPressMsg:` (App.Update and all sub-model `Update(msg, ctx)` signatures).
- **3.2** Any use of `msg.Type`/`msg.Runes`/`msg.Alt` SHALL be migrated to the v2 fields (`msg.Code`, etc.).
- **3.3** Any literal `" "` (space) key match SHALL become `"space"`.
- **3.4** `bubbles/key` bindings (the central `KeyMap`) SHALL be migrated to `charm.land/bubbles/v2/key`; `key.Matches`, `key.NewBinding`, and generated help SHALL continue to work.

### R-4 — Mouse events
- **4.1** The `handleMouse` dispatch SHALL be migrated from the single `tea.MouseMsg{Action,Button}` struct to the v2 typed messages (`tea.MouseWheelMsg`, `tea.MouseClickMsg`).
- **4.2** Button constants SHALL be renamed to v2 (`MouseButtonLeft`→`MouseLeft`, `MouseButtonWheelUp/Down`→`MouseWheelUp/Down`).
- **4.3** Wheel scrolling and click-to-select behavior SHALL be preserved exactly (all `update_mouse_test.go` cases pass after adapting the synthetic message construction to v2 types).

### R-5 — Lip Gloss v2
- **5.1** `internal/tui/theme.go` (`NewStyles`, palettes) SHALL compile against lipgloss v2, preserving the visible palette.
- **5.2** Color construction (`lipgloss.Color`, adaptive colors) SHALL use the v2 API; the `NO_COLOR` degraded path SHALL be preserved.
- **5.3** Layout helpers (`JoinVertical`/`JoinHorizontal`/`Width`/`Style.Render`) SHALL produce the same layout.

### R-6 — Bubbles components
- **6.1** `viewport` (comms, docs): field access like `.Width` SHALL move to `SetWidth()/Width()`; scroll methods used by the mouse handler (`LineUp/LineDown`) and `YOffset` reads in tests SHALL be migrated to their v2 equivalents.
- **6.2** `textinput` (mission, search, comms, ctxfile): `.Width =` assignments SHALL become `SetWidth()`; `Value/SetValue/Focus/Blur/Reset/Update` usage SHALL be migrated.
- **6.3** `spinner`: `Tick`/`TickMsg` usage SHALL be migrated.
- **6.4** `help`: `help.Model` + `ShortHelpView/FullHelpView` (contextual footer + manual) SHALL be migrated and continue to render from the KeyMap.

### R-7 — Tests & goldens
- **7.1** Golden determinism SHALL be pinned via `tea.WithColorProfile` (and/or the lipgloss v2 equivalent) instead of `termenv.TrueColor`.
- **7.2** Because v2's renderer may legitimately change byte output, golden files MAY be regenerated **once**, but the regeneration SHALL be reviewed as a visual diff and justified in the PR; unexplained content/layout changes are a defect, not an accepted regen.
- **7.3** All non-golden tests (Update logic, mouse, threat, field-count guard, agent parser) SHALL pass unchanged in intent (only message-type/APIs adapted).

### R-8 — Quality gates & CI
- **8.1** `go build ./...`, `go vet ./...`, `CI=true go test -race ./... -timeout 300s`, and `go run ./cmd/moonbase lint` SHALL all pass.
- **8.2** The GitHub Actions CI + Release workflows SHALL pass on the migration branch before merge.
- **8.3** tui package coverage SHALL remain within ~1% of the pre-migration figure (~91.5%).

### R-9 — Safety / rollback
- **9.1** The migration SHALL be done on a dedicated branch and merged via PR (not pushed straight to `main`), given its blast radius.
- **9.2** A clear rollback path SHALL exist (revert the merge); the commit SHALL be self-contained (deps + code + goldens together) so revert is clean.

## Risks

- **Wide blast radius**: ~35+ files in `internal/tui/` reference the affected APIs. Mitigate by leaning on the compiler (fix until it builds) and the golden/behavioral test net.
- **Golden drift**: v2's renderer/color handling may shift bytes. Mitigate with `WithColorProfile` pinning + reviewed one-time regen (AC-7.2).
- **Vanity module path** (`charm.land/...`): confirm availability/proxy resolution in CI early (a smoke `go get` before committing to the full migration).
- **Companion-version lockstep**: bubbletea/lipgloss/bubbles v2 must be mutually compatible versions — pin exact versions together.
