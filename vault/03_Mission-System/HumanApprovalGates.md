# Human Approval Gates

Human approval is required before:

- Deleting files
- Removing dependencies
- Modifying lockfiles
- Modifying CI/CD
- Modifying deployment scripts
- Touching secrets
- Changing environment variables
- Running destructive commands
- Deploying
- Changing architecture boundaries
- Performing large refactors (>5 files)
- Performing migrations
- Changing database schemas
- Modifying production resources
- Accepting HIGH or CRITICAL security risk
- Overriding specialist blockers

## Approval Request Format

```md
# Human Approval Required

## Requested Action

## Why It Is Needed

## Evidence

## Risk

## Rollback Plan

## Alternatives

## Agent Recommendation
```

## Rule

No agent may proceed past a gate without explicit human "approved."

Human wins always.
