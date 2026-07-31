---
name: testing-discipline
description: Effective Go testing — table-driven patterns, race detection, regression-first bug fixes, coverage strategy, and what not to test.
---

# Testing Discipline

## Table-Driven Tests

Use table-driven tests for functions with 3+ distinct cases:

```go
func TestParse_Cases(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    Result
        wantErr bool
    }{
        {name: "empty input", input: "", wantErr: true},
        {name: "valid", input: "abc", want: Result{Value: "abc"}},
        {name: "unicode", input: "日本語", want: Result{Value: "日本語"}},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := Parse(tt.input)
            if (err != nil) != tt.wantErr {
                t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
            }
            if got != tt.want {
                t.Errorf("got %v, want %v", got, tt.want)
            }
        })
    }
}
```

## Race Detection

- Always run `go test -race ./...` in CI and locally.
- Race failures are non-negotiable — fix immediately.
- Common race sources: shared map without mutex, goroutine capturing loop variable, unsynchronized struct field writes.

## Regression-First Bug Fixes

1. Write a test that reproduces the bug (must fail).
2. Fix the bug.
3. Confirm the test passes.
4. Never skip step 1 — without it, the bug can return silently.

## Coverage Strategy

- Aim for meaningful coverage, not 100% line count.
- Cover: happy path, error path, edge cases (nil, empty, max-length).
- Use `go test -coverprofile=cover.out` then `go tool cover -func=cover.out`.
- Critical paths (auth, payments, data integrity) require exhaustive coverage.

## Naming Convention

```
TestFunctionName_Scenario
TestLoadAgent_ValidFrontmatter
TestLoadAgent_MissingName
TestPipeline_RiskGateEscalation
```

## Test Isolation

- Each test sets up its own state — no shared mutable fixtures.
- Use `t.TempDir()` for filesystem tests (auto-cleaned).
- Use `t.Helper()` in test utility functions for accurate line reporting.
- Use `t.Parallel()` where safe to speed up suites.

## What Not to Test

- Unexported functions — test through their public callers.
- Visual/TUI rendering — test `Update()` logic, not pixel output.
- Third-party library internals — trust or vendor.
- Generated code — test the generator, not every output.
