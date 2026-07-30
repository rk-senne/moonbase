# Tasks — Bubble Tea v2 Migration

Execute on branch `feat/bubbletea-v2`. Each task should end type-checking (or with a
documented reason it can't until a later task). Do not merge to `main` until Phase 6
passes (R-9).

## Phase 0 — De-risk (do first)

- [ ] 0.1 Smoke-test the vanity module paths resolve in this environment **and** CI:
  `go get charm.land/bubbletea/v2@latest charm.land/lipgloss/v2@latest charm.land/bubbles/v2@latest`
  in a scratch clone. Record the exact mutually-compatible versions. (R-1, D-7)
- [ ] 0.2 Confirm the v2 color-profile pinning API for tests (bubbletea `WithColorProfile`
  and the lipgloss v2 profile setter) and update design D-5 with the exact calls. (R-7.1)
- [ ] 0.3 Create the branch and open a draft PR early so CI runs on every push. (R-9.1)

## Phase 1 — Dependencies & imports

- [ ] 1.1 Add the three v2 modules to `go.mod` at the pinned versions; remove the v1
  modules and `termenv` (if now unused). (R-1.1)
- [ ] 1.2 Search-and-replace import paths across the repo:
  `github.com/charmbracelet/bubbletea`→`charm.land/bubbletea/v2`,
  `…/lipgloss`→`charm.land/lipgloss/v2`, `…/bubbles`→`charm.land/bubbles/v2`. Keep the
  `tea` alias. (R-1.2)
- [ ] 1.3 `go mod tidy`; expect compile errors (addressed next) but a clean module graph. (R-1.3)

## Phase 2 — Program & Model interface

- [ ] 2.1 `cmd/moonbase/root.go`: `tea.NewProgram(tui.NewApp())` — drop `WithAltScreen`
  and `WithMouseCellMotion`. (R-2.3)
- [ ] 2.2 Extract `func (a App) renderFrame() string` = today's `View()` body. (D-2)
- [ ] 2.3 Add `func (a App) View() tea.View` wrapping `renderFrame()` and setting
  `AltScreen = true`, `MouseMode = tea.MouseModeCellMotion`. (R-2.1, R-2.2)

## Phase 3 — Keyboard

- [ ] 3.1 `App.Update`: `case tea.KeyMsg:` → `case tea.KeyPressMsg:`. (R-3.1)
- [ ] 3.2 Change sub-model `Update` parameter types to `tea.KeyPressMsg`
  (`model_dashboard.go`, `model_terminal.go`, `model_pipeline.go`) and all
  `handle*Keys(msg tea.KeyPressMsg)` handlers. (R-3.1)
- [ ] 3.3 Migrate `keys.go` to `charm.land/bubbles/v2/key`; verify `key.Matches`,
  `key.NewBinding`, and generated help/footer compile and behave. (R-3.4)
- [ ] 3.4 Audit & fix `msg.Type`/`msg.Runes`/`msg.Alt` → v2 (`msg.Code`, …) and any
  `" "` space match → `"space"`. (R-3.2, R-3.3)

## Phase 4 — Mouse

- [ ] 4.1 Restructure `handleMouse` (`update_mouse.go`) to dispatch on the v2 typed
  messages (`tea.MouseWheelMsg`, `tea.MouseClickMsg`); keep `mouseScroll`/`mouseClick`
  logic and the sidebar hit-map. (R-4.1)
- [ ] 4.2 Update `app.go` `case tea.MouseMsg:` → the v2 message case(s). (R-4.1)
- [ ] 4.3 Rename button constants (`MouseButtonLeft`→`MouseLeft`,
  `MouseButtonWheelUp/Down`→`MouseWheelUp/Down`). (R-4.2)
- [ ] 4.4 Update `update_mouse_test.go` synthetic events to the v2 message structs; all
  mouse tests pass. (R-4.3)

## Phase 5 — Lip Gloss v2 & Bubbles components

- [ ] 5.1 `theme.go`: migrate `NewStyles`, color construction, and the `NO_COLOR`
  degraded path to lipgloss v2. (R-5.1, R-5.2)
- [ ] 5.2 Fix remaining lipgloss render sites (`views_*.go`, `comms.go`, `docview.go`,
  `protocol.go`, `helpers_*.go`) — `JoinVertical/Horizontal`, `Width`, `Render`. (R-5.3)
- [ ] 5.3 `viewport` (comms/docs): `.Width` reads → `Width()`, writes → `SetWidth()`;
  verify/adapt `LineUp/LineDown`/`YOffset` used by the mouse handler + tests. (R-6.1)
- [ ] 5.4 `textinput`: `.Width =` → `SetWidth()` in `NewApp` and `helpers_comms.go`;
  verify `Value/SetValue/Focus/Blur/Reset/Update`. (R-6.2)
- [ ] 5.5 `spinner` (`Tick`/`TickMsg`) and `help` (`ShortHelpView/FullHelpView`) migrated;
  contextual footer + manual still render from the KeyMap. (R-6.3, R-6.4)
- [ ] 5.6 Package compiles: `go build ./...` clean.

## Phase 6 — Tests, goldens & gates

- [ ] 6.1 `golden_test.go`: replace `termenv` color pinning with the v2 profile API
  (D-5); render via `renderFrame()`. (R-7.1)
- [ ] 6.2 Run `TestGolden_*`. If they fail on byte diffs, **review each as a visual diff**;
  if the change is only the expected v2 renderer normalization (no layout/content change),
  regenerate once with `-update-golden` and note the justification in the PR. Any
  layout/content change is a defect to fix, not regen. (R-7.2)
- [ ] 6.3 Fix any remaining `*_test.go` that construct v1 messages or use changed APIs;
  field-count guard, threat, agent-parser tests pass unchanged in intent. (R-7.3)
- [ ] 6.4 Full gates: `go vet ./...`; `govulncheck ./...` (0); `CI=true go test -race
  ./... -count=1 -timeout 300s` (18 pkgs); `go run ./cmd/moonbase lint` (14). (R-8.1)
- [ ] 6.5 Coverage within ~1% of baseline (~91.5%). (R-8.3)
- [ ] 6.6 Manual smoke run: launch the TUI, exercise nav, comms scroll (wheel + keys),
  click-to-select, theme cycle, terminal/browser modes, resize. (G-1)
- [ ] 6.7 Confirm gh CI + Release workflows green on the branch; merge PR; delete branch. (R-8.2, R-9)

## Phase 7 — Changelog & release

- [ ] 7.1 CHANGELOG under `[Unreleased] → Changed`: "migrated the TUI to the Bubble Tea /
  Lip Gloss / Bubbles v2 stack (no behavior change)." (Internal-leaning, but a dependency
  major bump is worth noting.)
- [ ] 7.2 Cut the next release (MINOR bump — dependency/runtime change) once merged and CI
  is green.

## Out of scope (explicitly)

- New v2-only widgets, the new cursor/`View` features beyond alt-screen + mouse mode,
  view redesigns, or keybinding changes. Track separately if desired.
