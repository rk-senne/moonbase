---
name: design-patterns
description: The OO principles behind the patterns, picking between similar patterns, and when a pattern is overkill. From Head First Design Patterns.
---

# Design Patterns

## The Principles Behind the Patterns

- **Encapsulate what varies** — separate what changes from what doesn't. This motivates
  nearly every pattern.
- **Favor composition over inheritance** — inheritance fixes behavior at compile time;
  composition lets it be swapped at runtime.
- **Program to an interface, not an implementation.**
- **Open–Closed, but not everywhere** — blanket OCP adds abstraction and complexity.
  Target only the axes likely to change.
- **Least Knowledge (Demeter)** — avoid `a.getB().getC().doThing()`; every link is an
  undeclared dependency.
- **Hollywood Principle** — "don't call us, we'll call you": high-level components
  decide when low-level ones run.

## Telling Similar Patterns Apart

| Choice | Distinction |
|---|---|
| Factory Method vs Abstract Factory | Inheritance (subclass decides) vs composition (factory makes families) |
| Adapter vs Facade | Convert a mismatched interface vs simplify a subsystem — intent, not size |
| Strategy vs State | Client-configured algorithm vs behaviors the context moves through over time |
| Decorator vs Proxy | Add behavior vs control access |

## Patterns With Sharp Edges

- **Decorator** — overuse yields many small objects and unfollowable call chains.
- **Singleton** — the lazy accessor isn't thread-safe. Synchronizing it is far slower;
  eager static init is JVM-guaranteed; double-checked locking needs `volatile`; an enum
  is simplest.
- **Composite** — child-management in the shared interface buys uniformity at the cost
  of type safety; splitting it buys safety at the cost of client type checks.
- **Template Method** — abstract methods are required steps, hooks are optional with
  empty defaults.

## Don't Overengineer

The "Zen mind" caution: overusing patterns produces overengineered code. Introduce one
only for a real present need or a genuinely likely change, and prefer the simplest
thing that works. Pattern names are most valuable as shared vocabulary.
