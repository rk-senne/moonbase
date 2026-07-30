# Design: Adaptive Pipeline Depth

## Architecture Decision

Adaptive depth is implemented as a **pre-pipeline classifier** + **in-pipeline escalation hook** — not a new pipeline type. The key insight: `NewFast` already demonstrates the pattern (skip phases by setting `StatusSkipped`). We generalize this to three depth profiles while keeping the single `Pipeline` struct and execution loop unchanged.

**Key decisions:**

1. **Classifier lives in `internal/pipeline/`** — it's pipeline logic, not CLI logic. The CLI calls it, the pipeline owns it.
2. **Classification is deterministic** — pure string heuristics (length, keywords, path count). No AI call, no network. Microsecond latency.
3. **Escalation piggybacks on the existing rework mechanism** — `RouteToPhase` already re-routes. Escalation just "unskips" phases before re-routing.
4. **`NewFast` remains unchanged** — `--fast` is a hard override with no escalation. The new `NewAdaptive` constructor handles auto-depth.
5. **Flywheel additions are optional fields** — `omitempty` JSON tags, schema version stays at 1.

---

## Files Affected

| File | Change Type | Purpose |
|------|-------------|---------|
| `internal/pipeline/depth.go` | **new** | Complexity classifier + depth types + `NewAdaptive` constructor |
| `internal/pipeline/depth_test.go` | **new** | Table-driven tests for classifier |
| `internal/pipeline/pipeline.go` | modify | Add `Depth` and `DepthReason` fields to `Pipeline` struct; add `Escalate` method |
| `internal/pipeline/flywheel.go` | modify | Add `Depth`, `DepthReason`, `EscalatedFrom`, `EscalatedTo` fields to `FlywheelEntry` |
| `cmd/moonbase/mission_cmd.go` | modify | Add `--full` and `--depth` flags; mutual exclusivity check |
| `cmd/moonbase/mission.go` | modify | Add `runMissionAdaptive` function; wire depth into flywheel entries |
| `cmd/moonbase/mission_cmd.go` | modify | Route to adaptive/fast/full based on flags |

---

## Component Designs

### 1. Complexity Classifier (`internal/pipeline/depth.go`)

```go
package pipeline

// Depth represents the pipeline execution depth.
type Depth string

const (
    DepthTrivial Depth = "trivial"
    DepthSimple  Depth = "simple"
    DepthComplex Depth = "complex"
)

// DepthClassification holds the result of task complexity analysis.
type DepthClassification struct {
    Depth  Depth  // The classified depth
    Reason string // Human-readable explanation for flywheel/CLI output
}

// ClassifyTask analyzes a task description and returns the recommended
// pipeline depth. Reuses the reasoning-protocol task-scaling ladder:
//   - trivial: fix directly, verify builds (short, no complexity signals)
//   - simple:  read context → implement → test (moderate, some signals)
//   - complex: full protocol (long, multiple signals, multi-scope)
//
// Ambiguity resolves to simple — never under-estimates to trivial.
func ClassifyTask(task string) DepthClassification {
    signals := countComplexitySignals(task)
    length := len(task)
    paths := countFilePaths(task)

    // Complex: long tasks, many signals, multi-scope
    if length > 200 || signals >= 3 || paths >= 3 {
        return DepthClassification{
            Depth:  DepthComplex,
            Reason: complexReason(length, signals, paths),
        }
    }

    // Trivial: short tasks with zero complexity signals
    if length <= 80 && signals == 0 && paths <= 1 {
        return DepthClassification{
            Depth:  DepthTrivial,
            Reason: "short task, no complexity signals",
        }
    }

    // Default: simple
    return DepthClassification{
        Depth:  DepthSimple,
        Reason: simpleReason(length, signals, paths),
    }
}
```

**Complexity signal keywords** (drawn from reasoning-protocol vocabulary):

```go
// complexityKeywords are words/phrases that indicate non-trivial work.
// Grouped by category for maintainability.
var complexityKeywords = []string{
    // Feature addition
    "implement", "add", "create", "build", "introduce", "new endpoint",
    // Structural change
    "refactor", "redesign", "migrate", "restructure", "architecture",
    // System concerns
    "rate limit", "pagination", "authentication", "authorization",
    "caching", "concurrency", "performance",
    // Multi-step
    "and then", "followed by", "across all", "every",
}

// trivialKeywords are words that suggest minimal scope.
// Their presence alone doesn't override other signals.
var trivialKeywords = []string{
    "fix typo", "rename", "remove unused", "update comment",
    "fix import", "formatting", "whitespace", "spelling",
}
```

**Signal counting:**
- Each complexity keyword match = +1 signal
- Each trivial keyword match = -1 signal (can cancel, but floor at 0)
- Task referencing ≥ 3 file paths = +2 signals

**File path detection** reuses the `extractFilesChanged` heuristic already in `context.go` (look for strings with `/` and common extensions).

---

### 2. Pipeline Depth Fields

Add to `Pipeline` struct in `pipeline.go`:

```go
type Pipeline struct {
    // ... existing fields ...
    Depth       Depth  // Effective depth: trivial, simple, complex, or override value
    DepthReason string // Why this depth was selected (for flywheel + CLI display)
    Escalated   bool   // True if depth was promoted mid-pipeline
    OrigDepth   Depth  // Original depth before escalation (empty if no escalation)
}
```

---

### 3. Adaptive Pipeline Constructor

```go
// NewAdaptive creates a pipeline with phases configured for the given depth.
// Phases not included in the depth profile are pre-skipped.
func NewAdaptive(task string, depth Depth, reason string) *Pipeline {
    p := New(task)
    p.Depth = depth
    p.DepthReason = reason

    switch depth {
    case DepthTrivial:
        // Only Phase 3 + 4 (same as NewFast but with escalation enabled)
        for i := range p.Phases {
            n := p.Phases[i].Number
            if n != 3 && n != 4 {
                p.Phases[i].Status = StatusSkipped
            }
        }
    case DepthSimple:
        // Phase 1 + 3 + 4 (skip Architecture and Review initially)
        for i := range p.Phases {
            n := p.Phases[i].Number
            if n == 2 || n == 5 {
                p.Phases[i].Status = StatusSkipped
            }
        }
    case DepthComplex:
        // All phases active (same as New) — no changes needed
    }

    return p
}
```

---

### 4. Escalation Logic

Escalation is triggered when the risk gate returns MEDIUM or HIGH on a shallow pipeline. The pipeline "unskips" the necessary phases and re-routes.

```go
// Escalate promotes the pipeline to a deeper depth. It un-skips phases
// that the new depth requires and returns the phase to route to.
// Returns an error if the pipeline is already at the target depth.
func (p *Pipeline) Escalate(targetDepth Depth) error {
    if p.Depth == targetDepth || p.Depth == DepthComplex {
        return fmt.Errorf("already at depth %s, cannot escalate to %s", p.Depth, targetDepth)
    }

    p.OrigDepth = p.Depth
    p.Depth = targetDepth
    p.Escalated = true

    switch targetDepth {
    case DepthSimple:
        // Un-skip Phase 1 (Analysis)
        p.unskipPhase(1)
    case DepthComplex:
        // Un-skip Phases 1 and 2 (Analysis + Architecture)
        p.unskipPhase(1)
        p.unskipPhase(2)
    }

    return nil
}

// unskipPhase resets a skipped phase to pending so it can be executed.
func (p *Pipeline) unskipPhase(number int) {
    for i := range p.Phases {
        if p.Phases[i].Number == number && p.Phases[i].Status == StatusSkipped {
            p.Phases[i].Status = StatusPending
        }
    }
}
```

---

### 5. Escalation Integration in Mission Loop

The escalation hook lives in `runPipelineLoop` after the risk gate evaluation. The key change: when `reworkOnRisk` is true AND the pipeline is not at `DepthComplex`, MEDIUM/HIGH risk can trigger escalation *before* the normal rework routing.

```go
// In runPipelineLoop, after phase 4 completes, within the reworkOnRisk block:

if phase.Number == 4 && opts.reworkOnRisk {
    routing, rErr := p.ApplyRiskGate(output)
    
    // Escalation check: if depth is shallow and risk is non-LOW, escalate
    if opts.allowEscalation && routing.Level != RiskLow && p.Depth != DepthComplex {
        targetDepth := escalationTarget(p.Depth, routing.Level)
        if targetDepth != p.Depth {
            fmt.Printf("   ⬆️  Escalating: %s → %s (QA risk: %s)\n", p.Depth, targetDepth, routing.Level)
            p.Escalate(targetDepth)
            
            // Log escalation to flywheel
            flywheel.Append(FlywheelEntry{
                Timestamp:    time.Now().UTC(),
                TraceID:      p.TraceID,
                Phase:        phase.Number,
                Agent:        phase.AgentName,
                Task:         task,
                Outcome:      "escalated",
                RiskLevel:    string(routing.Level),
                Depth:        string(targetDepth),
                EscalatedFrom: string(p.OrigDepth),
                EscalatedTo:  string(targetDepth),
            })
            
            // Route to the earliest unskipped phase
            // (Phase 1 if newly added, or Phase 2 if escalating to complex)
            targetPhase := earliestPendingPhase(p)
            i = targetPhase - 1  // loop will increment
            continue
        }
    }
    
    // Normal risk gate handling (unchanged)
    shouldContinue, targetIdx := handleRiskGate(p, output)
    // ...
}
```

**Escalation target mapping:**

```go
// escalationTarget determines the depth to escalate to based on current
// depth and risk level.
func escalationTarget(current Depth, risk RiskLevel) Depth {
    switch {
    case risk == RiskHigh || risk == RiskCritical:
        return DepthComplex
    case risk == RiskMedium && current == DepthTrivial:
        return DepthSimple
    case risk == RiskMedium && current == DepthSimple:
        return DepthComplex
    default:
        return current
    }
}
```

**Critical invariant:** CRITICAL risk still stops the pipeline immediately — escalation only applies to MEDIUM and HIGH. The `ApplyRiskGate` call happens first, and if it returns CRITICAL, `p.Stop()` is called before escalation logic runs.

---

### 6. Flywheel Entry Additions

```go
type FlywheelEntry struct {
    // ... existing fields ...
    Depth         string `json:"depth,omitempty"`          // Pipeline depth for this run
    DepthReason   string `json:"depth_reason,omitempty"`   // Why this depth (first entry only)
    EscalatedFrom string `json:"escalated_from,omitempty"` // Original depth before escalation
    EscalatedTo   string `json:"escalated_to,omitempty"`   // New depth after escalation
}
```

All new fields use `omitempty` — existing entries without these fields remain valid. Schema version stays at 1.

---

### 7. CLI Flag Additions (`mission_cmd.go`)

```go
var missionFull bool
var missionDepth string

func init() {
    missionCmd.Flags().BoolVar(&missionFull, "full", false, "run all pipeline phases regardless of task complexity")
    missionCmd.Flags().StringVar(&missionDepth, "depth", "", "override auto-classification (trivial|simple|complex)")
    missionCmd.MarkFlagsMutuallyExclusive("fast", "full", "depth")
}
```

**Dispatch logic in mission command's Run:**

```go
switch {
case missionFast:
    runMissionFast(task)
case missionFull:
    runMissionFull(task) // same as current runMission, logs "override:full"
case missionDepth != "":
    depth := validateDepthFlag(missionDepth) // exits with error on invalid
    runMissionAdaptive(task, depth, "override:"+missionDepth)
default:
    // Auto-classify
    classification := pipeline.ClassifyTask(task)
    runMissionAdaptive(task, classification.Depth, classification.Reason)
}
```

---

### 8. Review Phase Addition After Escalation

When a trivial or simple pipeline escalates and eventually passes QA with LOW risk, Phase 5 (Review) is un-skipped and executed as the final gate. This ensures escalated pipelines get full review coverage.

```go
// After QA passes with LOW on an escalated pipeline:
if p.Escalated && routing.Level == RiskLow {
    p.unskipPhase(5) // Ensure review runs
}
```

For non-escalated trivial/simple pipelines that pass QA with LOW risk, Review remains skipped — the shallow path is validated.

---

### 9. pipelineLoopOptions Extension

```go
type pipelineLoopOptions struct {
    // ... existing fields ...
    allowEscalation bool // Enable mid-pipeline depth promotion
}
```

- `runMissionFast`: `allowEscalation: false` (--fast is a hard override)
- `runMissionAdaptive`: `allowEscalation: true`
- `runMission` (--full): `allowEscalation: false` (already at max depth)

---

## Execution Flow Diagrams

### Trivial (No Escalation)

```
ClassifyTask("fix typo in README") → trivial
    ↓
Phase 3 (Implementation)
    ↓
Phase 4 (QA) → LOW
    ↓
✅ Done (Review skipped)
```

### Trivial → Escalation to Simple

```
ClassifyTask("fix the auth check") → trivial
    ↓
Phase 3 (Implementation)
    ↓
Phase 4 (QA) → MEDIUM
    ↓
⬆️ Escalate: trivial → simple
    ↓
Phase 1 (Analysis) ← newly unskipped
    ↓
Phase 3 (Implementation) ← re-run with analysis context
    ↓
Phase 4 (QA) → LOW
    ↓
Phase 5 (Review) ← added because escalated
    ↓
✅ Done
```

### Simple → Escalation to Complex

```
ClassifyTask("add rate limiting") → simple
    ↓
Phase 1 (Analysis)
    ↓
Phase 3 (Implementation)
    ↓
Phase 4 (QA) → HIGH
    ↓
⬆️ Escalate: simple → complex
    ↓
Phase 2 (Architecture) ← newly unskipped
    ↓
Phase 3 (Implementation) ← re-run with design context
    ↓
Phase 4 (QA) → LOW
    ↓
Phase 5 (Review) ← added because escalated
    ↓
✅ Done
```

---

## Invariants

1. **QA always runs.** No depth profile skips Phase 4.
2. **Risk gate logic is untouched.** `ParseRiskGate`, `ApplyRiskGate`, `riskRouting` are not modified.
3. **MaxRework still applies.** Escalation increments `ReworkCount` (it routes back via the same mechanism).
4. **`--fast` is unchanged.** No escalation, informational risk gate only, same as today.
5. **CRITICAL always stops.** Escalation cannot override a CRITICAL verdict.
6. **Conditional phases are independent.** They evaluate their triggers regardless of depth.
7. **Classification is pure.** No side effects, no I/O, no AI calls — unit-testable.
8. **Flywheel is backward-compatible.** New fields are `omitempty`; old readers ignore them.

---

## Alternatives Considered

| Approach | Why Not |
|----------|---------|
| AI-based classification (ask an LLM to assess complexity) | Adds latency, cost, and non-determinism. The heuristic is good enough; flywheel data can tune it later. |
| User-prompted classification ("Is this trivial/simple/complex?") | Defeats the purpose of automation. Users can use `--depth` if they want control. |
| Three separate pipeline constructors (NewTrivial, NewSimple, NewComplex) | Creates code duplication. `NewAdaptive(task, depth, reason)` handles all three with a switch. |
| Disable escalation by default, opt-in with `--escalate` | Against the design principle: start shallow, escalate at a ceiling. Users shouldn't have to know about escalation. |
| Skip QA on trivial tasks | Violates the core invariant. Even "fix typo" can accidentally break something. |
