# Moonbase Gap Analysis — Research Library Audit

**Date:** 2026-07-28  
**Audited against:** Clean Architecture, Clean Code, Clean Coder, Philosophy of Software Design, Designing Data-Intensive Applications, The Pragmatic Programmer, The Phoenix Project, The C4 Model, Black Hat Go, MMIX Supplement

---

## Summary

| Metric | Count |
|--------|-------|
| **Total Gaps** | 33 |
| **HIGH** | 7 |
| **MEDIUM** | 14 |
| **LOW** | 12 |

### By Category

| Category | HIGH | MEDIUM | LOW | Total |
|----------|------|--------|-----|-------|
| Architecture | 1 | 4 | 3 | 8 |
| Code Quality | 2 | 2 | 2 | 6 |
| Security | 1 | 2 | 0 | 3 |
| Testing | 0 | 2 | 1 | 3 |
| Pipeline/Operations | 2 | 2 | 1 | 5 |
| Data/Reliability | 1 | 2 | 2 | 5 |
| Documentation | 0 | 0 | 3 | 3 |

---

## HIGH Severity

### SEC-1: Hook Guard Uses Blocklist — Trivially Bypassable

| Field | Value |
|-------|-------|
| **Book** | Black Hat Go |
| **Principle** | Input sanitization must be robust against evasion techniques |
| **Location** | `internal/tui/helpers_commands.go:55-67` (`isSafeHookCommand`) |
| **Category** | Security |

**Description:** The hook guard uses `strings.Contains(cmd, d)` against a blocklist of dangerous commands. This is trivially bypassable via: full paths (`/usr/bin/curl`), IFS splitting, aliasing, command substitution, or unlisted interpreters (`php`, `go run`, `deno`, `bun`, `aria2c`).

**Suggested Fix:** Switch from blocklist to allowlist. Parse the command with `strings.Fields()`, extract the base command name, and validate against an explicit allowlist:
```go
allowed := map[string]bool{"cat": true, "ls": true, "git": true, "echo": true, "wc": true, "head": true, "tail": true}
```

---

### PIPE-1: Flywheel & Checkpoint Systems Never Called at Runtime

| Field | Value |
|-------|-------|
| **Book** | Designing Data-Intensive Applications / The Phoenix Project |
| **Principle** | Reliability — data must be persisted; The Second Way — feedback loops |
| **Location** | `cmd/moonbase/mission.go`, `internal/pipeline/flywheel.go`, `internal/pipeline/checkpoint.go` |
| **Category** | Pipeline/Operations |

**Description:** The flywheel (`FlywheelLog.Append()`) and checkpoint (`SaveCheckpoint()`) infrastructure exists but is **never called** during actual mission execution — only in tests. `history.Save()` is also not called. The `moonbase flywheel` command will always show "No flywheel data yet." The entire self-improvement feedback loop is non-functional dead infrastructure.

**Suggested Fix:** Wire into `runMission()`:
1. `FlywheelLog.Append()` after each phase completes (phase number, agent, duration, outcome, output size, risk level)
2. `SaveCheckpoint()` after each phase transition
3. `history.Save()` at mission completion

---

### PIPE-2: PhaseTimeout Conflict — Config Ignored

| Field | Value |
|-------|-------|
| **Book** | MMIX Supplement / General |
| **Principle** | Constants must have a single source of truth |
| **Location** | `internal/tui/pipeline_exec.go:18` (120s) vs `internal/config/config.go:43` (300s) |
| **Category** | Pipeline/Operations |

**Description:** `PhaseTimeout` in `pipeline_exec.go` is hardcoded to 120 seconds, but `config.go` defaults to 300 seconds. The TUI uses its own constant, so config changes have **no effect** on actual pipeline execution.

**Suggested Fix:** Remove the hardcoded constant. Pass `cfg.PhaseTimeout` to the TUI at construction time or have `executePhase` read from the injected config.

---

### ARCH-1: Phase Definitions Leak Across Module Boundaries

| Field | Value |
|-------|-------|
| **Book** | Philosophy of Software Design |
| **Principle** | Information leakage — design knowledge should be encapsulated in one place |
| **Location** | `internal/pipeline/pipeline.go` (line 107) AND `cmd/moonbase/mission.go` (line 136) |
| **Category** | Architecture |

**Description:** `pipeline.New()` hardcodes 8 phases with operative names, agent names, and trigger specs. `runMissionFast()` in `mission.go` creates its own `fastPhases` slice with the **same information**. If phase 3's agent name changes, you must update both locations. Agent-name-to-phase-number mapping leaks across module boundaries.

**Suggested Fix:** The pipeline should own ALL phase knowledge. Add `pipeline.NewFast(task string)` or a `pipeline.WithPhases(nums ...int)` builder. The cmd layer should never construct `pipeline.Phase` structs directly.

---

### CQ-1: `runMission()` Too Large — Mixed Abstraction Levels

| Field | Value |
|-------|-------|
| **Book** | Clean Code |
| **Principle** | Functions should be small (5-20 lines), do one thing, at one level of abstraction |
| **Location** | `cmd/moonbase/mission.go` — `runMission()` (~105 lines) |
| **Category** | Code Quality |

**Description:** `runMission()` handles: loading agents, discovering context, creating the pipeline, iterating phases, resolving agents, handling conditional phases, deploying to backends, recording output, parsing meta, capturing git diffs, applying risk gates, and printing status. It mixes high-level orchestration with low-level details like `exec.Command("git", "diff")`.

**Suggested Fix:** Extract: `discoverProjectContext()`, `executePhase()`, `capturePostPhaseArtifacts()`, `reportPhaseResult()`. The function should read as: discover → create pipeline → for each phase: execute → assess → route.

---

### CQ-2: `runMission()` / `runMissionFast()` — 60% Code Duplication

| Field | Value |
|-------|-------|
| **Book** | Clean Code |
| **Principle** | DRY — Don't Repeat Yourself |
| **Location** | `cmd/moonbase/mission.go` — both functions |
| **Category** | Code Quality |

**Description:** `runMissionFast()` duplicates ~60% of `runMission()`: agent loading, discovery, deployment logic, diff capture, risk gate application. They differ only in which phases run and whether rework loops are allowed. If routing logic changes, you must remember to update both.

**Suggested Fix:** Extract a shared `executeMissionPhases(pipeline, registry, context, phases []Phase)` function. Or use a builder: `NewMission(task).WithPhases(3,4).Run()`.

---

### DATA-1: Backend Fallback Chain Bypassed

| Field | Value |
|-------|-------|
| **Book** | The Phoenix Project |
| **Principle** | Single Point of Failure — critical paths need fallback |
| **Location** | `cmd/moonbase/mission.go` — `deployToBackend()` |
| **Category** | Data/Reliability |

**Description:** The mission pipeline hardcodes `backend.Kiro{TrustTools: true}` rather than using `backend.Preferred()`. If kiro-cli is unavailable, fallback is clipboard (manual copy/paste). OpenAI/Anthropic backends exist but are unused by the mission pipeline despite being available.

**Suggested Fix:** In `deployToBackend()`, use `backend.Preferred()` which already implements priority detection with automatic fallback to OpenAI/Anthropic before clipboard.


---

## MEDIUM Severity

### ARCH-2: TUI Directly Imports 10+ Business Logic Packages

| Field | Value |
|-------|-------|
| **Book** | Clean Architecture |
| **Principle** | Dependency Rule — UI (framework detail) should depend on interfaces, not concrete implementations |
| **Location** | `internal/tui/app.go` |
| **Category** | Architecture |

**Description:** The TUI App struct imports `internal/pipeline`, `internal/backend`, `internal/discovery`, `internal/agents`, `internal/history`, `internal/chat`, `internal/watcher`, `internal/snippets`, and `internal/platform` directly. Replacing the TUI requires touching business logic.

**Suggested Fix:** Define interfaces (`AgentLister`, `PipelineRunner`, `ContextDiscoverer`) in the business-logic packages. Inject them into the App constructor. This decouples the delivery mechanism from core logic.

---

### ARCH-3: Main Contains Business Logic (385-line main.go)

| Field | Value |
|-------|-------|
| **Book** | Clean Architecture |
| **Principle** | Main as plugin — should only wire dependencies, not contain domain logic |
| **Location** | `cmd/moonbase/main.go` |
| **Category** | Architecture |

**Description:** `main.go` contains `runList()` (100+ lines of operative display logic with formatting, grouping, hardcoded fallback data), plus `extractNumbuh()`, `sourceTag()`, `loadAgentRegistry()`, `isTerminal()`. The hardcoded agent fallback table duplicates domain knowledge.

**Suggested Fix:** Extract `runList()` to `internal/roster/`. The cmd layer should only parse args, wire dependencies, and delegate.

---

### ARCH-4: Temporal Decomposition in Mission Execution

| Field | Value |
|-------|-------|
| **Book** | Philosophy of Software Design |
| **Principle** | Organize by knowledge, not by execution sequence |
| **Location** | `cmd/moonbase/mission.go` — `injectFileContext()`, `handleRiskGate()`, `handlePhaseFailure()` |
| **Category** | Architecture |

**Description:** Functions are organized around "what happens next" rather than knowledge boundaries. `injectFileContext()` knows pipeline context internals and file size limits. `handleRiskGate()` knows pipeline internals AND phase indexing. The cmd layer manages routing logic that belongs in the pipeline.

**Suggested Fix:** Move `injectFileContext()` into `pipeline` package. Add `p.ApplyRiskGateAndRoute()` that returns the new phase index — eliminate the need for the cmd layer to search through phases.

---

### ARCH-5: TUI App is a God Object (40+ Fields)

| Field | Value |
|-------|-------|
| **Book** | Philosophy of Software Design |
| **Principle** | General-purpose mechanisms over special-purpose aggregations |
| **Location** | `internal/tui/app.go` — App struct |
| **Category** | Architecture |

**Description:** The App struct holds state for every view, feature, and mode with no separation. Dashboard, pipeline, comms, file browser, and terminal state all live as flat fields. Each new feature adds more fields to the same struct.

**Suggested Fix:** Break into sub-models: `DashboardState`, `PipelineState`, `FileBrowserState`, `TerminalState`. The App becomes a container of nested Bubbletea models. `CommsState` is already extracted — apply the same pattern.

---

### CQ-3: Swallowed Errors in Mission Execution

| Field | Value |
|-------|-------|
| **Book** | Clean Code / The Pragmatic Programmer |
| **Principle** | Don't ignore errors; crash early |
| **Location** | `cmd/moonbase/mission.go` (multiple `_` discards), `internal/history/history.go:32-34` |
| **Category** | Code Quality |

**Description:** `cwd, _ := os.Getwd()` and `ctx, _ := discovery.Discover(cwd)` silently discard errors. In `history.go`, `home, _ := os.UserHomeDir()` could produce `"/.config/moonbase/history.json"` at filesystem root. `history.Load()` silently returns nil on corrupt JSON. 12+ similar sites across the codebase.

**Suggested Fix:** For `os.Getwd()`: return error (mission can't proceed without cwd). For discovery: log warning, continue with empty context. For history: distinguish "file not found" from "corrupt data." Add `--debug` logging for all discarded errors.

---

### CQ-4: Abbreviated/Confusing Names in TUI App

| Field | Value |
|-------|-------|
| **Book** | Clean Code |
| **Principle** | Use intention-revealing names |
| **Location** | `internal/tui/app.go` — App struct fields |
| **Category** | Code Quality |

**Description:** Fields named `ci`, `fi`, `ti2`, `fw`, `ctx` require mental mapping. `ctx` (platform.Context) conflicts with Go's `context.Context` convention. `pipelineCtx` vs `ctx` is confusing.

**Suggested Fix:** Rename: `ctx` → `platformCtx`, `ci` → `commsInput`, `fi` → `fileInput`, `ti2` → `termInput`, `fw` → `fileWatcher`.

---

### PIPE-3: No WIP Limit on Concurrent Conditional Phases

| Field | Value |
|-------|-------|
| **Book** | The Phoenix Project |
| **Principle** | WIP Limits / Theory of Constraints |
| **Location** | `cmd/moonbase/mission.go` — `runConditionalPhasesParallel()` |
| **Category** | Pipeline/Operations |

**Description:** All conditional phases fire simultaneously as goroutines with no concurrency limit. Could overwhelm AI backend rate limits and token budgets.

**Suggested Fix:** Add `MaxConcurrentPhases` config option (default 2-3). Use a buffered channel as semaphore. Add to `Config` as `max_concurrent_phases`.

---

### PIPE-4: Hardcoded Model Defaults Require Rebuild to Change

| Field | Value |
|-------|-------|
| **Book** | General |
| **Principle** | Frequently-changing defaults should be configurable without code changes |
| **Location** | `internal/chat/stream.go:80`, `internal/backend/openai.go:86`, `internal/backend/kimi.go:39`, `internal/backend/backends.go:122` |
| **Category** | Pipeline/Operations |

**Description:** Default model names (`claude-sonnet-4-20250514`, `gpt-4o`, `kimi-k3`, `llama3.1`) are hardcoded. Ollama has no env var override. When models deprecate, a rebuild is required.

**Suggested Fix:** Add `OLLAMA_MODEL` env var support. Move defaults to `config.yaml`:
```yaml
models:
  anthropic: claude-sonnet-4-20250514
  openai: gpt-4o
  ollama: llama3.1
  kimi: kimi-k3
```

---

### SEC-2: No Signature Verification for Self-Update Binary

| Field | Value |
|-------|-------|
| **Book** | Black Hat Go |
| **Principle** | Binary download integrity requires more than checksums from the same source |
| **Location** | `internal/updater/updater.go` |
| **Category** | Security |

**Description:** The updater has SHA256 checksum verification (good), but checksums come from the same GitHub release. If an attacker compromises the release, they can update both binary and checksums. No GPG/cosign signature verification.

**Suggested Fix:** Add cosign signature verification for release artifacts, or at minimum document the threat model explicitly. Short-term: ensure HTTP client has explicit timeouts.

---

### SEC-3: No Static Security Analysis in CI

| Field | Value |
|-------|-------|
| **Book** | Black Hat Go / General |
| **Principle** | Security-focused static analysis catches patterns `go vet` misses |
| **Location** | `.github/workflows/ci.yml`, `Makefile` |
| **Category** | Security |

**Description:** No `gosec`, `govulncheck`, `staticcheck`, or `golangci-lint` configured. CI only runs `go vet`. 8 direct + 30+ indirect dependencies are never scanned for CVEs. No `dependabot.yml` for automated updates.

**Suggested Fix:** Add to CI:
```yaml
- name: Govulncheck
  run: go install golang.org/x/vuln/cmd/govulncheck@latest && govulncheck ./...
- name: Staticcheck
  uses: dominikh/staticcheck-action@v1
```
Add `.github/dependabot.yml` for weekly gomod updates.

---

### TEST-1: No Race Condition Detection in CI

| Field | Value |
|-------|-------|
| **Book** | Black Hat Go |
| **Principle** | Go's race detector should be used on concurrent code |
| **Location** | `.github/workflows/ci.yml`, `Makefile` |
| **Category** | Testing |

**Description:** Neither CI nor Makefile uses `go test -race`. Codebase has goroutines in `chat/stream.go`, `tui/pipeline_exec.go`, and a mutex in `watcher/watcher.go`.

**Suggested Fix:** Add to CI: `go test -race ./... -count=1 -timeout 300s`. Add Makefile target `test-race`.

---

### TEST-2: No Coverage Enforcement in CI

| Field | Value |
|-------|-------|
| **Book** | General |
| **Principle** | CI should prevent coverage regression |
| **Location** | `.github/workflows/ci.yml` |
| **Category** | Testing |

**Description:** Makefile has a `coverage` target but CI doesn't run it or enforce a threshold. No coverage badge or PR reporting.

**Suggested Fix:** Add coverage gate (70% minimum):
```yaml
- name: Coverage
  run: |
    go test ./... -coverprofile=coverage.out -timeout 60s
    COVERAGE=$(go tool cover -func=coverage.out | tail -1 | awk '{print $3}' | tr -d '%')
    if (( $(echo "$COVERAGE < 70" | bc -l) )); then exit 1; fi
```

---

### DATA-2: No Schema Versioning for Persistent Data

| Field | Value |
|-------|-------|
| **Book** | Designing Data-Intensive Applications |
| **Principle** | Schema evolution — formats must be forward/backward compatible |
| **Location** | `internal/pipeline/flywheel.go`, `internal/history/history.go` |
| **Category** | Data/Reliability |

**Description:** Neither `FlywheelEntry` nor `Mission` JSON structs have a `schema_version` field. When fields are added, old entries become silently incomplete. No migration strategy exists.

**Suggested Fix:** Add `"schema_version": 1` to both structs. Handle missing fields with defaults on read. Document format in comments.

---

### DATA-3: Path Construction Scattered Across 14 Files

| Field | Value |
|-------|-------|
| **Book** | The Pragmatic Programmer |
| **Principle** | DRY — single authoritative representation for each piece of knowledge |
| **Location** | 14 files across `internal/` (history, flywheel, config, snippets, chat, logging, etc.) |
| **Category** | Data/Reliability |

**Description:** Each package independently constructs paths (`~/.moonbase/`, `~/.config/moonbase/`). History uses `~/.config/moonbase/`, flywheel uses `~/.moonbase/` — inconsistent split. `os.UserHomeDir()` + path joining duplicated everywhere.

**Suggested Fix:** Create `internal/paths` package: `paths.DataDir()` → `~/.moonbase/`, `paths.ConfigDir()` → `~/.config/moonbase/`. All packages import this single source of truth.


---

## LOW Severity

### ARCH-6: Presentation Logic in Pipeline Package

| Field | Value |
|-------|-------|
| **Book** | Clean Architecture / The Pragmatic Programmer |
| **Principle** | Business logic should not contain UI/presentation concerns |
| **Location** | `internal/pipeline/pipeline.go` — `statusIcon()` (lines 228-245), `StatusSummary()` |
| **Category** | Architecture |

**Description:** Pipeline package contains emoji-to-status mapping and human-readable formatting. A self-aware comment acknowledges: "NOTE: This is a PRESENTATION concern." 5 minutes of work to fix.

**Suggested Fix:** Move `statusIcon()` and `StatusSummary()` to the TUI or `internal/format` package. Pipeline exports only `PhaseStatus` enum.

---

### ARCH-7: Screaming Architecture (Acceptable)

| Field | Value |
|-------|-------|
| **Book** | Clean Architecture |
| **Principle** | Top-level structure should reveal the domain |
| **Location** | `internal/` directory structure |
| **Category** | Architecture |

**Description:** Packages organized by technical role (`tui/`, `backend/`, `pipeline/`) rather than use case (`mission/`, `deploy/`). The `tui/` package (53 files) is the largest. Acceptable for a single-binary CLI but worth noting for growth planning.

**Suggested Fix:** No immediate action. If the project grows significantly, consider regrouping by use case.

---

### ARCH-8: Cobra/Bubbletea Framework Leakage (Minor)

| Field | Value |
|-------|-------|
| **Book** | Clean Architecture |
| **Principle** | Frameworks as details — keep them at the edge |
| **Location** | `cmd/moonbase/root.go` |
| **Category** | Architecture |

**Description:** Root command directly instantiates `tui.NewApp()` and `tea.NewProgram()`. Minor — frameworks are already reasonably contained.

**Suggested Fix:** Have `root.go` call `tui.Run()` instead of importing `tea` package directly.

---

### CQ-5: Self-Aware Tech Debt Comment Not Actioned

| Field | Value |
|-------|-------|
| **Book** | Clean Coder |
| **Principle** | Don't leave known problems for later |
| **Location** | `internal/pipeline/pipeline.go:228-232` |
| **Category** | Code Quality |

**Description:** Comment says "NOTE: This is a PRESENTATION concern... move this function to the TUI layer." The author knew it was wrong and left it.

**Suggested Fix:** Do the refactoring (it's 5 minutes) or delete the comment and accept the placement.

---

### CQ-6: Temporal Coupling in Pipeline Advance

| Field | Value |
|-------|-------|
| **Book** | The Pragmatic Programmer |
| **Principle** | Don't require things to happen in a specific order unless necessary |
| **Location** | `cmd/moonbase/mission.go` — `p.Advance()` placement |
| **Category** | Code Quality |

**Description:** `p.Advance()` is called BEFORE the risk gate, requiring a `i = targetIdx - 1` hack when the gate routes backward.

**Suggested Fix:** Apply risk gate before advancing. Only call `p.Advance()` when the gate says proceed.

---

### TEST-3: No Windows CI Target

| Field | Value |
|-------|-------|
| **Book** | General |
| **Principle** | Test on all deployment platforms |
| **Location** | `.github/workflows/ci.yml` |
| **Category** | Testing |

**Description:** CI tests on ubuntu and macOS but not Windows. Clipboard package has Windows-specific code paths that are untested.

**Suggested Fix:** Add `windows-latest` to CI matrix, or explicitly document Windows as unsupported and exclude from goreleaser.

---

### PIPE-5: Makefile Release Missing LDFLAGS

| Field | Value |
|-------|-------|
| **Book** | General |
| **Principle** | Release builds should embed version metadata |
| **Location** | `Makefile:41-44` |
| **Category** | Pipeline/Operations |

**Description:** The `release` target doesn't pass `$(LDFLAGS)`, so manually-built release binaries report `version=dev`.

**Suggested Fix:** Add `$(LDFLAGS)` to each `go build` in the release target.

---

### DATA-4: Flywheel Log Has No Size Limit

| Field | Value |
|-------|-------|
| **Book** | Designing Data-Intensive Applications |
| **Principle** | Append-only logs require compaction |
| **Location** | `internal/pipeline/flywheel.go` |
| **Category** | Data/Reliability |

**Description:** JSONL flywheel log grows without bound. No rotation, compaction, or size cap.

**Suggested Fix:** Add `MaxEntries` or `MaxSizeMB` config. Rotate when limit exceeded.

---

### DATA-5: History Save Has No File Locking

| Field | Value |
|-------|-------|
| **Book** | Designing Data-Intensive Applications |
| **Principle** | Handle partial failures gracefully |
| **Location** | `internal/history/history.go` — `Save()` |
| **Category** | Data/Reliability |

**Description:** Read-modify-write cycle without advisory locking. Two concurrent missions could cause data loss. Unlikely for single-user CLI but architecturally unsound.

**Suggested Fix:** Add flock around the cycle, or switch to append-only JSONL format.

---

### DOC-1: Missing C4 Deployment Diagram

| Field | Value |
|-------|-------|
| **Book** | The C4 Model |
| **Principle** | Deployment diagrams show operational topology |
| **Location** | `docs/architecture.md` |
| **Category** | Documentation |

**Description:** No deployment diagram showing goreleaser → GitHub Releases → user binary → `~/.moonbase/` structure.

**Suggested Fix:** Add Mermaid deployment diagram.

---

### DOC-2: Component Diagram Model-Code Gap Risk

| Field | Value |
|-------|-------|
| **Book** | The C4 Model |
| **Principle** | Component diagrams drift when hand-maintained |
| **Location** | `docs/architecture.md` |
| **Category** | Documentation |

**Description:** Component diagram shows 7 pipeline components. No automation validates it still matches code.

**Suggested Fix:** Add last-verified date comment, or a `make verify-architecture` target.

---

### DOC-3: Missing Dynamic/Sequence Diagram

| Field | Value |
|-------|-------|
| **Book** | The C4 Model |
| **Principle** | Dynamic diagrams show runtime interaction for key use cases |
| **Location** | `docs/architecture.md` |
| **Category** | Documentation |

**Description:** Pipeline flow described as ASCII art in README but no formal sequence diagram showing runtime interactions including risk gates and rework loops.

**Suggested Fix:** Add Mermaid sequence diagram for the `moonbase mission` flow.

---

## Quick Wins

The top 5 easiest HIGH/MEDIUM fixes that deliver immediate value:

| # | Gap ID | Effort | Impact | Action |
|---|--------|--------|--------|--------|
| 1 | **PIPE-2** | 10 min | HIGH | Remove hardcoded `PhaseTimeout` constant in `pipeline_exec.go`, use config value |
| 2 | **ARCH-6** / **CQ-5** | 15 min | LOW→MEDIUM | Move `statusIcon()` + `StatusSummary()` to TUI package (resolves known tech debt) |
| 3 | **TEST-1** | 5 min | MEDIUM | Add `-race` flag to CI test step |
| 4 | **SEC-3** | 15 min | MEDIUM | Add `govulncheck` + `dependabot.yml` to CI |
| 5 | **PIPE-4** | 20 min | MEDIUM | Add `OLLAMA_MODEL` env var, move defaults to config |

---

## Strategic Improvements

Changes that require design thought, multi-file refactoring, or architectural decisions:

### 1. Unify Mission Execution (CQ-1, CQ-2, ARCH-1, ARCH-4)

**Goal:** Single mission execution engine with configurable phases.

**Design work needed:**
- Define `pipeline.NewFast()` or `pipeline.WithPhases()` builder
- Extract shared `executeMissionPhases()` from the two duplicated functions
- Move `injectFileContext()` and risk gate handling into pipeline package
- Pipeline owns ALL phase knowledge — cmd layer only configures and starts

**Risk:** Large refactor touching the core execution path. Needs thorough testing.

---

### 2. Wire the Flywheel (PIPE-1, DATA-2, DATA-4)

**Goal:** Close the feedback loop so `moonbase flywheel` provides real insights.

**Design work needed:**
- Decide what metrics to capture per phase (duration, tokens, outcome, risk)
- Add schema versioning before first real data is written
- Design log rotation strategy
- Decide: should flywheel data feed back into prompt context for future missions?

**Risk:** Low execution risk but needs product decisions about what to measure.

---

### 3. Allowlist-Based Command Security (SEC-1)

**Goal:** Replace bypassable blocklist with robust allowlist.

**Design work needed:**
- Define the complete set of safe commands for hook execution
- Decide granularity: base command only, or command + allowed flags?
- Handle edge cases: git subcommands, piped commands, shell builtins
- Consider: should the allowlist be configurable per-project?

**Risk:** Too restrictive breaks legitimate use cases. Needs user testing.

---

### 4. Decompose TUI God Object (ARCH-2, ARCH-5, CQ-4)

**Goal:** App struct becomes a container of focused sub-models.

**Design work needed:**
- Identify state boundaries: Dashboard, Pipeline, Comms, FileBrowser, Terminal
- Design message routing between sub-models
- `CommsState` is already extracted — use as template
- Define interfaces for cross-model communication

**Risk:** Large TUI refactor. Bubbletea's Elm architecture supports this well, but testing nested models requires careful design.

---

### 5. Centralize Path Management (DATA-3, CQ-3)

**Goal:** Single `internal/paths` package for all filesystem locations.

**Design work needed:**
- Audit all 14 files that construct paths
- Resolve the inconsistency: should everything live under `~/.moonbase/` or split config/data?
- Follow XDG Base Directory spec? (`XDG_DATA_HOME`, `XDG_CONFIG_HOME`)
- Replace `init()` patterns with explicit initialization

**Risk:** Low risk but touches many files. Can be done incrementally.

---

## Positive Findings

Not everything is a gap. These patterns should be replicated:

1. **`SafeEnv()` comments** — excellent security-boundary documentation explaining WHY each env var is included and what should NEVER be added
2. **`ComposePrompt()` design** — deep module with simple 3-param interface hiding 145 lines of security, caching, and size logic
3. **C4 diagrams** — Context, Container, and Component diagrams exist in Mermaid format (rare for a project this size)
4. **`Discover()` module** — genuinely deep module providing powerful functionality through a simple interface
5. **Backend abstraction** — `Preferred()` function effectively "defines errors out of existence" for the main use case

---

*Generated by gap-analysis pipeline. This is a working document — update as gaps are resolved.*
