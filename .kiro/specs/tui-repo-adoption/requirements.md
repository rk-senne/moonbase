# Requirements: TUI Repo Adoption & Detection

## Overview

In the moonbase TUI file browser, navigating the filesystem is currently
"dumb" — every directory looks the same, and moving into a project folder does
not make that project the *active* one. Operators cannot see at a glance which
folders are repositories, and opening a repo doesn't focus HQ on it.

This spec makes the browser **repo-aware**:
1. **Repo detection** — identify which directories are repositories/projects.
2. **Middle panel colouring** — repos are visually distinct in the file list.
3. **Left panel adoption** — entering (or opening) a repo makes it the active
   project, so HQ (left panel / project context) adopts that repo.

**Grounded in current code (confirmed):**
- `internal/projects/projects.go` already has `detectType(dir)` — manifest map
  (`go.mod`→go, `package.json`→node, `pom.xml`/`build.gradle`→java,
  `Cargo.toml`→rust) with a `.git` fallback → `"git"`. This is the detection
  primitive to reuse (currently unexported).
- `internal/tui/filebrowser.go` — the middle-panel browser. `refresh()` lists
  entries (hidden dotfiles skipped), `Enter()` descends + `os.Chdir`,
  `renderFileBrowser()` colours directories with `theme.Data.Info`, `fileIcon()`
  maps types to glyphs.
- The TUI theme exposes `theme.Data.{Active, Brand, Dim, Info, …}`.

> **AC-ID convention:** stable IDs `AC-{n}.{i}`.

---

## User Stories
- **US-1:** As an operator, when I open/enter a repo in the browser, HQ adopts it
  as the active project so subsequent actions target that repo.
- **US-2:** As an operator, I want repositories coloured in the middle panel so I
  can instantly tell which folders are repos versus plain directories.
- **US-3:** As an operator, I want detection to be accurate and cheap so browsing
  stays fast.

---

## Acceptance Criteria

### AC-1 — Repo Detection (shared primitive)
- **AC-1.1** — A single exported detector SHALL report whether a directory is a
  repo and its type. Reuse/extend `projects.detectType`: expose
  `projects.DetectType(dir) string` (empty = not a repo) and a convenience
  `projects.IsRepo(dir) bool`.
- **AC-1.2** — Detection SHALL treat a directory as a repo if it contains any
  known manifest (`go.mod`, `package.json`, `pom.xml`, `build.gradle`,
  `Cargo.toml`, …) **or** a `.git` directory (fallback type `"git"`).
- **AC-1.3** — Detection SHALL be a single-level `os.Stat` check per candidate
  (no deep walk); results for the visible entries SHALL be computed once per
  `refresh()` and cached on the entry, not recomputed per render frame.
- **AC-1.4** — Detection SHALL be safe on unreadable/permission-denied dirs
  (returns not-a-repo, no panic).

### AC-2 — Middle Panel Repo Colouring
- **AC-2.1** — In the file list, a directory that is a repo SHALL render in a
  **distinct repo colour** (a theme token, e.g. `theme.Data.Brand` or a new
  `theme.Data.Repo`), clearly different from a plain directory
  (`theme.Data.Info`) and from files (`theme.Data.Dim`).
- **AC-2.2** — A repo directory SHALL show a **repo glyph** (e.g. a distinct
  icon) in addition to colour, so the cue survives for colour-blind operators and
  monochrome terminals (do not rely on colour alone).
- **AC-2.3** — The repo type SHALL be indicated (icon variant or a short suffix
  tag such as `[go]`/`[node]`/`[git]`) without breaking the existing name
  truncation/width logic.
- **AC-2.4** — The currently-selected entry's existing highlight
  (`theme.Data.Active`, bold) SHALL still take precedence when a repo row is
  under the cursor.

### AC-3 — Left Panel / HQ Adoption
- **AC-3.1** — When the operator **enters** a directory that is a repo root
  (`Enter` on a repo dir), HQ SHALL adopt it as the **active project** (set the
  TUI's project/`projectDir` state to that path).
- **AC-3.2** — Adoption SHALL update the left panel / project context so it
  reflects the adopted repo's name and type.
- **AC-3.3** — Entering a **non-repo** directory SHALL NOT change the active
  project (plain navigation only).
- **AC-3.4** — Adoption SHALL be idempotent and reversible: navigating out
  (`Back`) does not silently un-adopt; the active project persists until a
  different repo is adopted (or an explicit clear).
- **AC-3.5** — If the opened path is *inside* a repo but not at its root, HQ
  SHALL adopt the nearest ancestor repo root (walk up until a repo marker or the
  filesystem root is found) — so opening a subfolder still adopts the repo.

### AC-4 — Cross-Cutting
- **AC-4.1** — No new runtime dependency; vanilla Go + existing lipgloss theme.
- **AC-4.2** — `go build/vet/test -race ./...` green; `moonbase lint` green.
- **AC-4.3** — Detection + adopt-root-walk + colour selection have unit tests
  (`TestFn_Scenario`); TUI render logic tested via `Update()`/render helper, not
  golden-pixel fragility (per test-alignment steering).

---

## Scope
**In:** exported repo detector; per-entry repo flag+type cached on `refresh()`;
middle-panel colour + glyph + type tag; `Enter`/open adoption with
nearest-ancestor root walk; left-panel reflection of the adopted project; tests.

**Out:** git status/branch/dirty indicators (separate feature); multi-repo
workspaces/monorepo sub-package detection beyond nearest-root; remote repo
detection; changing `projects.Discover()` scan roots.

---

## Dependencies
| Dependency | Status | Impact |
|-----------|--------|--------|
| `internal/projects` detectType | exists (unexported) | export + reuse (AC-1) |
| `internal/tui/filebrowser.go` | exists | colour + adoption wiring (AC-2/3) |
| TUI theme tokens | exists | repo colour (may add `Data.Repo`) |
| left-panel/project state | exists (`projectDir` in update_common/app) | adoption target (AC-3) |

## Risks
| Risk | Mitigation |
|------|-----------|
| Per-frame detection cost | AC-1.3 compute once per refresh, cache on entry |
| Colour-only cue fails a11y | AC-2.2 glyph + tag alongside colour |
| Adopting the wrong root from a subfolder | AC-3.5 nearest-ancestor walk with bounded stop |
| `.git` is a hidden entry (skipped in list) | detection uses `os.Stat`, independent of list filtering |
