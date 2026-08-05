# Design: TUI Repo Adoption & Detection

## ADR-summary
**Decision:** Add one shared repo detector, cache a repo flag/type on each browser
entry at `refresh()`, colour+glyph repos in the middle panel, and adopt the
nearest-ancestor repo root as HQ's active project on `Enter`/open.
**Why:** reuse the existing `detectType` primitive (single-level stat, cheap);
keep colour off the hot render path; make "open a repo → HQ focuses it" the
obvious mental model. **Reversibility:** additive fields + one theme token + a
guarded branch in `Enter()`; trivially revertible.

## Files Affected
| File | Change | Purpose |
|------|--------|---------|
| `internal/projects/projects.go` | modify | Export `DetectType(dir) string` + add `IsRepo(dir) bool` + `NearestRepoRoot(dir) (string,string)` |
| `internal/projects/projects_test.go` | modify | tests for exported detector + nearest-root walk |
| `internal/tui/filebrowser.go` | modify | `FileEntry{IsRepo bool; RepoType string}`; set in `refresh()`; colour+glyph+tag in `renderFileBrowser`; adopt in `Enter()` |
| `internal/tui/model_browser.go` / `update_common.go` | modify | set active `projectDir`/project on adoption; left-panel reflect |
| `internal/tui/theme*.go` | modify (optional) | add `Data.Repo` colour token (fallback to `Brand`) |
| `internal/tui/*_test.go` | new/modify | entry-flagging, colour selection, adoption via Enter |

## Components

### 1. Detection primitive (`internal/projects`)
```go
// DetectType returns the repo/project type for dir ("" if not a repo).
// Manifest match wins; ".git" is the fallback ("git"). Single-level stat.
func DetectType(dir string) string { /* current detectType body, exported */ }

// IsRepo reports whether dir is a repository/project root.
func IsRepo(dir string) bool { return DetectType(dir) != "" }

// NearestRepoRoot walks up from start until a repo marker is found.
// Returns (root, type) or ("","") at filesystem root. Bounded by path depth.
func NearestRepoRoot(start string) (string, string) {
    for d := start; ; d = filepath.Dir(d) {
        if t := DetectType(d); t != "" { return d, t }
        parent := filepath.Dir(d)
        if parent == d { return "", "" } // reached FS root
    }
}
```
`detectType` stays as a thin wrapper calling `DetectType` (back-compat).

### 2. Browser entry flagging (AC-1.3, cache-once)
```go
type FileEntry struct {
    Name string; IsDir bool; Size int64
    IsRepo bool; RepoType string   // NEW — set once in refresh()
}
```
In `refresh()`, for each **directory** entry: `t := projects.DetectType(filepath.Join(fb.dir, e.Name)); entry.RepoType=t; entry.IsRepo = t!=""`. Files are never repos. Computed once per refresh, read in render (no per-frame stat).

### 3. Middle-panel colour + glyph + tag (AC-2)
In `renderFileBrowser`, when building a non-selected directory row:
```go
style := lipgloss.NewStyle().Foreground(a.theme.Data.Info) // plain dir (today)
if entry.IsRepo {
    style = lipgloss.NewStyle().Foreground(repoColor(a.theme)).Bold(true)
}
```
- `repoColor(theme)` → `theme.Data.Repo` if defined, else `theme.Data.Brand`.
- Glyph: extend `fileIcon` (or a `repoIcon(repoType)`) so repos show a distinct
  mark (e.g. `◆`/a repo glyph) instead of the plain `📁`; type-specific variant
  optional.
- Type tag: append a short ` [go]`/`[node]`/`[git]` suffix **before** the
  existing width-truncation runs, so truncation still fits `listW`.
- Selected-row highlight (`Active`, bold) continues to take precedence (AC-2.4):
  the `i == fb.cursor` branch is unchanged; repo styling only affects the
  non-cursor branch (plus optionally a repo glyph in both).

### 4. Adoption on Enter/open (AC-3)
`FileBrowser.Enter()` today: descend into dir + `os.Chdir` + refresh. Add adoption:
```go
if entry.IsDir {
    newDir := filepath.Join(fb.dir, entry.Name)
    // AC-3.1/3.5: adopt nearest repo root at/above the entered dir
    if root, typ := projects.NearestRepoRoot(newDir); root != "" {
        fb.adopted = root; fb.adoptedType = typ   // browser signals adoption
    }
    fb.dir = newDir; os.Chdir(newDir); fb.cursor = 0; fb.refresh()
}
```
The browser exposes `AdoptedProject() (path, typ string, changed bool)`; the TUI
update loop (`update_common.go`) reads it after an `Enter` key and sets the HQ
active project (`projectDir`) + refreshes the left panel/project context
(AC-3.2). Non-repo entry → `NearestRepoRoot` returns the current active (or
"") → no change (AC-3.3). `Back` does not clear adoption (AC-3.4).

### 5. Left-panel reflection
The left panel already renders from the active project/`projectDir`. Adoption
just updates that state; the panel re-renders on the next frame showing the
adopted repo's name + type. No new panel plumbing.

## Data Flow
```
refresh():   dirs → projects.DetectType() → entry.IsRepo/RepoType (cached)
render():    entry.IsRepo → repoColor + glyph + [type] tag   (no I/O)
Enter(dir):  projects.NearestRepoRoot(dir) → adopted → update loop sets projectDir → left panel adopts
```

## Edge Cases
| Case | Handling |
|------|----------|
| Enter subfolder of a repo | AC-3.5 nearest-ancestor walk adopts the root |
| Dir with both manifest and `.git` | manifest type wins (DetectType order) |
| Permission-denied dir | DetectType stat fails → not-a-repo, no panic (AC-1.4) |
| `.git` hidden + skipped from list | detection uses stat, independent of list filter |
| Home/`/` reached in walk | `filepath.Dir(d)==d` stop → ("","") no adoption |
| Repo row under cursor | Active highlight precedence (AC-2.4) |

## Testing (AC-4.3)
| Test | Scenario |
|------|----------|
| `TestDetectType_Manifest/_Git/_None` | manifest, git-fallback, plain dir |
| `TestNearestRepoRoot_FromSubfolder` | walks up to root; ("","") at FS root |
| `TestBrowserRefresh_FlagsRepos` | entries get IsRepo/RepoType once |
| `TestRenderRow_RepoColourAndGlyph` | repo dir styled distinctly + glyph + tag |
| `TestEnter_AdoptsRepoRoot` | Enter repo → active project set |
| `TestEnter_NonRepo_NoAdoption` | Enter plain dir → project unchanged |

`go test -race ./...` + `moonbase lint` green. No golden-pixel tests (per steering).

## Reversibility / Migration
All additive: two `FileEntry` fields, one exported func + wrapper, one theme
token (with fallback), one guarded branch in `Enter`, one read in the update
loop. Delete to revert. No data/format migration.
