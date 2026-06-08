# Mission Template: TECH_DEBT_CLEANUP

## Default Flow

Numbuh 86 → Sector Z → Numbuh 3 or Numbuh 2 → Numbuh 4 → Numbuh 5

## Difficulty

SMALL to LARGE depending on scope.

## Conditional Agents

- Numbuh 0 for architecture impact
- Numbuh 362 for deployment assets
- Numbuh 274 for security-related cleanup
- Numbuh 999 for docs cleanup

## Required Outputs

- Decommissioning report (Numbuh 86)
- Legacy context (Sector Z)
- Safe removal plan (Numbuh 86)
- Implementation of removal (Numbuh 3)
- Verification (Numbuh 4)
- Final review (Numbuh 5)

## Approval Gates

- Human approval before any destructive deletion
- Sector Z must confirm load-bearing status before removal

## Success Criteria

- Evidence proves artifacts are unused
- Dynamic usage checked
- Removal is reversible or approved as destructive
- Tests pass after removal
- Documentation updated (stale refs removed)
- No unrelated changes included
