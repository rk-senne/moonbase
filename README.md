# 🌙 Moonbase

**K.N.D. Tactical Operations Terminal**

A 14-agent AI development pipeline with spec-driven methodology, human interaction protocols, and project-aware execution. Built for Kiro CLI, compatible with any AI backend.

```bash
moonbase init                    # make any project agent-ready
moonbase deploy 4 "check auth"  # deploy Numbuh 4 with a task
moonbase mission "add pagination to /users API"  # run full pipeline
```

---

## What It Does

- **14 AI operatives** with distinct roles, personalities, and scoped permissions
- **Spec-driven development** — agents discover `.kiro/specs/` and steering rules automatically
- **Human interaction protocol** — agents ask focused questions when uncertain, proceed when confident
- **Risk-gated pipeline** — QA classifies risk, work loops back until it holds
- **Project-aware** — discovers your stack, conventions, and specs before doing work
- **Self-auditing** — agents can scan their own codebase for bugs and tech debt

---

## Quick Start

```bash
# Build from source
make build

# Initialize a project for agent-driven development
cd my-project
moonbase init

# Deploy an agent
moonbase deploy 1 "analyze the user authentication flow"

# Run the full KND Council pipeline
moonbase mission "add rate limiting to the API"

# Check environment
moonbase status
```

---

## CLI Commands

| Command | Description |
|---------|-------------|
| `moonbase` | Launch the TUI dashboard |
| `moonbase init` | Scaffold `.kiro/` in any project (specs, steering, agents) |
| `moonbase deploy <n> [task]` | Deploy operative by numbuh (interactive kiro-cli session) |
| `moonbase mission <task>` | Run full KND Council pipeline on a task |
| `moonbase install [--all]` | Install agents to project's `.kiro/agents/` |
| `moonbase status` | Show environment health check |
| `moonbase lint` | Validate all agent `.md` files |
| `moonbase config` | Show current YAML configuration |
| `moonbase list` | Show operative roster |
| `moonbase help` | Operations manual |

**Pipe mode:**
```bash
echo "fix the auth bug" | moonbase              # pipe to KND Council
echo "check security" | moonbase deploy 274     # pipe to specific agent
```

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
Numbuh 4  → QA (verify, risk gate)
    ↓                    ↑
    ├── MEDIUM → fix ────┘ (max 2 rework loops)
    ├── HIGH → redesign (back to Numbuh 2)
    ├── CRITICAL → STOP (escalate to human)
    └── LOW ↓
Numbuh 5  → Review (final gate, PR package)
    ↓
Human Approval
```

**Conditional specialists** deploy based on content signals:

| Operative | Triggers When |
|-----------|--------------|
| Numbuh 0 | >5 files changed, core logic modified, new patterns |
| Numbuh 274 | Auth/secrets touched, new endpoints, dependency CVEs |
| Numbuh 362 | CI/CD, Docker, env vars, deployment config changed |
| Numbuh 9 | Version upgrades, breaking changes, migrations |
| Numbuh 13 | Edge case coverage needed, fragile flows, parsers |
| Numbuh 86 | Dead code discovered, unused deps, zombie features |
| Numbuh 999 | README needed, ADRs, changelogs |
| Sector Z | Old/mysterious code touched, legacy context needed |

---

## Agent Format

Each operative is a single self-contained `.md` file:

```yaml
---
name: numbuh-4
designation: Wallabee Beatles
role: QA / Verification
tools: [read, shell, grep, glob, code, knowledge, subagent]
shell:
  allowed_commands: ["go test ./...", "npm test", "mvn test"]
  read_only: true
routing:
  available: [numbuh-2, numbuh-3, numbuh-5, numbuh-274, numbuh-362, numbuh-0]
  trusted: [numbuh-3, numbuh-5]
pipeline_position: 4
triggers: null
---

# Numbuh 4 — QA / Verification

## Identity
Australian. Blunt. Brave. Evidence-driven...

## Purpose
Core question: "Does it hold when I hit it?"

## Output Formats
...

## Operating Protocol
(Evidence standard, human interaction, spec awareness, handoff protocol)
```

One file = one complete agent. Copy to any project's `.kiro/agents/` and it works.

---

## Project Structure

```
moonbase/
├── agents/              ← 14 agent .md files (source of truth)
├── doctrine/            ← 8 operating doctrine documents
├── .kiro/
│   ├── specs/           ← Feature specs (requirements, design, tasks)
│   ├── steering/        ← Project-wide dev rules
│   └── agents/          ← Installed agents for kiro-cli
├── internal/            ← Go source (15 packages)
│   ├── agents/          ← YAML frontmatter parser + registry
│   ├── pipeline/        ← Orchestrator, risk gate, triggers
│   ├── discovery/       ← Project context (.kiro/specs, stack detection)
│   ├── backend/         ← AI backend integrations (kiro-cli, clipboard)
│   ├── clipboard/       ← Cross-platform clipboard (macOS/Linux/Windows)
│   ├── config/          ← YAML config (no secrets)
│   ├── tui/             ← Bubbletea TUI (dashboard, pipeline, dossier)
│   └── ...              ← chat, history, watcher, docs, snippets
├── cmd/moonbase/        ← CLI entry point + subcommands
├── .github/workflows/   ← CI + release automation
├── .goreleaser.yml      ← Binary release config
├── Makefile
└── go.mod
```

---

## Key Capabilities

### Human Interaction Protocol

```
CERTAIN  → proceed silently
LIKELY   → proceed, label assumption
UNCERTAIN → ask the human (focused question + options + default)
UNKNOWN  → stop, ask, do not guess
```

### Project Discovery

When deployed, agents automatically find:
- `.kiro/specs/` — requirements, design, tasks (references AC-IDs)
- `.kiro/steering/` — project conventions and rules
- Build configs — detects Go/Java/Node/Python/Rust
- README — project overview

### Security

- **SafeEnv** — child processes get allowlisted env vars only (no secret leakage)
- **Hook guard** — blocks dangerous commands (curl, rm, python, eval, shell pipes)
- **Input validation** — agent IDs validated against `[a-zA-Z0-9-]`
- **File permissions** — all user data written with `0600`/`0700`
- **No secrets in config** — API keys come from environment variables only

---

## Build & Install

```bash
# Development
make run              # go run ./cmd/moonbase
make build            # go build -o bin/moonbase
make test             # go test ./...

# Install locally
cp bin/moonbase /usr/local/bin/

# Release (produces cross-platform binaries)
git tag v1.0.1 && git push --tags
# GitHub Actions builds: darwin/linux × amd64/arm64
```

---

## Quality

| Metric | Value |
|--------|-------|
| Tests | 144 |
| Packages tested | 15/15 (100%) |
| Go LOC | 9,266 |
| Direct dependencies | 6 |
| CI | GitHub Actions (vet + build + test on every push) |
| Releases | goreleaser (4 platform binaries on tag) |
| Agent lint | `moonbase lint` validates all 14 agents |

---

## Stack

| Layer | Tool | Why |
|-------|------|-----|
| Language | Go 1.24 | Single binary, fast, cross-platform |
| TUI | Bubbletea + Lipgloss | Elm architecture, terminal styling |
| Agents | Markdown + YAML frontmatter | Portable, readable, versionable |
| Backend | Kiro CLI (primary) | Tool execution, file access, multi-turn |
| Clipboard | pbcopy / xclip / xsel / wl-copy / clip | Cross-platform fallback |
| CI | GitHub Actions | Automated quality gates |
| Releases | goreleaser | Cross-platform binaries |

---

## Philosophy

1. **Pure markdown agents** — one file per operative, no assembly required
2. **Spec before code** — understand before building, spec the non-trivial
3. **Ask don't guess** — focused questions beat wrong assumptions
4. **Evidence over claims** — prove it or label it an assumption
5. **Self-auditing** — agents can scan their own system
6. **Tool-agnostic** — works with Kiro, Codex, OpenAI, Anthropic, Ollama
7. **Single binary** — no runtime deps, no install complexity
8. **Security by default** — env isolation, input validation, permission hardening

---

*"Kids Next Door... battle stations."*
