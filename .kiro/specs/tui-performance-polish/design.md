# Design — TUI Performance & Polish

## AC-1 Stream chunk coalescing
**Files:** `internal/tui/update_phase_stream.go`, `views_pipeline.go`, `model_pipeline.go`.

- Keep `pollPhaseStream` reading one chunk per message (unchanged contract), but in
  `handlePhaseChunk` accumulate text into `PhaseStreamBuf` and a pending "live"
  buffer without forcing a fresh glamour render each token.
- Approach: track the streaming message as a distinct live buffer on `PipelineModel`
  (e.g. `LiveAgent string`, `LiveBuf strings.Builder`). In `renderPipeline`, render
  completed agent messages via cached glamour (as today) and render the live buffer
  as **plain wrapped text** (`lipgloss` word-wrap, no glamour) — satisfies AC-1.2.
- On phase completion (`handlePhaseResult`), flush the live buffer into a normal
  `PipelineMsg` (which then renders through cached glamour once) and clear it.
- Coalescing (AC-1.1): the 60 Hz renderer already caps repaints; the win is avoiding
  glamour on the live message. Optionally add a lightweight time-based flush (accumulate
  chunks, apply on the next `spinner.TickMsg`) if per-token `Update` cost is still high.
- Generation guard (AC-1.3) stays: drop `msg.Gen != Pipeline.Gen` before appending.

## AC-2 Goroutine-leak proof
**Files:** `internal/tui/*_test.go` (new), `go.mod` (test dep `go.uber.org/goleak`),
possibly a slow mock backend in `internal/backend`.
- Add a test that starts a phase with a slow/blocking mock stream, cancels via
  `supersedeRunningMission`/`PipelineAbortedMsg`, drains, then `goleak.VerifyNone(t)`.
- Verify `backend.AsStream`/stream adapter selects on `ctx.Done()` and closes its
  channel; fix if it can block after cancel.

## AC-3 Unicode-safe truncation
**Files:** `internal/tui/views_pipeline.go` (persona header), `views_components.go`
(`renderHeader` mission segment).
- Replace any byte-length truncation with a `lipgloss.Width`-based truncator (reuse
  the rune-trim pattern already used in `renderSidebar`/`renderFileBrowser`). Extract
  a small `truncateToWidth(s string, max int) string` helper to avoid duplication.

## AC-4 Glamour v2 (gated)
**Files:** `internal/tui/render.go`, `go.mod`.
- Swap import to `charm.land/glamour/v2`; replace `WithAutoStyle()` with an explicit
  style (`"dark"`) + `WithWordWrap(width)`. Keep the per-width renderer + memo cache.
- Gate: if v2 changes output enough to break golden tests in ways that aren't clearly
  improvements, defer (document in CHANGELOG) rather than ship a half-migration.

## AC-5 Install-security guidance
**Files:** `internal/tools/tools.go`, `internal/tui/views_tools.go`.
- For tools whose only path is a script (`Manual` contains `curl`/`bash`), reword
  guidance to the download→verify(SHA256/GPG)→run pattern and, where known, include
  the official checksum/verify reference. No auto-execution (unchanged).

## Testing strategy
- Unit tests per AC; table-driven where applicable. Reuse mock backends.
- Regenerate goldens only if AC-1/AC-4 intentionally change rendered output.
- Full gates before handoff (see requirements).

## Rollout order (low-risk first)
AC-3 → AC-2 → AC-1 → AC-5 → AC-4 (gated last).
