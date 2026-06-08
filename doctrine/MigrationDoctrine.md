# Migration Doctrine

Migration standards for Moonbase.

---

## Core Principle

Never big-bang.

You do not drag a system into the future. You guide it across.

---

## Universal Migration Rules

- Every migration step must be independently deployable.
- Every migration step must be independently reversible.
- Every compatibility layer must have an expiry condition.
- Every shim must have an owner.
- Every cutover must have verification criteria.
- Every migration must have a decommissioning handoff.
- A bridge is not a home.

---

## Migration Plan Requirements

Every migration plan must include:

1. Current state — what exists now
2. Target state — what should exist after
3. Affected areas — files, modules, deps, configs, tests, docs, deployment
4. Breaking changes — what could break
5. Compatibility requirements — what must continue working
6. Phases — small, independently deployable steps
7. Verification per phase — how each phase is tested
8. Rollback per phase — how each phase can be undone
9. Cutover criteria — when do we switch fully
10. Decommissioning handoff — what Numbuh 86 removes after success
11. Documentation handoff — what Numbuh 999 records

---

## Compatibility Layer Rules

Every bridge/shim/adapter must include:

- Purpose
- Owner
- Consumers
- Expiry condition
- Removal signal
- Tests proving new path works
- Decommissioning handoff to Numbuh 86
- Documentation note for Numbuh 999

If any are missing: "Bridge without expiry detected."

---

## Phase Design

Each phase must answer:

- What changes?
- Can it be deployed alone?
- Can it be rolled back alone?
- How do we verify it worked?
- Does old path still function?
- Does new path function?
- Is parity confirmed?

---

## Migration Anti-Patterns

- Big-bang replacement
- "Temporary" code with no expiry
- Half-completed migrations left indefinitely
- Removing old path before new path is proven
- Migrating without deployment assessment
- Migrating without legacy context (call Sector Z)
- Migrating without security review when auth/secrets are involved

---

## Specialist Involvement

- Sector Z: before migrating old/unknown code
- Numbuh 362: when deployment is affected
- Numbuh 274: when security boundaries change
- Numbuh 86: after crossing is complete (remove old path)
- Numbuh 999: document the new route
- Numbuh 0: when architecture boundaries shift

---

## Final Rule

The old system is not stupid because it is old.

The new system is not good because it is new.

Both must be understood. Then the crossing can begin.

Never give up.
