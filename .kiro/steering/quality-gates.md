# Quality Gates

## Must Pass Before Done

Every change must clear these gates before being considered complete:

### Build & Test

```bash
go build ./...           # compiles cleanly
go vet ./...             # no suspicious constructs
go test -race ./...      # all tests pass, no data races
moonbase lint            # all 14 agent files valid
```

### Code Quality

- [ ] No new TODOs introduced (track in issues instead)
- [ ] No `_ = err` in production code (test helpers are fine)
- [ ] All new exported functions have doc comments
- [ ] Error messages include context (`"what failed: %w"`)
- [ ] Resources are cleaned up (defer Close, cancel contexts)

### Documentation

- [ ] CHANGELOG.md updated for user-visible changes
- [ ] README updated if CLI interface changed
- [ ] Agent `.md` files updated if agent behavior changed

### Git Discipline

- [ ] Commit messages follow conventional format: `type(scope): description`
- [ ] Types: `feat`, `fix`, `refactor`, `test`, `docs`, `chore`, `perf`
- [ ] One logical change per commit
- [ ] No secrets, no generated files, no binary artifacts

## Four-Lens Review

For non-trivial changes, verify through four lenses:

1. **Contract Fidelity** — does the code match the spec/requirements?
2. **Architecture Erosion** — does this respect package boundaries and patterns?
3. **Completeness** — are edge cases handled? Tests written? Docs updated?
4. **Intention Alignment** — does this solve what was actually asked for?

## Risk Classification

| Risk | Response |
|---|---|
| LOW (cosmetic, docs, test-only) | Self-merge after gates pass |
| MEDIUM (new feature, refactor) | Peer review + gates |
| HIGH (security, pipeline logic, agent format) | Design review + gates + manual test |
| CRITICAL (breaking change, data loss risk) | Human approval required before merge |

## Failure Response

- Gate failure = stop. Fix before proceeding.
- Two consecutive failures on the same gate = step back, diagnose root cause.
- Never skip a gate — if a gate is wrong, fix the gate first.
