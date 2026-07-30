# Requirements: Parallel Fan-Out Execution of Independent Conditional Specialists

## Overview

Conditional specialists (Numbuh 274 Security, Numbuh 999 Docs, Numbuh 362 Infra, Numbuh 0 Oversight, Numbuh 86 Dead-code, Sector Z Legacy) currently execute sequentially after QA. Each performs read-only analysis of the implementer's output — they share no mutable state and produce independent reports. Sequential execution wastes wall-clock time when multiple specialists trigger simultaneously.

This spec introduces bounded-concurrency fan-out for independent specialists, following the 2026 fan-out orchestration pattern (errgroup with semaphore, deterministic aggregation, partial-failure tolerance). Specialists that trigger after Phase 4 (QA) run concurrently, their outputs merge into the pipeline context in a stable order, and the risk gate + flywheel telemetry remain intact.

---

## User Stories

### US-1: Faster Multi-Specialist Missions
As a developer running a mission that triggers 3+ specialists, I want them to execute concurrently so that pipeline wall-clock time scales with the slowest specialist, not their sum.

### US-2: Deterministic Pipeline Output
As a developer reviewing mission history or replaying a pipeline, I want specialist outputs to appear in a consistent, reproducible order regardless of which specialist finishes first, so that diffs and comparisons are meaningful.

### US-3: Resilient Partial Execution
As a developer, I want one specialist failing (timeout, backend error) to not block the other specialists' results from merging into the pipeline context, so that useful analysis is preserved.

### US-4: Controllable Concurrency
As an operator on a resource-constrained machine (or using a rate-limited API backend), I want to configure the maximum number of concurrent specialists (or disable fan-out entirely), so that I can avoid overwhelming the backend or saturating the CPU.

### US-5: Preserved Risk Semantics
As a developer relying on the risk gate, I want the QA risk classification (which gates entry to specialists) and the final review gate to behave identically whether specialists run in parallel or sequentially, so that safety is not degraded.

---

## Acceptance Criteria

### AC-1: Independence Classification

#### AC-1.1: Read-Only Specialist Identification
- **WHEN** the pipeline evaluates which specialists to fan out
- **THEN** only specialists marked as independent (read-only analysis, no file writes, no shared mutable state) are eligible for parallel execution
- **SHALL** include: Numbuh 274, Numbuh 999, Numbuh 362, Numbuh 0, Numbuh 86, Sector Z
- **SHALL NOT** include any specialist whose agent frontmatter declares `shell.read_only: false` or `tools` containing `write`

#### AC-1.2: Independence Validated at Pipeline Build Time
- **WHEN** the pipeline is constructed (via `New()` or `NewFast()`)
- **THEN** the set of parallelizable specialists is computed from agent metadata, not hardcoded
- **SHALL** be derivable from agent frontmatter fields (`shell.read_only`, `tools` list)

---

### AC-2: Bounded Concurrent Execution

#### AC-2.1: Fan-Out After QA
- **WHEN** Phase 4 (QA) completes with `RiskLow` (no rework loop)
- **AND** two or more independent specialists are triggered
- **THEN** all triggered specialists begin execution concurrently (up to the concurrency cap)
- **SHALL** use `errgroup`-style bounded concurrency with a semaphore of configurable size

#### AC-2.2: Concurrency Cap Respected
- **WHEN** N specialists trigger and the concurrency cap is M (where N > M)
- **THEN** at most M specialists execute simultaneously; remaining queue and start as slots free
- **SHALL** default to `max_specialist_concurrency: 4` in config

#### AC-2.3: Context Propagation
- **WHEN** the pipeline is aborted (user presses Esc, context cancelled)
- **THEN** all in-flight specialist goroutines receive cancellation and terminate within PhaseTimeout
- **SHALL** reuse the existing `Pipeline.Ctx` / `Pipeline.Cancel` mechanism from `startNextPhase`

---

### AC-3: Deterministic Output Aggregation

#### AC-3.1: Stable Merge Order
- **WHEN** multiple specialists complete (in any order)
- **THEN** their outputs are recorded into `PipelineContext.PhaseOutputs` in a fixed order: phase number ascending (6, 7, 8, ..., Sector Z last)
- **SHALL** produce identical `PhaseOutputs` map entries regardless of completion order

#### AC-3.2: Unified Files-Changed Merge
- **WHEN** specialist outputs mention files
- **THEN** `PipelineContext.FilesChanged` is extended by de-duplicating file references from all specialist outputs in the same stable order
- **SHALL NOT** produce different `FilesChanged` lists across replays of the same checkpoint

---

### AC-4: Risk Gate and Pipeline Flow Preservation

#### AC-4.1: Risk Gate Precedes Fan-Out
- **WHEN** Phase 4 (QA) produces a risk verdict
- **THEN** the risk gate is applied BEFORE any specialists begin
- **SHALL** route to rework/redesign/stop as before; fan-out only activates on `RiskLow`

#### AC-4.2: Review Phase Receives All Specialist Outputs
- **WHEN** fan-out completes (all specialists done or failed)
- **THEN** Phase 5 (Review / Numbuh 5) receives the aggregated specialist context via `ForPhase(5)`
- **SHALL** include each specialist's output (or failure notice) in the input composed for review

---

### AC-5: Checkpointing and Flywheel Telemetry

#### AC-5.1: Checkpoint After Fan-Out
- **WHEN** the fan-out phase completes (all specialists resolved)
- **THEN** a checkpoint is saved capturing all specialist phase statuses and outputs
- **SHALL** persist partial results (completed specialists' outputs) even if some failed

#### AC-5.2: Per-Specialist Flywheel Entries
- **WHEN** each specialist completes or fails
- **THEN** a `FlywheelEntry` is appended with the specialist's phase, agent, outcome, duration, and output size
- **SHALL** record entries for all specialists (including failures) with outcome "complete", "failed", or "skipped"
- **SHALL** include a new `parallel_group` field linking entries from the same fan-out batch

---

### AC-6: Partial-Failure Handling

#### AC-6.1: One Failure Does Not Block Others
- **WHEN** specialist A errors (timeout, backend failure) while specialists B and C are still running
- **THEN** B and C continue to completion and their outputs merge normally
- **SHALL NOT** cancel other specialists due to one specialist's failure

#### AC-6.2: Failures Surfaced, Not Fatal
- **WHEN** one or more specialists fail during fan-out
- **THEN** failed specialists are recorded with `StatusFailed` and a summary in the pipeline chat
- **AND** the pipeline continues to Phase 5 (Review) with available results
- **SHALL NOT** mark the overall pipeline as failed due to specialist failure
- **SHALL** include failure details in the context provided to Numbuh 5

#### AC-6.3: All Specialists Fail
- **WHEN** every triggered specialist fails
- **THEN** the pipeline still advances to Phase 5 with a warning in the chat
- **SHALL** log the failure batch to flywheel with outcome "all_specialists_failed"

---

### AC-7: Configuration

#### AC-7.1: Concurrency Cap in Config
- **WHEN** `~/.config/moonbase/config.yaml` contains `max_specialist_concurrency: N`
- **THEN** the fan-out phase uses N as the semaphore size
- **SHALL** default to 4 when not specified
- **SHALL** accept values 1–16 (1 effectively disables parallelism, restoring sequential)

#### AC-7.2: Disable Fan-Out
- **WHEN** config contains `parallel_specialists: false`
- **THEN** specialists execute sequentially as before (backward-compatible behavior)
- **SHALL** default to `true` (parallel enabled)

#### AC-7.3: Per-Mission Override
- **WHEN** `moonbase mission --sequential "task"` is passed
- **THEN** fan-out is disabled for that mission regardless of config
- **SHALL** be reflected in the dry-run output

---

## Scope

### In Scope
- Classifying specialist independence from agent metadata
- Bounded-concurrency fan-out orchestration in `internal/pipeline/`
- Deterministic output aggregation into `PipelineContext`
- Checkpoint persistence of fan-out results
- Flywheel telemetry per specialist
- Config fields for concurrency cap and disable toggle
- TUI chat messages showing parallel execution status
- `--sequential` flag on `moonbase mission`

### Out of Scope
- Parallelizing mandatory phases (1–5) — these have data dependencies
- Cross-specialist communication during fan-out
- Distributed execution across multiple machines
- Dynamic specialist priority or ordering beyond phase number
- Streaming partial specialist output to the TUI during execution
- New specialist agents or changes to existing agent content

---

## Dependencies

| Dependency | Version | Purpose | Status |
|------------|---------|---------|--------|
| `golang.org/x/sync/errgroup` | latest | Bounded-concurrency goroutine coordination | New (stdlib-adjacent) |
| `sync` (stdlib) | Go 1.26 | Mutex for aggregation buffer | Already available |
| `context` (stdlib) | Go 1.26 | Cancellation propagation | Already used |

---

## Risks

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Race condition in PipelineContext writes | Medium | High | Specialists write to separate map keys (keyed by phase number); aggregation buffer protected by mutex |
| Backend rate limiting under concurrent load | Medium | Medium | Configurable concurrency cap; default 4 is conservative |
| Non-deterministic output in tests | Low | Medium | Fixed merge order by phase number; tests verify ordering |
| Flywheel entries interleaved with other pipeline runs | Low | Low | TraceID + parallel_group field for correlation |
| TUI rendering during concurrent phase updates | Medium | Medium | Batch chat messages; emit single "fan-out complete" update |
