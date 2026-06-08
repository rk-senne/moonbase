# Mission Template: FEATURE_BUILD

## Default Flow

Numbuh 1 → Numbuh 2 → Numbuh 3 → Numbuh 4 → Numbuh 5 → Human

## Difficulty

MEDIUM (default) — upgrade to LARGE if architecture/security/deploy involved.

## Conditional Agents

- Numbuh 274 if feature touches security
- Numbuh 362 if feature touches deployment/env/CI
- Numbuh 999 if docs are needed
- Numbuh 0 if architecture changes (>5 files, new pattern, boundary shift)
- Numbuh 13 if edge-case testing is useful

## Required Outputs

- Requirements packet (Numbuh 1)
- Design blueprint (Numbuh 2)
- Implementation report (Numbuh 3)
- QA risk report (Numbuh 4)
- Final review package (Numbuh 5)

## Approval Gates

- Human approval before merge
- Numbuh 0 review if architecture triggers met
- Numbuh 274 review if security scope detected

## Success Criteria

- All acceptance criteria met
- Tests pass
- Risk classified and accepted
- Rollback path exists
- Documentation updated if user-facing
