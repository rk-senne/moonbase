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

## Final Rule

Good architecture makes the system easier to change tomorrow.

Bad architecture makes the system harder to change tomorrow.

If a decision makes future work harder without clear present justification, it is debt wearing a suit.

Will future operatives thank us for this, or curse us?
