# Moonbase Research-Driven Improvements

**Date:** 2026-07-29  
**Scope:** All 10 books in `research/` cross-referenced against the post-refactor codebase

This document consolidates findings from a systematic study of all 10 books in the moonbase research library (Clean Architecture, Clean Code, The Clean Coder, A Philosophy of Software Design, The Pragmatic Programmer, The C4 Model, Designing Data-Intensive Applications, The Phoenix Project, Black Hat Go, and The MMIX Supplement) applied against the current codebase. Findings already implemented during this session are excluded from the active backlog and listed separately for reference. Each finding is deduplicated across sources — where multiple books flag the same issue, the strongest articulation is kept under a single ID.

---

## Execution Status (updated 2026-07-30)

**21 of 24 findings implemented**, across verified batches (each: build + vet +
`go test -race` + lint + per-finding QA). Plus a subsequence fuzzy agent-search UX
upgrade (not a numbered finding) and an O(1) agent-registry index.

**Done:** F-01, F-03, F-04, F-06, F-07, F-08, F-09, F-10, F-12, F-13, F-14, F-15, F-16,
F-17, F-18, F-19, F-20, F-21, F-22, F-23, F-24.

- **F-09** — graceful SIGINT/SIGTERM shutdown: `signal.NotifyContext` in
  `runMission`/`runMissionFast`, threaded through `deployToBackend`; on interrupt it
  marks the phase interrupted, writes an `"interrupted"` flywheel entry, saves a
  checkpoint, and prints a `moonbase replay <id>` hint. (commit 1e4398a)
- **F-20** — file-based mission WIP lock at `~/.moonbase/mission.lock` (PID + start,
  0600), liveness via `syscall.Kill(pid,0)`, stale/corrupt takeover, `--force` flag.
  (commit 1e4398a)

**Remaining (deferred — all MED, Medium-effort, architectural refactors):**
- **F-02** — full `mission.go` SRP extraction. `runMission` and `runMissionFast` still
  share a large phase loop; extract a single `runPipelineLoop(...)` used by both, and
  move `deployToBackend`/`fallbackDeploy` into `internal/backend`. Partially eased by
  F-04/F-13 already. (Effort: M)
- **F-11** — `ForPhase` phase-dependency spec (`internal/pipeline/context.go`): replace
  the hard-coded per-phase switch with a declarative `PhaseInputSpec` (required prior
  phases + max lengths) so adding/reordering phases is a data change, not code. (Effort: M)
- **F-05** — full `Pipeline` struct encapsulation (Out of Scope in the section below;
  33+ direct field-access sites — establish behaviour methods on new code only). (Effort: M)

### Continuing in a fresh session
This doc is the source of truth for what's left. To resume: `cd` into this repo and run
`kiro-cli chat --resume` (same directory), or `/chat load <file>` if you exported with
`/chat save`. Then point the new session at the three remaining items above (start with
F-02, then F-11). All work is committed on `main`.

---

## Summary

| Metric | Count |
|--------|-------|
| **Total findings** | 24 |
| **HIGH severity** | 8 |
| **MED severity** | 11 |
| **LOW severity** | 5 |

| Category | Count |
|----------|-------|
| Architecture | 6 |
| Code Quality | 5 |
| Data/Reliability | 4 |
| Security | 3 |
| Operations/DevOps | 2 |
| Documentation | 3 |
| Testing | 1 |

---

## Per-Book Findings

### Clean Architecture (Robert C. Martin)

#### F-01 | OCP Violation — Kimi Duplicates OpenAI's Entire SSE Parser

| Field | Detail |
|-------|--------|
| **Principle** | Open-Closed Principle (Ch. 8) + Information Hiding (POSD) |
| **Location** | `internal/backend/kimi.go` (lines 60–113) vs `internal/backend/openai.go` (lines 98–148) |
| **Current state** | `Kimi.Deploy()` is a near-verbatim copy of `OpenAI.Deploy()`'s SSE parsing logic (~50 lines). Only differences: base URL, env var, default model. The comment acknowledges wire-protocol identity yet duplicates the code. |
| **Recommendation** | Extract `streamChatCompletion(client *http.Client, url, apiKey, model, composed, task string) (string, error)` into `internal/backend/openai_stream.go`. Both backends become 3-line callers. Future OpenAI-compatible backends (Azure, LM Studio) become trivial. |
| **Severity** | MED |
| **Effort** | S |

#### F-02 | SRP Violation — `mission.go` Mixes Orchestration, Deployment, and Phase-Specific Logic

| Field | Detail |
|-------|--------|
| **Principle** | SRP (Ch. 7) + DRY / Common Closure (Ch. 13) + Functions do one thing (Clean Code Ch. 3) |
| **Location** | `cmd/moonbase/mission.go` (452 lines) — `runMission`, `runMissionFast`, `deployToBackend`, `executeAndRecordPhase` |
| **Current state** | The file serves three actors: mission orchestrator, backend deployer, and phase-specific injector. `runMission` and `runMissionFast` share ~85 lines of structural duplication. `executeAndRecordPhase` has hidden temporal coupling (phase 3/4 special-casing). |
| **Recommendation** | (1) Extract shared pipeline loop into `runPipelineLoop(p, reg, ctx, flywheel, opts)`. (2) Move `deployToBackend`/`fallbackDeploy` to `internal/backend/deploy.go`. (3) Replace phase-number checks with a `PhaseHooks` mechanism or `ctx.EnrichForPhase(num)`. |
| **Severity** | MED |
| **Effort** | M |

#### F-03 | DRY Violation — Agent Display Order Defined in Two Places

| Field | Detail |
|-------|--------|
| **Principle** | DRY / Common Closure Principle (Ch. 13) |
| **Location** | `internal/agents/registry.go` line 134 (`var agentOrder`) AND `internal/config/config.go` line 66 (`DefaultConfig().AgentOrder`) |
| **Current state** | The canonical agent display order is hardcoded independently in both locations. Adding/renaming an agent requires updating both. |
| **Recommendation** | Define single `var DefaultAgentOrder = [...]string{...}` in `agents` package. Config imports it: `AgentOrder: agents.DefaultAgentOrder[:]`. Registry references it. One change point. |
| **Severity** | LOW |
| **Effort** | S |

---

### Clean Code (Robert C. Martin)

#### F-04 | Error Suppression — Systematic `_, _ :=` Pattern in Production Code

| Field | Detail |
|-------|--------|
| **Principle** | Ch. 7 (Error Handling) — "Don't return null." Production standards say "Never `_ = fn()`" |
| **Location** | `cmd/moonbase/mission.go` lines 109–110, `cmd/moonbase/init.go` line 15, `cmd/moonbase/main.go` line 271 — 8 occurrences in non-test files |
| **Current state** | `os.Getwd()` and `os.UserHomeDir()` errors are discarded. If Getwd fails (deleted cwd, permissions), the pipeline runs with empty context — a silent failure that produces confusing results. Violates the project's own steering rule. |
| **Recommendation** | Create `func mustGetwd() string` and `func mustUserHomeDir() string` helpers that `fmt.Fprintf(os.Stderr, ...)` + `os.Exit(1)` on error. Replace all bare `_, _ :=` calls. Turns silent corruption into loud, obvious failure. |
| **Severity** | HIGH |
| **Effort** | S |

#### F-05 | Pipeline Struct Exposes All Fields — Law of Demeter Violation

| Field | Detail |
|-------|--------|
| **Principle** | Ch. 6 (Objects vs Data Structures) — "Objects hide data behind abstractions" |
| **Location** | `internal/pipeline/pipeline.go` (all-public fields), consumed in `cmd/moonbase/mission.go` (33 direct field accesses) |
| **Current state** | The `cmd` layer directly reads/writes `p.Context.RiskLevel`, `p.Phases[i].Status`, `p.Context.Diff`, etc. Impossible to change pipeline internals without modifying the CLI. |
| **Recommendation** | Introduce behavior methods: `p.RecordDiff(diff)`, `p.CurrentRiskLevel()`, `p.MarkPhaseComplete(num, output)`. Confine field access to the `pipeline` package. Establish pattern for new code; migrate incrementally. |
| **Severity** | MED |
| **Effort** | M |

#### F-06 | Presentation Logic in Domain Package (Dead TODO)

| Field | Detail |
|-------|--------|
| **Principle** | Dependency Rule (Clean Arch Ch. 22) + Comments (Clean Code Ch. 4) — "Don't use a comment when you can use a function" |
| **Location** | `internal/pipeline/pipeline.go`, `statusIcon()` (line ~229) and `StatusSummary()` (line ~220) |
| **Current state** | Pipeline (core domain) contains emoji mapping and human-readable formatting. A NOTE comment acknowledges this should move but no issue tracks it. Steering says "No TODOs." |
| **Recommendation** | Move `statusIcon()` and `StatusSummary()` to `internal/tui/helpers_pipeline.go`. Pipeline exposes only `Phase.Status` enum. Remove the NOTE comment. |
| **Severity** | LOW |
| **Effort** | S |

---

### The Clean Coder (Robert C. Martin)

#### F-07 | No Integration Tests for CLI-to-Pipeline Mission Path

| Field | Detail |
|-------|--------|
| **Principle** | Ch. 8 (Testing Strategies) — Test Automation Pyramid requires coverage at every layer |
| **Location** | `cmd/moonbase/mission.go` (`runMission`, `runMissionFast`, `executeAndRecordPhase`) |
| **Current state** | CLI tests cover flag parsing and cobra wiring but `runMission()` has no integration test validating end-to-end flow: create pipeline → mock backend → execute phases → risk gate → checkpoint. This is the primary user-facing code path. |
| **Recommendation** | Add `cmd/moonbase/mission_integration_test.go` with `//go:build integration` tag. Exercise: 5-phase LOW→completion, fast 2-phase, MEDIUM→rework loop, CRITICAL→stop. Mock backend already exists. |
| **Severity** | HIGH |
| **Effort** | M |

#### F-08 | PhaseTimeout Race — Retry Budget Exceeds Phase Deadline

| Field | Detail |
|-------|--------|
| **Principle** | Ch. 10 (Estimation) — estimates are distributions; compound errors across sequential tasks |
| **Location** | `internal/pipeline/pipeline.go` line 96 (`PhaseTimeout: 5*time.Minute`), `cmd/moonbase/mission.go` line 297 (360s retry budget) |
| **Current state** | All phases get flat 5-minute timeout. `deployToBackend` uses 120s × 3 attempts = 360s, which exceeds PhaseTimeout (300s). Phase can timeout at 300s while retry still has budget — causing confusing double-failures. |
| **Recommendation** | (a) Per-phase timeout config: `PhaseTimeouts map[int]time.Duration`. (b) Pass phase timeout as context deadline to `deployToBackend` — retry budget derived from remaining deadline, not independently computed. |
| **Severity** | MED |
| **Effort** | S |

#### F-09 | No Graceful Shutdown on SIGINT — Checkpoint/Flywheel Data Loss

| Field | Detail |
|-------|--------|
| **Principle** | Ch. 11 (Pressure) — "Don't Panic. Slow down." Handle interrupts gracefully. |
| **Location** | `cmd/moonbase/mission.go`, `runMission()` — no signal handling |
| **Current state** | Ctrl+C during a mission terminates immediately. Flywheel entry for in-progress phase never written, no checkpoint saved, backend subprocess may be orphaned. TUI has `PipelineAbortedMsg` but CLI path doesn't wire signals. |
| **Recommendation** | Add `signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)` at top of `runMission`. On cancellation: mark phase failed, save checkpoint with "interrupted" status, write flywheel entry. Enables graceful `moonbase replay`. |
| **Severity** | MED |
| **Effort** | M |

---

### A Philosophy of Software Design (John Ousterhout)

#### F-10 | Risk Gate Parsing — Define Errors Out of Existence via Structured Meta

| Field | Detail |
|-------|--------|
| **Principle** | Define Errors Out of Existence (Ch. 10) |
| **Location** | `internal/pipeline/riskgate.go`, `extractRiskLevel()` (lines 56–107) |
| **Current state** | Uses 3 fragile strategies (regex-based) to parse risk level from free-form QA output. Falls through to `RiskUnknown` (treated as MEDIUM) on format mismatch. Meanwhile `internal/pipeline/meta.go` (`ParseMeta`) already extracts structured `__moonbase_meta` JSON blocks — but isn't used as primary path in risk parsing. |
| **Recommendation** | At top of `ParseRiskGate`, add: `if meta := ParseMeta(qaOutput); meta != nil && meta.Risk != "" { return matchRiskLevel(meta.Risk) }`. The regex strategies become a fallback for non-conforming agents. Eliminates the most common failure mode. |
| **Severity** | HIGH |
| **Effort** | S |

#### F-11 | `ForPhase` Hard-Codes Phase Dependencies — Shallow Module

| Field | Detail |
|-------|--------|
| **Principle** | Deep Modules / Information Hiding |
| **Location** | `internal/pipeline/context.go`, `ForPhase()` (lines 48–100) |
| **Current state** | Giant switch-case hard-codes which prior phases each phase needs. Adding a phase or changing information flow requires editing this function. Phase ordering knowledge leaks into context assembly. |
| **Recommendation** | Define `PhaseInputSpec` per phase (e.g., `[]int` of required prior phases + max lengths). Store in Phase struct. `ForPhase` becomes a generic loop over the spec. Single change point for phase dependency changes. |
| **Severity** | MED |
| **Effort** | M |

---

### The Pragmatic Programmer (David Thomas, Andrew Hunt)

#### F-12 | Broken Contract — `Discover()` Returns Error It Never Produces

| Field | Detail |
|-------|--------|
| **Principle** | Design by Contract / Assertive Programming |
| **Location** | `internal/discovery/discovery.go`, `Discover()` (lines 69–100) |
| **Current state** | Signature returns `(*ProjectContext, error)` but always returns `(ctx, nil)`. Sub-discoveries swallow errors. Callers correctly ignore the error (`ctx, _ := discovery.Discover(cwd)`) but the contract is dishonest. |
| **Recommendation** | Change signature to `Discover(dir string) *ProjectContext` (infallible by design). Optionally accumulate warnings in `ProjectContext.Warnings []string`. Honest API contract. |
| **Severity** | LOW |
| **Effort** | S |

#### F-13 | `deployToBackend` Hard-Codes Kiro — Not Backend-Orthogonal

| Field | Detail |
|-------|--------|
| **Principle** | Orthogonality / Decoupling |
| **Location** | `cmd/moonbase/mission.go`, `deployToBackend()` (lines 412–450) |
| **Current state** | Creates `&backend.Kiro{TrustTools: true}` directly. Other backends (OpenAI, Anthropic, Ollama) reachable from `moonbase deploy` via `backend.Preferred()` but NOT from mission pipeline. The mission loop is coupled to a specific backend. |
| **Recommendation** | Use `backend.Preferred()` or accept a `Backend` interface parameter injected at mission start. Apply retry uniformly regardless of backend. Makes `moonbase mission` work with any configured backend. |
| **Severity** | HIGH |
| **Effort** | M |

---

### The C4 Model (Simon Brown)

#### F-14 | Missing Dynamic Diagram — Pipeline Execution Flow

| Field | Detail |
|-------|--------|
| **Principle** | Dynamic Diagram for Runtime Behavior (Ch. 7) |
| **Location** | `docs/architecture.md` — has Levels 1–3 static diagrams, no dynamic diagram |
| **Current state** | Architecture doc doesn't illustrate the runtime behavior of mission execution (phase sequencing, risk gate loops, conditional parallel phases). This is complex temporal behavior not inferable from static diagrams. |
| **Recommendation** | Add a Mermaid sequence diagram showing: User→CLI→Pipeline→Backend→Agent roundtrip, risk gate decision with rework loop, conditional phase parallel execution. ~40 lines. |
| **Severity** | HIGH |
| **Effort** | S |

#### F-15 | Missing Deployment Diagram — Filesystem Distribution Model

| Field | Detail |
|-------|--------|
| **Principle** | Deployment Diagram (Ch. 8) |
| **Location** | `docs/architecture.md` — no deployment/distribution diagram |
| **Current state** | No diagram shows how binary, agents, config, and data files distribute across the user's filesystem (binary in PATH, ~/.moonbase/, project/.kiro/). |
| **Recommendation** | Add C4 deployment diagram: Developer Machine → binary, ~/.moonbase/ (agents, config, flywheel, checkpoints), project/.kiro/ (specs, steering, skills), network calls to AI backends. ~30 lines Mermaid. |
| **Severity** | MED |
| **Effort** | S |

#### F-16 | Model-Code Gap — Prompt Composer Shown as Separate Container

| Field | Detail |
|-------|--------|
| **Principle** | Model-Code Gap (Ch. 11) |
| **Location** | `docs/architecture.md` Container Diagram lists "Prompt Composer" as separate container |
| **Current state** | In code it's just `discovery.ComposePrompt()` — a single function in the discovery package, not a separate package. Diagram overstates its architectural significance. |
| **Recommendation** | Demote Prompt Composer to a component within the Discovery container in the diagram. Update relationship arrows. 5-line edit. |
| **Severity** | LOW |
| **Effort** | S |

---

### Designing Data-Intensive Applications (Martin Kleppmann)

#### F-17 | No Schema Versioning on Persisted Data Structs

| Field | Detail |
|-------|--------|
| **Principle** | Ch. 4 (Encoding and Evolution) — persisted formats need schema version for forward/backward compat |
| **Location** | `internal/pipeline/flywheel.go` (`FlywheelEntry`), `internal/pipeline/checkpoint.go` (`Checkpoint`), `internal/history/history.go` (`Mission`) |
| **Current state** | None carry a `"v"` field. Flywheel is append-only (never rewritten), so old entries persist forever. Adding a field risks silent data loss (unparseable entries skipped). Checkpoint and history have same issue. |
| **Recommendation** | Add `SchemaVersion int \`json:"v"\`` (starting at 1) to all three structs. On read, branch on version. Document evolution contract: "new fields optional; removing/renaming bumps version." ~30 lines total. |
| **Severity** | HIGH |
| **Effort** | S |

#### F-18 | Flywheel Append Lacks `file.Sync()` — Crash Durability Gap

| Field | Detail |
|-------|--------|
| **Principle** | Ch. 7 (Transactions) — durability requires fsync |
| **Location** | `internal/pipeline/flywheel.go` → `Append()` (lines 49–62) |
| **Current state** | Opens with `O_APPEND|O_WRONLY`, writes JSON + newline, closes. No `file.Sync()` call — OS crash can lose the last entry. Single-user CLI makes concurrent writes unlikely but not impossible in CI. |
| **Recommendation** | Add `file.Sync()` before `file.Close()`. Optionally add doc comment: "single-writer assumed; no file locking." 1–2 lines. |
| **Severity** | MED |
| **Effort** | S |

#### F-19 | Checkpoint Writes Are Not Atomic — Defeats Crash Recovery Purpose

| Field | Detail |
|-------|--------|
| **Principle** | Ch. 3 (Storage) — temp-then-rename prevents half-written files |
| **Location** | `internal/pipeline/checkpoint.go` → `SaveCheckpoint()` (line 36: `os.WriteFile`) |
| **Current state** | Uses `os.WriteFile` directly. Unlike `writeHistory()` (which correctly uses temp+rename), a crash mid-write produces a truncated JSON blob. Next `LoadCheckpoint` fails with unmarshal error — mission state lost. The checkpoint exists FOR crash recovery; non-atomic write defeats its purpose. |
| **Recommendation** | Apply temp-then-rename pattern from `writeHistory()`: write to `path + ".tmp"`, then `os.Rename()`. 5-line change, internal consistency. |
| **Severity** | HIGH |
| **Effort** | S |

---

### The Phoenix Project (Gene Kim, George Spafford, Kevin Behr)

#### F-20 | No WIP Limit — Multiple Missions Can Run Simultaneously

| Field | Detail |
|-------|--------|
| **Principle** | Second Way (Feedback) + Theory of Constraints — visualize and limit WIP |
| **Location** | `cmd/moonbase/mission.go` → `runMission()` |
| **Current state** | Nothing prevents multiple simultaneous `moonbase mission` invocations. No lock file, no queue. Multiple terminals hitting the same backend exhausts rate limits or token budgets. Flywheel records after-the-fact but provides no flow control. |
| **Recommendation** | Implement file-based mission lock (`~/.moonbase/mission.lock` with PID + start time). If locked: "Mission in progress (PID 1234, 3m ago). Use --force to override." Simplest WIP=1 constraint. |
| **Severity** | MED |
| **Effort** | M |

#### F-21 | Flywheel Lacks Lead-Time Metrics

| Field | Detail |
|-------|--------|
| **Principle** | The Three Ways — measure lead time, not just work time |
| **Location** | `cmd/moonbase/flywheel_cmd.go`, `internal/pipeline/flywheel.go` → `FlywheelEntry` |
| **Current state** | Records `DurationMs` per phase (work time) but not gap between phases (wait/queue time). Cannot show total mission lead time or identify true bottleneck. |
| **Recommendation** | Add `MissionStartedAt` timestamp to first entry per trace. Derive lead time from first/last entries sharing a `TraceID`. `moonbase flywheel` reports average mission lead time and longest phase. |
| **Severity** | LOW |
| **Effort** | S |

---

### Black Hat Go (Steele, Patten, Kottmann)

#### F-22 | Updater HTTP Client Allows TLS < 1.2

| Field | Detail |
|-------|--------|
| **Principle** | Ch. 11 (Cryptography) — enforce minimum TLS; updater downloads executable code |
| **Location** | `internal/updater/updater.go` → `downloadFile()` (line 196) |
| **Current state** | Uses `&http.Client{Timeout: downloadTimeout}` with default transport — no `TLSClientConfig`. Unlike OpenAI/Anthropic clients (which set `MinVersion: tls.VersionTLS12`), the updater allows TLS 1.0/1.1. Checksum also downloaded over same weak connection, so MITM can substitute both. |
| **Recommendation** | Apply same `TLSClientConfig{MinVersion: tls.VersionTLS12}` pattern from `openai.go`. Define dedicated `updaterHTTPClient` with connection-phase timeouts. ~15 lines. |
| **Severity** | HIGH |
| **Effort** | S |

#### F-23 | SSE Parsers Have No Total Response Size Bound — OOM Risk

| Field | Detail |
|-------|--------|
| **Principle** | Ch. 4 (HTTP) — network parsers must bound memory usage |
| **Location** | `internal/backend/openai.go` → `Deploy()` SSE loop, `internal/chat/stream.go` → `streamFrom()` |
| **Current state** | `bufio.NewScanner` has 64KB per-line limit (safe), but total accumulated `result.String()` is unbounded. A malicious OpenAI-compatible endpoint (user-configurable `OPENAI_BASE_URL`) sending millions of small chunks could OOM. |
| **Recommendation** | Add `const maxResponseSize = 10 << 20` (10MB). Check `result.Len()` on each append; break with truncation error if exceeded. Mirrors `maxBinarySize` in updater. ~5 lines per backend. |
| **Severity** | MED |
| **Effort** | S |

#### F-24 | `splitOnOperators` — `||` Handling Is Accidentally Correct

| Field | Detail |
|-------|--------|
| **Principle** | Ch. 2 (Parsers) — incomplete metacharacter handling creates bypass risk |
| **Location** | `internal/tui/helpers_commands.go` → `splitOnOperators()` (lines 139–150) |
| **Current state** | Doc says "splits on &&, ||, ;, and |" but `||` is NOT explicitly replaced — it's caught by the `|` split producing an empty string filtered by `p != ""`. Accidentally correct but fragile to refactoring. |
| **Recommendation** | Add `cmd = strings.ReplaceAll(cmd, "||", "\x00")` before the `|` split. Add test case for `cmd1 || dangerous_cmd`. Matches documented intent. |
| **Severity** | LOW |
| **Effort** | S |

---

### The MMIX Supplement (Martin Ruckert)

#### (Subsumed — No Standalone Findings)

The MMIX Supplement's relevant insight (O(n) history I/O) is captured as part of DDIA findings. The book's focus on low-level algorithmic efficiency at the instruction level does not yield additional actionable findings against this Go CLI codebase — the performance-relevant observation about `history.go` linear scans is attributed to Knuth's sorting/searching principles but the recommendation aligns with DDIA's log-structured approach and is already covered by the schema evolution finding (F-17). The history O(n) issue is tracked below as a secondary concern within F-17's scope.

**Note:** History's read-modify-write pattern (`internal/history/history.go` → `Save()`, `Load()`, `GetByID()`) is O(n) per operation. Currently bounded by realistic usage (~100 missions). When schema versioning (F-17) is implemented, consider migrating history to JSONL (append-only) for consistency with flywheel pattern. This is captured as future work within F-17, not a separate finding.

---

### The Pragmatic Programmer — Additional Finding (Unplanned Work)

This finding from The Phoenix Project (work-type tracking) is low-severity and grouped with Phoenix findings above (F-21). No additional standalone Pragmatic Programmer findings beyond those already merged into F-02 and F-12.

---

## Prioritized Backlog

Top findings ordered by severity (HIGH first), then effort (S before M):

| # | ID | Title | Severity | Effort |
|---|----|-------|----------|--------|
| 1 | F-04 | Error suppression — `_, _ :=` pattern in production code | HIGH | S |
| 2 | F-10 | Risk gate: use structured meta as primary path | HIGH | S |
| 3 | F-17 | Schema versioning on persisted data structs | HIGH | S |
| 4 | F-19 | Checkpoint writes not atomic (temp+rename) | HIGH | S |
| 5 | F-22 | Updater TLS minimum version enforcement | HIGH | S |
| 6 | F-14 | Dynamic diagram — pipeline execution flow | HIGH | S |
| 7 | F-07 | Integration tests for CLI mission path | HIGH | M |
| 8 | F-13 | `deployToBackend` hard-codes Kiro backend | HIGH | M |
| 9 | F-01 | Extract shared SSE streaming function | MED | S |
| 10 | F-08 | Phase timeout race — retry exceeds deadline | MED | S |

---

## Already Addressed (for reference)

The following improvements were implemented during this research session and are NOT included in the backlog above:

- **Architecture documentation** — `docs/architecture.md` created with C4 System Context, Container, and Component diagrams (Levels 1–3)
- **Contributor documentation** — project structure, package purposes, and build/test workflows documented
- **Knowledge base indexing** — all 10 research books indexed for agent-accessible search
- **Research methodology** — systematic cross-reference process established (3 parallel studies covering all 10 books)

---

## Explicitly Out of Scope / Won't Fix

| Item | Rationale |
|------|-----------|
| **History migration to JSONL** | Current O(n) is bounded by realistic usage (~100 missions). Complexity of migration + backward compat outweighs benefit today. Revisit if history exceeds 500 entries. Tracked as future work within F-17. |
| **TLS certificate pinning for updater** | Minimum TLS version (F-22) provides sufficient protection. Pinning adds brittleness (cert rotation) without proportional security gain for a CLI tool downloading from GitHub. |
| **Flywheel file locking (flock)** | Single-user CLI model makes concurrent writes near-impossible. Adding `Sync()` (F-18) handles crash durability. A doc comment noting "single-writer assumed" is sufficient. WIP lock (F-20) prevents the scenario at a higher level. |
| **Phoenix "unplanned work" `--type` flag** | Nice-to-have for long-term analytics but low signal-to-noise today. Would add CLI surface area for minimal benefit at current project scale. |
| **Full Pipeline struct encapsulation** | F-05 recommends incremental migration. Full rewrite to hide all fields would touch 33+ access sites — too disruptive for current value. Establish pattern on new code only. |
