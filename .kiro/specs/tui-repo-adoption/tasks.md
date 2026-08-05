# Tasks: TUI Repo Adoption & Detection

Phased, behaviour-additive. Each task: implement → `go build ./... && go vet ./...
&& go test -race ./...` green → `moonbase lint` green → one conventional commit.
Refs `requirements.md` (AC-n.i) + `design.md` (§). Effort: *S*<½d, *M*~1d.

## Phase R1 — Detection primitive (shared)
- [ ] **TR1.1** Export `DetectType(dir) string` in `internal/projects/projects.go`
  (keep `detectType` as a thin wrapper); add `IsRepo(dir) bool`. — AC-1.1/1.2 — *S*
- [ ] **TR1.2** Add `NearestRepoRoot(start) (root, typ string)` — walk up, bounded
  by FS root; safe on unreadable dirs. — AC-1.4/3.5 — *S*
- [ ] **TR1.3** Tests: `TestDetectType_Manifest/_Git/_None`,
  `TestNearestRepoRoot_FromSubfolder`, permission-denied safety. — AC-4.3
- [ ] **R1 gate:** build/vet/test-race green; lint green.

## Phase R2 — Middle-panel colour + glyph (AC-2)
- [ ] **TR2.1** Add `IsRepo bool` + `RepoType string` to `FileEntry`; populate in
  `refresh()` via `projects.DetectType` (dirs only; once per refresh). — AC-1.3 — *S*
- [ ] **TR2.2** Add repo colour token: `theme.Data.Repo` (fallback `Brand`) +
  `repoColor(theme)` helper. — AC-2.1 — *S*
- [ ] **TR2.3** In `renderFileBrowser`, style repo dirs with `repoColor`+bold, add
  a distinct repo glyph and a ` [type]` tag **before** width truncation; keep
  selected-row (`Active`) precedence. — AC-2.1/2.2/2.3/2.4 — *M*
- [ ] **TR2.4** Tests: `TestBrowserRefresh_FlagsRepos`,
  `TestRenderRow_RepoColourAndGlyph` (assert style/glyph, not pixels). — AC-4.3
- [ ] **R2 gate:** build/vet/test-race green; lint green; manual: repos visibly
  distinct in the browser.

## Phase R3 — Left-panel / HQ adoption (AC-3)
- [ ] **TR3.1** `FileBrowser` tracks `adopted`/`adoptedType`; on `Enter` into a
  dir, set them from `projects.NearestRepoRoot(newDir)`. Expose
  `AdoptedProject() (path, typ string, changed bool)`. — AC-3.1/3.5 — *M*
- [ ] **TR3.2** In the TUI update loop (`update_common.go`), after an Enter key in
  the browser, read `AdoptedProject()`; if changed, set the active
  project/`projectDir` and refresh the left panel/project context. — AC-3.2 — *M*
- [ ] **TR3.3** Non-repo Enter → no change; `Back` → no un-adopt. — AC-3.3/3.4 — *S*
- [ ] **TR3.4** Tests: `TestEnter_AdoptsRepoRoot`, `TestEnter_NonRepo_NoAdoption`,
  adopt-from-subfolder. — AC-4.3
- [ ] **R3 gate:** build/vet/test-race green; lint green; manual: entering a repo
  updates the left panel to that repo.

## Definition of Done
1. AC-1..AC-4 satisfied with cited tests.
2. `go build/vet/test -race ./...` green; `moonbase lint` green.
3. Repos are colour+glyph+tag distinct in the middle panel (a11y-safe).
4. Entering/opening a repo (or a subfolder of one) adopts it as HQ's active
   project; plain dirs don't.
5. Additive only — no regression to existing browser navigation.

## Traceability
| Task | AC | § |
|------|----|---|
| TR1.* | 1.1/1.2/1.4/3.5 | §1 |
| TR2.* | 1.3/2.* | §2/§3 |
| TR3.* | 3.* | §4/§5 |

## Non-Goals
git status/branch/dirty badges; monorepo sub-package detection beyond nearest
root; remote-repo detection; changing `projects.Discover()` roots.
