# Tasks: Token & Cost Observability

## Milestone 1: Usage Interface & Backend Extraction

### Task 1: Create UsageInfo type and optional interfaces
- **Requirements:** AC-1.1, AC-1.2
- **Files:** `internal/backend/usage.go` (new)
- **Action:** Create `UsageInfo` struct with `PromptTokens`, `CompletionTokens`, `TotalTokens`, `Model` fields. Define `UsageReporter` and `RawUsageReporter` interfaces. Add doc comments explaining the optional-interface pattern (same as `RawDeployer`).
- **Test:** `go build ./...` passes. Interfaces compile correctly with explicit type assertions in a test file.
- **Status:** pending

### Task 2: Extend streamChatCompletion to extract usage from SSE
- **Requirements:** AC-1.3, AC-1.5
- **Files:** `internal/backend/openai_stream.go`
- **Action:**
  1. Add `StreamOptions *streamOptions` field to `openaiRequest` with `IncludeUsage: true`.
  2. Expand the SSE event struct to include `Usage *struct{...}` field.
  3. After the SSE parse loop, check if a `usage` event was received and return `*UsageInfo`.
  4. Change return signature to `(string, *UsageInfo, error)`.
  5. Update all callers of `streamChatCompletion` (OpenAI.Deploy, Kimi.Deploy).
- **Test:** `go build ./...` passes. Add test in `internal/backend/openai_test.go` with mock SSE response including a final `usage` event. Verify `UsageInfo` is populated.
- **Status:** pending

### Task 3: Implement RawUsageReporter on OpenAI backend
- **Requirements:** AC-1.3
- **Files:** `internal/backend/openai.go`
- **Action:** Add `DeployRawWithUsage(composed, task string) (string, *UsageInfo, error)` method to `OpenAI`. Extract shared logic from `Deploy` into a helper. `Deploy` calls the helper and discards usage (backward compat). Verify `OpenAI` satisfies both `Backend` and `RawUsageReporter` via compile-time assertion: `var _ RawUsageReporter = (*OpenAI)(nil)`.
- **Test:** `go build ./...` passes. Existing OpenAI tests still pass. New test verifies `RawUsageReporter` type assertion succeeds.
- **Status:** pending

### Task 4: Implement RawUsageReporter on Kimi backend
- **Requirements:** AC-1.5
- **Files:** `internal/backend/kimi.go`
- **Action:** Same pattern as Task 3. Add `DeployRawWithUsage` method to `Kimi`. Reuse `streamChatCompletion`'s new return signature.
- **Test:** `go build ./...` passes. Compile-time assertion `var _ RawUsageReporter = (*Kimi)(nil)`.
- **Status:** pending

### Task 5: Implement UsageReporter on Anthropic backend
- **Requirements:** AC-1.4
- **Files:** `internal/backend/backends.go`, `internal/chat/stream.go`
- **Action:**
  1. Add `Usage *StreamUsage` field to `chat.StreamChunk` (populated on `Done=true` chunk).
  2. In the Anthropic streaming handler (internal/chat), parse `message_start` for `input_tokens` and `message_delta` for `output_tokens`. Set on the final chunk.
  3. Add `DeployWithUsage` method to `Anthropic` that collects `StreamUsage` from the final chunk and returns `*UsageInfo`.
  4. Keep existing `Deploy` method unchanged (backward compat).
- **Test:** `go build ./...` passes. Test with mock stream that emits usage on final chunk. Verify `UsageInfo` populated correctly.
- **Status:** pending

### Task 6: Verify non-reporting backends return nil
- **Requirements:** AC-1.6
- **Files:** `internal/backend/usage_test.go` (new)
- **Action:** Write tests asserting that `Kiro`, `Codex`, `Ollama`, `Clipboard` do NOT implement `UsageReporter` or `RawUsageReporter` (type assertion returns `ok=false`).
- **Test:** All assertions pass. `go test ./internal/backend/...` passes.
- **Status:** pending

### Checkpoint: Milestone 1
- [ ] `UsageInfo`, `UsageReporter`, `RawUsageReporter` exist and compile
- [ ] `streamChatCompletion` returns `(string, *UsageInfo, error)`
- [ ] OpenAI, Kimi implement `RawUsageReporter`
- [ ] Anthropic implements `UsageReporter`
- [ ] kiro-cli, codex, ollama, clipboard do NOT implement usage interfaces
- [ ] `go build ./...` passes
- [ ] `go test ./internal/backend/...` passes

---

## Milestone 2: DeployComposed & Pipeline Integration

### Task 7: Update DeployComposed to return usage
- **Requirements:** AC-2.1
- **Files:** `internal/backend/deploy.go`
- **Action:**
  1. Change `DeployComposed` signature to `(ctx, composed, task, timeout) → (string, *UsageInfo, error)`.
  2. Inside the retry function, check backend for `RawUsageReporter` first, then `RawDeployer`, then `UsageReporter`, then `Backend.Deploy`. Propagate `*UsageInfo` from whichever path is taken (nil for non-reporters).
  3. Update clipboard fallback to return `nil` usage.
- **Test:** `go build ./...` passes (will fail until callers in mission.go are updated — do Task 8 next).
- **Status:** pending

### Task 8: Update executeAndRecordPhase to capture usage
- **Requirements:** AC-2.2
- **Files:** `cmd/moonbase/mission.go`
- **Action:**
  1. `executeAndRecordPhase` calls `DeployComposed` and receives `(output, usage, err)`.
  2. If `usage != nil`, populate `PromptTokens`, `CompletionTokens`, `TotalTokens`, `Model` on the `FlywheelEntry`.
  3. Compute `EstimatedCostUSD` using the pricing table (from Task 10).
  4. Return usage to caller for budget tracking (change `executeAndRecordPhase` return to `(string, *backend.UsageInfo, error)`).
- **Test:** `go build ./...` passes. `go test ./cmd/moonbase/...` passes.
- **Status:** pending

### Task 9: Update parallel conditional execution
- **Files:** `cmd/moonbase/mission.go` (function `runConditionalPhasesParallel`)
- **Action:** Update the goroutine to use `DeployComposed`'s new signature. Discard usage for parallel phases (no budget enforcement on conditionals). Optionally record usage in flywheel for cost tracking.
- **Test:** `go build ./...` passes.
- **Status:** pending

### Checkpoint: Milestone 2
- [ ] `DeployComposed` returns `(string, *UsageInfo, error)`
- [ ] `executeAndRecordPhase` populates token fields on FlywheelEntry
- [ ] Parallel phases compile with new signature
- [ ] `go build ./...` && `go test ./...` pass

---

## Milestone 3: Pricing & Flywheel Persistence

### Task 10: Create pricing module
- **Requirements:** AC-3.1, AC-3.2
- **Files:** `internal/pipeline/pricing.go` (new), `internal/pipeline/pricing_test.go` (new)
- **Action:**
  1. Define `ModelPrice` struct: `PromptPer1M float64`, `CompletionPer1M float64`.
  2. Implement `DefaultPricing() map[string]ModelPrice` with embedded prices for gpt-4o, gpt-4o-mini, gpt-4.1, gpt-4.1-mini, claude-sonnet-4, claude-haiku-3.5, kimi-k3.
  3. Implement `EstimateCost(model string, promptTokens, completionTokens int, pricing map[string]ModelPrice) float64`.
  4. Handle unknown model → returns 0.
- **Test:** Table-driven tests: known model returns correct cost, unknown model returns 0, zero tokens returns 0.
- **Status:** pending

### Task 11: Extend FlywheelEntry with token/cost fields
- **Requirements:** AC-4.1, AC-4.2
- **Files:** `internal/pipeline/flywheel.go`, `internal/pipeline/flywheel_test.go`
- **Action:**
  1. Add fields to `FlywheelEntry`: `PromptTokens`, `CompletionTokens`, `TotalTokens`, `Model`, `EstimatedCostUSD` (all `omitempty`).
  2. Keep `currentSchemaVersion = 1` (per evolution contract).
  3. Add test: marshal entry with token fields → unmarshal → verify fields present.
  4. Add test: unmarshal legacy entry without token fields → verify zero values (no error).
- **Test:** `go test ./internal/pipeline/...` passes. Existing flywheel tests unmodified and passing.
- **Status:** pending

### Task 12: Add pricing and budget to config
- **Requirements:** AC-3.1, AC-6.1, AC-6.4
- **Files:** `internal/config/config.go`, `internal/config/config_test.go`
- **Action:**
  1. Add `ModelPricing map[string]pipeline.ModelPrice` field to `Config` (yaml key: `model_pricing`). If import cycle, define a local `ModelPriceConfig` struct and convert.
  2. Add `TokenBudget TokenBudgetConfig` field (yaml key: `token_budget`).
  3. `TokenBudgetConfig`: `MaxTokensPerMission int` (default 0 = unlimited), `WarnThresholdPct int` (default 80).
  4. `DefaultConfig()` sets `ModelPricing` to nil (use embedded defaults), `TokenBudget` to zero values.
  5. Add helper: `func EffectivePricing(cfg Config) map[string]pipeline.ModelPrice` that merges config overrides with defaults.
- **Test:** Test loading config with/without pricing section. Test `EffectivePricing` merges correctly.
- **Status:** pending

### Checkpoint: Milestone 3
- [ ] `EstimateCost` works correctly for known and unknown models
- [ ] `FlywheelEntry` serializes with new fields, deserializes old entries cleanly
- [ ] Config loads pricing and budget sections
- [ ] `go build ./...` && `go test ./...` pass

---

## Milestone 4: Budget Enforcement

### Task 13: Add budget enforcement to pipeline loop
- **Requirements:** AC-6.1, AC-6.2, AC-6.3, AC-6.4
- **Files:** `cmd/moonbase/mission.go`
- **Action:**
  1. At the start of `runPipelineLoop`, load `TokenBudgetConfig` from config.
  2. Track `missionTokens int` accumulator (sum of `TotalTokens` from each phase's usage).
  3. After each successful phase with non-nil usage:
     - Add `usage.TotalTokens` to accumulator.
     - If `MaxTokensPerMission > 0` and accumulator exceeds budget: stop pipeline, log `"budget_exceeded"` to flywheel, save checkpoint, print stop message.
     - Else if accumulator exceeds `WarnThresholdPct` of budget: print warning, continue.
  4. If budget is 0 (default): skip all budget checks.
- **Test:** Integration test with mock backend returning fixed usage. Verify warning printed at threshold. Verify pipeline stops at budget limit. Verify no checks when budget is 0.
- **Status:** pending

### Checkpoint: Milestone 4
- [ ] Budget warning prints at threshold
- [ ] Pipeline stops when budget exceeded
- [ ] Flywheel records `"budget_exceeded"` outcome
- [ ] No enforcement when budget is 0
- [ ] `go test ./...` passes

---

## Milestone 5: Flywheel Display

### Task 14: Add token/cost aggregation to flywheel command
- **Requirements:** AC-5.1, AC-5.2, AC-5.3, AC-5.4, AC-5.5
- **Files:** `cmd/moonbase/flywheel_cmd.go`
- **Action:**
  1. After existing analysis, add a new section if ANY entries have `TotalTokens > 0`.
  2. **Totals:** Sum prompt/completion tokens and estimated cost across all entries.
  3. **Per-agent:** Group by `Agent` field, compute average tokens and cost per invocation. Sort by cost descending.
  4. **Per-mission:** Group by `TraceID`, compute total tokens/cost per mission. Show avg, max, and identify the most expensive mission.
  5. **Cost-heavy phase:** Rank phases by average `EstimatedCostUSD`. Highlight the top one with its % of average mission cost.
  6. **Graceful degradation:** If no entries have token data, skip the section entirely. If some entries lack data, exclude from averages (count entries with data separately).
  7. Format large numbers with K/M suffixes (e.g., "142K tokens").
- **Test:** Write a test function that feeds entries (some with tokens, some without) and verifies the output contains expected sections and values.
- **Status:** pending

### Task 15: Add --cost flag to flywheel command
- **Files:** `cmd/moonbase/flywheel_cmd.go`
- **Action:** Add `--cost` flag that shows ONLY the cost section (for quick checks). Without the flag, cost section is shown alongside existing output (no behavior change for existing users).
- **Test:** Verify `--cost` produces cost-only output. Verify without flag, full output is shown.
- **Status:** pending

### Checkpoint: Milestone 5
- [ ] `moonbase flywheel` shows token/cost summary when data exists
- [ ] Per-agent breakdown sorted by cost
- [ ] Per-mission averages displayed
- [ ] Cost-heavy phase highlighted
- [ ] Entries without tokens gracefully excluded from averages
- [ ] `--cost` flag works
- [ ] `go build ./...` && `go test ./...` pass

---

## Milestone 6: Integration & Polish

### Task 16: Integration test with full pipeline
- **Files:** `cmd/moonbase/mission_integration_test.go`
- **Action:** Extend existing integration test to use a mock backend that implements `RawUsageReporter`. Verify flywheel entries contain token/cost data. Verify budget enforcement triggers at the expected phase.
- **Test:** `go test -tags integration ./cmd/moonbase/...` passes.
- **Status:** pending

### Task 17: Update CHANGELOG and docs
- **Files:** `CHANGELOG.md`, `README.md`
- **Action:**
  1. Add entry under `## [Unreleased]`: `feat(flywheel): token consumption and cost observability per mission phase`
  2. Update README "Key Capabilities" or "Flywheel" section to mention cost tracking.
  3. Add config example in README showing `model_pricing` and `token_budget`.
- **Test:** Manual review. `moonbase lint` passes.
- **Status:** pending

### Final Checkpoint
- [ ] All milestones complete
- [ ] `go build ./...` passes
- [ ] `go test -race ./...` passes
- [ ] `moonbase lint` passes
- [ ] `moonbase flywheel` shows cost data with test flywheel entries
- [ ] Budget enforcement works end-to-end
- [ ] CHANGELOG updated
- [ ] No TODOs introduced
