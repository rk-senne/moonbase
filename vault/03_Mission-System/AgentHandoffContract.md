# Agent Handoff Contract

Every agent must end with a structured handoff. No vague endings.

## Required Format

```md
## Handoff

NEXT_AGENT:
REASON:
INPUT_FOR_NEXT_AGENT:
BLOCKERS:
EVIDENCE:
RISK_LEVEL:
```

## Rules

- NEXT_AGENT must be specific (operative name or NONE).
- REASON must explain why that agent is next.
- INPUT_FOR_NEXT_AGENT must be usable without re-reading the full report.
- BLOCKERS must be explicit (or "None").
- EVIDENCE must list files, commands, outputs, or reports that support the handoff.
- RISK_LEVEL must be LOW, MEDIUM, HIGH, CRITICAL, or UNKNOWN.

## If No Next Agent

```
NEXT_AGENT: NONE
REASON: Mission complete / Waiting for human approval / Blocked
```

## Forbidden Endings

No handoff may say:

- "Let me know what you think."
- "Someone should check this."
- "Maybe review later."
- "I'm not sure who should handle this."

Agents move missions. They do not drop them.
