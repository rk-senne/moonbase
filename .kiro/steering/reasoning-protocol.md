# Reasoning Protocol

## Task Scaling

Calibrate effort to complexity:

| Complexity | Action |
|---|---|
| Trivial (typo, rename, one-liner) | Fix directly, verify builds |
| Simple (add field, new test, small refactor) | Read context → implement → test |
| Complex (new feature, architecture change) | Full protocol below |

## Full Protocol (Complex Tasks)

### 1. Retrieval First

- Read the relevant code before making claims about it.
- Check `.kiro/specs/` for existing requirements and design decisions.
- Check `.kiro/steering/` for project conventions.
- Identify the package boundaries the change touches.

### 2. Alternative Generation

- Generate 2–3 approaches for any non-trivial design decision.
- Evaluate trade-offs: complexity, testability, blast radius, reversibility.
- Select with reasoning — state why the chosen approach wins.
- If trade-offs are close, present options to the human.

### 3. Adversarial Review

Before considering work done, challenge it:

- **Hidden assumptions** — what am I taking for granted that might not hold?
- **Edge cases** — empty input, nil, max-length, concurrent access, context cancelled?
- **Scaling risks** — does this work with 1 agent? 100 agents? 1000 history entries?
- **Integration risk** — does this break existing callers or pipeline flow?
- **Security** — does this introduce injection, path traversal, or info leakage?

### 4. Self-Verification

- Audit for missing steps: did I update tests? Docs? Changelog?
- Check for unsupported claims: did I say "this is safe" without evidence?
- Verify the change compiles and tests pass.
- Confirm alignment with the original request — scope creep is a bug.

## Workspace Awareness

- Follow existing patterns in the package you're modifying.
- Don't introduce new dependencies without justification.
- Don't change architecture as a side-effect of a feature.
- If existing code contradicts steering, flag it — don't silently diverge.

## Decision Records

For significant decisions (new package, architecture change, dependency addition):
- State the decision, alternatives considered, and rationale.
- Record in the PR description or a spec file — not just in commit messages.
