---
name: architecture-diagrams
description: C4 levels, notation rules that let a diagram stand alone, and which diagrams are worth maintaining. From The C4 Model (Brown).
---

# Architecture Diagrams

## Four Levels of Zoom

1. **System Context** — the system, its users, the systems it talks to. Best tool for
   pinning down what's in scope and what isn't.
2. **Container** — separately deployable/runnable units, their responsibilities and
   technologies. Not a Docker container; every inter-container call is out-of-process.
3. **Component** — groupings inside one container. Optional and volatile.
4. **Code** — rarely worth drawing; the code answers those questions.

Agree the vocabulary first — teams mean different things by "component".

## Notation Rules

A diagram must survive being extracted from its page:

- A **title** stating type and scope, plus a **key** for every shape, color, line style.
- Put **Name, Type, Technology, Description** inside each box, and tag the type
  (`[Person]`, `[Container: Go]`). Name-only boxes are the top ambiguity source.
- **Label every relationship** with a description ("reads data from") *and* the
  protocol. Solid = synchronous, dashed = asynchronous.
- Check arrow direction by reading the label aloud: "UI makes API requests to Backend."
- Color may encode new-vs-existing, ownership, or debt — but define it in the key and
  keep it readable color-blind and in black and white.

## Do Diagram the Technology

"It's an implementation detail" produces useless diagrams. Include technologies and
protocols; a rough up-front choice ("MySQL or PostgreSQL") is fine.

## Scope Discipline

- Keep deployment off container diagrams. Use **one deployment diagram per
  environment** and prioritize production — that's what an incident needs.
- Model other teams' systems as opaque boxes; reaching inside encodes coupling.

## Prefer a Model Over Drawings

Define elements and relationships once as a graph; render many views from it, version
controlled and diff-able. When components aren't physically evident in the code, the
diagram documents an intention, not a reality — say so.
