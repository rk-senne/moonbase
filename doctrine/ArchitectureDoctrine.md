# Architecture Doctrine

Architecture standards for Moonbase.

---

## Core Principle

Architecture is not abstraction for its own sake.

Architecture is the set of decisions that are expensive to change later.

Make those decisions well. Make them reversible when possible. Document them always.

---

## Foundational Rules

- Prefer simple over clever.
- Prefer boring technology over exciting technology unless the mission demands excitement.
- Prefer composition over inheritance.
- Prefer explicit over implicit.
- Prefer small interfaces over large ones.
- Prefer flat structures over deep nesting.
- Do not add abstraction until the second or third concrete need appears.

---

## Boundaries

- Every module should have clear responsibility.
- Dependencies flow inward (toward domain logic), not outward (toward infrastructure).
- Interfaces define boundaries. Implementations live behind them.
- Do not let infrastructure decisions leak into domain logic.
- Do not let domain logic leak into presentation.

---

## Component Structure

Cohesion and coupling are measurable, not matters of taste.

**Cohesion — a tension to balance, never to maximize:**

- **REP** — the granule of reuse is the granule of release. What others depend on must be versioned and released as a whole.
- **CCP** — gather what changes together, for the same reason. This is SRP at component scale.
- **CRP** — do not force a component's users to depend on what they do not need.

REP and CCP push components larger; CRP pushes them smaller. Young code should favour CCP (developability); shared code slides toward REP/CRP (reuse). Partitioning is expected to be revisited — treat it as dynamic, not settled.

**Coupling — hard rules:**

- **ADP — no cycles in the component dependency graph.** Not "few." None. A cycle fuses components into a single de facto release unit and drags unrelated dependencies into every test build. Break it by inverting one edge with DIP, or by extracting a component both sides depend on.
- **SDP** — depend in the direction of stability. Stability is driven by how many things depend on you, not by how well written you are.
- **SAP** — a component that is heavily depended upon must be abstract, or it cannot be extended.

**The Main Sequence.** With A = abstractness and I = instability, D = |A + I − 1| measures distance from the ideal. Two exclusion zones:

- **Zone of Pain** — stable and concrete. Rigid, widely depended upon, impossible to extend.
- **Zone of Uselessness** — abstract with no dependents. Detritus that still occupies attention.

When D grows over time, the structure is drifting. Report it with the numbers.

**Boundary cost.** A full boundary with reciprocal interfaces is expensive; a Strategy interface is a one-directional DIP; a Facade is cheaper still but leaves the client with transitive dependencies. Build the full boundary at the inflection point where its cost drops below the cost of not having it — not before, and not after the mess sets in.

**Enforcement.** Prefer package-by-component, and let the compiler enforce the rule via visibility (`internal`, package-private). A rule enforced only by review discipline erodes the moment a deadline appears.

---

## Duplication

Distinguish two things that look identical:

- **True duplication** — the copies must always change together. Unify it.
- **Accidental duplication** — the copies merely resemble each other today and will diverge (a stored record shape that happens to match a screen shape). Unifying it manufactures coupling between independent concepts.

Collapsing accidental duplication looks like cleanliness at review time and hurts a year later.

---

## Decision Records

Significant architectural decisions must be recorded as ADRs.

An ADR is needed when:

- A new pattern is introduced.
- A dependency is added that affects multiple modules.
- A boundary is created or changed.
- A trade-off is made that future operatives must understand.
- A previously considered option is explicitly rejected.

Format: Context → Decision → Consequences → Alternatives → Reversibility.

---

## Dependency Direction

```
cmd/        → internal/
internal/   → (no outward dependency)
agents/     → loaded by internal/agents
doctrine/   → loaded as resources
```

- `internal/` packages may depend on each other, but avoid circular imports.
- External packages are imported only where needed, not globally.
- Interfaces live where they are used, not where they are implemented.

---

## Patterns

### Accept Interfaces, Return Structs

Functions should accept interface parameters and return concrete types.

This keeps dependencies narrow and testing simple.

### Configuration

- Configuration comes from environment variables or config files.
- No hardcoded magic values in business logic.
- Defaults must be safe and documented.
- Sensitive config uses environment variables, never committed files.

### Error Handling

- Errors are values. Handle them explicitly.
- Wrap errors with context as they travel up the stack.
- Do not swallow errors silently.
- Use structured error types when callers need to distinguish failure modes.

### State Management

- Minimise shared mutable state.
- If state must be shared, protect it with clear ownership.
- Prefer message-passing over shared memory where the runtime supports it.

---

## Anti-Patterns

Do not:

- Create abstractions before you have two concrete cases.
- Add layers "for future flexibility" without a current need.
- Split a module just because it has many lines (split by responsibility, not size).
- Introduce a framework when a library suffices.
- Introduce a library when the standard library suffices.
- Create god objects that know everything.
- Create util/helper packages that become dumping grounds.

---

## Moonbase-Specific Architecture

- Agents are declarative configs + markdown prompts. They are not compiled code.
- The TUI (Bubble Tea) uses the Model-Update-View pattern. Respect it.
- Backend providers are behind an interface. Adding a provider must not change the orchestration layer.
- The pipeline is the agent execution flow. It must remain simple and traceable.
- Doctrine is loaded as resources, not compiled into agents.

---

## Reversibility

Before introducing an architectural change, answer:

- Can this be undone if it fails?
- What is the blast radius?
- Who else is affected?
- What breaks if we revert?
- Is there a smaller version we can try first?

If the answer to all of these is "unclear," the change needs more investigation before proceeding.

---

## Reference Canon

The rules above are not invented here. They are drawn from, and traceable to, the
research corpus indexed in `research/`:

| Source | What it grounds |
|---|---|
| *Clean Architecture* (Robert C. Martin) | Dependency Rule, SOLID, REP/CCP/CRP, ADP/SDP/SAP, the Main Sequence, partial boundaries, Humble Object, package-by-component, true vs accidental duplication |
| *The C4 Model* (Simon Brown) | Context → Container → Component → Code levels, notation rules, deployment-per-environment, models-as-code, the model–code gap |
| *Monolith to Microservices* (Sam Newman) | Independent deployability, strangler fig, branch by abstraction, parallel run, tracer write, shared-database hazards, sagas vs 2PC |
| *Head First Design Patterns* (Freeman & Robson) | Encapsulate what varies, composition over inheritance, program to interfaces, Least Knowledge, pattern-overuse caution |
| *The Clean Coder* (Robert C. Martin) | Do no harm to structure, TDD's three laws, estimates vs commitments, the marshes-and-messes inflection point, crisis discipline |
| *Algorithms Notes* (GoalKicker) | Complexity bounds, algorithm preconditions, degenerate-input discipline |
| *Java Notes* (GoalKicker) / *Learning Java* (Loy, Niemeyer, Leuck) | Language-level correctness: resource lifetimes, boxing traps, concurrency primitives, bounded pools, charset boundaries |
| *A Philosophy of Software Design* (Ousterhout) | Complexity symptoms and causes, zero tolerance, strategic vs tactical, deep modules, information leakage, temporal decomposition, define errors out of existence, design it twice |
| *The Pragmatic Programmer* (Thomas, Hunt) | No broken windows, DRY as knowledge, the orthogonality test, reversibility, tracer bullets vs prototypes, design by contract, crash early, balance resources |
| *The Phoenix Project* (Kim, Behr, Spafford) | The Three Ways, Theory of Constraints, four types of work, WIP control, queue theory, deployment automation, technical debt as compounding interest |
| *Designing Data-Intensive Applications* (Kleppmann) | Reliability/scalability/maintainability, percentiles and tail amplification, storage engine trade-offs, encoding evolution, replication lag, isolation levels, fencing tokens, idempotence |
| *Black Hat Go* (Steele, Patten, Kottmann) | Defensive Go: mutual TLS verification, contextual template escaping, middleware chain termination, network boundary handling |

Corresponding progressive skills live in `skills/`: `architecture-boundaries`,
`architecture-diagrams`, `design-patterns`, `incremental-migration`,
`distributed-data`, `algorithmic-complexity`.

---

## Final Rule

Good architecture makes the system easier to change tomorrow.

Bad architecture makes the system harder to change tomorrow.

If a decision makes future work harder without clear present justification, it is debt wearing a suit.

Will future operatives thank us for this, or curse us?
