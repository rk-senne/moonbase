# Decommissioning Doctrine

Decommissioning standards for Moonbase.

---

## Core Principle

Do not delete based on vibes.

If it no longer serves the mission, it must be revived, documented, quarantined, or decommissioned — with evidence.

---

## Universal Rules

- Old does not mean useless.
- Unused does not mean safe to remove.
- Sentiment is not a dependency.
- Dynamic usage must be checked.
- Decommissioning requires evidence.
- Human approval required before destructive removal unless explicitly authorised.
- Grep is not a god. It is a witness.

---

## Decommissioning Categories

| Category | Meaning |
|----------|---------|
| KEEP | Active, serves the mission |
| KEEP AND DOCUMENT | Strange/messy but has confirmed purpose |
| QUARANTINE | May be unused, removal risk unclear |
| DECOMMISSION CANDIDATE | Appears safe to remove after confirmation |
| DECOMMISSION APPROVED | Proven unused, non-load-bearing, safe to remove |
| ESCALATE | Specialist required before decision |

---

## Evidence Requirements

Minimum for removal:

1. No static references found
2. No dynamic/runtime usage suspected (or checked)
3. No config/build/deployment dependency
4. No docs or tests require it (or those are also stale)
5. Removal plan includes rollback path
6. Human approval for destructive action

Evidence chain: One signal = suspicion. Two = investigation. Three clean = candidate. Deletion requires proof.

---

## Hidden Usage Risks

Always check for:

- Reflection
- Dependency injection
- Plugin/MCP loading
- CLI command registration
- Build tags
- Generated code
- Templates
- JSON/YAML config references
- Environment variables
- Shell scripts
- CI/CD references
- Test fixtures
- External consumers
- Documentation examples
- Runtime string references

If hidden usage is possible → QUARANTINE or ESCALATE.

---

## Inspection Areas

1. Dead code (unreferenced functions/files/modules/routes)
2. Unused dependencies
3. Deprecated APIs
4. Duplicate logic
5. Stale configs
6. TODO/FIXME debt
7. Zombie features
8. Test rot
9. Documentation rot
10. Ownership gaps

---

## Specialist Routing

- Architecture concerns → Numbuh 0
- Security rot → Numbuh 274
- Deployment configs → Numbuh 362
- Migration bridges → Numbuh 9
- Documentation updates → Numbuh 999
- Legacy unknowns → Sector Z
- Implementation of removal → Numbuh 3
- Final review → Numbuh 5

---

## Final Rule

A decommissioning order without proof is just yelling.

Sentiment is not a dependency.

But neither is impatience a justification for reckless deletion.

Diagnose. Confirm. Classify. Then issue the order.
