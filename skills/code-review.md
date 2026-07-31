---
name: code-review
description: Code review methodology — the four lenses (contract fidelity, architecture, completeness, intent alignment) and constructive review etiquette.
---

# Code Review

## The Four Lenses

Review every change through these four perspectives:

### 1. Contract Fidelity

Does the code match the spec/requirements?

- Acceptance criteria satisfied (reference AC-IDs if available).
- Edge cases from the spec are handled.
- Error conditions produce the documented behavior.
- API contracts (request/response shapes) match the design.

### 2. Architecture Erosion

Does this respect the system's structure?

- Package boundaries honored (no circular imports, no reaching into internals).
- Existing patterns followed (if the codebase uses X, new code should too).
- Dependency direction correct (domain does not import infrastructure).
- No new dependencies without justification.
- No hidden coupling (global state, implicit ordering requirements).

### 3. Completeness

Is anything missing?

- Tests: happy path, error path, edge cases.
- Error handling: no swallowed errors, context in wraps.
- Resource cleanup: defers for Close, cancel contexts.
- Documentation: exported functions have doc comments.
- Observability: errors logged at boundaries, not buried.

### 4. Intention Alignment

Does this solve what was actually asked for?

- Scope: no feature creep (extra features not in the spec).
- Simplicity: is this the simplest solution that works?
- No premature abstraction (YAGNI — build what you need now).
- User experience: does the end result match what was requested?

## Constructive Etiquette

### Giving Reviews

- Critique the code, not the author.
- Ask questions instead of making demands: "What happens if X is nil?" not "You forgot nil check."
- Prefix optional suggestions: "Nit:" or "Optional:" for non-blocking feedback.
- Explain WHY, not just WHAT: "This could race because..." not "Add a mutex."
- Acknowledge good work — reviewing is not just finding faults.
- Approve when concerns are minor — do not block on style preferences.

### Receiving Reviews

- Assume good intent — reviewers want the code to be better.
- Address every comment (fix it or explain why not).
- Do not take feedback personally — it is about the code.
- Ask for clarification when a comment is unclear.
- Thank reviewers for catching issues.

## Review Workflow

1. Self-review first: re-read your own diff before requesting review.
2. Small PRs: easier to review thoroughly (<400 lines ideal).
3. Context in description: what changed, why, what was tested.
4. Respond promptly: do not let review comments go stale.
5. Resolve threads: mark addressed items as resolved.

## Red Flags to Always Call Out

- Swallowed errors (`_ = fn()` in production code).
- Hardcoded secrets or credentials.
- Missing input validation on public boundaries.
- Unbounded allocations (slices growing without limit).
- Race conditions (shared state without synchronization).
- Breaking changes without version bump or migration path.
