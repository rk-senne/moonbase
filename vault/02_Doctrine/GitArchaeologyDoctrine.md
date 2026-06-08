# Git Archaeology Doctrine

Legacy investigation standards for Moonbase.

---

## Core Principle

Check git history before judgment.

Respect the ghosts. Do not worship the ruins.

History is evidence, not law.

---

## Universal Rules

- Do not assume old code is useless because it is old.
- Do not assume old code is sacred because it survived.
- Always check git history BEFORE making judgments.
- If you can find the original commit message, that is gold.
- Produce actionable verdicts, not ghost stories.
- Cryptic opening is allowed. Vague verdict is not.

---

## Archaeology Tools

Primary investigation commands:

- `git log` — commit history, messages, timestamps
- `git blame` — who last touched each line and when
- `git show` — specific commit content
- `git diff` — compare versions
- `git shortlog` — author activity summary
- `git rev-list` — commit listing and counting
- `git grep` — search across history
- `git tag` — version markers
- `git branch -a` — all branches including old/abandoned

---

## Investigation Areas

1. Git History — commits, authors, timestamps, messages, reverts
2. Blame — who last touched and why
3. Previous Versions — what it looked like before
4. Deleted Code — was related code removed?
5. Dependency History — when added, upgraded, abandoned
6. Test History — added, removed, skipped, changed
7. Documentation History — did old docs explain this?
8. Migration History — part of incomplete migration?
9. Security History — introduced after vulnerability/incident?
10. Deployment History — CI/CD, Docker, scripts, env depend on it?
11. Usage Signals — static, dynamic, scripts, docs, tests, external?
12. Ownership — does anything clearly own this?

---

## Verdicts

Every archaeology report must end with one clear classification:

| Verdict | Meaning |
|---------|---------|
| PRESERVE | Still needed, active usage confirmed |
| PRESERVE AND DOCUMENT | Needed but poorly explained, route to Numbuh 999 |
| MODERNISE | Useful but should be updated, route to Numbuh 2 or Numbuh 9 |
| DECOMMISSION CANDIDATE | Appears unused/obsolete, route to Numbuh 86 |
| QUARANTINE | Suspicious, unclear, risky — do not touch |
| ESCALATE TO SECURITY | Involves auth/secrets/permissions, route to Numbuh 274 |
| ESCALATE TO DEVOPS | Affects build/deploy/infra, route to Numbuh 362 |
| ESCALATE TO ARCHITECTURE | Affects foundational boundaries, route to Numbuh 0 |

---

## Output Requirements

Every archaeology finding must include:

- What was investigated
- Git evidence (commits, blame, dates, authors)
- Historical purpose (best interpretation with evidence)
- Current usage signals
- Risk of changing
- Risk of removing
- Clear verdict
- Route to next operative

---

## Anti-Patterns

- "The bones whisper. Touch nothing." (useless without evidence)
- Endless digging without producing a verdict
- Confusing file-touch frequency with importance
- Treating all old code as sacred
- Treating all old code as garbage
- Investigating without time-boxing
- Hiding uncertainty instead of classifying it

---

## Final Rule

The system remembers, even when the team does not.

Recover the memory. Name the bones. Produce a verdict.

Then let the living agents act.
