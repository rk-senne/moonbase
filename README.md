# 🌙 Moonbase

**K.N.D. Tactical Operations Terminal**

A 14-agent AI development pipeline with spec-driven methodology, human interaction protocols, and project-aware execution. Built for Kiro CLI, compatible with any AI backend.

```
moonbase              # launch command center
moonbase deploy 4     # deploy Numbuh 4 (QA)
moonbase mission      # run the full KND Council pipeline
```

---

## What It Does

- **14 AI operatives** with distinct roles, personalities, and scoped permissions
- **Spec-driven development** — agents discover and work with `.kiro/specs/` and steering rules
- **Human interaction protocol** — agents ask focused questions when uncertain instead of guessing
- **Evidence-based pipeline** — every claim requires proof, every handoff includes context
- **Risk-gated flow** — QA classifies risk, work loops back until it holds

---

## The Pipeline

```
Human Request
    ↓
Numbuh 1  → Requirements (ACs, scope, risks)
    ↓
Numbuh 2  → Design (blueprint, trade-offs, file impact)
    ↓
Numbuh 3  → Implementation (code, tests, build)
    ↓
Numbuh 4  → QA (verify, risk gate: LOW/MEDIUM/HIGH/CRITICAL)
    ↓                    ↑
    ├── MEDIUM → fix ────┘
    ├── HIGH → redesign (back to Numbuh 2)
    └── LOW ↓
Numbuh 5  → Review (final gate, PR package)
    ↓
Human Approval
```

**Specialists** deploy conditionally:
- **Numbuh 0** — Architecture oversight (>5 files, core logic, boundaries)
- **Numbuh 9** — Migration (upgrades, breaking changes, compatibility)
- **Numbuh 13** — Chaos testing (edge cases, weird inputs, fuzz)
- **Numbuh 86** — Decommissioning (dead code, unused deps, tech debt)
- **Numbuh 274** — Security audit (auth, injection, secrets, CVEs)
- **Numbuh 362** — DevOps (CI/CD, Docker, deployment, health checks)
- **Numbuh 999** — Documentation (READMEs, ADRs, changelogs, guides)
- **Sector Z** — Legacy archaeology (git history, forgotten code, the WHY)

---

## Structure

```
moonbase/
├── agents/                  ← 14 self-contained .md agent files
│   ├── numbuh-0.md         (System Architect)
│   ├── numbuh-1.md         (Analyst)
│   ├── numbuh-2.md         (Architect)
│   ├── numbuh-3.md         (Implementer)
│   ├── numbuh-4.md         (QA)
│   ├── numbuh-5.md         (Reviewer)
│   ├── numbuh-9.md         (Migration)
│   ├── numbuh-13.md        (Chaos Tester)
│   ├── numbuh-86.md        (Decommissioning)
│   ├── numbuh-274.md       (Security)
│   ├── numbuh-362.md       (DevOps)
│   ├── numbuh-999.md       (Documentation)
│   ├── sector-z.md         (Legacy Archaeology)
│   └── knd-council.md      (Full Pipeline)
├── doctrine/                ← Operating principles & protocols
│   ├── HumanInteractionDoctrine.md
│   ├── ProjectDiscoveryProtocol.md
│   ├── SpecDrivenDevelopmentDoctrine.md
│   ├── MoonbaseEngineeringDoctrine.md
│   ├── ArchitectureDoctrine.md
│   ├── GoDoctrine.md
│   └── SuperTesterDoctrine.md
├── docs/                    ← Project documentation
│   ├── agent-format.md     (agent file format spec)
│   └── design.md           (TUI architecture)
├── internal/                ← Go TUI source code
├── cmd/moonbase/            ← CLI entry point
├── go.mod / go.sum
├── Makefile
└── README.md
```

---

## Agent Format

Each agent is a single `.md` file with:

1. **YAML frontmatter** — machine-readable metadata (tools, permissions, routing, hooks)
2. **Markdown body** — full identity, output formats, behaviour rules, verification checklist
3. **Operating Protocol** — universal section (evidence standard, human interaction, spec awareness, handoff protocol)

```yaml
---
name: numbuh-4
designation: Wallabee Beatles
role: QA / Verification
tools: [read, shell, grep, glob, code, knowledge, subagent]
shell:
  allowed_commands: ["mvn test", "npm test", "go test ./...", ...]
  read_only: true
routing:
  available: [numbuh-2, numbuh-3, numbuh-5, numbuh-274, numbuh-362, numbuh-0]
pipeline_position: 4
shortcut: ctrl+shift+4
---

# Numbuh 4 — QA / Verification
...
```

No JSON configs. No external references. One file = one complete agent.

---

## Key Capabilities

### Human Interaction Protocol

Agents use a 4-level uncertainty model:
- **CERTAIN** — proceed silently
- **LIKELY** — proceed, label assumption
- **UNCERTAIN** — ask the human (focused question with options and default)
- **UNKNOWN** — stop, ask, do not guess

### Spec-Driven Awareness

Agents automatically look for project specs:
```
.kiro/specs/*/requirements.md    → numbered acceptance criteria
.kiro/specs/*/design.md          → architecture decisions
.kiro/specs/*/tasks.md           → implementation steps
.kiro/steering/*.md              → project-wide rules
```

When specs exist, agents reference AC-IDs, follow designs, and update task status.
When no spec exists, agents work from code patterns and suggest specs for non-trivial work.

### Evidence Standard

Every agent must prove what they claim:
- Numbuh 1: requirements cite user request or label assumptions
- Numbuh 3: implementation lists changed files and tests run
- Numbuh 4: every finding includes reproduction evidence
- Numbuh 274: every vulnerability needs attack vector + remediation
- Sector Z: every legacy claim needs git evidence

---

## Stack

| Layer | Tool | Why |
|-------|------|-----|
| Language | Go 1.22+ | Single binary, fast, cross-platform |
| TUI | Bubbletea + Lipgloss | Elm architecture, terminal styling |
| Agents | Markdown + YAML frontmatter | Portable, readable, versionable |
| Pipeline | Kiro CLI / any AI backend | Tool-agnostic orchestration |

---

## Build

```bash
make run        # go run cmd/moonbase/main.go
make build      # go build -o bin/moonbase
make test       # go test ./...
```

---

## Philosophy

1. **Pure markdown agents** — one file per operative, no assembly required
2. **Spec before code** — understand before building, spec the non-trivial
3. **Ask don't guess** — focused questions beat wrong assumptions
4. **Evidence over claims** — prove it or label it an assumption
5. **Tool-agnostic** — works with Kiro, Codex, OpenAI, Anthropic, Ollama
6. **Single binary** — no runtime deps, no install complexity

---

*"Kids Next Door... battle stations."*
