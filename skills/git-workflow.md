---
name: git-workflow
description: Git workflow discipline — conventional commits, atomic changes, branch strategy, PR hygiene, and safe history management.
---

# Git Workflow

## Conventional Commits

Format: `type(scope): description`

Types:
- `feat` — new user-facing feature
- `fix` — bug fix
- `refactor` — code restructuring (no behavior change)
- `test` — adding or updating tests
- `docs` — documentation only
- `chore` — tooling, CI, dependencies
- `perf` — performance improvement

Rules:
- Subject line ≤72 characters, imperative mood ("add", not "added").
- Body explains WHY, not WHAT (the diff shows what).
- Breaking changes: add `!` after type or `BREAKING CHANGE:` in footer.

```
feat(pipeline): add retry logic for transient backend failures

Backends occasionally return 503 during high load. Retry up to 3 times
with exponential backoff (1s, 2s, 4s) before surfacing the error.

Closes #142
```

## Atomic Commits

- One logical change per commit — reviewable in isolation.
- If you must explain two unrelated things, it's two commits.
- Stage specific files (`git add path/to/file`) over `git add .`.
- Separate refactoring commits from feature commits.

## Branch Strategy

- `main` — always deployable, protected.
- `feat/<name>` — feature branches, short-lived (days, not weeks).
- `fix/<name>` — bug fix branches.
- Delete branches after merge — no stale branches.

## Pull Request Hygiene

- Title: concise (<70 chars), describes the outcome.
- Description: summary of changes, what was tested, blocked items.
- Small PRs merge faster — split large features into incremental PRs.
- Self-review before requesting others: re-read the diff as a reviewer would.
- Address all review comments or explicitly explain disagreement.

## Safe History Management

- Never force-push to shared branches (main, develop).
- Prefer new commits over `--amend` on pushed commits.
- Use `git rebase -i` only on local, unpushed branches.
- `git reset --hard` and `git clean -f` require explicit intent — they destroy work.

## Conflict Resolution

- Rebase feature branches onto main before merging (linear history).
- Resolve conflicts in the feature branch, not in main.
- After resolving, re-run tests to catch semantic conflicts the merge missed.

## Hooks and CI

- Pre-commit: lint, format, type-check (fast, local feedback).
- CI: full build + test + vet + vulnerability scan on every push.
- Never skip hooks (`--no-verify`) without documenting why.
