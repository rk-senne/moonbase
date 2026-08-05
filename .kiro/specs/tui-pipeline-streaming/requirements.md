# Requirements: TUI Pipeline Streaming

## Overview

When a mission runs, each pipeline phase blocks on
`be.Deploy → exec …CombinedOutput()` and only surfaces its output **after the
entire (multi-minute) kiro-cli call finishes**. The TUI shows a static
"Phase N starting…" + spinner the whole time, so it looks frozen and feels laggy
— the AI backend's output is not passed to the TUI incrementally.

This spec routes pipeline phases through **incremental streaming** so agent
output appears live, while preserving the existing completion contract
(`PhaseResultMsg`) so the risk-gate, advance, and fan-out logic are unchanged.

**Grounded in current code (confirmed):**
- `internal/backend/backend.go:13` — `Backend` interface exposes only
  `Deploy(agent, ctx, task) (string, error)` (synchronous).
- `internal/backend/backends.go:24/52` — `Kiro.Deploy` runs
  `cmd.CombinedOutput()` (blocking, whole output at end).
- `internal/tui/pipeline_exec.go:84/88` — `executePhase` calls `be.Deploy`
  inside a goroutine, returns one `PhaseResultMsg` at the end.
- **COMMS already streams** (the pattern to reuse): `chat.Stream(conv) <-chan
  chat.StreamChunk` → `pollStream(ch)` emits `streamChunkMsg{text,done,err}`
  (`helpers_comms.go:60`) → `handleStreamChunk` re-polls (`update_comms.go:123`).
  `StreamChunk` carries `Text`, `Done`, `Err`.

> **AC-ID convention:** stable IDs `AC-{n}.{i}`.

---

## User Stories
- **US-1:** As an operator, I want to see each agent's output appear *as it is
  produced* so the pipeline never looks frozen.
- **US-2:** As an operator, I want phase completion, risk gates, and advancing to
  work exactly as before — streaming is a display improvement, not a behaviour
  change.
- **US-3:** As a maintainer, I want backends without streaming to still work
  (graceful fallback), with no regression to existing tests.

---

## Acceptance Criteria

### AC-1 — Streaming backend capability (opt-in, with fallback)
- **AC-1.1** — A streaming capability SHALL be available for backends that can
  produce incremental output, surfaced as a channel of `chat.StreamChunk`
  (reuse the existing type: `Text`, `Done`, `Err`).
- **AC-1.2** — The **Kiro** backend SHALL implement streaming by reading the
  `kiro-cli chat` process **stdout incrementally** (`StdoutPipe` + scanner under
  an `exec.CommandContext`), emitting a chunk per line/segment, then a terminal
  `Done` chunk (carrying any non-zero-exit error).
- **AC-1.3** — Backends that do NOT implement streaming SHALL be adapted to a
  **one-shot stream**: a single chunk with the full `Deploy` output, then `Done`
  — so the pipeline code path is uniform and non-streaming backends still work.
- **AC-1.4** — The streaming command SHALL respect the per-phase timeout and
  context cancellation (kills the process, closes the channel, no goroutine leak).

### AC-2 — Pipeline consumes the stream (live output)
- **AC-2.1** — `executePhase` SHALL start the phase stream and poll it, emitting
  a **pipeline-scoped chunk message** for each non-terminal chunk (distinct from
  the COMMS `streamChunkMsg`).
- **AC-2.2** — Each chunk SHALL append its text to the current phase's live
  output in the pipeline view (the chat/output buffer), so the operator sees the
  agent "typing".
- **AC-2.3** — The full phase output SHALL be accumulated across chunks so the
  final result equals what the blocking path produced.

### AC-3 — Completion parity (unchanged downstream)
- **AC-3.1** — On the terminal `Done` chunk, a `PhaseResultMsg{Phase, Output:
  <accumulated>, Elapsed}` SHALL be dispatched so the existing
  `handlePhaseResult` (risk gate at phase 4, `Advance`, fan-out, MISSION COMPLETE)
  runs **unchanged**.
- **AC-3.2** — On error or timeout, a `PhaseResultMsg{Err}` SHALL be dispatched
  (same failure UX as today: ❌ + retry/skip hint).
- **AC-3.3** — The QA risk-gate parsing, parallel-specialist fan-out, and
  conditional-phase skipping SHALL NOT be modified by this change.

### AC-4 — COMMS isolation
- **AC-4.1** — The pipeline chunk message SHALL be a **separate type** from the
  COMMS `streamChunkMsg`; `handleStreamChunk` and COMMS streaming behaviour SHALL
  be untouched.

### AC-5 — Abort / cancel / timeout
- **AC-5.1** — `PipelineAbortedMsg`/`Cancel()` SHALL stop an in-flight phase
  stream promptly (process killed, channel drained/closed).
- **AC-5.2** — Per-phase timeout SHALL still fire and produce a timeout
  `PhaseResultMsg` (no change to `PhaseTimeout` semantics).

### AC-6 — Backward compatibility & no regression
- **AC-6.1** — Simulated mode (no/clipboard backend) SHALL behave exactly as
  today.
- **AC-6.2** — All existing pipeline/exec tests SHALL pass unchanged
  (`PhaseResultMsg` shape and handlers preserved).

### AC-7 — Cross-cutting
- **AC-7.1** — No new runtime dependency (uses `os/exec`, `bufio`, existing
  `chat.StreamChunk`).
- **AC-7.2** — `go build/vet/test -race ./...` green; `moonbase lint` green.
- **AC-7.3** — New behaviour has tests: Kiro stream chunking, one-shot fallback,
  timeout/cancel closes channel, chunk→PhaseResultMsg completion parity, COMMS
  untouched.

---

## Scope
**In:** streaming capability + Kiro stdout streaming + one-shot fallback;
`executePhase` streaming consume; a pipeline chunk message + Update routing +
live view append; completion still via `PhaseResultMsg`; timeout/abort; tests.

**Out:** COMMS changes; changing the risk-gate/fan-out logic; token/cost metering
changes; multiplexing multiple phases' streams (phases run one at a time except
fan-out, which stays batch); backends other than Kiro gaining native streaming
(they use the one-shot fallback).

---

## Dependencies
| Dependency | Status | Impact |
|-----------|--------|--------|
| `chat.StreamChunk` type | exists | reused as the chunk unit (AC-1.1) |
| `pollStream` pattern | exists (COMMS) | pattern to mirror for the pipeline |
| `Kiro.Deploy`/`DeployRaw` | exists (CombinedOutput) | add a streaming sibling |
| `handlePhaseResult` | exists | completion contract preserved (AC-3) |

## Risks
| Risk | Mitigation |
|------|-----------|
| Goroutine/process leak on abort | AC-1.4/5.1 CommandContext kill + channel close + tests |
| Output differs from blocking path | AC-2.3 accumulate; test parity vs Deploy |
| COMMS regression | AC-4.1 separate message type; COMMS tests untouched |
| Non-streaming backend breaks | AC-1.3 one-shot fallback; AC-6 tests |
| Chunk flooding the update loop | coalesce by line; poll one chunk per cmd (same as COMMS) |
