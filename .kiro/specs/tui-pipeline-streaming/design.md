# Design: TUI Pipeline Streaming

## ADR-summary
**Decision:** Give the pipeline the same incremental-streaming treatment COMMS
already has: a `StreamingBackend` capability (Kiro streams `kiro-cli` stdout;
other backends fall back to a one-shot stream wrapping `Deploy`), consumed by
`executePhase` via a poll loop that emits pipeline chunk messages, and finalised
with the **existing** `PhaseResultMsg` so nothing downstream changes.
**Why:** the blocking `CombinedOutput()` is the sole reason the pipeline looks
frozen; reusing the proven `pollStream` pattern is low-risk and keeps the
risk-gate/fan-out contract intact. **Reversibility:** additive interface + a
streaming sibling + a rewritten `executePhase` body (same signature/return
message); revert by pointing `executePhase` back at `be.Deploy`.

## Files Affected
| File | Change | Purpose |
|------|--------|---------|
| `internal/backend/backend.go` | modify | add optional `StreamingBackend` interface + `AsStream(be)` adapter (fallback) |
| `internal/backend/backends.go` | modify | `Kiro.DeployStream(ctx, …) (<-chan chat.StreamChunk, error)` via StdoutPipe+scanner |
| `internal/backend/stream_adapter.go` | new | one-shot fallback: wrap `Deploy` as a single-chunk stream |
| `internal/tui/helpers_messages.go` | modify | add `PhaseChunkMsg{Phase int; Text string; Done bool; Err error}` |
| `internal/tui/pipeline_exec.go` | modify | `executePhase` streams: poll → `PhaseChunkMsg`; on Done → `PhaseResultMsg` |
| `internal/tui/update_pipeline.go` / `app.go` | modify | route `PhaseChunkMsg` → append live text + re-poll |
| `internal/tui/views_pipeline.go` | modify (light) | render the live-growing phase output |
| `*_test.go` | new/modify | streaming, fallback, timeout/cancel, completion parity, COMMS untouched |

## Components

### 1. Streaming capability (backend)
```go
// backend.go
type StreamingBackend interface {
    Backend
    DeployStream(ctx context.Context, agent agents.Agent,
        pc *discovery.ProjectContext, task string) (<-chan chat.StreamChunk, error)
}

// AsStream returns a chunk channel for ANY backend: native stream if the
// backend implements StreamingBackend, else a one-shot stream wrapping Deploy.
func AsStream(ctx context.Context, be Backend, agent agents.Agent,
    pc *discovery.ProjectContext, task string) (<-chan chat.StreamChunk, error) {
    if sb, ok := be.(StreamingBackend); ok {
        return sb.DeployStream(ctx, agent, pc, task)
    }
    return oneShotStream(ctx, be, agent, pc, task), nil // stream_adapter.go
}
```

### 2. Kiro streaming (backends.go)
```go
func (k *Kiro) DeployStream(ctx context.Context, agent agents.Agent,
    pc *discovery.ProjectContext, task string) (<-chan chat.StreamChunk, error) {
    composed := task // caller already composed (parity with Deploy path)
    args := []string{"chat", "--trust-all-tools", "--no-interactive", "--", composed}
    cmd := exec.CommandContext(ctx, "kiro-cli", args...)
    cmd.Env = SafeEnv()
    stdout, err := cmd.StdoutPipe()
    if err != nil { return nil, err }
    cmd.Stderr = cmd.Stdout // fold stderr in (CombinedOutput parity)  [or capture separately]
    if err := cmd.Start(); err != nil { return nil, err }

    ch := make(chan chat.StreamChunk)
    go func() {
        defer close(ch)
        sc := bufio.NewScanner(stdout)
        sc.Buffer(make([]byte, 64*1024), 1024*1024)
        for sc.Scan() {
            select {
            case ch <- chat.StreamChunk{Text: sc.Text() + "\n"}:
            case <-ctx.Done():
                _ = cmd.Process.Kill(); return
            }
        }
        werr := cmd.Wait()
        ch <- chat.StreamChunk{Done: true, Err: werr} // Err carries non-zero exit
    }()
    return ch, nil
}
```
- Timeout/cancel: `exec.CommandContext` + the `ctx.Done()` select kill the
  process and close the channel (AC-1.4/5.1). No leak.
- `SafeEnv()` reused (same as `DeployRaw`). Prompt composed by the caller (the
  phase), identical to today.

### 3. One-shot fallback (stream_adapter.go)
```go
func oneShotStream(ctx context.Context, be Backend, agent agents.Agent,
    pc *discovery.ProjectContext, task string) <-chan chat.StreamChunk {
    ch := make(chan chat.StreamChunk, 1)
    go func() {
        defer close(ch)
        out, err := be.Deploy(agent, pc, task)
        if err == nil && out != "" { ch <- chat.StreamChunk{Text: out} }
        ch <- chat.StreamChunk{Done: true, Err: err}
    }()
    return ch
}
```
Non-streaming backends behave exactly like today (one blob, then done) — no
regression (AC-1.3/6).

### 4. `executePhase` streaming rewrite (pipeline_exec.go)
Keep the signature and the retry/timeout semantics; swap the body to stream:
```go
func executePhase(ctx, phase, reg, be, projectCtx, pctx, phaseTimeout) tea.Cmd {
    return func() tea.Msg {
        agent := reg.GetByName(phase.AgentName) // (unchanged guards)
        composed := discovery.ComposePrompt(agent.Prompt, projectCtx, pctx.ForPhase(phase.Number))
        tctx, cancel := context.WithTimeout(ctx, phaseTimeout)   // cancel stored/deferred appropriately
        ch, err := backend.AsStream(tctx, be, *agent, projectCtx, composed)
        if err != nil { cancel(); return PhaseResultMsg{Phase, Err: err, …} }
        // stash ch + start time + a *bytes.Buffer accumulator on pipeline state,
        // then hand control to the poll loop:
        return pollPhaseStream(phase.Number, ch, start, cancel /*, buf*/)
    }
}

func pollPhaseStream(phaseNum int, ch <-chan chat.StreamChunk, start time.Time, cancel context.CancelFunc) tea.Cmd {
    return func() tea.Msg {
        chunk, ok := <-ch
        if !ok || chunk.Done {
            cancel()
            return PhaseResultMsg{Phase: phaseNum, Output: /*accumulated*/, Err: chunk.Err, Elapsed: time.Since(start)}
        }
        return PhaseChunkMsg{Phase: phaseNum, Text: chunk.Text}
    }
}
```
- **Accumulation:** store the channel + a growing buffer on
  `a.views.Pipeline` (e.g. `StreamCh`, `StreamBuf`, `StreamStart`, `StreamCancel`,
  `StreamPhase`). The Update loop appends each chunk to `StreamBuf` and the live
  view, then re-polls; on Done it builds the `PhaseResultMsg` from `StreamBuf`.
  (Mirrors COMMS `pollStream`/`handleStreamChunk` exactly, but pipeline-scoped.)

### 5. Message + routing (helpers_messages.go, app.go/update_pipeline.go)
```go
type PhaseChunkMsg struct { Phase int; Text string }
```
`app.go` Update: `case PhaseChunkMsg: return a.handlePhaseChunk(msg)`.
```go
func (a App) handlePhaseChunk(msg PhaseChunkMsg) (tea.Model, tea.Cmd) {
    a.views.Pipeline.StreamBuf.WriteString(msg.Text)
    a.appendLivePhaseOutput(msg.Text)         // AC-2.2 live view
    return a, pollPhaseStream(msg.Phase, a.views.Pipeline.StreamCh, a.views.Pipeline.StreamStart, a.views.Pipeline.StreamCancel)
}
```
On Done, `pollPhaseStream` returns `PhaseResultMsg` (Output = `StreamBuf.String()`)
→ existing `handlePhaseResultUpdate`/`handlePhaseResult` runs unchanged (AC-3).

### 6. View (views_pipeline.go)
Render the in-progress phase's `StreamBuf` (tail) beneath the "Phase N …"
header while `Running`; the spinner badge stays for the active phase. On
completion the existing summary line is appended as today.

## Data Flow
```
executePhase → AsStream(ctx,be,…) ─┬─ Kiro: kiro-cli stdout → chunks
                                   └─ others: Deploy() → one chunk
   ↓ pollPhaseStream (one chunk/cmd)
PhaseChunkMsg → append StreamBuf + live view → re-poll
   … (repeat) …
Done → PhaseResultMsg{Output:StreamBuf} → handlePhaseResult (risk gate/advance)  [UNCHANGED]
```

## Edge Cases
| Case | Handling |
|------|----------|
| Backend not streaming | one-shot fallback (AC-1.3) |
| Timeout mid-stream | CommandContext fires → Wait returns err → Done{Err:timeout} → PhaseResultMsg{Err} |
| Abort (PipelineAborted) | `StreamCancel()` + `Cancel()` kill process, channel closes, poll returns Done |
| Non-zero kiro-cli exit (e.g. arg error) | `cmd.Wait` err → Done{Err} → failure UX |
| Huge line | scanner buffer raised to 1 MB |
| Empty output | Done only → PhaseResultMsg{Output:""} (same as today) |

## Testing (AC-7.3)
| Test | Scenario |
|------|----------|
| `TestKiroDeployStream_ChunksThenDone` | fake stdout → ordered chunks + terminal Done (use a stub command / `everything`-style) |
| `TestOneShotStream_WrapsDeploy` | non-streaming backend → single chunk + Done, text == Deploy output |
| `TestAsStream_SelectsNativeVsFallback` | StreamingBackend vs plain Backend |
| `TestPollPhaseStream_DoneBuildsPhaseResult` | accumulated buffer → PhaseResultMsg.Output parity |
| `TestExecutePhase_Timeout_ClosesChannel` | ctx timeout → Done{Err}, no leak |
| `TestHandlePhaseChunk_AppendsAndRepolls` | chunk appends to buffer + returns re-poll cmd |
| `TestComms_StreamUnchanged` | COMMS streamChunkMsg/handleStreamChunk path intact |

`go test -race ./...` + `moonbase lint` green. No golden-pixel tests.

## Reversibility / Migration
Additive: one interface, one Kiro method, one adapter file, one message type,
one Update case, a rewritten `executePhase` body (same return contract), a few
`Pipeline` state fields. Revert `executePhase` to call `be.Deploy` to roll back.
No data/format migration.
