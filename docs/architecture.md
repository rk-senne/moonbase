# Moonbase Architecture

## 1. Foundations

Moonbase's architecture draws from six intellectual foundations:

- **Clean Architecture** (Robert C. Martin) — The dependency rule governs package structure. Inner layers (domain entities, use cases) have zero knowledge of outer layers (frameworks, drivers). Dependencies always point inward.
- **Clean Code** (Robert C. Martin) — Small functions with single responsibilities, intention-revealing names, DRY principle. Each file targets one responsibility, max ~300 lines before splitting.
- **The Clean Coder** (Robert C. Martin) — Professional discipline: comprehensive test coverage (1,218 tests across 16 packages), estimation through evidence, TDD where practical.
- **The C4 Model** (Simon Brown) — Architecture visualization at four zoom levels. This document uses C4's Context, Container, and Component diagrams to communicate structure without ambiguity.
- **OpenAI Practical Guide to Building Agents** — Guardrails as first-class concerns, orchestration patterns for multi-agent pipelines, structured handoffs, flywheel logging for continuous improvement.
- **Kiro CLI Patterns** — Specs as requirements contracts, steering rules as project conventions, skills as reusable knowledge, hooks for lifecycle events, flywheel for self-improvement loops.

---

## 2. C4 Context Diagram (Level 1)

The system boundary and its external actors.

```mermaid
C4Context
    title Moonbase - System Context

    Person(dev, "Developer", "Uses CLI to run agent pipelines, deploy operatives, review output")

    System(moonbase, "Moonbase", "14-agent AI development pipeline. Single Go binary with TUI dashboard.")

    System_Ext(ai_backends, "AI Backends", "kiro-cli, OpenAI, Anthropic, Kimi, Ollama")
    System_Ext(project_ctx, "Project Context", ".kiro/ directory: specs, steering, skills, prompts")
    System_Ext(cmux, "cmux", "Optional terminal multiplexer integration for interactive sessions")
    System_Ext(github, "GitHub", "Code hosting, CI/CD, release automation")

    Rel(dev, moonbase, "Deploys agents, runs missions, reviews output")
    Rel(moonbase, ai_backends, "Sends composed prompts, receives agent output")
    Rel(moonbase, project_ctx, "Discovers specs, steering rules, skills, stack info")
    Rel(moonbase, cmux, "Launches interactive terminal sessions")
    Rel(dev, github, "Pushes code, triggers CI, creates PRs")
```

---

## 3. C4 Container Diagram (Level 2)

Internal containers within the single moonbase binary.

```mermaid
C4Container
    title Moonbase - Container Diagram

    Person(dev, "Developer")

    System_Boundary(moonbase, "Moonbase Binary") {
        Container(cli, "CLI Layer", "Cobra", "Subcommand routing: deploy, mission, init, status, guide, history")
        Container(tui, "TUI Dashboard", "Bubbletea + Lipgloss", "Elm-architecture terminal UI with views for dashboard, pipeline, dossier, comms")
        Container(pipeline, "Pipeline Orchestrator", "Go", "Phase execution state machine, risk gate, rework loops, flywheel logging")
        Container(backends, "Backend Adapters", "Go", "7 backends: kiro-cli, openai, anthropic, kimi, ollama, clipboard, cmux")
        Container(registry, "Agent Registry", "Go", "YAML frontmatter parser, multi-directory merge (built-in → user → project)")
        Container(discovery, "Discovery Engine", "Go", "Scans .kiro/specs, steering, skills; detects stack; assembles ProjectContext")
        Container(composer, "Prompt Composer", "Go", "Cache-aware prompt composition with trust boundaries and size limits")
        Container(config, "Config Manager", "Go + YAML", "User preferences from ~/.moonbase/config.yml, no secrets stored")
    }

    Rel(dev, cli, "moonbase deploy/mission/init")
    Rel(dev, tui, "moonbase (no args) launches dashboard")
    Rel(cli, pipeline, "Starts pipeline execution")
    Rel(tui, pipeline, "Visualizes and controls pipeline")
    Rel(pipeline, backends, "Deploys agents to AI")
    Rel(pipeline, registry, "Loads agent definitions")
    Rel(backends, composer, "Gets composed prompt")
    Rel(composer, discovery, "Fetches project context")
    Rel(pipeline, config, "Reads backend preference, trust settings")
```

---

## 4. C4 Component Diagram (Level 3) — Pipeline Package

Zooming into `internal/pipeline/`, the core orchestration engine.

```mermaid
C4Component
    title Pipeline Package - Components

    Container_Boundary(pipeline_pkg, "internal/pipeline") {
        Component(pipeline, "Pipeline", "State machine managing phase execution order, rework loops, and fast-mode skipping")
        Component(context, "PipelineContext", "Accumulates phase outputs, files changed, decisions, risk level, and git diff")
        Component(riskgate, "RiskGate", "Parses QA verdict (LOW/MEDIUM/HIGH/CRITICAL), determines routing target phase")
        Component(triggers, "TriggerEvaluator", "Evaluates conditional specialist activation based on keyword and file-count signals")
        Component(checkpoint, "Checkpoint", "Serializes pipeline state to JSON for crash recovery and replay")
        Component(flywheel, "FlywheelLog", "Append-only JSONL session logger for self-improvement analysis")
        Component(meta, "PhaseMeta", "Parses __moonbase_meta JSON blocks from agent output for structured routing")
    }

    Rel(pipeline, context, "Records phase outputs, reads accumulated state")
    Rel(pipeline, riskgate, "Routes after QA based on risk verdict")
    Rel(pipeline, triggers, "Checks whether conditional specialists should activate")
    Rel(pipeline, checkpoint, "Saves/loads state for recovery")
    Rel(pipeline, flywheel, "Logs every phase execution")
    Rel(pipeline, meta, "Extracts structured metadata from agent output")
    Rel(riskgate, context, "Reads QA output to parse risk level")
    Rel(triggers, context, "Reads all phase outputs for signal detection")
```

---

## 5. Dependency Rule

Clean Architecture's dependency rule: source code dependencies point inward. Outer layers know about inner layers, never the reverse.

```
┌─────────────────────────────────────────────────────────┐
│  Frameworks & Drivers (outermost)                       │
│  cmd/moonbase (Cobra), internal/tui (Bubbletea),        │
│  internal/backend (HTTP clients, exec calls)            │
├─────────────────────────────────────────────────────────┤
│  Interface Adapters                                     │
│  internal/backend.Backend (interface),                  │
│  internal/discovery (project scanning),                 │
│  internal/discovery.ComposePrompt (prompt assembly)     │
├─────────────────────────────────────────────────────────┤
│  Use Cases                                              │
│  internal/pipeline (orchestration, risk routing,        │
│  trigger evaluation, checkpoint, flywheel)              │
├─────────────────────────────────────────────────────────┤
│  Entities (innermost)                                   │
│  internal/agents.Agent, internal/pipeline.Phase,        │
│  internal/pipeline.RiskLevel, internal/pipeline.Pipeline│
│  internal/discovery.ProjectContext                      │
└─────────────────────────────────────────────────────────┘
         ↑ Dependencies point inward (never outward)
```

**Layer mapping:**

| Layer | Packages | Characteristics |
|-------|----------|----------------|
| Entities | `agents.Agent`, `pipeline.Phase`, `pipeline.RiskLevel`, `discovery.ProjectContext` | Pure data structures, no framework imports, no I/O |
| Use Cases | `pipeline.Pipeline`, `pipeline.RiskGate`, `pipeline.TriggerEvaluator` | Business logic, depends only on entities and interfaces |
| Interface Adapters | `backend.Backend` interface, `discovery.Discover()`, `discovery.ComposePrompt()` | Adapts external systems to use-case needs |
| Frameworks & Drivers | `cmd/moonbase` (Cobra), `tui` (Bubbletea), `backend.Kiro` (exec), `backend.OpenAI` (HTTP) | External framework code, highest churn, lowest architectural significance |

**Key constraint:** The pipeline package never imports `cmd/`, `tui/`, or concrete backend implementations. It works through the `Backend` interface, which is defined in the `backend` package and injected at construction time.

---

## 6. Key Design Decisions

| Decision | Rationale | Trade-offs |
|----------|-----------|------------|
| **Single Go binary** | Zero runtime dependencies. `go build` produces one file that runs anywhere. No Python envs, no Node modules, no Docker required. | Larger binary size (~23MB). No hot-reload for agent changes (must rebuild for Go logic). |
| **Markdown agents** | Agents are `.md` files with YAML frontmatter — portable, human-readable, diffable, versionable. One file = one complete agent. | No compiled validation at write time. Must parse and validate at load time. |
| **Pipeline over single-agent** | Separation of concerns (analysis ≠ implementation ≠ QA). Risk gates between phases catch errors before they compound. Rework loops are bounded (max 2). | Higher latency for trivial tasks (mitigated by `--fast` mode). More complex orchestration code. |
| **Multiple backends** | Tool-agnostic philosophy. Works with kiro-cli, OpenAI, Anthropic, Kimi, Ollama, or clipboard fallback. Users aren't locked into one vendor. | Must maintain 7 adapter implementations. Lowest-common-denominator feature set across backends. |
| **SafeEnv for child processes** | Security boundary: child processes (AI backends spawned via exec) receive only allowlisted environment variables. Prevents secret leakage through `os.Environ()`. | Must explicitly allowlist new env vars when adding features. May break backends that expect undocumented env vars. |
| **Flywheel logging** | Self-improvement over time. Append-only JSONL captures every phase execution. `moonbase flywheel` surfaces patterns: which agents get reworked, which phases are slow, where risk gates fire. | Disk usage grows unbounded (mitigated: small entries, ~200 bytes each). Privacy consideration for task descriptions logged to disk. |
| **Cache-aware prompt composition** | Prompt ordering optimized for LLM prefix caching (Claude, GPT). Static content first (steering, agent prompt), dynamic content last (task). Higher cache hit rate = lower cost + latency. | Rigid composition order. Adding new context sections requires considering cache impact. |
| **Multi-directory agent merge** | Three-tier priority: project `.kiro/agents/` > user `~/.moonbase/agents/` > built-in `agents/`. Projects can override any agent. | Name collisions resolved silently by priority. Can be confusing when a project override shadows a built-in. |
