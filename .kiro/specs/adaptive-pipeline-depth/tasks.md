# Tasks: Adaptive Pipeline Depth

## Milestone 1: Complexity Classifier

### Task 1: Create depth types and classifier
- **Requirements:** AC-1.1, AC-1.2, AC-1.3, AC-1.4
- **Files:** `internal/pipeline/depth.go` (new)
- **Action:** Create `Depth` type (`trivial`, `simple`, `complex`), `DepthClassification` struct, `ClassifyTask(task string) DepthClassification` function. Implement `countComplexitySignals` using keyword matching (complexity keywords add +1, trivial keywords subtract 1, floor at 0). Implement `countFilePaths` reusing the heuristic from `extractFilesChanged` in `context.go`. Classification logic: >200 chars OR ≥3 signals OR ≥3 paths → complex; ≤80 chars AND 0 signals AND ≤1 path → trivial; else → simple.
- **Test:** `go build ./...` passes.
- **Status:** pending

### Task 2: Write classifier tests
- **Requirements:** AC-1.1, AC-1.2, AC-1.3
- **Files:** `internal/pipeline/depth_test.go` (new)
- **Action:** Table-driven tests covering: trivial tasks ("fix typo in README", "rename getUserName to getUsername", "remove unused import"), complex tasks ("add rate limiting to the API with per-user quotas and Redis backing", "implement pagination for /users endpoint with cursor-based navigation and Redis caching", a 250-char task), simple tasks ("fix the auth check", "update the error message in login", "add a test for the parser"), edge cases (empty string → simple, 81-char no-signal → simple, exactly 80-char no-signal → trivial).
- **Test:** `go test ./internal/pipeline/ -run TestClassifyTask` passes with all cases.
- **Status:** pending

### Checkpoint: Classifier
- [ ] `ClassifyTask` returns correct depth for trivial/simple/complex examples
- [ ] Ambiguous tasks resolve to `simple`, never `trivial`
- [ ] No external dependencies — pure string logic
- [ ] `go test ./internal/pipeline/ -run TestClassifyTask` passes

---

## Milestone 2: Adaptive Pipeline Constructor

### Task 3: Add Depth fields to Pipeline struct
- **Requirements:** AC-5.1, AC-6.1
- **Files:** `internal/pipeline/pipeline.go`
- **Action:** Add fields to `Pipeline` struct: `Depth Depth`, `DepthReason string`, `Escalated bool`, `OrigDepth Depth`. Update `New()` to set `Depth: DepthComplex` and `DepthReason: "full pipeline (default)"`. Update `NewFast()` to set `Depth: "override:fast"` and `DepthReason: "explicit --fast flag"`.
- **Test:** `go build ./...` passes. Existing tests pass (new fields have zero-value defaults).
- **Status:** pending

### Task 4: Implement NewAdaptive constructor
- **Requirements:** AC-2.1, AC-2.2, AC-2.3, AC-2.4
- **Files:** `internal/pipeline/depth.go`
- **Action:** Create `NewAdaptive(task string, depth Depth, reason string) *Pipeline`. For `DepthTrivial`: skip phases 1, 2, 5. For `DepthSimple`: skip phases 2, 5. For `DepthComplex`: same as `New()`. Set `Depth` and `DepthReason` on the returned pipeline. Conditional phases (6, 7, 8) remain pending (their `Conditional` flag handles them independently).
- **Test:** `go build ./...` passes.
- **Status:** pending

### Task 5: Test NewAdaptive
- **Requirements:** AC-2.1, AC-2.2, AC-2.3, AC-2.4
- **Files:** `internal/pipeline/depth_test.go`
- **Action:** Add tests: `TestNewAdaptive_Trivial` (phases 1,2,5 skipped; 3,4 pending; conditionals pending), `TestNewAdaptive_Simple` (phases 2,5 skipped; 1,3,4 pending), `TestNewAdaptive_Complex` (all mandatory phases pending). Verify `Depth` and `DepthReason` are set correctly.
- **Test:** `go test ./internal/pipeline/ -run TestNewAdaptive` passes.
- **Status:** pending

### Checkpoint: Constructor
- [ ] `NewAdaptive` produces correct phase layouts for all three depths
- [ ] Conditional phases are unaffected by depth selection
- [ ] `Pipeline.Depth` and `Pipeline.DepthReason` are populated
- [ ] All existing pipeline tests still pass

---

## Milestone 3: Escalation Logic

### Task 6: Implement Escalate method
- **Requirements:** AC-3.1, AC-3.2, AC-3.3, AC-3.5
- **Files:** `internal/pipeline/depth.go`
- **Action:** Add `Escalate(targetDepth Depth) error` method to `Pipeline`. Validates current depth < target depth. Sets `OrigDepth`, `Escalated`, updates `Depth`. Calls `unskipPhase` for newly required phases. Add helper `unskipPhase(number int)` that resets a phase from `StatusSkipped` to `StatusPending`. Add `escalationTarget(current Depth, risk RiskLevel) Depth` — maps MEDIUM on trivial → simple, MEDIUM on simple → complex, HIGH on anything → complex.
- **Test:** `go build ./...` passes.
- **Status:** pending

### Task 7: Test escalation
- **Requirements:** AC-3.1, AC-3.2, AC-3.3, AC-3.4
- **Files:** `internal/pipeline/depth_test.go`
- **Action:** Add tests: `TestEscalate_TrivialToSimple` (Phase 1 unskipped, Phase 2 remains skipped), `TestEscalate_TrivialToComplex` (Phases 1+2 unskipped), `TestEscalate_SimpleToComplex` (Phase 2 unskipped), `TestEscalate_AlreadyComplex` (returns error), `TestEscalationTarget_Medium` (trivial→simple, simple→complex), `TestEscalationTarget_High` (any→complex), `TestEscalationTarget_Critical` (no escalation — returns current depth, CRITICAL stops separately).
- **Test:** `go test ./internal/pipeline/ -run TestEscalat` passes.
- **Status:** pending

### Task 8: Review phase un-skip after escalation
- **Requirements:** AC-3.1, AC-3.2
- **Files:** `internal/pipeline/depth.go`
- **Action:** In `Escalate`, when escalating to `DepthSimple` or `DepthComplex`, also un-skip Phase 5 (Review). This ensures escalated pipelines get full review coverage. Document: "Non-escalated trivial/simple pipelines skip Review when QA passes LOW; escalated pipelines always include Review."
- **Test:** Add test case: `TestEscalate_ReviewUnskipped` — verify Phase 5 goes from Skipped to Pending after any escalation.
- **Status:** pending

### Checkpoint: Escalation
- [ ] `Escalate` correctly unskips the right phases for each transition
- [ ] `escalationTarget` maps risk levels to correct target depths
- [ ] Phase 5 (Review) is un-skipped on escalation
- [ ] MaxRework limit still applies (escalation uses `RouteToPhase` which increments rework count)
- [ ] CRITICAL risk is not handled by escalation (it stops the pipeline separately)
- [ ] `go test ./internal/pipeline/ -run TestEscalat` passes

---

## Milestone 4: Flywheel Integration

### Task 9: Add depth fields to FlywheelEntry
- **Requirements:** AC-5.1, AC-5.2, AC-5.3, AC-5.4
- **Files:** `internal/pipeline/flywheel.go`
- **Action:** Add fields to `FlywheelEntry`: `Depth string \`json:"depth,omitempty"\``, `DepthReason string \`json:"depth_reason,omitempty"\``, `EscalatedFrom string \`json:"escalated_from,omitempty"\``, `EscalatedTo string \`json:"escalated_to,omitempty"\``. Do NOT increment `currentSchemaVersion` (optional fields per evolution contract).
- **Test:** `go build ./...` passes. Existing flywheel tests pass unchanged.
- **Status:** pending

### Task 10: Test flywheel backward compatibility
- **Requirements:** AC-5.4
- **Files:** `internal/pipeline/flywheel_test.go`
- **Action:** Add test that marshals a `FlywheelEntry` with new fields set and verifies JSON output includes them. Add test that marshals a `FlywheelEntry` with new fields empty and verifies they're omitted from JSON. Add test that unmarshals a legacy entry (no depth fields) without error.
- **Test:** `go test ./internal/pipeline/ -run TestFlywheel` passes.
- **Status:** pending

### Checkpoint: Flywheel
- [ ] New fields appear in JSON when set
- [ ] New fields are absent from JSON when empty (omitempty)
- [ ] Legacy entries without depth fields parse without error
- [ ] Schema version remains 1

---

## Milestone 5: CLI Integration

### Task 11: Add --full and --depth flags
- **Requirements:** AC-4.1, AC-4.2, AC-4.3, AC-4.4, AC-4.5
- **Files:** `cmd/moonbase/mission_cmd.go`
- **Action:** Add `var missionFull bool` and `var missionDepth string`. Register flags: `--full` (bool), `--depth` (string, valid values: trivial, simple, complex). Use cobra's `MarkFlagsMutuallyExclusive("fast", "full", "depth")` for mutual exclusivity. Add validation in the Run func: if `--depth` is set, validate it's one of the three valid values.
- **Test:** `go build ./...` passes. `moonbase mission --fast --full "test"` prints mutual exclusivity error.
- **Status:** pending

### Task 12: Implement runMissionAdaptive
- **Requirements:** AC-2.1, AC-2.2, AC-2.3, AC-4.5, AC-5.1, AC-5.2, AC-6.1, AC-6.2
- **Files:** `cmd/moonbase/mission.go`
- **Action:** Create `runMissionAdaptive(task string, depth pipeline.Depth, reason string)`. Structure: acquire lock → discover context → create pipeline via `pipeline.NewAdaptive(task, depth, reason)` → print depth announcement (`"   Depth: %s (%s)"`) → run pipeline loop with `allowEscalation: true` → log depth on every flywheel entry → log `depth_reason` on first entry only → run conditional phases → print summary. The function replaces `runMission` as the default code path.
- **Test:** `go build ./...` passes.
- **Status:** pending

### Task 13: Wire dispatch logic
- **Requirements:** AC-4.1, AC-4.2, AC-4.3, AC-4.5
- **Files:** `cmd/moonbase/mission_cmd.go`
- **Action:** Update the mission command's Run function dispatch: `--fast` → `runMissionFast` (unchanged); `--full` → current `runMission` (renamed to `runMissionFull`, sets depth `"override:full"`); `--depth <val>` → `runMissionAdaptive(task, val, "override:<val>")`. Default (no flags) → `classification := pipeline.ClassifyTask(task); runMissionAdaptive(task, classification.Depth, classification.Reason)`.
- **Test:** `go build ./...` passes. `moonbase mission --dry-run "fix typo"` should show trivial depth in the dry-run output (integration-level).
- **Status:** pending

### Task 14: Wire escalation into pipeline loop
- **Requirements:** AC-3.1, AC-3.2, AC-3.3, AC-3.4, AC-3.5, AC-6.2
- **Files:** `cmd/moonbase/mission.go`
- **Action:** In `runPipelineLoop`, after the QA phase (phase.Number == 4) and when `opts.allowEscalation` is true: check if `routing.Level != RiskLow` and `p.Depth != DepthComplex`. If so, compute `targetDepth := escalationTarget(p.Depth, routing.Level)`. If targetDepth != p.Depth: print escalation message, call `p.Escalate(targetDepth)`, log escalation flywheel entry, find earliest pending phase index, set `i = idx - 1`, continue. CRITICAL risk is handled first (unchanged) — escalation only applies to MEDIUM/HIGH on non-complex pipelines.
- **Test:** `go build ./...` passes.
- **Status:** pending

### Task 15: Add allowEscalation to pipelineLoopOptions
- **Requirements:** AC-3.1, AC-4.1
- **Files:** `cmd/moonbase/mission.go`
- **Action:** Add `allowEscalation bool` to `pipelineLoopOptions`. Set to `true` in `runMissionAdaptive`, `false` in `runMissionFast` (unchanged) and `runMissionFull` (already at max depth — escalation is a no-op but explicitly disabled for clarity).
- **Test:** `go build ./...` passes.
- **Status:** pending

### Checkpoint: CLI Integration
- [ ] `moonbase mission "fix typo"` auto-classifies and runs at trivial depth
- [ ] `moonbase mission --full "fix typo"` runs full pipeline
- [ ] `moonbase mission --depth simple "fix typo"` forces simple depth
- [ ] `moonbase mission --fast --full "x"` errors with mutual exclusivity message
- [ ] Depth is printed at mission start
- [ ] Escalation message appears when QA triggers it
- [ ] `go build ./...` and `go test ./...` pass

---

## Milestone 6: Integration Testing

### Task 16: Integration test for auto-classification
- **Requirements:** AC-1.1, AC-1.2, AC-1.3, AC-4.5
- **Files:** `cmd/moonbase/mission_integration_test.go`
- **Action:** Add test cases (using mock backend) that verify: trivial task → only phases 3+4 invoked; simple task → phases 1+3+4 invoked; complex task → phases 1+2+3+4+5 invoked. Use the existing mock backend pattern from the file.
- **Test:** `go test ./cmd/moonbase/ -run TestMissionAdaptive -tags integration` passes.
- **Status:** pending

### Task 17: Integration test for escalation
- **Requirements:** AC-3.1, AC-3.2, AC-3.4
- **Files:** `cmd/moonbase/mission_integration_test.go`
- **Action:** Add test: mock backend returns MEDIUM risk from Phase 4 on a trivial pipeline → verify Phase 1 is executed after escalation → verify Phase 3 re-runs → verify final QA passes. Add test: mock returns HIGH → verify escalation to complex (phases 1+2 added). Add test: mock returns CRITICAL → verify pipeline stops (no escalation).
- **Test:** `go test ./cmd/moonbase/ -run TestMissionEscalation -tags integration` passes.
- **Status:** pending

### Task 18: Flywheel output verification
- **Requirements:** AC-5.1, AC-5.2, AC-5.3
- **Files:** `cmd/moonbase/mission_integration_test.go`
- **Action:** After an adaptive mission run (with mock backend), read the flywheel JSONL output and verify: all entries have `"depth"` field set; first entry has `"depth_reason"`; escalation entry has `"escalated_from"` and `"escalated_to"` fields; schema version is 1.
- **Test:** `go test ./cmd/moonbase/ -run TestMissionFlywheel -tags integration` passes.
- **Status:** pending

### Checkpoint: Integration
- [ ] Auto-classification → correct phases invoked
- [ ] Escalation → correct phases added and re-run
- [ ] CRITICAL → pipeline stops (no escalation)
- [ ] Flywheel entries contain depth metadata
- [ ] All tests pass: `go test ./... -tags integration`

---

## Milestone 7: Dry-Run and Docs

### Task 19: Update dry-run to show depth
- **Requirements:** AC-6.1
- **Files:** `cmd/moonbase/mission_dryrun.go`
- **Action:** Update `runMissionDryRun` to: auto-classify the task; print the classification result (`"   Depth: <depth> (<reason>)"`); show which phases would be active vs skipped based on the classified depth. If `--depth` flag is set, use that instead of auto-classification.
- **Test:** `moonbase mission --dry-run "fix typo"` shows depth as trivial, phases 3+4 active, others skipped.
- **Status:** pending

### Task 20: Update CHANGELOG
- **Requirements:** Changelog discipline
- **Files:** `CHANGELOG.md`
- **Action:** Add entry under `[Unreleased]`: `feat(pipeline): adaptive pipeline depth — auto-classifies task complexity and selects minimum viable depth, escalating mid-run if QA flags insufficient analysis`. Add sub-entries for `--full` and `--depth` flags.
- **Test:** Changelog entry exists and follows format.
- **Status:** pending

### Task 21: Update README CLI table
- **Requirements:** Documentation accuracy
- **Files:** `README.md`
- **Action:** Update the CLI Commands table to document `--full` and `--depth` flags on the mission command. Add a brief "Adaptive Depth" section to the Pipeline documentation explaining the three tiers and escalation behavior.
- **Test:** README accurately describes the new flags.
- **Status:** pending

### Checkpoint: Docs & Polish
- [ ] `moonbase mission --dry-run "task"` shows depth classification
- [ ] CHANGELOG updated with feature entry
- [ ] README documents new flags and adaptive behavior
- [ ] `go build ./...` passes

---

## Final Verification

- [ ] `go build ./...` — clean
- [ ] `go test ./...` — all pass
- [ ] `go vet ./...` — clean
- [ ] `go test ./... -race` — no data races
- [ ] `moonbase lint` — all 14 agents valid
- [ ] `moonbase mission --fast "x"` — unchanged behavior (no escalation, informational gate)
- [ ] `moonbase mission --full "x"` — full pipeline (same as old default)
- [ ] `moonbase mission "fix typo"` — classifies trivial, runs 2 phases
- [ ] `moonbase mission "add pagination to /users API"` — classifies complex, runs full
- [ ] `moonbase mission --depth simple "anything"` — forces simple depth
- [ ] `moonbase mission --fast --full "x"` — mutual exclusivity error
- [ ] Flywheel entries contain `depth` field
- [ ] Escalation scenario: trivial task → MEDIUM QA → escalates to simple → passes
- [ ] Risk gate logic is identical (diff `riskgate.go` shows zero changes)
