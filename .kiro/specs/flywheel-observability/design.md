# Design: Token & Cost Observability

## Architecture Decision

The design uses Go's interface-assertion pattern to extend backends without breaking the existing `Backend` interface. Backends that can report usage implement an additional optional interface; the pipeline checks at call time and degrades gracefully to nil when usage isn't available. This is the same pattern already used for `RawDeployer`.

**Key decisions:**
1. **Optional interface over signature change** — adding return values to `Backend.Deploy` would break all 7 implementations and all callers. A new `UsageReporter` interface lets each backend opt in.
2. **Usage extracted from SSE final chunk** — OpenAI/Kimi include a `usage` field in the last streamed SSE event when `stream_options.include_usage: true`. This avoids a second API call.
3. **Cost computed at write time** — the flywheel entry stores estimated cost, not just tokens. This avoids needing the price table at read time and lets historical data remain accurate even if prices change later.
4. **Schema version stays at 1** — per the existing evolution contract, new optional fields don't require a version bump. Readers tolerate missing fields via `omitempty` + zero values.
5. **Budget enforcement in `runPipelineLoop`** — budget checks happen after each phase completion, reusing the existing loop structure.
6. **`DeployComposed` signature changes** — this is an internal function (unexported-level coordination), so changing its return to `(string, *UsageInfo, error)` has bounded blast radius (only `mission.go` and its test call it).

---

## Files Affected

| File | Change Type | Purpose |
|------|-------------|---------|
| `internal/backend/usage.go` | **new** | `UsageInfo` struct, `UsageReporter`, `RawUsageReporter` interfaces |
| `internal/backend/openai_stream.go` | modify | Parse `usage` from final SSE event; add `stream_options` to request |
| `internal/backend/openai.go` | modify | Implement `RawUsageReporter` on `OpenAI`, return usage from stream |
| `internal/backend/kimi.go` | modify | Implement `RawUsageReporter` on `Kimi` (same SSE wire format) |
| `internal/backend/backends.go` | modify | `Anthropic.Deploy` → also implement `UsageReporter`, extract usage from stream |
| `internal/backend/deploy.go` | modify | `DeployComposed` returns `(string, *UsageInfo, error)` |
| `internal/backend/usage_test.go` | **new** | Unit tests for usage extraction |
| `internal/pipeline/flywheel.go` | modify | Add token/cost fields to `FlywheelEntry` |
| `internal/pipeline/flywheel_test.go` | modify | Test new fields serialize/deserialize correctly |
| `internal/pipeline/pricing.go` | **new** | Price table type, default prices, cost calculation |
| `internal/pipeline/pricing_test.go` | **new** | Tests for cost calculation |
| `internal/config/config.go` | modify | Add `ModelPricing` and `TokenBudget` config sections |
| `cmd/moonbase/mission.go` | modify | Pass usage to flywheel; budget enforcement in loop |
| `cmd/moonbase/flywheel_cmd.go` | modify | Add token/cost aggregation sections to output |
| `cmd/moonbase/flywheel_cmd_test.go` | **new** | Test aggregation logic |

---

## Component Designs

### 1. Usage Return Interface (`internal/backend/usage.go`)

```go
package backend

// UsageInfo holds token consumption from a single backend call.
// Nil UsageInfo means the backend doesn't report usage (not an error).
type UsageInfo struct {
    PromptTokens     int    // Tokens in the prompt/input
    CompletionTokens int    // Tokens in the completion/output
    TotalTokens      int    // Sum (some APIs report this independently)
    Model            string // Model name that served the request
}

// UsageReporter is an optional interface backends implement to return token usage.
// Check with type assertion: if ur, ok := be.(UsageReporter); ok { ... }
type UsageReporter interface {
    DeployWithUsage(agent agents.Agent, context *discovery.ProjectContext, task string) (string, *UsageInfo, error)
}

// RawUsageReporter is the usage-aware variant of RawDeployer.
type RawUsageReporter interface {
    DeployRawWithUsage(composed string, task string) (string, *UsageInfo, error)
}
```

### 2. OpenAI/Kimi SSE Usage Extraction (`internal/backend/openai_stream.go`)

The OpenAI streaming API includes usage in the final SSE event when requested:

```json
data: {"id":"...","choices":[],"usage":{"prompt_tokens":42,"completion_tokens":128,"total_tokens":170}}
```

Changes to `streamChatCompletion`:

```go
// Add to openaiRequest:
type streamOptions struct {
    IncludeUsage bool `json:"include_usage"`
}

type openaiRequest struct {
    Model         string         `json:"model"`
    Messages      []openaiMessage `json:"messages"`
    Stream        bool           `json:"stream"`
    StreamOptions *streamOptions `json:"stream_options,omitempty"`
}

// New return signature:
func streamChatCompletion(client *http.Client, baseURL, apiKey, model, composed, task string) (string, *UsageInfo, error) {
    // ... existing setup ...
    body := openaiRequest{
        // ... existing fields ...
        StreamOptions: &streamOptions{IncludeUsage: true},
    }

    // In the SSE parse loop, extract usage from events:
    var usage *UsageInfo
    // ... parse loop ...
    // After choices processing:
    if event.Usage != nil && event.Usage.TotalTokens > 0 {
        usage = &UsageInfo{
            PromptTokens:     event.Usage.PromptTokens,
            CompletionTokens: event.Usage.CompletionTokens,
            TotalTokens:      event.Usage.TotalTokens,
            Model:            model,
        }
    }

    return result.String(), usage, nil
}
```

The SSE event struct expands:

```go
var event struct {
    Choices []struct {
        Delta struct {
            Content string `json:"content"`
        } `json:"delta"`
        FinishReason *string `json:"finish_reason"`
    } `json:"choices"`
    Usage *struct {
        PromptTokens     int `json:"prompt_tokens"`
        CompletionTokens int `json:"completion_tokens"`
        TotalTokens      int `json:"total_tokens"`
    } `json:"usage"`
}
```

### 3. Anthropic Usage Extraction

Anthropic's streaming API sends usage in two places:
- `message_start` event: `message.usage.input_tokens`
- `message_delta` event: `usage.output_tokens`

Since Anthropic streaming goes through `internal/chat.Stream`, the `chat.StreamChunk` type needs a `Usage` field. Alternatively, the `Anthropic` backend can make a non-streaming call (simpler) or parse the final chunk's metadata.

**Chosen approach:** Extend `chat.StreamChunk` with an optional `Usage` field populated on the final chunk. The Anthropic backend collects it after the stream completes.

```go
// In internal/chat/stream.go, extend StreamChunk:
type StreamChunk struct {
    Text  string
    Done  bool
    Err   error
    Usage *StreamUsage // populated on Done=true chunk
}

type StreamUsage struct {
    InputTokens  int
    OutputTokens int
}
```

### 4. DeployComposed Signature Change (`internal/backend/deploy.go`)

```go
// DeployComposed returns (output, usage, error).
// Usage is nil when the backend doesn't report it.
func DeployComposed(ctx context.Context, composed, task string, timeout time.Duration) (string, *UsageInfo, error) {
    be := Preferred()

    // ... existing retry logic ...

    // Inside the retry function:
    if raw, ok := be.(RawUsageReporter); ok {
        result, usage, deployErr = raw.DeployRawWithUsage(composed, task)
    } else if raw, ok := be.(RawDeployer); ok {
        result, deployErr = raw.DeployRaw(composed, task)
        // usage stays nil
    } else if ur, ok := be.(UsageReporter); ok {
        result, usage, deployErr = ur.DeployWithUsage(agents.Agent{}, nil, composed)
    } else {
        result, deployErr = be.Deploy(agents.Agent{}, nil, composed)
    }

    return output, usage, nil
}
```

### 5. Flywheel Entry Extension (`internal/pipeline/flywheel.go`)

```go
type FlywheelEntry struct {
    SchemaVersion    int       `json:"v"`
    Timestamp        time.Time `json:"timestamp"`
    TraceID          string    `json:"trace_id"`
    Phase            int       `json:"phase"`
    Agent            string    `json:"agent"`
    Task             string    `json:"task"`
    Outcome          string    `json:"outcome"`
    RiskLevel        string    `json:"risk_level,omitempty"`
    DurationMs       int64     `json:"duration_ms"`
    OutputSize       int       `json:"output_size"`
    ReworkCount      int       `json:"rework_count"`
    // Token/cost observability (added without schema version bump per evolution contract)
    PromptTokens     int       `json:"prompt_tokens,omitempty"`
    CompletionTokens int       `json:"completion_tokens,omitempty"`
    TotalTokens      int       `json:"total_tokens,omitempty"`
    Model            string    `json:"model,omitempty"`
    EstimatedCostUSD float64   `json:"estimated_cost_usd,omitempty"`
}
```

### 6. Price Table (`internal/pipeline/pricing.go`)

```go
package pipeline

// ModelPrice holds per-million-token pricing for a model.
type ModelPrice struct {
    PromptPer1M     float64 `yaml:"prompt"`
    CompletionPer1M float64 `yaml:"completion"`
}

// DefaultPricing returns embedded prices for well-known models.
// Users can override via config. Prices are public information (no secrets).
func DefaultPricing() map[string]ModelPrice {
    return map[string]ModelPrice{
        "gpt-4o":                    {PromptPer1M: 2.50, CompletionPer1M: 10.00},
        "gpt-4o-mini":               {PromptPer1M: 0.15, CompletionPer1M: 0.60},
        "gpt-4.1":                   {PromptPer1M: 2.00, CompletionPer1M: 8.00},
        "gpt-4.1-mini":              {PromptPer1M: 0.40, CompletionPer1M: 1.60},
        "claude-sonnet-4-20250514":  {PromptPer1M: 3.00, CompletionPer1M: 15.00},
        "claude-haiku-3.5":          {PromptPer1M: 0.80, CompletionPer1M: 4.00},
        "kimi-k3":                   {PromptPer1M: 0.70, CompletionPer1M: 2.80},
    }
}

// EstimateCost computes USD cost given token counts and a pricing table.
// Returns 0 if the model is not in the table.
func EstimateCost(model string, promptTokens, completionTokens int, pricing map[string]ModelPrice) float64 {
    price, ok := pricing[model]
    if !ok {
        return 0
    }
    promptCost := float64(promptTokens) / 1_000_000 * price.PromptPer1M
    completionCost := float64(completionTokens) / 1_000_000 * price.CompletionPer1M
    return promptCost + completionCost
}
```

### 7. Config Extensions (`internal/config/config.go`)

```go
type Config struct {
    // ... existing fields ...

    // Token/cost observability
    ModelPricing map[string]ModelPrice `yaml:"model_pricing,omitempty"` // Override default model prices
    TokenBudget  TokenBudgetConfig     `yaml:"token_budget,omitempty"` // Per-mission token budget
}

type TokenBudgetConfig struct {
    MaxTokensPerMission int `yaml:"max_tokens_per_mission"` // 0 = unlimited (default)
    WarnThresholdPct    int `yaml:"warn_threshold_pct"`     // Warn at this % of budget (default 80)
}
```

Note: `ModelPrice` is defined in `internal/pipeline/pricing.go` and imported by config. If this creates a cycle, define the YAML struct inline in config and convert at load time.

### 8. Budget Enforcement (`cmd/moonbase/mission.go`)

Budget checking is added inside `runPipelineLoop` after each successful phase:

```go
// After executeAndRecordPhase succeeds:
if usage != nil && budgetCfg.MaxTokensPerMission > 0 {
    missionTokens += usage.TotalTokens
    pct := (missionTokens * 100) / budgetCfg.MaxTokensPerMission

    if missionTokens > budgetCfg.MaxTokensPerMission {
        fmt.Printf("   🛑 Token budget exceeded (%dK / %dK). Pipeline stopped.\n",
            missionTokens/1000, budgetCfg.MaxTokensPerMission/1000)
        flywheel.Append(FlywheelEntry{..., Outcome: "budget_exceeded"})
        pipeline.SaveCheckpoint(p, checkpointDir)
        return
    }

    if pct >= budgetCfg.WarnThresholdPct {
        fmt.Printf("   ⚠️  Token budget: %d%% used (%dK / %dK)\n",
            pct, missionTokens/1000, budgetCfg.MaxTokensPerMission/1000)
    }
}
```

### 9. Flywheel Display (`cmd/moonbase/flywheel_cmd.go`)

New section appended after existing analysis:

```
   💰 Token & Cost Summary:
      Total tokens:     1.2M prompt / 340K completion
      Total est. cost:  $8.42

   💰 Cost per Agent (avg per invocation):
      numbuh-3 (Implementation)   $2.14  (68K prompt, 18K completion)
      numbuh-1 (Analysis)         $0.89  (32K prompt, 6K completion)
      numbuh-4 (QA)               $0.72  (28K prompt, 8K completion)
      numbuh-2 (Architecture)     $0.51  (22K prompt, 5K completion)

   💰 Cost per Mission (avg):
      Avg tokens/mission:  142K
      Avg cost/mission:    $1.68
      Most expensive:      20250730T... ($4.21, 380K tokens)

   💰 Cost-Heavy Phase:
      Phase 3 (Implementation) — avg $2.14/invocation (42% of mission cost)
```

Logic:
- Skip the entire section if no entries have token data
- Group by agent name for per-agent averages
- Group by trace_id for per-mission totals
- Rank phases by average `estimated_cost_usd` for cost-heavy detection

---

## Data Flow

```
Backend.Deploy/DeployRaw
    ↓ (implements UsageReporter/RawUsageReporter)
returns (output string, *UsageInfo, error)
    ↓
DeployComposed (internal/backend/deploy.go)
    ↓ returns (output, *UsageInfo, error)
executeAndRecordPhase (cmd/moonbase/mission.go)
    ↓ computes cost via pricing table
    ↓ records in FlywheelEntry{PromptTokens, CompletionTokens, Model, EstimatedCostUSD}
    ↓ checks budget (warn / stop)
FlywheelLog.Append → ~/.moonbase/flywheel.jsonl
    ↓
moonbase flywheel (reads JSONL, aggregates, displays)
```

---

## Alternatives Considered

### A: Change `Backend.Deploy` signature directly
- Pro: Simple, single path
- Con: Breaks all 7 backends and all callers; clipboard/ollama would return nil always
- **Rejected:** Too much churn for backends that can never report usage

### B: Separate token-counting middleware (count tokens before sending)
- Pro: Works for all backends including CLI-based ones
- Con: Token counting libraries are large dependencies; counts are approximate; doesn't match actual provider billing
- **Rejected:** Provider-reported usage is more accurate and free

### C: Store only tokens, compute cost at display time
- Pro: Prices update automatically when you change config
- Con: Historical cost becomes unreliable if prices change; display needs config access
- **Rejected:** Write-time cost is simpler and historically accurate

---

## Edge Cases

- **Retry:** If a phase retries 2 times, each attempt may produce usage. Only the successful attempt's usage is recorded in the final flywheel entry. Failed attempts' usage is lost (acceptable — it's a small fraction).
- **Parallel conditional phases:** Each parallel goroutine returns its own usage. Budget enforcement doesn't apply to parallel phases (they're fire-and-forget specialists).
- **Model name mismatch:** If the API returns a different model name than requested (e.g., alias resolution), use the returned model for pricing lookup.
- **Zero-token response:** Some API errors return usage with 0 completion tokens. Record as-is.
- **Rate limit / timeout:** Failed phases record no usage (consistent with current behavior of recording outcome "failed").
