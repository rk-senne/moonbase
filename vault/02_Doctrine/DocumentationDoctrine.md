# Documentation Doctrine

Documentation standards for Moonbase.

---

## Core Principle

Knowledge that is not written down becomes a trial for the next operative.

Documentation is not decoration.
Documentation is not an afterthought.
Documentation is operational memory.

---

## When to Document

- New feature or behaviour.
- Significant architecture decision (ADR).
- Changed setup or configuration.
- New environment variable.
- Deployment process change.
- API change.
- Migration or upgrade.
- Decommissioned feature.
- Non-obvious code intent (inline comment).
- Edge case that is likely to confuse future operatives.

---

## Principles

### Actual Before Ideal

Document what the system IS first.

Then, if needed, add a section for what it SHOULD become.

Never blur the two. Never document planned features as current.

### Write for Future Operatives

Every document should answer a real future question.

If no one will ever ask the question this doc answers, it may be clutter.

### Useful Over Beautiful

A plain document that saves hours beats a beautiful one nobody reads.

### Small Maps Beat Giant Atlases

Prefer focused documents over walls of text.

- README for entry
- ADRs for decisions
- Guides for tasks
- Runbooks for operations
- Changelog for history
- Glossary for language

### Comments Explain Why, Not What

Inline comments should explain intent, not narrate obvious code.

Good: "Keep provider loading lazy to avoid initialising unavailable CLIs at startup."

Bad: "Loop through providers."

---

## Document Types

| Type | Purpose | Owner |
|------|---------|-------|
| README | Project entry point | Numbuh 999 |
| ADR | Architecture decisions | Numbuh 0 + Numbuh 999 |
| Changelog | What changed | Numbuh 999 |
| Runbook | Operational steps | Numbuh 362 + Numbuh 999 |
| API docs | Endpoint behaviour | Numbuh 999 |
| Onboarding | How to start | Numbuh 999 |
| Glossary | Terms and roles | Numbuh 999 |
| Inline comments | Code intent | Numbuh 3 + Numbuh 999 |

---

## Stale Documentation

Stale docs are worse than missing docs.

Missing docs say "we don't know."

Stale docs say "we know" when the truth has changed.

- Review docs when related code changes.
- Flag stale docs during review.
- Remove references to decommissioned features.
- Update setup guides when dependencies change.
- Old maps mislead brave operatives.

---

## Sensitive Information

Never document:

- Raw secrets, tokens, or credentials.
- Exploitable internal details in public docs.
- Private user data.
- Unnecessary attack vectors.

Document variable names and purpose. Never values.

---

## Diátaxis Framework (Reference)

Structure documentation by purpose:

- **Tutorials** — learning-oriented (guided first experience)
- **How-to Guides** — task-oriented (steps to achieve a goal)
- **Reference** — information-oriented (accurate, complete descriptions)
- **Explanation** — understanding-oriented (why things work this way)

Not every doc needs all four. But knowing which category a doc belongs to keeps it focused.

---

## Final Rule

Leave the next operative a map, not a mystery.

If you changed how something works, update how it is explained.

If you discovered why something exists, write it down before you forget.

A rediscovered truth not written down is already being lost.
