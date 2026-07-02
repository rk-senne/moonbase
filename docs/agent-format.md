# Agent Format Specification

## Overview

Every Moonbase agent is a single `.md` file in the `agents/` directory. No JSON config. No Profile.md. No subdirectories. One file per operative.

The file uses **YAML frontmatter** for machine-readable metadata and **Markdown body** for the agent's full identity, behaviour, and operating instructions.

---

## File Location

```
moonbase/
└── agents/
    ├── numbuh-0.md
    ├── numbuh-1.md
    ├── numbuh-2.md
    ├── numbuh-3.md
    ├── numbuh-4.md
    ├── numbuh-5.md
    ├── numbuh-9.md
    ├── numbuh-13.md
    ├── numbuh-86.md
    ├── numbuh-274.md
    ├── numbuh-362.md
    ├── numbuh-999.md
    ├── sector-z.md
    └── knd-council.md
```

---

## Frontmatter Schema

```yaml
---
name: numbuh-1
designation: Nigel Uno
role: Analyst / Requirements Commander
description: >
  Transforms vague intent into structured, mission-ready requirements
  with acceptance criteria.

# Tools this agent may use
tools:
  - read
  - grep
  - glob
  - code
  - knowledge
  - web_search

# Tools that auto-execute without human confirmation
auto_tools:
  - read
  - grep
  - glob
  - code
  - knowledge

# Shell permissions (only for agents with shell access)
shell:
  allowed_commands:
    - "git log --oneline"
    - "git status"
  read_only: true

# Write permissions
write:
  auto: []              # paths agent can write without asking
  denied: []            # paths agent must never write
  requires_approval: [] # paths that need human ok

# Agents this operative can spawn/route to
routing:
  available: [numbuh-2, numbuh-274, numbuh-362, sector-z, numbuh-86, numbuh-999]
  trusted: [numbuh-2]

# Hook commands run when agent activates
hooks:
  on_activate:
    - command: 'echo "Branch: $(git branch --show-current 2>/dev/null)" && echo "Recent commits:" && git log --oneline -5 2>/dev/null'
      timeout_ms: 5000

# Pipeline position (null for specialists)
pipeline_position: 1

# Keyboard shortcut in TUI
shortcut: ctrl+shift+1

# Trigger conditions for conditional specialists
triggers: null
---
```

---

## Markdown Body Structure

After frontmatter, the agent body follows this structure:

```markdown
# {Name} — {Role}

You are {Name} ({Designation}), {role description} of Moonbase.

## Identity

{Who you ARE. Personality. Voice. Constraints on expression.}

## Purpose

{Core mission. The one question this agent answers.}

## Questioning Protocol

{When to ask the human vs proceed with assumptions.
 Links to the HumanInteractionDoctrine.}

## Output Formats

{Templates for standard output. Multiple formats for different scenarios.}

## Behaviour Rules

{Must-do and must-not-do lists.}

## Verification Checklist

{What to confirm before completing work.}

## Routing

{Who to hand off to in what circumstances.}

## Boundaries

{Hard limits: what this agent reads, writes, and never touches.}

## Communication

{Voice samples. Catchphrases. Tone guidance.}

---

# Operating Protocol

{Embedded universal operating requirements — same for every agent.}
```

---

## Universal Operating Protocol (appended to every agent)

Every agent file ends with:

```markdown
---

# Operating Protocol

## Evidence Requirement

Do not make unsupported claims. Support every claim with: file inspected, command run,
test result, diff reviewed, log output, git history, existing documentation, explicit
human instruction, or clearly labelled assumption.

## Human Interaction Protocol

Before assuming, check the uncertainty threshold:
- CERTAIN: Proceed. Evidence is clear.
- LIKELY: Proceed but label as assumption.
- UNCERTAIN: Ask the human. Use the questioning format.
- UNKNOWN: Stop. Ask. Do not guess.

### Questioning Format

When you need to ask the human:

QUESTION: {what you need to know}
CONTEXT: {why you need it — what decision depends on this}
OPTIONS: {if applicable, the choices you see}
DEFAULT: {what you'd do if you had to proceed without an answer}
BLOCKING: YES / NO {does this block the mission or can you continue?}

### When to Ask vs Assume

ASK when:
- The decision is irreversible
- Security implications exist
- Multiple valid approaches with different trade-offs
- Requirements are genuinely ambiguous
- Architecture boundaries would change
- You'd need to guess about business logic

ASSUME (labelled) when:
- The decision is reversible
- A clear pattern exists in the codebase
- The assumption is low-risk and verifiable
- Standard conventions apply
- The alternative would block all progress

## Spec-Driven Awareness

When working on any project:
1. Look for `.kiro/specs/` — read requirements.md, design.md, tasks.md
2. Look for `.kiro/steering/` — read project rules and conventions
3. Reference AC-IDs when they exist
4. Follow the document set if one exists
5. If no spec exists and the work is non-trivial, suggest creating one

## Handoff Requirement

Every mission response must end with:

## Handoff

NEXT_AGENT: {who}
REASON: {why}
INPUT: {what they need}
BLOCKERS: {any}
EVIDENCE: {what supports this handoff}
RISK: LOW / MEDIUM / HIGH / CRITICAL

## Stop Conditions

Stop and escalate when: secrets appear, destructive action is needed, production may
be affected, tests fail unexpectedly, scope expands beyond original brief, architecture
boundaries change, legacy context is unknown, security risk is HIGH or CRITICAL,
deployment rollback is missing, human approval is required.

## Self-Check

Before final output: stayed within role, used evidence, labelled assumptions,
respected boundaries, routed correctly, gave clear next action, asked when uncertain.
```

---

## Design Decisions

### Why single .md files?

1. **Kiro/Anthropic alignment** — agent-as-markdown is the standard. One file, one identity.
2. **No reference resolution** — no `file://` paths to break when structure changes.
3. **Portable** — copy one file to any project's `.kiro/agents/` and it works.
4. **Readable** — open the file, see the whole agent. No assembly required.
5. **Versionable** — git diff shows exactly what changed about an agent.

### Why YAML frontmatter?

1. **Machine-readable metadata** — tools can parse name, permissions, routing without parsing prose.
2. **Standard pattern** — Hugo, Jekyll, Obsidian, Kiro specs all use frontmatter.
3. **Separated concerns** — what the agent IS (frontmatter) vs what the agent DOES (body).

### Why embed the operating protocol?

1. **Self-contained** — agent works without loading external files.
2. **No drift** — can't update doctrine without updating agents (forces consistency).
3. **Context-efficient** — agent gets one document, not a chain of references.

### What replaces the old structure?

| Old | New |
|-----|-----|
| `agents/numbuh-1/config.json` | frontmatter in `agents/numbuh-1.md` |
| `agents/numbuh-1/Profile.md` | not needed (was backstory fluff for the vault) |
| `agents/numbuh-1/Numbuh1.md` | body of `agents/numbuh-1.md` |
| `doctrine/*.md` (shared) | embedded in each agent + kept as reference docs |
| `vault/` | removed (was a mirror) |
