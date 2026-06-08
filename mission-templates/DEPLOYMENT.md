# Mission Template: DEPLOYMENT

## Default Flow

Numbuh 362 → Numbuh 274 → Numbuh 5 → Human

## Difficulty

MEDIUM to CRITICAL depending on scope.

## Conditional Agents

- Numbuh 9 if deployment includes migration
- Numbuh 999 for runbook
- Sector Z if old deployment scripts are involved

## Required Outputs

- Deployment assessment (Numbuh 362)
- Secrets review (Numbuh 274)
- Health checks defined (Numbuh 362)
- Rollback plan (Numbuh 362)
- Runbook if new process (Numbuh 999)
- Final review (Numbuh 5)

## Approval Gates

- Human approval required before any deployment
- Numbuh 274 must clear secrets/security
- Numbuh 362 must confirm health check + rollback exist

## Success Criteria

- Build passes
- Health check responds
- Rollback tested or documented
- Secrets sealed
- No deployment through known vulnerability
- Runbook exists for new processes
