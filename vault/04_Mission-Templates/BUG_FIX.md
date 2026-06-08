# Mission Template: BUG_FIX

## Default Flow

Numbuh 1 → Numbuh 3 → Numbuh 4 → Numbuh 5

## Difficulty

SMALL (default) — upgrade if bug is in core logic or security-relevant.

## Optional Agents

- Numbuh 2 if design flaw is discovered
- Numbuh 13 if bug is edge-case-heavy
- Sector Z if bug is in legacy code
- Numbuh 274 if security implication exists

## Required Outputs

- Bug definition + reproduction steps (Numbuh 1)
- Fix report + regression test (Numbuh 3)
- QA confirmation (Numbuh 4)
- Review package (Numbuh 5)

## Approval Gates

- Human approval if fix touches >3 files or shared logic

## Success Criteria

- Bug reproduced and fixed
- Regression test added
- No unrelated changes
- Tests pass
