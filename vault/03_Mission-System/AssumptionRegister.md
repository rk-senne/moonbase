# Assumption Register

Every assumption must be logged. Do not hide assumptions inside confident language.

## Format

```md
ASSUMPTION:
WHY_MADE:
RISK_IF_WRONG:
WHO_CAN_CONFIRM:
CAN_PROCEED: YES / NO
```

## Rules

- If an assumption affects architecture → route to Numbuh 0 or Numbuh 2.
- If an assumption affects security → route to Numbuh 274.
- If an assumption affects deployment → route to Numbuh 362.
- If an assumption affects requirements → route to Numbuh 1.
- If an assumption is HIGH risk → must be confirmed before proceeding.
- If an assumption is LOW risk and labelled → may proceed.

## Anti-Pattern

Do not present assumptions as facts.

Bad: "The config is loaded at startup."

Good: "ASSUMPTION: The config is loaded at startup. Based on existing pattern in config.go. Risk if wrong: runtime panic. Can proceed: YES (low risk, verifiable by Numbuh 4)."
