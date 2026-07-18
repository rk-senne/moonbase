# Moonbase How-To Guide

A practical walkthrough for getting started with moonbase and using it effectively.

---

## Table of Contents

- [Installation](#installation)
- [First-Time Setup](#first-time-setup)
- [Making a Project Agent-Ready](#making-a-project-agent-ready)
- [Deploying a Single Agent](#deploying-a-single-agent)
- [Running a Full Mission](#running-a-full-mission)
- [Using the TUI Dashboard](#using-the-tui-dashboard)
- [Configuring Backends](#configuring-backends)
- [Writing Custom Skills](#writing-custom-skills)
- [Writing Stored Prompts](#writing-stored-prompts)
- [Using the Flywheel](#using-the-flywheel)
- [Working with cmux](#working-with-cmux)
- [Pipeline Observability](#pipeline-observability)
- [Customizing Agents](#customizing-agents)
- [Troubleshooting](#troubleshooting)

---

## Installation

### From Source

```bash
git clone git@github.com:rk-senne/moonbase.git
cd moonbase
make build
cp bin/moonbase /usr/local/bin/
```

### From Release

Download the latest binary from the [Releases page](https://github.com/rk-senne/moonbase/releases) for your platform (darwin/linux, amd64/arm64).

### Verify Installation

```bash
moonbase version
moonbase status
```

---

## First-Time Setup

Run setup once to install agents globally:

```bash
moonbase setup
```

This copies the 14 agent definitions to `~/.moonbase/agents/` so they are available in any project.

Check everything is working:

```bash
moonbase status
moonbase list
```

---

## Making a Project Agent-Ready

Navigate to any project and run:

```bash
cd my-project
moonbase init
```

This creates:

```
.kiro/
  specs/           -- Feature specifications (requirements, design, tasks)
  steering/        -- Project-wide rules agents follow
  agents/          -- Project-specific agent overrides (optional)
  skills/          -- Domain knowledge for agents
  prompts/         -- Reusable named workflows
```

### Adding Steering Rules

Create `.kiro/steering/dev-rules.md` with your project conventions:

```markdown
# Development Rules

## Stack
- Python 3.12, FastAPI, SQLAlchemy
- pytest for tests
- ruff for linting

## Conventions
- All endpoints return JSON with a consistent envelope
- Errors use problem+json format (RFC 9457)
- Tests live next to source files as *_test.py
```

Agents will automatically discover and follow these rules.

### Adding Skills

Create `.kiro/skills/api-patterns/SKILL.md`:

```markdown
# API Patterns

## Pagination
All list endpoints support cursor-based pagination:
- Query params: ?cursor=<token>&limit=<int>
- Response includes: { items: [], next_cursor: "..." }

## Authentication
All endpoints require Bearer token in Authorization header.
Token validation happens in middleware (app/middleware/auth.py).
```

Skills give agents domain-specific knowledge about your project.

---

## Deploying a Single Agent

Deploy an agent with a specific task:

```bash
# Analyst -- break down requirements
moonbase deploy 1 "analyze the user authentication flow"

# Architect -- design a solution
moonbase deploy 2 "design pagination for the /users endpoint"

# Implementer -- write code
moonbase deploy 3 "implement rate limiting middleware"

# QA -- verify changes
moonbase deploy 4 "verify the pagination implementation"

# Reviewer -- final review
moonbase deploy 5 "review changes for PR readiness"
```

### Specialist Agents

```bash
# Security audit
moonbase deploy 274 "audit the auth token handling"

# DevOps
moonbase deploy 362 "set up the CI pipeline"

# Documentation
moonbase deploy 999 "write API docs for the new endpoints"

# Tech debt cleanup
moonbase deploy 86 "find and remove dead code in src/legacy/"

# Edge case testing
moonbase deploy 13 "find edge cases in the date parser"

# Migration support
moonbase deploy 9 "upgrade from React 18 to React 19"

# Legacy archaeology
moonbase deploy z "explain what src/old_processor.py does"
```

### Pipe Mode

Pipe context directly to agents:

```bash
git diff | moonbase deploy 4             # Pipe diff to QA
cat error.log | moonbase deploy 3        # Pipe logs to implementer
echo "fix the auth bug" | moonbase       # Pipe to full council
```

### Deploy to cmux

If you use cmux (macOS AI terminal), deploy agents into split panes:

```bash
moonbase deploy --cmux 3 "implement feature X"
moonbase deploy --cmux 4 "verify the changes"
```

Each agent gets its own pane with notification rings when done.

---

## Running a Full Mission

The full KND Council pipeline runs all phases automatically:

```bash
moonbase mission "add pagination to the /users API"
```

Pipeline flow:
```
Analyst -> Architect -> Implementer -> QA -> Reviewer
                                       |
                            Risk Gate:  |
                            LOW    -> proceed to Review
                            MEDIUM -> back to Implementer (max 2 reworks)
                            HIGH   -> back to Architect
                            CRITICAL -> STOP (human required)
```

### Fast Mode

Skip analysis/architecture for trivial or well-specified tasks:

```bash
moonbase mission --fast "fix typo in README"
```

### Trace Mode

See detailed timing and observability output:

```bash
moonbase mission --trace "add caching to the API"
```

This prints trace IDs, phase start/end timestamps, output sizes, and duration per phase.

### Dry Run

Preview what the pipeline would do without executing:

```bash
moonbase mission --dry-run "refactor the auth module"
```

---

## Using the TUI Dashboard

Launch the interactive dashboard:

```bash
moonbase
```

### Key Bindings

| Key | Action |
|-----|--------|
| Up/Down, j/k | Navigate roster |
| Enter | Open dossier / deploy |
| m | New mission |
| p | Project navigator |
| W | Document viewer |
| C | Open COMMS (chat) |
| M | Launch cmux/tmux |
| ? | Help manual |
| T | Cycle theme |
| q | Quit |

### Pipeline View

When a mission is running:
| Key | Action |
|-----|--------|
| n | Next phase |
| r | Retry phase |
| s | Skip phase |
| esc | Back (press twice to abort) |

---

## Configuring Backends

Moonbase supports multiple AI backends. Configure in `~/.config/moonbase/config.yaml` or via environment variables.

### Kiro CLI (recommended for interactive)

```bash
# Just install kiro-cli -- moonbase auto-detects it
which kiro-cli
```

### OpenAI

```bash
export OPENAI_API_KEY=sk-...
export OPENAI_MODEL=gpt-4o         # optional, default: gpt-4o
```

### Anthropic

```bash
export ANTHROPIC_API_KEY=sk-ant-...
```

### Kimi (Moonshot AI)

```bash
export MOONSHOT_API_KEY=sk-...
export KIMI_MODEL=kimi-k3           # optional, default: kimi-k3
# Models: kimi-k3 (1M ctx), kimi-k2.7-code, kimi-k2.6, kimi-k2.5
```

### Ollama (local)

```bash
ollama pull llama3.1
# moonbase auto-detects ollama
```

### Backend Priority

Moonbase selects the best available backend automatically:
kiro-cli > codex > openai > anthropic > kimi > ollama > clipboard

Override in config:

```yaml
# ~/.config/moonbase/config.yaml
default_backend: anthropic
pipeline_backend: openai    # Use different backend for pipeline analysis phases
```

---

## Writing Custom Skills

Skills are domain knowledge that agents incorporate into their prompts.

### Structure

```
.kiro/skills/
  docker-build/
    SKILL.md
  api-patterns/
    SKILL.md
  database/
    SKILL.md
```

### Example Skill

`.kiro/skills/error-handling/SKILL.md`:

```markdown
# Error Handling Patterns

## HTTP Errors
All API errors use RFC 9457 problem+json format:
{
  "type": "https://api.example.com/errors/not-found",
  "title": "Resource Not Found",
  "status": 404,
  "detail": "User with ID 123 does not exist"
}

## Internal Errors
- Wrap errors with context: fmt.Errorf("fetching user %d: %w", id, err)
- Never expose internal details to clients
- Log full error chain server-side with trace ID

## Retry Policy
- Retry on 429, 500, 502, 503, 504
- Exponential backoff: 1s, 2s, 4s (max 3 attempts)
- Circuit breaker after 5 consecutive failures
```

Agents automatically discover and incorporate skills when deployed in your project.

---

## Writing Stored Prompts

Stored prompts are reusable workflows. Create `.kiro/prompts/*.md` files.

### Example: Implementation Prompt

`.kiro/prompts/implement.md`:

```markdown
# Implement from Spec

Read the latest spec in .kiro/specs/ and implement all remaining tasks.
For each task:
1. Read the acceptance criteria
2. Implement the code
3. Write tests that verify the ACs
4. Run the test suite
5. Move to the next task

When all tasks are complete, run the full test suite and report results.
```

### Example: Diagnose Prompt

`.kiro/prompts/diagnose.md`:

```markdown
# Diagnose Bug

1. Read the bug report from the user
2. Reproduce the issue by reading relevant code
3. Write a failing test that demonstrates the bug
4. Fix the code to make the test pass
5. Run the full test suite to verify no regressions
6. Summarize: root cause, fix applied, prevention strategy
```

---

## Using the Flywheel

The flywheel learns from your pipeline execution history.

### How It Works

Every mission logs execution data:
- Which phases took longest
- Which agents got reworked
- Where risk gates triggered
- What failed and why

### View Insights

```bash
moonbase flywheel
```

Output shows:
- Total missions run
- Average duration per phase
- Rework rate (how often QA sends work back)
- Risk level distribution
- Phases that fail most often

### Improve Over Time

Use flywheel insights to:
1. Add steering rules for common mistakes
2. Add skills for domains agents struggle with
3. Adjust agent prompts for clarity
4. Tune pipeline config (timeouts, retries)

---

## Working with cmux

[cmux](https://github.com/manaflow-ai/cmux) is a macOS terminal built for AI coding agents.

### Setup

```bash
brew tap manaflow-ai/cmux
brew install --cask cmux
```

### Deploy Agents in Panes

```bash
moonbase deploy --cmux 3 "implement feature"
moonbase deploy --cmux 4 "verify changes"
```

Each agent runs in its own cmux split pane. You get notification rings when they finish.

### Pipeline Notifications

When running `moonbase mission`, cmux users automatically get:
- Notification when each phase completes
- Alert on CRITICAL risk (attention needed)
- Notification when the full mission finishes

### Enable in Config

```yaml
# ~/.config/moonbase/config.yaml
use_cmux: true
```

### TUI Integration

Press `M` in the moonbase TUI to open a cmux workspace (falls back to tmux if cmux is not installed).

---

## Pipeline Observability

### Trace IDs

Every pipeline run gets a unique trace ID for correlation:

```bash
moonbase mission --trace "your task"
# Prints: trace: 20260718T120000-a3b4c5d6
```

### Checkpoints

Pipeline state is saved to `~/.moonbase/checkpoints/` as JSON. If a mission crashes, the state is preserved for analysis.

### Mission History

```bash
moonbase history              # Show past missions
moonbase history --json       # Machine-readable output
moonbase replay <id>          # Replay a previous mission
moonbase export <id>          # Export mission details
```

### Config Tuning

```yaml
# ~/.config/moonbase/config.yaml
phase_timeout_seconds: 300    # 5 min per phase (increase for large tasks)
max_output_size: 100000       # 100KB output limit per phase
max_retries: 1                # Retry failed phases once before failing
enable_trace: true            # Generate trace IDs for all runs
```

---

## Customizing Agents

### Override for a Project

Copy an agent to `.kiro/agents/` and modify:

```bash
moonbase install --all    # Installs all agents to .kiro/agents/
# Edit .kiro/agents/numbuh-3.md to customize the implementer
```

Project agents override built-in agents with the same name.

### Agent Format

Each agent is a single `.md` file with YAML frontmatter:

```yaml
---
name: numbuh-3
designation: Kuki Sanban
role: Implementer
tools:
  - read
  - write
  - shell
  - grep
  - glob
  - code
shell:
  allowed_commands: ["go test ./...", "npm test", "make build"]
  read_only: false
guardrails:
  max_turns: 50
  max_output: 100000
handoff:
  format: structured
  required: ["next_agent", "reason", "evidence", "risk"]
hooks:
  on_activate:
    - command: 'git status --short'
      timeout_ms: 3000
  pre_tool_use:
    - command: '.kiro/hooks/check-secrets.sh'
      timeout_ms: 5000
  on_complete:
    - command: '.kiro/hooks/log-completion.sh'
      timeout_ms: 3000
pipeline_position: 3
---

# Numbuh 3 -- Implementer

## Identity
...

## Purpose
...
```

### Adding Guardrails

```yaml
guardrails:
  max_turns: 30          # Stop after 30 LLM turns
  max_output: 50000      # Max 50K chars output
  stop_words:            # Emergency stop triggers
    - "ABORT"
    - "SECURITY_VIOLATION"
  output_rules:          # Output must contain these
    - "## Handoff"       # Enforce handoff protocol
```

### Adding Hooks

```yaml
hooks:
  pre_tool_use:
    - command: '.kiro/hooks/check-secrets.sh'
      timeout_ms: 5000
  on_complete:
    - command: '.kiro/hooks/flywheel-log.sh'
      timeout_ms: 3000
```

---

## Troubleshooting

### "No backend available"

```bash
moonbase status    # Shows which backends are detected

# Fix: install at least one
# Option 1: kiro-cli
# Option 2: export OPENAI_API_KEY=sk-...
# Option 3: export ANTHROPIC_API_KEY=sk-ant-...
# Option 4: export MOONSHOT_API_KEY=sk-...
# Option 5: ollama pull llama3.1
```

### "Agent not found"

```bash
moonbase list      # Shows loaded agents
moonbase lint      # Validates agent files

# Fix: install agents
moonbase setup              # Global install
moonbase install --all      # Project-local install
```

### Pipeline gets stuck

- Press `esc` twice in TUI to abort
- Check `moonbase flywheel` for patterns
- Increase timeout: `phase_timeout_seconds: 600` in config
- Check `~/.moonbase/checkpoints/` for saved state

### Build fails after changes

```bash
go build ./...     # Must pass
go test ./...      # Must pass
moonbase lint      # Validates agent definitions
```

---

## Quick Reference

```bash
moonbase                          # Launch TUI
moonbase init                     # Make project agent-ready
moonbase deploy <n> "task"        # Deploy single agent
moonbase mission "task"           # Full pipeline
moonbase mission --fast "task"    # Skip analysis for simple tasks
moonbase mission --trace "task"   # With observability output
moonbase deploy --cmux <n> "task" # Deploy in cmux pane
moonbase flywheel                 # Show learning insights
moonbase status                   # Health check
moonbase list                     # Show all agents
moonbase lint                     # Validate agents
moonbase history                  # Past missions
moonbase config                   # Show config
```
