# Design: Parallel Fan-Out Execution of Independent Conditional Specialists

## Architecture Decision

The fan-out lives entirely in `internal/pipeline/` as a new `fanout.go` file, orchestrated by the existing `startNextPhase` / `handlePhaseResult` flow in the TUI layer. The pattern follows the 2026 fan-out orchestration model: identify independent work, dispatch concurrently with bounded parallelism via `errgroup` + semaphore, collect results into a thread-safe aggregation buffer, then merge deterministically into the pipeline context.

**Key decisions:**

1. **errgroup with weighted semaphore** — `golang.org/x/sync/errgroup` provides structured concurrency with cancellation. A semaphore (channel of size N) caps active goroutines. This is the standard Go fan-out pattern for 2026.
2. **Phase-number-keyed isolation** — Each specialist writes to its own `PhaseOutputs[phaseNum]` slot. No two specialists share a key. The aggregation mutex only guards the merge step, not individual execution.
3. **Aggregation buffer before context merge** — Results collect into a local `[]FanOutResult` slice, then merge into `PipelineContext` in phase-number order after all goroutines complete. This guarantees determinism.
4. **Partial failure via `errgroup` tolerance** — Unlike vanilla errgroup (which cancels all on first error), we use a custom collector that records per-specialist success/failure without cancelling siblings.
5. **New TUI message: `FanOutCompleteMsg`** — A single message carries all results to the Elm update loop, avoiding N individual `PhaseResultMsg` interleaving with other updates.
6. **Config-driven** — `max_specialist_concurrency` and `parallel_specialists` fields in Config. Pipeline reads these at construction time.

---

## Files Affected

| File | Change Type | Purpose |
|------|-------------|---------|
| `internal/pipeline/fanout.go` | **new** | Fan-out orchestrator: `RunSpecialists()`, `FanOutResult`, semaphore logic |
| `internal/pipeline/fanout_test.go` | **new** | Unit tests for fan-out: concurrency, ordering, partial failure, cancellation |
| `internal/pipeline/pipeline.go` | modify | Add `ParallelSpecialists bool`, `MaxSpecialistConcurrency int` to `Pipeline` struct; populate from config |
| `internal/pipeline/context.go` | modify | Add `MergeSpecialistResults([]FanOutResult)` method for deterministic merge |
| `internal/pipeline/checkpoint.go` | modify | Include specialist fan-out status in `Checkpoint` struct (new optional field) |
| `internal/pipeline/flywheel.go` | modify | Add `ParallelGroup string` field to `FlywheelEntry` |
| `internal/config/config.go` | modify | Add `MaxSpecialistConcurrency int` and `ParallelSpecialists bool` fields |
| `internal/tui/pipeline_exec.go` | modify | Replace sequential specialist loop with fan-out dispatch; handle `FanOutCompleteMsg` |
| `internal/tui/views_pipeline.go` | modify | Render parallel execution status in pipeline view |
| `cmd/moonbase/mission.go` | modify | Add `--sequential` flag |
| `go.mod` | modify | Add `golang.org/x/sync` dependency |

---

## Component Designs

### 1. Fan-Out Orchestrator (`internal/pipeline/fanout.go`)

```go
package pipeline

import (
    "context"
    "fmt"
    "sort"
    "sync"
    "time"
)

// FanOutResult captures the outcome of a single specialist execution.
type FanOutResult struct {
    Phase    int           // Phase number of the specialist
    Agent    string        // Agent name (e.g., "numbuh-274")
    Output   string        // Agent output (empty on failure)
    Err      error         // Non-nil if specialist failed
    Duration time.Duration // Wall-clock execution time
    Status   PhaseStatus   // StatusComplete or StatusFailed
}

// FanOutConfig controls parallel specialist execution.
type FanOutConfig struct {
    MaxConcurrency int           // Semaphore size (default 4)
    PhaseTimeout   time.Duration // Per-specialist timeout
    TraceID        string        // For flywheel correlation
}

// SpecialistFunc is the execution function for a single specialist.
// Matches the signature needed to invoke a backend with an agent.
type SpecialistFunc func(ctx context.Context, phase Phase) (string, error)

// RunSpecialists executes triggered specialists concurrently with bounded parallelism.
// Returns results sorted by phase number for deterministic aggregation.
// Partial failures do NOT cancel other specialists.
func RunSpecialists(
    ctx context.Context,
    phases []Phase,
    execute SpecialistFunc,
    cfg FanOutConfig,
) []FanOutResult {
    if len(phases) == 0 {
        return nil
    }

    // Clamp concurrency
    maxConc := cfg.MaxConcurrency
    if maxConc <= 0 {
        maxConc = 4
    }
    if maxConc > 16 {
        maxConc = 16
    }

    // Semaphore: buffered channel limits active goroutines
    sem := make(chan struct{}, maxConc)

    var mu sync.Mutex
    results := make([]FanOutResult, 0, len(phases))

    var wg sync.WaitGroup
    for _, phase := range phases {
        wg.Add(1)
        go func(p Phase) {
            defer wg.Done()

            // Acquire semaphore slot
            select {
            case sem <- struct{}{}:
                defer func() { <-sem }()
            case <-ctx.Done():
                mu.Lock()
                results = append(results, FanOutResult{
                    Phase:  p.Number,
                    Agent:  p.AgentName,
                    Err:    ctx.Err(),
                    Status: StatusFailed,
                })
                mu.Unlock()
                return
            }

            start := time.Now()
            output, err := execute(ctx, p)
            elapsed := time.Since(start)

            result := FanOutResult{
                Phase:    p.Number,
                Agent:    p.AgentName,
                Output:   output,
                Duration: elapsed,
            }
            if err != nil {
                result.Err = err
                result.Status = StatusFailed
            } else {
                result.Status = StatusComplete
            }

            mu.Lock()
            results = append(results, result)
            mu.Unlock()
        }(phase)
    }

    wg.Wait()

    // Sort by phase number for deterministic ordering
    sort.Slice(results, func(i, j int) bool {
        return results[i].Phase < results[j].Phase
    })

    return results
}

// TriggeredSpecialists evaluates which conditional specialists should fire
// and returns only those that pass their trigger check.
func TriggeredSpecialists(phases []Phase, pctx *PipelineContext) []Phase {
    var triggered []Phase
    for _, p := range phases {
        if !p.Conditional {
            continue
        }
        result := EvaluateTrigger(p.TriggerSpec, pctx)
        if result.Invoke {
            triggered = append(triggered, p)
        }
    }
    return triggered
}
```

**Design rationale:**
- `sync.WaitGroup` + mutex instead of errgroup because we explicitly do NOT want first-error cancellation. Each specialist is independent.
- Semaphore via buffered channel is idiomatic Go and avoids the `golang.org/x/sync/semaphore` weighted API complexity.
- Results sorted post-collection rather than pre-allocated by index to handle the case where some specialists are cancelled before starting.

---

### 2. Deterministic Aggregation (`internal/pipeline/context.go` addition)

```go
// MergeSpecialistResults writes fan-out results into the pipeline context
// in phase-number order. Failed specialists are recorded with an error marker.
func (pc *PipelineContext) MergeSpecialistResults(results []FanOutResult) {
    for _, r := range results {
        if r.Err != nil {
            pc.PhaseOutputs[r.Phase] = fmt.Sprintf("[SPECIALIST FAILED: %v]", r.Err)
        } else {
            pc.PhaseOutputs[r.Phase] = r.Output
        }
        // Extract files from successful specialist output
        if r.Output != "" {
            files := extractFilesChanged(r.Output)
            for _, f := range files {
                if !contains(pc.FilesChanged, f) {
                    pc.FilesChanged = append(pc.FilesChanged, f)
                }
            }
        }
    }
}
```

The caller (`RunSpecialists`) guarantees results arrive sorted by phase number, so the merge is deterministic. Files are appended in phase order — same result every time.

---

### 3. Pipeline Struct Additions (`internal/pipeline/pipeline.go`)

```go
// Added to Pipeline struct:
type Pipeline struct {
    // ... existing fields ...
    ParallelSpecialists     bool  // Whether to fan-out specialists (default true)
    MaxSpecialistConcurrency int  // Semaphore size for fan-out (default 4)
}

// In New():
func New(task string) *Pipeline {
    return &Pipeline{
        // ... existing ...
        ParallelSpecialists:      true,
        MaxSpecialistConcurrency: 4,
    }
}
```

The config loader sets these from `Config.ParallelSpecialists` and `Config.MaxSpecialistConcurrency`.

---

### 4. Flywheel Integration

```go
// FlywheelEntry gains:
type FlywheelEntry struct {
    // ... existing fields ...
    ParallelGroup string `json:"parallel_group,omitempty"` // Groups entries from same fan-out batch
}
```

The fan-out orchestrator generates a `parallelGroup` ID (`traceID + "-fanout"`) and passes it to all entries in the batch. This enables `moonbase flywheel` to correlate parallel specialist durations.

---

### 5. Checkpoint Extension

```go
// Checkpoint gains an optional field (schema version stays at 1 per evolution contract):
type Checkpoint struct {
    // ... existing fields ...
    SpecialistResults map[int]string `json:"specialist_results,omitempty"` // Phase → "complete"/"failed"
}
```

The field is optional — legacy checkpoints without it are handled gracefully (nil map = no specialist data).

---

### 6. TUI Integration (`internal/tui/pipeline_exec.go`)

New message type for the Elm architecture:

```go
// FanOutCompleteMsg carries all specialist results as a single batch update.
type FanOutCompleteMsg struct {
    Results []pipeline.FanOutResult
}
```

The `handlePhaseResult` for Phase 4 (QA) changes:
- After `ApplyRiskGate` returns `RiskLow`, instead of calling `Advance()` + `startNextPhase()` (which iterates sequentially), it dispatches a `tea.Cmd` that runs `RunSpecialists` and returns `FanOutCompleteMsg`.
- A new `handleFanOutComplete(msg FanOutCompleteMsg) tea.Cmd` method merges results, appends chat messages per specialist, then advances to Phase 5.

```go
func (a *App) startFanOut() tea.Cmd {
    return func() tea.Msg {
        state := a.views.Pipeline.State

        // Identify triggered specialists
        specialists := state.Phases[5:] // phases after Review (indices 5, 6, 7 = phases 6, 7, 8)
        triggered := pipeline.TriggeredSpecialists(specialists, state.Context)

        if len(triggered) == 0 {
            return FanOutCompleteMsg{Results: nil}
        }

        // Mark all as running
        for i := range triggered {
            triggered[i].StartPhase()
        }

        cfg := pipeline.FanOutConfig{
            MaxConcurrency: state.MaxSpecialistConcurrency,
            PhaseTimeout:   state.PhaseTimeout,
            TraceID:        state.TraceID,
        }

        // Execute function wraps the backend call
        execute := func(ctx context.Context, phase pipeline.Phase) (string, error) {
            agent := a.registry.GetByName(phase.AgentName)
            if agent == nil {
                return "", fmt.Errorf("agent %s not found", phase.AgentName)
            }
            phaseInput := state.Context.ForPhase(phase.Number)
            composed := discovery.ComposePrompt(agent.Prompt, a.projectCtx, phaseInput)
            return a.env.Backend.Active.Deploy(*agent, a.projectCtx, composed)
        }

        results := pipeline.RunSpecialists(a.views.Pipeline.Ctx, triggered, execute, cfg)
        return FanOutCompleteMsg{Results: results}
    }
}
```

---

### 7. Config Additions (`internal/config/config.go`)

```go
type Config struct {
    // ... existing fields ...

    // Parallel specialist execution (fan-out orchestration).
    ParallelSpecialists     bool `yaml:"parallel_specialists,omitempty"`      // Enable fan-out (default true)
    MaxSpecialistConcurrency int  `yaml:"max_specialist_concurrency,omitempty"` // Concurrency cap (default 4, range 1-16)
}

// In DefaultConfig():
func DefaultConfig() Config {
    return Config{
        // ... existing ...
        ParallelSpecialists:      true,
        MaxSpecialistConcurrency: 4,
    }
}
```

---

### 8. Sequential Fallback

When `ParallelSpecialists` is `false` (or `--sequential` flag), the pipeline reverts to the existing behavior: iterating specialists via `startNextPhase` one at a time. No code path is removed — the fan-out is an additive optimization gated by config.

```go
// In handlePhaseResult, after risk gate returns RiskLow:
if state.ParallelSpecialists && !a.sequentialOverride {
    return a.startFanOut()
}
// else: existing sequential Advance() + startNextPhase()
```

---

### 9. Pipeline Flow (Updated)

```
Phase 1 (Analysis) → Phase 2 (Architecture) → Phase 3 (Implementation)
    → Phase 4 (QA) → [Risk Gate]
        ├── MEDIUM/HIGH/CRITICAL → rework/stop (unchanged)
        └── LOW → Fan-Out Gate
                    ├── parallel_specialists=false → sequential (existing)
                    └── parallel_specialists=true →
                         ┌──────────────────────────────────────┐
                         │  Fan-Out (bounded concurrency = N)   │
                         │                                      │
                         │  ┌─ Numbuh 0 (Oversight)     ─┐     │
                         │  ├─ Numbuh 274 (Security)    ─┤     │
                         │  ├─ Numbuh 362 (Infra)       ─┤ concurrent
                         │  ├─ Numbuh 86 (Dead-code)    ─┤     │
                         │  ├─ Numbuh 999 (Docs)        ─┤     │
                         │  └─ Sector Z (Legacy)        ─┘     │
                         │                                      │
                         │  → Aggregate (sort by phase #)       │
                         │  → Merge into PipelineContext         │
                         │  → Checkpoint                        │
                         │  → Flywheel entries                  │
                         └──────────────────────────────────────┘
                    → Phase 5 (Review / Numbuh 5)
    → Human Approval
```

---

## Concurrency Safety Analysis

| Shared State | Access Pattern | Protection |
|---|---|---|
| `PipelineContext.PhaseOutputs` | Written AFTER fan-out (single-threaded merge) | No concurrent write — safe |
| `FanOutResult` slice | Written by goroutines during fan-out | `sync.Mutex` in `RunSpecialists` |
| Semaphore channel | Read/write by goroutines | Channel semantics (safe by design) |
| `FlywheelLog.Append` | Called per-specialist after fan-out | Single-writer (sequential after collect) |
| TUI state | Updated only via `FanOutCompleteMsg` in Elm loop | Single-threaded (Bubble Tea guarantee) |

**Key insight:** The aggregation buffer (`[]FanOutResult`) is the only shared mutable state during fan-out. After `wg.Wait()`, all mutations are done and the slice is read-only. The merge into `PipelineContext` happens in the caller's goroutine (the Elm update loop), so no concurrent access to `PipelineContext` occurs.

---

## Alternatives Considered

| Approach | Pros | Cons | Decision |
|---|---|---|---|
| `errgroup` with `SetLimit()` | Clean API, built-in context cancellation | First error cancels all — wrong for partial tolerance | Rejected |
| Worker pool with job queue | Flexible, reusable | Over-engineered for ≤6 specialists | Rejected |
| `sync.WaitGroup` + semaphore channel | Full control over cancellation semantics, idiomatic | Slightly more code than errgroup | **Selected** |
| No parallelism (status quo) | Zero risk | Wastes wall-clock time on multi-specialist missions | Rejected as default |

---

## Testing Strategy

| Scenario | Test Type | Key Assertion |
|---|---|---|
| 3 specialists, all succeed | Unit | Results sorted by phase number; all `StatusComplete` |
| 1 of 3 fails | Unit | 2 complete, 1 failed; pipeline continues |
| All fail | Unit | All `StatusFailed`; `FanOutCompleteMsg` still delivered |
| Concurrency cap respected | Unit | With cap=1, specialists run sequentially (verify via timestamps) |
| Context cancellation | Unit | All in-flight specialists receive `ctx.Err()` |
| Deterministic ordering | Unit | Run 100 iterations; output order never varies |
| Config `parallel_specialists: false` | Integration | Falls back to sequential execution |
| Checkpoint contains specialist results | Unit | Serialized checkpoint has `specialist_results` field |
| Flywheel entries have parallel_group | Unit | All entries in batch share the same group ID |
