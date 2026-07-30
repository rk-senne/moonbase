# Tasks: Parallel Fan-Out Execution of Independent Conditional Specialists

## Milestone 1: Core Fan-Out Infrastructure

### Task 1: Add `golang.org/x/sync` dependency
- **Requirements:** AC-2.1
- **Files:** `go.mod`, `go.sum`
- **Action:** `go get golang.org/x/sync@latest`. Pin the resolved version. Although the primary implementation uses `sync.WaitGroup` + channel semaphore, this dependency provides `errgroup` for potential future use and signals intent.
- **Test:** `go mod tidy` succeeds. `go build ./...` passes.
- **Status:** pending

### Task 2: Create fan-out orchestrator
- **Requirements:** AC-2.1, AC-2.2, AC-3.1, AC-6.1
- **Files:** `internal/pipeline/fanout.go` (new)
- **Action:** Implement `FanOutResult` struct, `FanOutConfig` struct, `SpecialistFunc` type, `RunSpecialists()` function, and `TriggeredSpecialists()` helper. `RunSpecialists` uses `sync.WaitGroup` + buffered-channel semaphore for bounded concurrency. Results collected under mutex, sorted by phase number before return. Individual specialist errors do not cancel siblings.
- **Test:** `go build ./...` passes.
- **Status:** pending

### Task 3: Unit tests for fan-out orchestrator
- **Requirements:** AC-2.1, AC-2.2, AC-3.1, AC-6.1, AC-6.2, AC-6.3
- **Files:** `internal/pipeline/fanout_test.go` (new)
- **Action:** Table-driven tests covering:
  - All specialists succeed → results sorted by phase number
  - One of three fails → two complete, one failed, all returned
  - All fail → all `StatusFailed`, non-nil results slice
  - Concurrency cap = 1 → sequential execution (verify with timing)
  - Concurrency cap = 4, 6 specialists → max 4 concurrent (verify with atomic counter)
  - Context cancellation → all pending specialists get `ctx.Err()`
  - Empty phases list → returns nil
  - Deterministic ordering → run 50 iterations, verify stable order
- **Test:** `go test ./internal/pipeline/ -run TestRunSpecialists -race` passes.
- **Status:** pending

### Checkpoint: Fan-Out Core
- [ ] `RunSpecialists` compiles and returns deterministically sorted results
- [ ] Partial failure does not cancel other specialists
- [ ] Concurrency cap is respected (verified by test)
- [ ] Context cancellation propagates to all goroutines
- [ ] `go test -race ./internal/pipeline/` passes

---

## Milestone 2: Pipeline Integration

### Task 4: Add parallel config fields to Pipeline struct
- **Requirements:** AC-7.1, AC-7.2
- **Files:** `internal/pipeline/pipeline.go`
- **Action:** Add `ParallelSpecialists bool` (default `true`) and `MaxSpecialistConcurrency int` (default `4`) to `Pipeline` struct. Set defaults in `New()`. Validate concurrency range 1–16 (clamp if outside).
- **Test:** `go build ./...` passes. Existing pipeline tests still pass.
- **Status:** pending

### Task 5: Add config fields for specialist parallelism
- **Requirements:** AC-7.1, AC-7.2
- **Files:** `internal/config/config.go`
- **Action:** Add `ParallelSpecialists bool` field with yaml tag `parallel_specialists,omitempty` (default `true`). Add `MaxSpecialistConcurrency int` field with yaml tag `max_specialist_concurrency,omitempty` (default `4`). Update `DefaultConfig()`.
- **Test:** `go build ./...` passes. `go test ./internal/config/` passes. Config round-trips through YAML correctly.
- **Status:** pending

### Task 6: Add config test cases for new fields
- **Requirements:** AC-7.1, AC-7.2
- **Files:** `internal/config/config_test.go`
- **Action:** Add table-driven tests:
  - Default values: `ParallelSpecialists=true`, `MaxSpecialistConcurrency=4`
  - YAML with `parallel_specialists: false` → disables fan-out
  - YAML with `max_specialist_concurrency: 2` → sets cap to 2
  - Missing fields → defaults apply
  - Out-of-range concurrency (0, 17) → clamped to bounds
- **Test:** `go test ./internal/config/ -run TestParallelSpecialists` passes.
- **Status:** pending

### Task 7: Add `MergeSpecialistResults` to PipelineContext
- **Requirements:** AC-3.1, AC-3.2, AC-6.2
- **Files:** `internal/pipeline/context.go`
- **Action:** Add `MergeSpecialistResults(results []FanOutResult)` method. Iterates results (pre-sorted by phase number), writes output to `PhaseOutputs[phase]`, extracts files changed from successful outputs. Failed specialists get a `[SPECIALIST FAILED: <error>]` marker in their phase output slot.
- **Test:** `go test ./internal/pipeline/ -run TestMergeSpecialistResults` passes.
- **Status:** pending

### Task 8: Test deterministic merge
- **Requirements:** AC-3.1, AC-3.2
- **Files:** `internal/pipeline/context_test.go` (or existing file)
- **Action:** Test `MergeSpecialistResults` with:
  - Results arriving in various orders → always stored by phase key
  - Files extracted and de-duplicated in stable order
  - Failed results produce marker text, not empty string
  - Mixed success/failure → context has both markers and real output
- **Test:** `go test ./internal/pipeline/ -run TestMerge` passes.
- **Status:** pending

### Checkpoint: Pipeline Integration
- [ ] `Pipeline` struct carries `ParallelSpecialists` and `MaxSpecialistConcurrency`
- [ ] Config loads and saves new fields correctly
- [ ] `MergeSpecialistResults` produces deterministic output
- [ ] `go test ./internal/pipeline/` passes
- [ ] `go test ./internal/config/` passes

---

## Milestone 3: Checkpoint & Flywheel

### Task 9: Extend Checkpoint with specialist status
- **Requirements:** AC-5.1
- **Files:** `internal/pipeline/checkpoint.go`
- **Action:** Add `SpecialistResults map[int]string` field to `Checkpoint` struct (json tag: `specialist_results,omitempty`). In `SaveCheckpoint`, populate from fan-out results if present (map phase number → "complete"/"failed"). Schema version stays at 1 per evolution contract (new optional field).
- **Test:** `go test ./internal/pipeline/ -run TestCheckpoint` passes. Legacy checkpoints (without field) still load correctly.
- **Status:** pending

### Task 10: Extend FlywheelEntry with parallel_group
- **Requirements:** AC-5.2
- **Files:** `internal/pipeline/flywheel.go`
- **Action:** Add `ParallelGroup string` field to `FlywheelEntry` (json tag: `parallel_group,omitempty`). No schema version bump (new optional field). The fan-out caller sets this to `traceID + "-fanout"` for all entries in a batch.
- **Test:** `go test ./internal/pipeline/ -run TestFlywheel` passes. Existing entries without the field parse correctly.
- **Status:** pending

### Task 11: Test checkpoint and flywheel extensions
- **Requirements:** AC-5.1, AC-5.2
- **Files:** `internal/pipeline/checkpoint_test.go`, `internal/pipeline/flywheel_test.go`
- **Action:** Add tests:
  - Checkpoint with `SpecialistResults` serializes and deserializes correctly
  - Checkpoint without `SpecialistResults` (legacy) loads with nil map
  - FlywheelEntry with `ParallelGroup` marshals to JSON with the field
  - FlywheelEntry without `ParallelGroup` marshals without the field (omitempty)
- **Test:** `go test ./internal/pipeline/ -run TestCheckpoint -run TestFlywheel` passes.
- **Status:** pending

### Checkpoint: Observability
- [ ] Checkpoint includes specialist results (optional field)
- [ ] Legacy checkpoints still load without error
- [ ] Flywheel entries carry `parallel_group` for correlation
- [ ] `go test ./internal/pipeline/` passes

---

## Milestone 4: TUI Dispatch

### Task 12: Add `FanOutCompleteMsg` type
- **Requirements:** AC-2.1, AC-4.2
- **Files:** `internal/tui/pipeline_exec.go`
- **Action:** Define `FanOutCompleteMsg struct { Results []pipeline.FanOutResult }`. This carries the batch result from the fan-out goroutine back to the Elm update loop.
- **Test:** `go build ./...` passes.
- **Status:** pending

### Task 13: Implement `startFanOut` method
- **Requirements:** AC-2.1, AC-2.3, AC-4.1, AC-6.1
- **Files:** `internal/tui/pipeline_exec.go`
- **Action:** Add `(a *App) startFanOut() tea.Cmd` method. Returns a `tea.Cmd` that:
  1. Identifies triggered specialists via `pipeline.TriggeredSpecialists`
  2. If none triggered, returns `FanOutCompleteMsg{Results: nil}`
  3. Constructs `SpecialistFunc` wrapping the backend deployment
  4. Calls `pipeline.RunSpecialists` with pipeline context
  5. Returns `FanOutCompleteMsg` with results
- **Test:** `go build ./...` passes.
- **Status:** pending

### Task 14: Implement `handleFanOutComplete` method
- **Requirements:** AC-3.1, AC-4.2, AC-5.1, AC-5.2, AC-6.2, AC-6.3
- **Files:** `internal/tui/pipeline_exec.go`
- **Action:** Add `(a *App) handleFanOutComplete(msg FanOutCompleteMsg) tea.Cmd` method:
  1. Call `state.Context.MergeSpecialistResults(msg.Results)`
  2. For each result, append chat message (✅ or ❌ with summary)
  3. Mark specialist phases as Complete/Failed/Skipped in pipeline state
  4. Save checkpoint
  5. Append flywheel entries (one per specialist, with `ParallelGroup`)
  6. Skip to Phase 5 (Review) — set `state.Current` to Review index, start it
  7. If no specialists triggered, advance directly to Review
- **Test:** `go build ./...` passes.
- **Status:** pending

### Task 15: Wire fan-out into Phase 4 result handler
- **Requirements:** AC-4.1, AC-7.2
- **Files:** `internal/tui/pipeline_exec.go`
- **Action:** In `handlePhaseResult`, after risk gate returns `RiskLow`:
  - If `state.ParallelSpecialists` is true, call `a.startFanOut()` instead of `Advance() + startNextPhase()`
  - If false (or `--sequential` override), preserve existing sequential behavior
  - Add chat message: "⚡ Fan-out: N specialists triggered (concurrency: M)"
- **Test:** `go build ./...` passes. Manual test with `--sequential` falls back to sequential.
- **Status:** pending

### Task 16: Handle `FanOutCompleteMsg` in TUI Update loop
- **Requirements:** AC-2.1
- **Files:** `internal/tui/app.go` (or `update_pipeline.go` if split exists)
- **Action:** Add case `FanOutCompleteMsg` to the `Update()` message switch. Delegate to `a.handleFanOutComplete(msg)`. Return resulting cmd.
- **Test:** `go build ./...` passes.
- **Status:** pending

### Checkpoint: TUI Dispatch
- [ ] `startFanOut` dispatches concurrent specialists
- [ ] `handleFanOutComplete` merges results and advances to Review
- [ ] Sequential fallback works when `ParallelSpecialists=false`
- [ ] Chat messages show per-specialist outcomes
- [ ] `go build ./...` passes
- [ ] `go test ./internal/tui/... -race` passes

---

## Milestone 5: CLI & Config Wiring

### Task 17: Add `--sequential` flag to mission command
- **Requirements:** AC-7.3
- **Files:** `cmd/moonbase/mission.go`
- **Action:** Add `--sequential` bool flag. When set, the created pipeline has `ParallelSpecialists = false`. Reflected in dry-run output as "Specialists: sequential (override)".
- **Test:** `moonbase mission --sequential --dry-run "test"` shows sequential mode. `go build ./...` passes.
- **Status:** pending

### Task 18: Wire config into pipeline construction
- **Requirements:** AC-7.1, AC-7.2
- **Files:** `internal/tui/pipeline_exec.go` (or wherever pipeline is constructed for TUI missions)
- **Action:** After creating a `Pipeline` via `New(task)`, apply config:
  ```go
  cfg := config.Load()
  p.ParallelSpecialists = cfg.ParallelSpecialists
  p.MaxSpecialistConcurrency = cfg.MaxSpecialistConcurrency
  ```
  Clamp concurrency to 1–16 range.
- **Test:** Set `max_specialist_concurrency: 2` in config, run mission, verify only 2 concurrent. `go build ./...` passes.
- **Status:** pending

### Task 19: Update dry-run output for parallel info
- **Requirements:** AC-7.3
- **Files:** `cmd/moonbase/mission.go`
- **Action:** In `runMissionDryRun`, add section showing:
  - "Specialists: parallel (concurrency: N)" or "Specialists: sequential"
  - List which specialists would be triggered (reuse existing trigger eval)
- **Test:** `moonbase mission --dry-run "add auth"` shows specialist section. `go build ./...` passes.
- **Status:** pending

### Checkpoint: CLI Wiring
- [ ] `--sequential` flag disables fan-out for that mission
- [ ] Config values flow through to pipeline construction
- [ ] Dry-run shows parallel/sequential mode and concurrency
- [ ] `go build ./...` passes

---

## Milestone 6: Integration Testing & Polish

### Task 20: Integration test — full fan-out pipeline
- **Requirements:** AC-2.1, AC-3.1, AC-4.2, AC-5.1, AC-5.2, AC-6.2
- **Files:** `internal/pipeline/integration_test.go`
- **Action:** Add `//go:build integration` test that:
  1. Creates a pipeline with mock backend returning canned specialist outputs
  2. Runs through Phase 1–4 (mock), gets `RiskLow`
  3. Triggers fan-out with 3 specialists (mock 1 to fail)
  4. Verifies: 2 outputs merged, 1 failure marker, checkpoint saved, 3 flywheel entries
  5. Phase 5 receives all context via `ForPhase(5)`
- **Test:** `go test -tags=integration ./internal/pipeline/ -run TestFanOut` passes.
- **Status:** pending

### Task 21: Integration test — sequential fallback
- **Requirements:** AC-7.2
- **Files:** `internal/pipeline/integration_test.go`
- **Action:** Same setup as Task 20 but with `ParallelSpecialists=false`. Verify specialists execute one at a time (timing-based assertion with mock delays).
- **Test:** `go test -tags=integration ./internal/pipeline/ -run TestSequentialFallback` passes.
- **Status:** pending

### Task 22: Update pipeline view rendering for fan-out status
- **Requirements:** AC-2.1
- **Files:** `internal/tui/views_pipeline.go`
- **Action:** When specialists are in fan-out, show a grouped display:
  ```
  ⚡ Fan-Out [3 specialists, concurrency: 4]
    🔄 Numbuh 274 (Security) — running...
    ✅ Numbuh 0 (Oversight) — 2.3s
    🔄 Numbuh 362 (Infra) — running...
  ```
  Use existing `statusIcon` function for consistency.
- **Test:** `go build ./...` passes. Visual verification in TUI.
- **Status:** pending

### Task 23: Update PhaseInputSpec for Review to include specialists
- **Requirements:** AC-4.2
- **Files:** `internal/pipeline/phase_input_spec.go`
- **Action:** Update `phaseInputSpecs[5]` (Review phase) to include specialist phases (6, 7, 8) in `RequiresPhases` with appropriate headers and truncation limits. Add headers like `"## Security Analysis (from Phase 7 — Numbuh 274)"`.
- **Test:** `go test ./internal/pipeline/ -run TestPhaseInputSpec` passes. `ForPhase(5)` includes specialist outputs when present.
- **Status:** pending

### Checkpoint: Integration & Polish
- [ ] Full pipeline integration test passes with mock backend
- [ ] Sequential fallback integration test passes
- [ ] TUI shows grouped fan-out status
- [ ] Phase 5 (Review) receives specialist context
- [ ] `go test ./...` passes
- [ ] `go test -race ./...` passes

---

## Final Verification

- [ ] `go build ./...` — clean
- [ ] `go test ./...` — all pass
- [ ] `go vet ./...` — clean
- [ ] `go test -race ./...` — no data races
- [ ] `moonbase lint` — all 14 agents valid
- [ ] Fan-out executes N specialists concurrently (verified by test timing)
- [ ] Results merge in phase-number order (verified by determinism test)
- [ ] Partial failure preserves successful results (verified by unit test)
- [ ] Config `parallel_specialists: false` falls back to sequential (verified by integration test)
- [ ] Config `max_specialist_concurrency: 1` serializes execution (verified by unit test)
- [ ] `--sequential` flag overrides config (verified by build + dry-run)
- [ ] Checkpoint persists specialist results (verified by checkpoint test)
- [ ] Flywheel entries carry `parallel_group` (verified by flywheel test)
- [ ] No new TODOs introduced
- [ ] No `_ = err` in production code
