---
name: refactoring-safely
description: Safe refactoring practices — characterization tests first, small behavior-preserving steps, keeping the build green throughout.
---

# Refactoring Safely

## Characterization Tests First

Before changing any code, write tests that capture current behavior:

1. Identify the function/module to refactor.
2. Write tests that document what it currently does (including quirks).
3. Verify tests pass against the existing code.
4. Only then begin refactoring — tests are your safety net.

```go
// Characterization test — captures existing behavior before refactoring
func TestParseConfig_CurrentBehavior(t *testing.T) {
    // This test documents how the function handles edge cases TODAY
    got := ParseConfig("")
    // Even if this behavior is wrong, capture it first
    if got.Timeout != 30*time.Second {
        t.Errorf("default timeout = %v, want 30s", got.Timeout)
    }
}
```

## Small Behavior-Preserving Steps

Each step must:
- Change structure without changing behavior.
- Compile and pass all tests.
- Be independently revertable.

Step examples:
- Extract method (move code to a named function).
- Rename for clarity.
- Inline a trivial wrapper.
- Replace conditional with polymorphism.
- Extract interface.
- Move function to a more appropriate package.

## Keep the Build Green

- Run `go build ./...` and `go test ./...` after every step.
- Never make two structural changes in one commit.
- If a step breaks tests, revert it and try a smaller step.
- Commit after each green step — creates a safe rollback point.

## When to Refactor

Refactor BEFORE adding a feature when:
- The current structure makes the feature awkward to add.
- You need to understand the code anyway (refactoring builds understanding).
- The change is small enough to keep the PR focused.

Do NOT refactor when:
- The code works and you are not modifying it.
- The refactoring is unrelated to your current task (file an issue instead).
- You cannot write characterization tests (too risky without a safety net).

## Common Refactoring Patterns

| Smell | Refactoring |
|-------|-------------|
| Long function (>50 lines) | Extract method |
| Duplicated logic | Extract shared helper |
| Deep nesting (>3 levels) | Early returns / guard clauses |
| God struct (too many fields) | Split into focused types |
| Package dependency cycle | Extract interface at the boundary |
| Stringly-typed code | Introduce typed constants or enums |

## Verification Checklist

- [ ] Characterization tests written and passing before changes.
- [ ] Each commit compiles and passes all tests.
- [ ] No behavior changes mixed with structural changes.
- [ ] Original test suite still passes after refactoring.
- [ ] No new public API surface unless intentional.
- [ ] Performance not regressed (profile if in doubt).
