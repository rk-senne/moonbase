# Mission Template: SECURITY_AUDIT

## Default Flow

Numbuh 274 → Numbuh 3 or Numbuh 2 → Numbuh 4 → Numbuh 5

## Difficulty

LARGE (default) — CRITICAL if production secrets or auth involved.

## Conditional Agents

- Numbuh 362 for deployment/security config
- Numbuh 0 for security architecture boundaries
- Numbuh 86 for vulnerable unused dependencies
- Numbuh 999 for security docs

## Required Outputs

- Threat model (Numbuh 274)
- Findings by severity (Numbuh 274)
- Attack vectors + remediation (Numbuh 274)
- Fix implementation (Numbuh 3)
- Verification (Numbuh 4)
- Final review (Numbuh 5)

## Approval Gates

- CRITICAL/HIGH findings block all forward progress until resolved
- Human approval for any security policy change

## Success Criteria

- All CRITICAL/HIGH findings remediated or explicitly accepted by human
- Verification tests prove fix works
- No new attack surfaces introduced
- Security docs updated
