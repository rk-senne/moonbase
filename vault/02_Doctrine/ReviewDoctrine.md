# Review Doctrine

Final review standards for Moonbase.

---

## Core Principle

Review the whole package, not only the code.

If the human would need to guess what happened, the review is not complete.

---

## Final Review Checklist

1. Does the result match the mission objective?
2. Were unrelated changes introduced?
3. Is the implementation readable and maintainable?
4. Are tests meaningful and actually run?
5. Are risks identified and honestly classified?
6. Is the rollback plan believable?
7. Does this need architecture review? (>5 files, new pattern, boundary change)
8. Does this need security review? (auth, secrets, shell, file access, webhooks)
9. Does this need deployment review? (CI/CD, Docker, env vars, infra)
10. Is the approval package complete enough for a human to decide?

---

## Review Questions

- Does the PR summary match the diff?
- Does the implementation match the requirements?
- Do the tests prove what they claim?
- Are the risk notes logically complete?
- Is the scope clean?
- Are there contradictions between agent reports?
- Is this package intellectually honest?

---

## Review Verdicts

- **APPROVED FOR HUMAN REVIEW** — no blocking gaps, minor notes allowed
- **SEND BACK TO NUMBUH 3** — implementation issue (quality, test, naming, minor bug)
- **SEND BACK TO NUMBUH 2** — design issue (gap, wrong abstraction, state flow unclear)
- **SEND BACK TO NUMBUH 1** — requirements issue (objective unclear, ACs incomplete)
- **ESCALATE TO NUMBUH 0** — architecture triggers met
- **ESCALATE TO SPECIALIST** — security, DevOps, migration, decommissioning, legacy
- **BLOCKED** — package not reviewable (missing evidence, unknown risk, human would guess)

---

## Architecture Review Triggers

Recommend Numbuh 0 when:

- More than 5 files changed
- Core logic changed
- New architectural pattern introduced
- Tool or pipeline abstraction changed
- AI backend abstraction changed
- Major dependency introduced
- Maintainability risk affects future operatives

---

## Evidence in Review

The reviewer must compare claims against evidence:

- "Tests pass" → where is the test output?
- "No regressions" → what was checked?
- "Low risk" → does the diff agree?
- "Rollback plan" → is it actually reversible?
- "Small change" → does the file count match?

If claims lack evidence, send back or block.

---

## Final Rule

The reviewer is the last shield.

If something passes review that should not have, the mission pays later.

Do not approve because everyone is tired.

Do not approve because it looks fine.

Approve because the evidence says it is ready.
