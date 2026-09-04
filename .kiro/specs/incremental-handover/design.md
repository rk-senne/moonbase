# Incremental Handover — Design

## Intent

Shrink the unit of work and verification from a whole phase to a single increment,
and let verification of increment N−1 run concurrently with construction of
increment N.

The agent-level half is already in place: Numbuh 3 works in increments and
maintains a Green Ledger, Numbuh 4 audits that ledger row by row. That gives small
batches today. This spec covers the pipeline-level half, which needs Go changes and
is therefore kept separate for review.

## Why, stated as values rather than process

Human team process is shaped by two constraints agents don't share: context
switching is expensive, and colleagues cannot audit each other's every claim — so
process substitutes trust and batching.

Neither holds here. Handover costs an agent nothing, and every claim can carry
machine-checkable evidence. So the rules should suit what agents are actually good
at:

- **Small batches, because they are free.** No context-switch penalty means no
  reason to accumulate work before handing it off.
- **Evidence instead of trust.** A claim without attached proof isn't a weaker
  claim, it's an unverified one.
- **Concurrency only where it is safe.** Verification is read-only, so it can run
  alongside construction without conflicting.

The one human practice that gets *stronger* here, not weaker: the expectation must
exist before the implementation. An agent shown code writes tests that agree with
it. That is a property of pattern completion, not of discipline, so it needs a
structural guard rather than a cultural one.

## The constraint that shapes the design

`pipeline.IsIndependentSpecialist` permits concurrency only when the phase is
conditional **and** the agent is read-only or lacks the `write` tool:

```go
func IsIndependentSpecialist(phase Phase, tools []string, shellReadOnly *bool) bool {
	if !phase.Conditional { return false }
	if shellReadOnly != nil && *shellReadOnly { return true }
	for _, t := range tools { if t == "write" { return false } }
	return true
}
```

This is correct and must not be relaxed. Two agents writing Go files in one package
collide on shared identifiers, and Go's one-package-per-directory model offers no
isolation to fall back on.

Therefore: **do not parallelise writing. Parallelise roles.**

## Proposed model

### Increment as a first-class unit

The phase model has no sub-phase concept — `MaxRework`, `ReworkCount` and
`RetryPhase()` all operate on whole phases. Add:

```go
// Increment is one independently verifiable change within a phase.
type Increment struct {
    Seq         int         // order within the phase
    Expectation string      // what must be true
    TestName    string      // the test asserting it
    Status      PhaseStatus // reuses the existing status enum
    Evidence    string      // command output proving Status
}
```

Attached to `Phase` as `Increments []Increment`. A phase with no increments behaves
exactly as today, so this is additive and existing pipelines keep working.

### Pipelined verification

Today: phase 3 completes, then phase 4 starts. Proposed: as each increment reaches
`StatusComplete` it becomes eligible for verification while the implementer
continues. One writer, one reader, no write conflicts.

Needs:

1. A queue of completed increments. The TUI already streams phase output via
   `chat.StreamChunk`, so the transport pattern exists.
2. Verification results attached to an increment rather than a phase, so one
   failing increment routes back without discarding the phase's verified work.
3. A greenness rule: all increments `StatusComplete` **and** every increment
   verified. Partial greenness is explicitly not green.

### Interaction with the risk gate

The risk gate stays exactly where it is. Per-increment verification does not
replace the phase-4 gate; it front-loads defect discovery so the gate has less to
find. "QA should find nothing" remains the target — finding defects earlier makes
that more nearly true, not less.

## What this does not change

- The risk gate and its LOW/MEDIUM/HIGH/CRITICAL routing.
- `MaxRework` escalation to a human.
- Read-only enforcement for parallel specialists.
- Any agent's tool permissions.

## Open questions

1. **Granularity is agent-judged.** Should the pipeline enforce a ceiling (reject an
   increment touching more than N files) or trust judgement and let the ledger audit
   catch abuse?
2. **Failure semantics mid-stream.** If increment 3 of 7 fails while 6 is being
   built, does the implementer stop immediately, finish the current increment, or
   continue? Stopping is safest; finishing the current one wastes least.
3. **Does the TUI need an increment-level view,** or is the ledger inside phase
   output enough? The latter is cheaper and probably sufficient to start.

## Sequencing

Stage 1 (agent behaviour) is done. Stage 2 should be the `Increment` type and ledger
persistence with **no concurrency at all** — the data model first, sequential,
verifiable in isolation. Only then Stage 3, concurrent verification.

Do not build Stage 3 before Stage 2 has run in anger. Concurrency layered onto a
data model that hasn't proven itself produces bugs that are very hard to attribute.
