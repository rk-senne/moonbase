# Agent Quality Scores

Every agent output should be self-scored before handoff.

## Score Categories

| Dimension | Question |
|-----------|----------|
| CLARITY | Can the next agent understand without re-reading? |
| EVIDENCE | Are claims supported by commands/files/output? |
| COMPLETENESS | Are all required sections present? |
| RISK_CONTROL | Are risks identified and classified? |
| NEXT_STEP_QUALITY | Is the handoff specific and actionable? |

## Scale

1 = Missing or unusable
2 = Present but weak
3 = Acceptable minimum
4 = Good
5 = Excellent

## Rule

If any score is below 3, the agent should revise before handoff.

## Score Format

```md
## Output Quality

CLARITY: X/5
EVIDENCE: X/5
COMPLETENESS: X/5
RISK_CONTROL: X/5
NEXT_STEP_QUALITY: X/5

REVISION_REQUIRED: YES / NO
```

## Verification

Self-assessed by the producing agent. Verified by Numbuh 5 at final review.

If Numbuh 5 disagrees with self-assessment, she may send back for revision.
