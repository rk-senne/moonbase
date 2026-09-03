---
name: distributed-data
description: Owning data across service boundaries — shared databases, sagas vs two-phase commit, splitting tables. From Monolith to Microservices (Newman).
---

# Distributed Data

## Independent Deployability Is the Test

Can you release one service without deploying anything else? That needs loose coupling
and explicit, stable contracts. A **shared database is the worst enemy** of it: nobody
can say what is hidden versus public, or who may mutate what. The only narrow
exceptions are read-only static reference data and a schema *deliberately* published as
a managed interface. Everything else is an unowned write path.

## Avoid Two-Phase Commit

2PC holds distributed locks, breaks isolation (an observable window where participants
disagree), adds latency, and fails in ways needing manual repair — availability drops as
participants increase. **If you truly need atomicity, don't split that data.**

## Sagas Give You No Atomicity

A saga is a sequence of local transactions, so partial completion is normal. Recover
either way:

- **Backward** — compensating transactions, which are *semantic*. You cannot unsend an
  email, only send a cancellation.
- **Forward** — retry, which requires persisted state.

Reorder steps to fail early. Choose **orchestrated** (central coordinator, clearer
visibility, more coupling — fine when one team owns the flow) or **choreographed**
(event-driven, looser coupling — better across teams). Always keep a documented way to
determine a saga's current state.

## Splitting Storage

- **Database-per-bounded-context**, even inside a monolith, so the seam exists before
  you need it.
- **Split table** column-by-column when one table serves two contexts. When a column is
  written from several places, ownership belongs to the service owning that entity's
  state machine — everyone else asks.
- **Foreign key into code** costs latency (mitigate with bulk lookups or caching) and
  loses database-enforced referential integrity.

## Static Reference Data

Duplicate per service; a versioned reference schema; a statically linked library for
small rarely-changing sets; or a service when it must stay consistent.
