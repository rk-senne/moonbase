# Testing Doctrine

Testing standards for Moonbase.

---

## Core Principle

Tests are not paperwork. They are the system's immune response.

A test that exists but is never run is a lie.
A test that passes but tests nothing is theatre.
A test that was deleted without understanding is a wound.

---

## When to Test

- New behaviour: write tests.
- Bug fix: write a regression test that fails without the fix.
- Refactor: existing tests must still pass. Add tests if coverage is weak.
- Migration: verify parity between old and new paths.
- Edge cases: if Numbuh 13 finds it, pin it with a test.

---

## What to Test

- Public API behaviour (inputs → outputs).
- Error cases and failure modes.
- Boundary conditions.
- State transitions.
- Integration points (if feasible).
- User-facing behaviour (not internal implementation details).

---

## What Not to Test

- Private implementation details that may change.
- Trivial getters/setters with no logic.
- Third-party library internals.
- Things already covered by the type system.

---

## Test Quality

- Tests must be deterministic. No flaky tests allowed without documented reason.
- Tests must be independent. No order dependency between tests.
- Tests must be fast. Slow tests go in a separate suite if needed.
- Tests must be readable. A test is documentation of expected behaviour.
- Test names must describe the scenario: `TestLoadConfig_MissingFile_ReturnsError`.

---

## Test Structure

Use Arrange-Act-Assert (AAA) or Given-When-Then:

```
// Arrange: set up inputs and dependencies
// Act: call the function under test
// Assert: verify the result
```

For table-driven tests (Go):

```go
tests := []struct {
    name     string
    input    string
    expected string
    wantErr  bool
}{
    {"empty input", "", "", true},
    {"valid input", "hello", "HELLO", false},
}
```

---

## Coverage

- Coverage is a signal, not a target.
- 100% coverage does not mean 100% quality.
- Low coverage in changed areas is a warning sign.
- Critical paths (auth, data persistence, command execution) must have strong coverage.

---

## Running Tests

- Tests must pass before handoff to QA.
- Tests must pass before PR approval.
- Tests must pass in CI.
- If tests cannot be run (missing dependency, environment issue), state it explicitly.

---

## Deleting Tests

- Do not delete tests without understanding what they protect.
- If behaviour is removed, the test can be removed.
- If behaviour changed, the test must be updated, not deleted.
- If a test is flaky, fix it or document the flakiness. Do not silently skip.

---

## Test Environments

- Tests should not depend on production state.
- Tests should not require network access unless explicitly integration tests.
- Tests should not modify shared state.
- Tests should clean up after themselves.

---

## Evidence Requirement

When reporting test results:

- State which tests were run.
- State pass/fail counts.
- State what was NOT tested and why.
- Never claim "all tests pass" without running them.

---

## Final Rule

A system without tests is a system running on faith.

Faith does not scale.

Write the test. Run the test. Trust the test. Maintain the test.
