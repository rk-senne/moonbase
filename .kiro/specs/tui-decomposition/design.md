# internal/tui Decomposition — Design

## Problem

`internal/tui` is 8,005 production lines across 71 files — roughly 38% of all
production code in one package. Per CCP (gather what changes for the same reason,
separate what changes for different reasons), a package this size has several
independent reasons to change: visual design, key bindings, view rendering, event
handling, and pipeline streaming all live together.

The symptoms are the ones *A Philosophy of Software Design* names directly:

- **Change amplification** — a rendering tweak sits in the same package as event
  handling and multiplexer integration.
- **Cognitive load** — 71 files must be treated as one unit of understanding.

## The constraint that blocks a naive split

All 25 view functions are methods on `App`:

```go
func (a App) renderHeader(breadcrumb string) string
func (a App) renderSidebar(width int, maxH int) string
func (a App) renderMainPanel(width int, maxH int) string
// ...22 more
```

Moving `views_*.go` into `internal/tui/views` would therefore require `views` to
import `App`, while `App.View()` must call into `views` — a circular import, which
Go forbids outright.

This is the standard Bubble Tea monolith shape and the reason such packages stay
large. It is not an accident to be tidied away; it needs an actual design change.

## Measured extractability

Files with no reference to `App` or `Model` total **1,923 lines (24%)**. Those can
move without touching the circular dependency. The remaining ~6,000 lines are
`App`-coupled and need the interface change described in Stage 3.

## Staged plan

Each stage is independently shippable and independently revertible. Do not attempt
them as one change — the migration patterns in `skills/incremental-migration.md`
apply to internal refactors just as much as to service extraction.

### Stage 1 — pure algorithms out (DONE)

`fuzzy.go` → `internal/fuzzy`, exported as `fuzzy.Match`. A pure scoring function
with no presentation concern, now at 100% coverage in its own package. Reduced
`internal/tui` by 88 lines and proved the pattern.

### Stage 2 — presentation primitives out (NEXT)

Move to `internal/tui/chrome`:

| File | Lines |
|---|---|
| `theme.go` | 179 |
| `portraits.go` | 190 |
| `threat.go` | 146 |
| `animations.go` | 119 |
| `personas.go` | 57 |
| `styles.go` | 11 |

~700 lines. These change for one reason: visual design.

Cost to be aware of before starting: `Styles` is referenced in 15 files and
`Theme` in 5, so those become `chrome.Styles` / `chrome.Theme`. Mechanical, but it
touches ~20 files and the compiler must drive it. `moonbaseTheme`,
`newDegradedStyles`, `NextTheme`, `ThemeByName` and `ThemeCount` are referenced
from tests, so `theme_test.go` and `personas_test.go` move too.

### Stage 3 — break the App coupling on views

This is the stage that unlocks the remaining ~6,000 lines, and it is the real
design work. Apply the **Humble Object** split
(`skills/architecture-boundaries.md`): convert view methods into functions taking
a narrow, purpose-built view model rather than the whole `App`.

```go
// Before — needs all of App, cannot move out of the package.
func (a App) renderSidebar(width, maxH int) string

// After — depends on a small struct, movable and directly testable.
type SidebarView struct {
    Agents   []AgentRow
    Selected int
    Width    int
    MaxH     int
}
func RenderSidebar(v SidebarView) string
```

`App.View()` then builds view models and passes them down. Dependencies point one
way, so `views` can become its own package. Each view converts independently, so
this is a per-view sequence rather than a single cutover — do the smallest one
first as a tracer bullet, confirm the shape, then continue.

Secondary benefit: view models are directly testable without constructing an
`App`, which is what currently makes rendering awkward to test.

### Stage 4 — split update/event handling

Once views are out, `update_*.go` and `helpers_*.go` can be grouped by the events
they handle. Re-measure before planning this stage; Stage 3 will have changed the
shape enough that planning it now would be guesswork.

## Guardrails already in place

`scripts/check-file-size.sh` freezes every currently-oversized file at its present
length and fails CI on growth, so the package cannot get worse while this proceeds
incrementally. The baseline entries for `internal/tui/*` are the debt register for
this work — an entry disappearing is the signal a stage landed.

## Explicitly not doing

Splitting for its own sake. The goal is separating reasons to change, not hitting
a file count. `views_util.go` and the tiny `model_*.go` accessors (8–28 lines
each) are fine where they are; moving them would add imports and buy nothing.
