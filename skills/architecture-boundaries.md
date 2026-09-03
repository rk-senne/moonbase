---
name: architecture-boundaries
description: Dependency direction, component cohesion/coupling metrics, and where to draw or defer a boundary. From Clean Architecture (Martin).
---

# Architecture Boundaries

## The Dependency Rule

Source dependencies point inward, toward higher-level policy. Inner code must never
name anything — type, function, variable, or data format — declared further out.
"Level" means distance from I/O: higher-level policy sits farther out and changes less.

## Cohesion (balance, don't maximize)

- **REP** — the granule of reuse is the granule of release. Version it.
- **CCP** — gather what changes together for the same reason (SRP for components).
- **CRP** — don't force users to depend on what they don't need.

REP/CCP grow components; CRP shrinks them. Young code favors CCP; shared code slides
toward REP/CRP. Partitioning is dynamic — re-evaluate it.

## Coupling

- **ADP** — no cycles, ever. Break one by inverting an edge with DIP or extracting a
  shared component. A cycle fuses components into one release unit.
- **SDP** — depend in the direction of stability.
- **SAP** — stable components must be abstract to stay extensible.
- **D = |A + I − 1|**. Stable+concrete = Zone of Pain (rigid). Abstract with no
  dependents = Zone of Uselessness.

## Boundary Options (cheapest first)

1. **Facade** — no inversion; client keeps transitive dependencies.
2. **Strategy interface** — one-directional DIP.
3. **Full boundary** with reciprocal interfaces — expensive, strongest.

Build the full boundary at the inflection point where its cost drops below the cost of
not having it.

## Keep Details Deferrable

Database, web, and framework are details behind boundaries — maximize the number of
decisions not made. Prefer package-by-component and let the compiler enforce the rule
(package-private, `internal`); review discipline erodes under deadlines.

## Testability

Humble Object split: testable logic produces a plain view model (strings, booleans,
enums); the humble edge — screen, socket, driver — stays thin. Business rules must run
with no server, database, or framework booted.
