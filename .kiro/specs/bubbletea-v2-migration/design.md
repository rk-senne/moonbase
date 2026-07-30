# Design — Bubble Tea v2 Migration

## Strategy

A **big-bang migration on a dedicated branch**, compiler-driven. The `tea.Model`
interface change (`View() string` → `View() tea.View`) cannot be done piecemeal, and
Go lets both major versions coexist only across *different* modules — not within one
package — so `internal/tui` migrates as a unit. We lean on the compiler ("fix until it
builds") and the golden + behavioral test net to prove behavior is preserved.

Recommended order (each step should at least type-check the affected files):
1. **Deps & imports** — bump `go.mod`, run search-and-replace on import paths.
2. **Program construction** — simplify `root.go`.
3. **`View()` signature** — the single top-level change (see D-2).
4. **Keyboard** — `KeyMsg`→`KeyPressMsg`, `key` package.
5. **Mouse** — typed messages.
6. **Lip Gloss v2** — `theme.go` + render sites.
7. **Bubbles components** — viewport/textinput/spinner/help width & key changes.
8. **Tests & goldens** — color pinning, one-time reviewed regen.
9. **Gates + CI** — full verification, then PR.

## Design Decisions

### D-1 — Branch + PR, not direct-to-main
Given the blast radius (~35 files), this goes on `feat/bubbletea-v2` and merges via PR after CI is green (overrides the usual direct-push convenience for this one change; ref R-9).

### D-2 — Bound the `View()` churn with a `renderFrame()` seam
Only `App.View()` changes signature. Introduce an unexported `func (a App) renderFrame() string` that contains today's `View()` body (the `switch a.view { ... }` returning the composed string). The new `View()` becomes a thin wrapper:

```go
func (a App) renderFrame() string { /* old View() body */ }

func (a App) View() tea.View {
    v := tea.NewView(a.renderFrame())
    v.AltScreen = true
    v.MouseMode = tea.MouseModeCellMotion
    return v
}
```

**Payoff:** every `renderDashboard/renderComms/...` helper keeps returning `string`
unchanged, and the golden harness calls `renderFrame()` (a string) — so the goldens do
**not** depend on how `tea.View` renders. This isolates the interface change to two
methods.

### D-3 — Keyboard
`App.Update`'s `case tea.KeyMsg:` → `case tea.KeyPressMsg:`. The sub-model method
signatures (`DashboardModel.Update(msg tea.KeyMsg, ctx AppContext)`,
`TerminalModel.Update`, etc.) change their parameter type to `tea.KeyPressMsg`. The
central `KeyMap` in `keys.go` migrates to `charm.land/bubbles/v2/key`; `key.Matches`,
`key.NewBinding`, and the generated help/footer keep working. Audit for `msg.Type`,
`msg.Runes`, `msg.Alt`, and the literal `" "` space match (→ `"space"`).

### D-4 — Mouse (restructure `handleMouse`)
v2 splits the one `tea.MouseMsg{Action,Button}` into typed messages. `handleMouse` moves
from a `switch msg.Button` to dispatch by message type, preserving `mouseScroll`/
`mouseClick`:

```go
case tea.MouseWheelMsg:  // Button: MouseWheelUp / MouseWheelDown
case tea.MouseClickMsg:  // Button: MouseLeft
```

In `app.go`, the single `case tea.MouseMsg:` becomes the relevant typed cases (or one
handler that type-switches). `update_mouse_test.go` synthetic events change from
`tea.MouseMsg{Action:…,Button:…}` to the v2 message structs. Button constants:
`MouseButtonLeft`→`MouseLeft`, `MouseButtonWheelUp/Down`→`MouseWheelUp/Down`.

### D-5 — Golden color pinning
Replace `lipgloss.SetColorProfile(termenv.TrueColor)` in `golden_test.go` with the v2
mechanism (`tea.WithColorProfile(...)` for programs, and/or the lipgloss v2 profile API
for direct `Render` calls in tests). Because the golden harness renders via
`renderFrame()` (D-2) rather than a running program, the harness pins the lipgloss v2
color profile directly before calling `renderFrame()`. Confirm the exact v2 profile API
during Task 1 and record it here.

### D-6 — Bubbles width/height & scroll API
Mechanical field→method changes:
- `viewport`: reads `vp.Width` → `vp.Width()`; writes → `vp.SetWidth(n)`; verify
  `LineUp/LineDown` and `YOffset` names (used by `update_mouse.go` and its tests) against
  the v2 viewport; adapt if renamed.
- `textinput`: `ti.Width = n` → `ti.SetWidth(n)` (in `NewApp` and `helpers_comms.go`).
- `spinner`, `help`: adapt `Tick`/`TickMsg` and `ShortHelpView/FullHelpView` per guide.

### D-7 — Version & path smoke test first
Before touching code, do a throwaway `go get charm.land/bubbletea/v2@latest` (+ lipgloss/
bubbles v2) to confirm the vanity path resolves through the module proxy in this
environment and in CI. Pin the three v2 modules to mutually-compatible exact versions.

### D-8 — Optional parallel execution
Areas 4–7 (mouse, lipgloss/theme, bubbles components) are largely independent once the
imports/`View()`/key changes land. They can be delegated to separate operatives, but
they converge at "package compiles + goldens pass," so a single integrator (or sequential
execution) is safer than parallel edits to shared files like `app.go`.

## File Impact Map

| Area | Files | Requirements |
|---|---|---|
| Deps | `go.mod`, `go.sum` | R-1 |
| Program | `cmd/moonbase/root.go` | R-2.3 |
| Model/View + dispatch | `internal/tui/app.go` | R-2, R-3.1, R-4 |
| Sub-model Updates | `model_dashboard.go`, `model_terminal.go`, `model_pipeline.go` | R-3.1 |
| Key/handlers | `update_common.go`, `update_comms.go`, `update_dossier.go`, `update_mission.go`, `update_dashboard.go`, `keys.go` | R-3 |
| Mouse | `update_mouse.go`, `update_mouse_test.go` | R-4 |
| Styling | `theme.go`, `views_*.go`, `comms.go`, `docview.go`, `protocol.go`, `helpers_*.go` | R-5 |
| Components | `comms.go`, `docview.go`, `helpers_comms.go`, `views_help.go`, spinner sites | R-6 |
| Tests/goldens | `golden_test.go`, `testdata/*.golden`, misc `*_test.go` | R-7 |

## Test Alignment

| Requirement | Verified by |
|---|---|
| R-1 deps | `go build ./...`, `go mod tidy`, `govulncheck` |
| R-2 View/declarative | build + `TestGolden_*` (via `renderFrame`) + manual smoke run |
| R-3 keyboard | all `Update` key tests (state/keys/help_gen), `moonbase lint` unaffected |
| R-4 mouse | `update_mouse_test.go` (wheel per view, click-select, offset check) |
| R-5 lipgloss | `TestGolden_*` (byte diff reviewed once), `NO_COLOR` degraded test |
| R-6 components | comms/docs/mission/search Update tests + goldens |
| R-7 goldens | reviewed one-time regen; determinism across repeated runs |
| R-8 gates | build/vet/govulncheck/`-race`/lint + gh CI + Release |
| R-9 rollback | single squashed PR revertability |

## Estimated Effort

Medium-large. Mechanical breadth (imports, KeyMsg→KeyPressMsg, width setters) is fast with
search-and-replace; the judgment work is D-2 (`View()` seam), D-5 (golden color pinning),
and the one-time golden regen review. Budget for a full verification loop on both CI
runners (ubuntu + macos).
