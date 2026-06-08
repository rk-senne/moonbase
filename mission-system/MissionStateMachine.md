# Mission State Machine

## States

```
INTAKE
ANALYSIS
DESIGN
IMPLEMENTATION
QA
CHAOS_TESTING
SECURITY_REVIEW
FINAL_REVIEW
DEPLOYMENT_REVIEW
DOCUMENTATION
MIGRATION
DECOMMISSIONING
ARCHAEOLOGY
HUMAN_APPROVAL
DONE
BLOCKED
ESCALATED
```

## State Rules

Every agent output must either:

1. Move the mission to the next state.
2. Send work back to a previous state.
3. Escalate to a specialist.
4. Block the mission with a clear reason.
5. Mark the mission done.

No agent may end vaguely.

## Default Pipeline

```
INTAKE → ANALYSIS (Numbuh 1) → DESIGN (Numbuh 2) → IMPLEMENTATION (Numbuh 3) → QA (Numbuh 4) → FINAL_REVIEW (Numbuh 5) → HUMAN_APPROVAL → DONE
```

## Conditional States

| State | Trigger |
|-------|---------|
| SECURITY_REVIEW | Auth, secrets, permissions, shell, file access, webhooks, AI tool boundaries, dependency CVEs |
| DEPLOYMENT_REVIEW | CI/CD, Docker, deployment scripts, env vars, infrastructure, runtime config |
| MIGRATION | Version upgrades, framework changes, library replacements, API deprecations |
| DECOMMISSIONING | Dead code, unused dependencies, stale configs, duplicate logic |
| ARCHAEOLOGY | Old, mysterious, undocumented, or risky legacy code |
| DOCUMENTATION | README, ADR, changelog, onboarding, setup, migration, security, or deployment docs |
| CHAOS_TESTING | Edge case coverage needed, fragile flows, user-facing inputs |

## State Transitions

| From | To | Trigger |
|------|----|---------|
| INTAKE | ANALYSIS | Human provides request |
| ANALYSIS | DESIGN | Numbuh 1 completes mission brief |
| DESIGN | IMPLEMENTATION | Numbuh 2 completes blueprint |
| IMPLEMENTATION | QA | Numbuh 3 completes code + tests |
| QA | SECURITY_REVIEW | Scope touches security |
| QA | FINAL_REVIEW | Numbuh 4 rates LOW, no security scope |
| QA | IMPLEMENTATION | Numbuh 4 rates MEDIUM (rework) |
| QA | DESIGN | Numbuh 4 rates HIGH (redesign) |
| QA | BLOCKED | Numbuh 4 rates CRITICAL |
| SECURITY_REVIEW | FINAL_REVIEW | Numbuh 274 clears |
| FINAL_REVIEW | HUMAN_APPROVAL | Numbuh 5 approves |
| FINAL_REVIEW | IMPLEMENTATION | Numbuh 5 sends back |
| HUMAN_APPROVAL | DEPLOYMENT_REVIEW | Deploy needed |
| HUMAN_APPROVAL | DONE | No deploy needed |
| DEPLOYMENT_REVIEW | DONE | Numbuh 362 confirms healthy |
| Any | BLOCKED | Stop condition triggered |
| Any | ARCHAEOLOGY | Legacy context required |
