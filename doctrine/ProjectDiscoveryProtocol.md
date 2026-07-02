# Project Discovery Protocol

How Moonbase agents discover, orient to, and work within any project's specification — regardless of project type, language, or framework.

> **The core idea:** Before touching code, orient to the project. A spec-aware agent produces better work in one pass than a spec-blind agent does in three.

---

## Step 1 — Discover the Project Context

When activated in a project directory, immediately look for:

```
Priority 1 (Spec System):
├── .kiro/specs/*/requirements.md     ← What we're building (numbered ACs)
├── .kiro/specs/*/design.md           ← How it's architected
├── .kiro/specs/*/tasks.md            ← Step-by-step implementation plan
└── .kiro/steering/*.md               ← Project-wide rules and conventions

Priority 2 (Standard Docs):
├── docs/requirements.md
├── docs/design.md
├── docs/decisions-adr.md
├── docs/roadmap.md
├── docs/agent-brief.md
└── ARCHITECTURE.md / DESIGN.md

Priority 3 (Build/Config Context):
├── package.json / pom.xml / go.mod / Cargo.toml  ← Stack detection
├── tsconfig.json / vitest.config.*               ← Tooling
├── .eslintrc* / .prettierrc*                     ← Code style
├── Dockerfile / docker-compose.*                 ← Deployment
└── Makefile / justfile                           ← Build commands

Priority 4 (Conventions from Code):
├── Existing patterns in src/                     ← How things are done
├── Test patterns in tests/ or __tests__/         ← How things are tested
├── README.md                                     ← Project overview
└── CONTRIBUTING.md                               ← Process conventions
```

---

## Step 2 — Orient (the 30-second check)

Before doing any work, answer these internally:

1. **What stack is this?** (language, framework, build tool, test runner)
2. **Does a spec exist?** (requirements, design, tasks)
3. **Are there steering rules?** (coding conventions, naming, patterns)
4. **What's the current state?** (what's built, what's in progress, what's planned)
5. **Are there AC-IDs I should reference?** (AC-1.1, AC-2.3, etc.)

If a spec exists → follow it. Reference AC-IDs in your work.
If no spec exists → work from code patterns and state it.
If work is non-trivial and no spec exists → suggest creating one (but don't block on it).

---

## Step 3 — Work Spec-Aware

### For Numbuh 1 (Analyst)

When producing requirements:
- Check if `.kiro/specs/` or `docs/requirements.md` already exists
- Extend existing specs rather than creating parallel ones
- Use the AC-ID format: `AC-{section}.{number}` (e.g., AC-1.1, AC-2.3)
- Reference existing ACs when the work relates to them
- Suggest creating a spec folder if the project doesn't have one

### For Numbuh 2 (Architect)

When producing designs:
- Read existing `design.md` and `decisions-adr.md` first
- Reference steering rules for technology choices
- Align with existing architecture rather than proposing alternatives without reason
- If making a significant decision, format it as an ADR
- Reference which ACs the design satisfies

### For Numbuh 3 (Implementer)

When writing code:
- Read `tasks.md` for the current step's scope
- Read `requirements.md` for the ACs you're implementing
- Read `steering/*.md` for coding conventions
- Reference AC-IDs in code comments for significant logic: `// Implements AC-1.3`
- Follow patterns from existing code over personal preference
- Check task status and update it when done

### For Numbuh 4 (QA)

When verifying:
- Check each AC by ID: pass or fail
- Report format should reference ACs: `AC-1.1: PASS (evidence: ...)`
- Check that tests actually validate the ACs they claim to cover
- Flag ACs that have no corresponding test

### For Numbuh 5 (Reviewer)

When reviewing:
- Verify AC coverage: every AC in scope has evidence of completion
- Check that design decisions match what was implemented
- Verify steering rules are followed
- Include AC pass/fail summary in the review

---

## Step 4 — When No Spec Exists

Many projects won't have formal specs. That's fine. The agents still work. They just operate from code-as-truth instead of spec-as-truth.

**Behaviour when spec-less:**
- Read existing code patterns as the implicit spec
- Match naming, structure, error handling, and test patterns
- Label your understanding of the "implicit requirements" as assumptions
- Suggest spec creation only when:
  - The work is large (>5 files, new feature, architectural change)
  - Ambiguity would cause waste
  - Multiple interpretations exist

**What to say:**
> "No formal spec found. Working from existing code patterns. ASSUMPTION: [what I'm inferring]. If this is wrong, let me know."

---

## Step 5 — Creating Specs for New Work

When agents or humans decide a spec is needed, create this minimal set:

```
.kiro/specs/{feature-name}/
├── requirements.md    ← ACs in WHEN/THEN/SHALL format
├── design.md          ← Architecture, files affected, data flow
└── tasks.md           ← Step-by-step implementation tasks
```

Optionally add:
- `.kiro/steering/{rule}.md` — for project-wide rules that aren't feature-specific

### Minimal requirements.md

```markdown
# Requirements: {Feature Name}

## User Story
As a {role}, I want to {action}, so that {benefit}.

## Acceptance Criteria

### AC-1.1: {Name}
- WHEN {trigger}
- THEN {expected behaviour}
- SHALL {constraint}

## Scope
### In Scope
### Out of Scope

## Risks
```

### Minimal design.md

```markdown
# Design: {Feature Name}

## Approach
{What and why}

## Files Affected
| File | Change | Purpose |
|------|--------|---------|

## Data Flow
{Step 1 → Step 2 → Step 3}

## Edge Cases
| Scenario | Handling |
```

### Minimal tasks.md

```markdown
# Tasks: {Feature Name}

## Task 1: {Description}
- Requirements: AC-1.1
- Files: `path/to/file`
- Action: {What to do}
- Test: {How to verify}
- Status: pending
```

---

## Integration with the 10 Principles

| Principle | How agents apply it |
|-----------|-------------------|
| Spec before code | Read/create spec before implementing |
| Interfaces first | Check if abstractions exist; don't skip them |
| Ship increments | Work on one task/AC at a time, verify, move on |
| Cheap-now first | Flag cheap-to-establish seams early |
| Decisions recorded | Write ADRs for significant choices |
| Single source of truth | Don't duplicate specs; extend existing ones |
| Test ships with code | Every implementation includes its tests |
| Requirements traceable | Reference AC-IDs in code, tests, and reports |
| Boundary you control | Don't spec external systems; interface them |
| Governance built-in | Security/compliance checked during the pipeline, not after |

---

## Quick Reference Card

```
BEFORE WORK:
  1. Discover: look for .kiro/specs/ and .kiro/steering/
  2. Orient: stack, spec status, conventions, current state
  3. Reference: use AC-IDs if they exist

DURING WORK:
  4. Follow: steering rules > existing patterns > personal preference
  5. Reference: cite ACs in code comments and reports
  6. Record: significant decisions → ADR format

AFTER WORK:
  7. Verify: pass/fail per AC-ID
  8. Update: task status in tasks.md if it exists
  9. Handoff: include spec references in handoff notes
```

---

*The spec serves the project, not the other way around. Lightweight for small work. Rigorous for large work. Always honest about what exists and what doesn't.*
