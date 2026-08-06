# Requirements — TUI Performance & Polish

Research-backed follow-ups to the shipped TUI work (global keys, mission submit,
personality feedback, mission indicator, lag fixes, OS-aware mux, tools install).
Sources: Bubble Tea cursed-renderer wiki (60 Hz frame diff, `View()` per update),
Glamour v2 notes (pure render), oneuptime goroutine-leak guide (`goleak`),
sysdig/atomicobject install-security analyses, Go string-builder benchmarks.

## AC-1 — Stream chunk coalescing (perf)
- **AC-1.1** High-frequency pipeline stream chunks MUST be coalesced so `View()`/
  glamour is not rebuilt per token. Rendered output MUST be visually equivalent.
- **AC-1.2** The live (in-flight) streaming message MUST render as plain wrapped
  text; glamour formatting applies once the phase completes.
- **AC-1.3** No streamed content may be lost or reordered; the generation guard
  (`Pipeline.Gen`) MUST still discard superseded-mission chunks.

## AC-2 — Goroutine-leak proof (correctness)
- **AC-2.1** A test MUST assert no goroutines leak when a mission is aborted and
  when a new mission supersedes a running one (using `go.uber.org/goleak`).
- **AC-2.2** The streaming backend MUST observe `ctx` cancellation and close its
  channel; a test with a slow mock backend MUST confirm no leak on cancel.

## AC-3 — Unicode-safe width truncation (correctness)
- **AC-3.1** Persona headers and the header mission indicator MUST truncate using
  visual width (`lipgloss.Width`), never byte length, so emoji/CJK never overflow
  or corrupt the layout at narrow widths.

## AC-4 — Glamour v2 migration (perf/correctness) — gated
- **AC-4.1** Markdown rendering SHOULD migrate to `charm.land/glamour/v2` (Lip
  Gloss v2), replacing removed `WithAutoStyle`. If migration risks regressions it
  MUST be deferred and documented, not partially applied.

## AC-5 — Install-security guidance (security)
- **AC-5.1** Manual/script-install tools MUST present checksum/GPG verification
  guidance (download → verify → run), not a bare `curl | bash` line.
- **AC-5.2** No behavior change to package-manager installs; still confirmation-gated.

## Non-goals
- No new TUI views or CLI commands. No dependency additions beyond `goleak` (test-only).

## Quality gates (all must pass)
`go build ./...`, `go vet ./...`, `staticcheck ./...`, `go test -race ./...`,
`moonbase lint`. Update CHANGELOG. No golden drift unless intended + regenerated.
