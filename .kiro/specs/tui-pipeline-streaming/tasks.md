# Tasks: TUI Pipeline Streaming

Phased. Each task: implement → `go build ./... && go vet ./... && go test -race
./...` green → `moonbase lint` green → one conventional commit. Refs
`requirements.md` (AC-n.i) + `design.md` (§). *S*<½d, *M*~1d.

## Phase S1 — Backend streaming capability
- [ ] **TS1.1** Add `StreamingBackend` interface + `AsStream(ctx,be,…)` adapter in
  `internal/backend/backend.go`. — AC-1.1 — *S*
- [ ] **TS1.2** `internal/backend/stream_adapter.go`: `oneShotStream` wrapping
  `Deploy` (single chunk + Done); used for non-streaming backends. — AC-1.3 — *S*
- [ ] **TS1.3** `Kiro.DeployStream` in `backends.go` via `exec.CommandContext` +
  `StdoutPipe` + scanner (1 MB buf), chunk-per-line, terminal Done carrying
  `cmd.Wait` err; `SafeEnv()`; ctx-kill on cancel. — AC-1.2/1.4 — *M*
- [ ] **TS1.4** Tests: `TestKiroDeployStream_ChunksThenDone`,
  `TestOneShotStream_WrapsDeploy`, `TestAsStream_SelectsNativeVsFallback`,
  `TestExecutePhase_Timeout_ClosesChannel` (channel closes, no leak). — AC-7.3
- [ ] **S1 gate:** build/vet/test-race green; lint green; no new deps.

## Phase S2 — Pipeline consumes the stream (live output)
- [ ] **TS2.1** Add `PhaseChunkMsg{Phase int; Text string}` in
  `helpers_messages.go`; add pipeline stream state
  (`StreamCh`, `StreamBuf`, `StreamStart`, `StreamCancel`, `StreamPhase`). — AC-2.1 — *S*
- [ ] **TS2.2** Rewrite `executePhase` body to `backend.AsStream` + return
  `pollPhaseStream`; keep signature + timeout; on Done build
  `PhaseResultMsg{Output: StreamBuf}`. — AC-2.3/3.1 — *M*
- [ ] **TS2.3** `app.go` Update: `case PhaseChunkMsg → handlePhaseChunk` (append
  to `StreamBuf` + live view, re-poll). — AC-2.2 — *S*
- [ ] **TS2.4** `views_pipeline.go`: render the in-progress phase's `StreamBuf`
  tail under the phase header while running. — AC-2.2 — *S*
- [ ] **TS2.5** Tests: `TestPollPhaseStream_DoneBuildsPhaseResult` (Output
  parity), `TestHandlePhaseChunk_AppendsAndRepolls`. — AC-7.3
- [ ] **S2 gate:** build/vet/test-race green; lint green; manual: a mission shows
  live agent output.

## Phase S3 — Completion parity, abort, no-regression
- [ ] **TS3.1** Verify `handlePhaseResult` (risk gate @4, `Advance`, fan-out,
  MISSION COMPLETE) is unchanged and driven by the streamed `PhaseResultMsg`;
  add a regression test asserting the risk-gate still fires on phase 4. — AC-3.1/3.3 — *S*
- [ ] **TS3.2** Wire abort/cancel: `PipelineAborted`/`Cancel()` also calls
  `StreamCancel()`; confirm process killed + channel closed. — AC-5.1 — *S*
- [ ] **TS3.3** COMMS isolation test: `streamChunkMsg`/`handleStreamChunk`
  untouched (`TestComms_StreamUnchanged`); simulated-mode unchanged. — AC-4.1/6.1 — *S*
- [ ] **TS3.4** Full suite + lint; rebuild + reinstall the binary so the running
  TUI gets streaming. — AC-7.2
- [ ] **S3 gate:** `go test -race ./...` + `moonbase lint` green; existing
  pipeline/exec tests pass unchanged; a mission streams and completes with the
  same risk-gate/advance behaviour as before.

## Definition of Done
1. Pipeline phases stream incrementally; the TUI never looks frozen (US-1).
2. `PhaseResultMsg` completion contract preserved → risk gate/fan-out/advance
   unchanged (AC-3), existing tests green (AC-6.2).
3. Kiro streams via stdout; other backends use the one-shot fallback (AC-1).
4. Timeout/abort kill the process and close the channel, no leak (AC-1.4/5).
5. COMMS untouched (AC-4); no new deps (AC-7.1).
6. `go build/vet/test -race ./...` + `moonbase lint` green; binary rebuilt/installed.

## Traceability
| Task | AC | § |
|------|----|---|
| TS1.* | 1.1/1.2/1.3/1.4/7.3 | §1/§2/§3 |
| TS2.* | 2.*/3.1 | §4/§5/§6 |
| TS3.* | 3.*/4.1/5.*/6.* | §4/edge-cases |

## Non-Goals
COMMS changes; risk-gate/fan-out logic changes; cost/token metering; multiplexed
concurrent phase streams; native streaming for non-Kiro backends.
