# Mission Template: REFACTOR

## Default Flow

Numbuh 1 → Numbuh 2 → Numbuh 0 → Numbuh 3 → Numbuh 4 → Numbuh 5

## Difficulty

MEDIUM to LARGE.

## Conditional Agents

- Sector Z if old code is involved
- Numbuh 86 if dead code is found during refactor
- Numbuh 999 if architecture docs or ADR needed
- Numbuh 274 if refactor changes security boundaries

## Required Outputs

- Refactor objective (Numbuh 1)
- Current structure + desired structure (Numbuh 2)
- Architecture review (Numbuh 0)
- Implementation report (Numbuh 3)
- QA verification (Numbuh 4)
- Final review (Numbuh 5)

## Approval Gates

- Numbuh 0 must approve before implementation begins
- Human approval before merge

## Success Criteria

- Behaviour preserved (tests still pass)
- Structure improved
- No unrelated changes
- ADR created if new pattern introduced
- Rollback path exists
