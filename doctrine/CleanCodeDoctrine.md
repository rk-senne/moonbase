# Clean Code Doctrine

Principles from Robert C. Martin's Clean Code, adapted for AI agent development.
These apply to both the code agents write AND the moonbase codebase itself.

---

## Functions

- Functions should do one thing, do it well, and do it only.
- A function that does more than one thing should be split.
- The ideal function is 5-15 lines. Anything over 30 needs justification.
- Function names should describe what the function does, not how.
- Arguments: fewer is better. Zero is ideal, three is a warning sign.

## Naming

- Names should reveal intent. If a name requires a comment, the name is wrong.
- Use pronounceable, searchable names.
- Class/struct names are nouns. Function names are verbs.
- Do not encode type information in names (no Hungarian notation).
- Naming consistency across the codebase is more important than individual cleverness.

## Comments

- Comments explain WHY, not WHAT. Code should explain what.
- If you need a comment to explain what code does, the code should be rewritten.
- Good comments: legal headers, intent explanation, warning of consequences, TODO with ticket.
- Bad comments: redundant (repeating the code), misleading, journal comments, noise.

## Error Handling

- Error handling is one thing. A function that handles errors should do little else.
- Wrap errors with context: what operation failed, with what input.
- Never return nil/null when you could return an error.
- Never ignore errors silently. Handle, wrap, or propagate -- but never swallow.

## DRY (Do Not Repeat Yourself)

- Duplication is the root of all software evil.
- Every piece of knowledge should have a single, authoritative representation.
- If you change something in one place and must change it in another, you have duplication.
- Extract shared logic into a single source of truth.

## The Boy Scout Rule

Always leave the code cleaner than you found it.

## Tests

- Tests are first-class citizens. They deserve the same care as production code.
- One assert per test (conceptually). Test one behaviour at a time.
- Tests should be fast, independent, repeatable, self-validating, and timely (FIRST).
- Test names should describe the scenario: TestFunction_Scenario_ExpectedResult.

## Boundaries

- Keep third-party code at arm's length. Wrap it behind interfaces you control.
- Do not let framework details leak into business logic.
- Dependencies point inward: outer layers depend on inner layers, never the reverse.

---

## Applied to Moonbase Agents

When agents write code, they follow these principles. When agents review code, they check for violations of these principles. The specific application:

| Agent | Clean Code Responsibility |
|-------|---------------------------|
| Numbuh 3 (Implementer) | Write clean code. Small functions, clear names, wrapped errors. |
| Numbuh 4 (QA) | Verify clean code. Flag long functions, vague names, bare errors. |
| Numbuh 5 (Reviewer) | Enforce clean code. Reject PRs with Boy Scout Rule violations. |
| Numbuh 86 (Tech Debt) | Find unclean code. Identify duplication, dead code, complexity. |
