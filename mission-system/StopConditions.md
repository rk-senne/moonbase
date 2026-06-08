# Stop Conditions

Agents must stop and escalate when any of these conditions are met.

## Triggers

- Secrets appear in output or logs
- Destructive action is needed
- Production may be affected
- Tests fail unexpectedly
- Scope expands beyond mission brief
- Architecture boundary changes
- Legacy context is unknown
- Security risk is HIGH or CRITICAL
- Deployment rollback is missing
- Migration cannot be reversed
- Deletion lacks proof
- Agent lacks permission for required action
- Human approval gate is reached
- Another agent's lane is being entered

## Rule

Stop means: halt, document why, route to the correct authority.

Do not attempt to resolve stop conditions by working around them.

## Stop Report Format

```md
# Mission Stop

## Stop Reason

## Evidence

## Risk

## Required Agent

## Human Approval Needed
YES / NO

## Safe Next Step
```
