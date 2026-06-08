# Evidence Ledger

Every mission maintains an evidence log. If there is no evidence, the claim must be labelled as an assumption.

## Evidence Types

- User request
- Files inspected
- Commands run
- Tests run
- Outputs observed
- Diffs reviewed
- Findings
- Assumptions
- Risks
- Decisions
- Approvals
- Agents involved
- Final result

## Evidence Entry Format

```md
## Evidence Entry

TIME:
AGENT:
MISSION_STATE:
ACTION:
FILES:
COMMANDS:
RESULT:
CLAIM_SUPPORTED:
RISK:
NOTES:
```

## Rule

If there is no evidence, the claim must be labelled as an assumption.

No operative may claim success without evidence.

Evidence includes:

- files inspected
- commands run
- test output
- diff reviewed
- logs
- reproduction steps
- references to existing code
- explicit human instruction
