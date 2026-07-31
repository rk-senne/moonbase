# Contributing

Moonbase follows its own steering rules (`.kiro/steering/`) — production standards, test alignment, reasoning protocol, and quality gates.

## Quality gates (must pass before merge)

```bash
go build ./...                          # compiles cleanly
go vet ./...                            # no suspicious constructs
staticcheck ./...                       # 0 findings (CI gate)
govulncheck ./...                       # 0 affecting vulnerabilities
CI=true go test -race ./... -count=1    # all packages, no data races
go run ./cmd/moonbase lint              # all 14 agents valid
```

CI (GitHub Actions) runs vet + build + govulncheck + `-race` tests on ubuntu **and** macos, plus a lint job (agent lint + staticcheck) on every push and PR.

## Conventions

- **Commits**: conventional format — `type(scope): description` (`feat`, `fix`, `refactor`, `test`, `docs`, `chore`, `perf`). One logical change per commit.
- **Tests**: every exported function has a test; table-driven where there are 3+ cases; `t.TempDir()` for filesystem tests; mock backends for pipeline tests. Integration tests use `//go:build integration`.
- **Spec-driven**: non-trivial features get a spec in `.kiro/specs/<feature>/` (requirements → design → tasks) before implementation.
- **Errors**: wrap with context (`fmt.Errorf("what failed: %w", err)`); never swallow errors.
- **Agents**: never modify agent content from Go code — the `agents/` `.md` files are the source of truth; pass the body through as-is.

## Branch & release flow

- Large/breaking changes go on a `feat/*` branch and merge via PR after CI is green (small fixes may go direct to `main`).
- Releases: move `[Unreleased]` CHANGELOG entries under a new version, tag `vX.Y.Z`, push the tag — goreleaser builds the cross-platform binaries and checksums.
- Semver: MAJOR = breaking CLI/agent/config change · MINOR = new feature · PATCH = fix/perf/docs.

## Dogfooding

Moonbase can improve itself: run `moonbase mission "<task>"` on the moonbase repo, or deploy a specialist directly (`moonbase deploy 4 "audit the risk gate"`). Its operatives can scan their own codebase — several of this project's features were built by its own council.
