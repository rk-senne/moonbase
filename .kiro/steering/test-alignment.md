# Test Alignment

## Principle

Tests are part of the spec. If a feature isn't tested, it isn't done.
Tests document behavior — a reader should understand the contract from tests alone.

## Rules

- Every exported function has at least one test.
- Table-driven tests for functions with 3+ distinct cases.
- Test files live in the same package (`*_test.go`), not a separate `_test` package.
- `go test -race ./...` must pass — no data races tolerated.
- Integration tests use build tag: `//go:build integration`

## Naming Convention

```
TestFunctionName_Scenario
TestLoadAgent_ValidFrontmatter
TestLoadAgent_MissingName
TestPipeline_RiskGateEscalation
```

## Cross-Check Matrix

| Change Type | Required Test |
|---|---|
| New exported function | Unit test with at least happy + error path |
| New CLI command | Integration test exercising full command |
| New agent frontmatter field | Parser test (valid + invalid values) |
| New pipeline phase | Mock backend test with phase transitions |
| Bug fix | Regression test reproducing the bug first |
| New config option | Test for default value + override |
| TUI model change | `Update()` test with relevant messages |

## Test Quality

- Tests assert behavior, not implementation. Don't test private internals directly.
- One logical assertion per test case (table-driven rows count as separate cases).
- No test interdependence — each test sets up its own state.
- Use `t.Helper()` in test utility functions.
- Use `t.TempDir()` for file system tests — never write to real project dirs.
- Use `testdata/` for fixture files that tests need to read.

## What NOT to Test

- Unexported functions tested only through their public callers.
- TUI rendering (visual output is fragile) — test `Update()` logic instead.
- Third-party library behavior — trust it or vendor it.

## Mock Strategy

- Mock AI backends for pipeline tests (interface: `Backend`).
- Real `.md` files from `agents/` for parser tests — not synthetic fixtures.
- `httptest.Server` for any HTTP integration tests.
- No mocking file system — use `t.TempDir()` with real files.
