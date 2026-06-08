# Mission Control Doctrine

Moonbase is not an agent chatroom. Moonbase is mission control.

## Architecture

```
doctrine/           = what agents believe and obey (law)
mission-system/     = how missions move (process)
mission-templates/  = repeatable workflows
permissions/        = what agents may touch (safety)
```

## Mission System Files

The operational process lives in `mission-system/`:

- `MissionStateMachine.md` — states, transitions, pipeline flow
- `AgentHandoffContract.md` — structured handoff format
- `EvidenceLedger.md` — proof and traceability
- `AssumptionRegister.md` — logged assumptions
- `HumanApprovalGates.md` — required human approval triggers
- `ConflictResolution.md` — authority when agents disagree
- `StopConditions.md` — when to halt and escalate
- `MissionMemory.md` — post-completion records
- `AgentQualityScores.md` — output quality scoring
- `PostMissionReview.md` — lessons learned

## Mission Difficulty

| Rating | Pipeline |
|--------|----------|
| TINY | Single agent, no pipeline |
| SMALL | 2–3 agents |
| MEDIUM | Full core pipeline (N1→N2→N3→N4→N5) |
| LARGE | Full pipeline + specialists |
| CRITICAL | Full pipeline + all gates active |

## Core Rules

- Every agent output must move the mission forward or explicitly block it.
- Every handoff must be structured and actionable.
- Every claim must be supported by evidence.
- Every assumption must be labelled.
- Every destructive action requires human approval.
- Human wins always.

## Final Law

Never let personality outrank evidence.
