# Requirements: Token & Cost Observability

## Overview

Moonbase's flywheel records timing, outcome, and output size per pipeline phase — but has zero visibility into token consumption or cost. Developers running missions have no way to know how many tokens a mission consumed, which phases are expensive, or whether a budget is being exceeded.

This spec adds token/cost observability to the pipeline: backends that expose usage data return it; the pipeline captures per-phase tokens and estimated cost; the flywheel persists this alongside existing fields; and `moonbase flywheel` surfaces aggregated insights including cost-heavy phase detection and optional budget enforcement.

---

## Current State (Confirmed)

| Component | Status |
|-----------|--------|
| `Backend` interface (`internal/backend/backend.go`) | Returns `(string, error)` — no usage metadata |
| `RawDeployer` interface | Returns `(string, error)` — same |
| `streamChatCompletion` (OpenAI/Kimi SSE) | Streams deltas but discards final `usage` object from SSE |
| `Anthropic.Deploy` (via `chat.Stream`) | Collects text chunks, ignores `message_delta` usage event |
| `FlywheelEntry` struct | Has `v`, timing, outcome, output_size — no token fields |
| `moonbase flywheel` command | Shows duration, rework rate, risk distribution — no cost data |
| Config (`internal/config/config.go`) | No pricing or budget fields |

---

## User Stories

### US-1: See Token Consumption Per Mission
As a developer, I want to see how many prompt and completion tokens each mission consumed so that I can understand the cost of my pipeline runs.

### US-2: Identify Expensive Phases
As a developer, I want to see which pipeline phases consume the most tokens so that I can optimize prompts or skip unnecessary phases.

### US-3: Estimate Dollar Cost
As a developer, I want token counts translated to estimated dollar cost (using a configurable price table) so that I can budget my AI usage without checking provider dashboards.

### US-4: Budget Guard
As a developer running many missions, I want to set a per-mission token budget with a warning threshold so that I get alerted before a mission burns through excessive tokens.

### US-5: Historical Cost Trends
As a developer, I want `moonbase flywheel` to show aggregate cost per agent and per mission over time so that I can spot cost inflation and compare mission complexity.

---

## Acceptance Criteria

### AC-1: Usage Return Interface

#### AC-1.1: UsageInfo Type
- **GIVEN** a new `UsageInfo` struct in `internal/backend/`
- **THEN** it contains `PromptTokens int`, `CompletionTokens int`, `TotalTokens int`, `Model string`
- **AND** backends return `*UsageInfo` (nil when usage is unavailable)

#### AC-1.2: UsageReporter Optional Interface
- **GIVEN** the existing `Backend` interface is unchanged (backward compatible)
- **THEN** a new optional interface `UsageReporter` exists:
  ```go
  type UsageReporter interface {
      DeployWithUsage(agent agents.Agent, context *discovery.ProjectContext, task string) (string, *UsageInfo, error)
  }
  ```
- **AND** a companion `RawUsageReporter` for pre-composed prompts:
  ```go
  type RawUsageReporter interface {
      DeployRawWithUsage(composed string, task string) (string, *UsageInfo, error)
  }
  ```

#### AC-1.3: OpenAI Returns Usage
- **WHEN** the OpenAI backend completes a streaming request
- **THEN** it parses the final SSE `usage` field (present when `stream_options.include_usage: true`) or falls back to the non-streaming `usage` object
- **AND** returns a populated `*UsageInfo` with model name

#### AC-1.4: Anthropic Returns Usage
- **WHEN** the Anthropic backend completes a streaming request
- **THEN** it extracts `input_tokens` and `output_tokens` from the `message_delta` event's `usage` field
- **AND** returns a populated `*UsageInfo`

#### AC-1.5: Kimi Returns Usage
- **WHEN** the Kimi backend completes a streaming request
- **THEN** it returns usage from the OpenAI-compatible `usage` SSE field (same wire format as OpenAI)

#### AC-1.6: Non-Reporting Backends Return Nil
- **WHEN** `kiro-cli`, `codex`, `ollama`, or `clipboard` backends are used
- **THEN** usage is `nil` (not an error — usage is best-effort)
- **AND** the pipeline proceeds normally without usage data

### AC-2: Per-Phase Token Capture

#### AC-2.1: DeployComposed Extracts Usage
- **WHEN** `DeployComposed` calls a backend
- **THEN** it checks if the backend implements `RawUsageReporter` (preferred) or `UsageReporter`
- **AND** returns usage alongside output: signature becomes `DeployComposed(...) (string, *UsageInfo, error)`

#### AC-2.2: Pipeline Records Tokens
- **WHEN** `executeAndRecordPhase` completes a phase
- **THEN** the returned `*UsageInfo` (if non-nil) is recorded in the flywheel entry

### AC-3: Cost Estimation

#### AC-3.1: Price Table in Config
- **GIVEN** a new config section `model_pricing` in `~/.config/moonbase/config.yaml`
- **THEN** it maps model names to per-million-token prices:
  ```yaml
  model_pricing:
    gpt-4o:
      prompt: 2.50      # USD per 1M prompt tokens
      completion: 10.00 # USD per 1M completion tokens
    claude-sonnet-4-20250514:
      prompt: 3.00
      completion: 15.00
    kimi-k3:
      prompt: 0.70
      completion: 2.80
  ```
- **AND** reasonable defaults are embedded for common models (user can override)
- **AND** no secrets or API keys are stored (prices are public information)

#### AC-3.2: Cost Calculated Per Entry
- **WHEN** a flywheel entry has token counts AND the model is in the price table
- **THEN** `estimated_cost_usd` is computed and stored in the entry
- **AND** if the model is unknown, cost is `0` (not an error) and a field `cost_estimated: false` differentiates from zero-cost

### AC-4: Flywheel Persistence

#### AC-4.1: Schema Evolution
- **GIVEN** the existing `currentSchemaVersion = 1` and the evolution contract ("new optional fields may be added without bumping the version")
- **THEN** new fields are added to `FlywheelEntry` as `omitempty`:
  ```go
  PromptTokens     int     `json:"prompt_tokens,omitempty"`
  CompletionTokens int     `json:"completion_tokens,omitempty"`
  TotalTokens      int     `json:"total_tokens,omitempty"`
  Model            string  `json:"model,omitempty"`
  EstimatedCostUSD float64 `json:"estimated_cost_usd,omitempty"`
  ```
- **AND** `currentSchemaVersion` remains `1` (these are optional additions)
- **AND** existing readers (current `moonbase flywheel`) continue to work without modification on new entries (zero values are fine)

#### AC-4.2: Backward Compatibility
- **WHEN** reading legacy entries (v=1 without token fields, or v=0)
- **THEN** token/cost fields default to zero
- **AND** aggregations treat zero-token entries as "no data" (excluded from averages, included in totals as 0)

### AC-5: Flywheel Aggregation & Display

#### AC-5.1: Total Token/Cost Summary
- **WHEN** running `moonbase flywheel`
- **THEN** the output includes total prompt tokens, completion tokens, and estimated cost across all recorded missions

#### AC-5.2: Per-Agent Token/Cost
- **WHEN** running `moonbase flywheel`
- **THEN** the output includes a breakdown of tokens and cost per agent name (e.g., "numbuh-1: 45K prompt, 12K completion, $0.38")

#### AC-5.3: Per-Mission Token/Cost
- **WHEN** running `moonbase flywheel`
- **THEN** the output includes average and max tokens/cost per mission (grouped by trace_id)

#### AC-5.4: Cost-Heavy Phase Detection
- **WHEN** running `moonbase flywheel`
- **THEN** phases are ranked by average cost
- **AND** the top cost-consuming phase is highlighted (analogous to existing "Longest Phase")

#### AC-5.5: Graceful Degradation
- **WHEN** flywheel entries have no token data (older entries or non-reporting backends)
- **THEN** the cost section shows "N/A" or "(no usage data)" instead of zeros
- **AND** entries without data are excluded from averages but contribute to counts

### AC-6: Token Budget

#### AC-6.1: Budget Config
- **GIVEN** new config fields:
  ```yaml
  token_budget:
    max_tokens_per_mission: 500000   # hard cap (0 = unlimited)
    warn_threshold_pct: 80           # warn at 80% of budget
  ```
- **THEN** the pipeline checks cumulative tokens after each phase

#### AC-6.2: Warning Threshold
- **WHEN** cumulative tokens for the current mission exceed `warn_threshold_pct` of `max_tokens_per_mission`
- **THEN** a warning is printed: `⚠️ Token budget: 82% used (410K / 500K)`
- **AND** execution continues

#### AC-6.3: Hard Cap
- **WHEN** cumulative tokens for the current mission exceed `max_tokens_per_mission`
- **THEN** the pipeline stops with: `🛑 Token budget exceeded (520K / 500K). Pipeline stopped.`
- **AND** the flywheel entry records outcome `"budget_exceeded"`
- **AND** a checkpoint is saved for resume

#### AC-6.4: Budget Disabled by Default
- **WHEN** `max_tokens_per_mission` is `0` or not set
- **THEN** no budget enforcement occurs (unlimited)

---

## Non-Goals

- Real-time cost streaming in TUI (future enhancement — this spec is CLI-first)
- Automatic model selection based on cost (out of scope)
- Integration with billing APIs to get actual invoiced cost (we estimate only)
- Token counting for non-API backends (kiro-cli, ollama CLI) — they return nil usage
- Persisting raw API responses for audit (too large, privacy concerns)

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| OpenAI streaming doesn't include `usage` by default | No token data for OpenAI | Add `stream_options: {include_usage: true}` to request body |
| Price table becomes stale | Inaccurate cost estimates | Embed defaults but allow user override; document update process |
| Token budget stops important missions | User frustration | Budget is opt-in (disabled by default), warn before stopping |
| Breaking `DeployComposed` signature | All callers must update | New function `DeployComposedWithUsage` or add usage to return values with compile-time check |
| Large flywheel files with extra fields | Marginally larger logs | Fields are omitempty; tokens add ~50 bytes per entry |
