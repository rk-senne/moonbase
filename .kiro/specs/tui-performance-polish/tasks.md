# Tasks — TUI Performance & Polish

Ordered low-risk → high-risk. Each task ends green (build/vet/staticcheck/test-race/lint).

- [ ] **T1 (AC-3)** Add `truncateToWidth` helper; apply to persona header
  (`views_pipeline.go`) and header mission segment (`views_components.go`).
  Test: long designation + narrow width stays within bounds.

- [ ] **T2 (AC-2)** Add `go.uber.org/goleak` (test-only). Add leak tests for
  mission abort and mission supersede. Add/confirm a slow mock backend that honors
  `ctx`. Fix any post-cancel channel block in the stream adapter.

- [ ] **T3 (AC-1)** Introduce a live streaming buffer on `PipelineModel`; render it
  as plain wrapped text; flush to a glamour-rendered `PipelineMsg` on phase complete.
  Keep the generation guard. Tests: chunk accumulation, live-vs-completed rendering,
  stale-gen still dropped. Regenerate pipeline golden if output intentionally changes.

- [ ] **T4 (AC-5)** Reword script-install `Manual` guidance to download→verify→run
  with checksum/GPG references. Test: manual tools surface a verification step.

- [ ] **T5 (AC-4, gated)** Migrate `render.go` to `charm.land/glamour/v2`
  (`"dark"` style + `WithWordWrap`). If goldens/tests regress non-trivially, revert
  and document deferral in CHANGELOG. Otherwise regenerate goldens.

- [ ] **T6** Update CHANGELOG `[Unreleased]`. Run full gates. Final review.

## Definition of done
All ACs met or explicitly deferred with rationale; all quality gates green;
CHANGELOG updated; no unintended golden drift.
