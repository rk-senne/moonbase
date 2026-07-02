# Requirements: Moonbase v2

## Overview

Moonbase v2 aligns the Go TUI and CLI with the new pure-markdown agent format. The Go code currently reads `.json` configs — it needs to read `.md` files with YAML frontmatter instead. Beyond that, moonbase needs to become a practical tool for daily development: deploying agents into any project, orchestrating the pipeline, and providing useful project awareness.

## User Stories

### US-1: Agent Deployment
As a developer, I want to deploy any moonbase agent into my current project so that I get specialised AI assistance without manually assembling prompts.

### US-2: Pipeline Execution
As a developer, I want to run the KND Council pipeline on a task so that requirements, design, implementation, QA, and review happen in sequence with risk-gating.

### US-3: Project Awareness
As a developer, I want agents to automatically discover my project's specs, steering rules, and conventions so that they produce contextually correct work without me repeating myself.

### US-4: Agent Installation
As a developer, I want to install moonbase agents into a project's `.kiro/agents/` directory so that Kiro CLI can use them natively.

---

## Acceptance Criteria

### AC-1.1: YAML Frontmatter Parsing
- **WHEN** the agent registry loads from the agents/ directory
- **THEN** it reads `.md` files and parses YAML frontmatter for metadata
- **SHALL** extract: name, designation, role, description, tools, auto_tools, shell, write, routing, hooks, pipeline_position, shortcut, triggers

### AC-1.2: Full Agent Body as Prompt
- **WHEN** an agent is deployed to a backend
- **THEN** the full markdown body (everything after frontmatter) is used as the system prompt
- **SHALL** include everything from `# {Name}` through the Operating Protocol section

### AC-1.3: Backward Compatibility
- **WHEN** `.json` agent files exist alongside `.md` files
- **THEN** the loader reads `.md` files only (`.json` is deprecated)
- **SHALL** log a warning if only `.json` files are found

### AC-2.1: Pipeline Phase Execution
- **WHEN** the KND Council pipeline runs
- **THEN** each phase deploys the correct agent with the correct prompt and accumulated context
- **SHALL** pass the output of each phase as input to the next

### AC-2.2: Risk Gate Logic
- **WHEN** Numbuh 4 (QA) reports a risk level
- **THEN** the pipeline routes accordingly: LOW → Numbuh 5, MEDIUM → back to Numbuh 3, HIGH → back to Numbuh 2, CRITICAL → stop
- **SHALL** enforce max 2 rework loops before escalating to human

### AC-2.3: Conditional Phase Triggers
- **WHEN** a conditional specialist has trigger conditions defined in frontmatter
- **THEN** the pipeline evaluates whether to invoke them based on the mission context
- **SHALL** announce why a conditional phase was triggered or skipped

### AC-3.1: Project Context Discovery
- **WHEN** an agent is deployed in a project directory
- **THEN** it discovers `.kiro/specs/`, `.kiro/steering/`, build configs, and README
- **SHALL** inject discovered context into the agent's prompt as a preamble

### AC-3.2: Steering Rules Injection
- **WHEN** `.kiro/steering/*.md` files exist in the project
- **THEN** their content is prepended to the agent prompt as project rules
- **SHALL** respect `inclusion: manual` frontmatter (skip unless explicitly requested)

### AC-4.1: Install Command
- **WHEN** `moonbase install` is run in a project directory
- **THEN** it copies/symlinks agent `.md` files into `.kiro/agents/`
- **SHALL** only install agents the user selects (or all with `--all`)

### AC-4.2: Agent Selection
- **WHEN** `moonbase install` runs without `--all`
- **THEN** it prompts interactively for which agents to install
- **SHALL** show agent name, role, and one-line description for selection

---

## Scope

### In Scope
- YAML frontmatter parser for `.md` agent files
- Updated Agent struct matching new frontmatter schema
- Pipeline execution with risk-gate routing
- Project context discovery and injection
- `moonbase install` command
- Backend integration (kiro-cli as primary)
- TUI updates to reflect new agent metadata

### Out of Scope
- Multi-repo pipeline (single project for now)
- Cloud/hosted agent deployment
- Agent marketplace or sharing
- Plugin system for custom agents
- Real-time collaboration between agents (sequential pipeline only)

## Dependencies
- YAML parsing library for Go (e.g., `gopkg.in/yaml.v3`)
- Existing Bubbletea TUI framework
- Kiro CLI installed for primary backend

## Risks
- YAML frontmatter parsing edge cases (multi-line strings, special chars)
- Pipeline context accumulation may exceed model context window
- Steering rules with `inclusion: manual` need frontmatter detection in non-agent `.md` files

## Rollback Note
- Agent `.md` files are the source of truth; Go code is the consumer. Rolling back Go changes doesn't affect agents.
- The old `.json` files have been removed; no backward path needed.
