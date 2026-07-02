# Requirements: Moonbase v3 — Production-Ready

## Overview

Moonbase v3 closes the gap between "builds and tests pass" and "useful daily tool." The focus is: real pipeline execution in the TUI, proper CLI deploy flow that actually works with kiro-cli, test coverage for untested packages, config modernisation, and removal of dead code.

This is the "production hardening" release.

---

## User Stories

### US-1: Real TUI Pipeline
As a developer, I want the TUI pipeline to actually invoke agents via the backend so that I can watch the council work in real-time, not just see fake messages.

### US-2: Working CLI Deploy
As a developer, I want `moonbase deploy 4` to actually start an interactive kiro-cli session with Numbuh 4's full prompt so that I get real AI assistance in my terminal.

### US-3: Config Modernisation
As a developer, I want moonbase config to use YAML (matching the agent format), support preferences (default backend, theme, agent order), and never store API keys in plaintext files.

### US-4: Test Confidence
As a developer, I want test coverage for the CLI commands, backend selection, and critical TUI logic so that future changes don't silently break features.

### US-5: Clean Codebase
As a developer contributing to moonbase, I want no dead code, no stale stubs, no packages that do nothing — so the codebase matches its documentation.

---

## Acceptance Criteria

### AC-1: TUI Real Pipeline Execution

#### AC-1.1: TUI Mission Invokes Backend
- **WHEN** user presses `enter` on the mission briefing in the TUI
- **THEN** the pipeline executes by invoking agents through the preferred backend
- **SHALL** show real agent output in the pipeline chat view (streaming if possible)

#### AC-1.2: TUI Pipeline Risk Gate
- **WHEN** Numbuh 4's output is received in the TUI pipeline
- **THEN** the risk gate is applied and the pipeline routes accordingly
- **SHALL** visually indicate rework (🔁), show risk level in sidebar, and allow user to override

#### AC-1.3: TUI Pipeline Abort
- **WHEN** user presses `esc` during active pipeline execution
- **THEN** the current backend invocation is cancelled and the pipeline stops
- **SHALL** show "Mission aborted by human" status

#### AC-1.4: TUI Conditional Trigger Display
- **WHEN** a conditional phase is evaluated
- **THEN** the TUI shows whether it triggered or was skipped, and why
- **SHALL** use the ⚡ icon for triggered and ⊘ for skipped with reason in the chat

---

### AC-2: CLI Deploy Flow

#### AC-2.1: Deploy Starts Interactive Session
- **WHEN** `moonbase deploy 4` is run and kiro-cli is available
- **THEN** an interactive kiro-cli session starts with Numbuh 4's composed prompt as system context
- **SHALL** hand full stdin/stdout to the user (not capture output)

#### AC-2.2: Deploy Without kiro-cli Copies to Clipboard
- **WHEN** `moonbase deploy 4` is run and kiro-cli is NOT available
- **THEN** the composed prompt is copied to clipboard
- **SHALL** show the prompt length, agent name, and project context summary

#### AC-2.3: Deploy Accepts Task Argument
- **WHEN** `moonbase deploy 4 "check the auth flow"`
- **THEN** the task is appended to the composed prompt
- **SHALL** include the task in both kiro-cli invocation and clipboard copy

---

### AC-3: Config Modernisation

#### AC-3.1: YAML Config File
- **WHEN** moonbase loads configuration
- **THEN** it reads from `~/.config/moonbase/config.yaml`
- **SHALL** fall back to defaults if file doesn't exist, support old JSON path with migration warning

#### AC-3.2: Config Schema
- **WHEN** the config file is loaded
- **THEN** it supports: `default_backend`, `theme`, `agent_order`, `agents_dir`
- **SHALL NOT** store API keys (those come from environment variables)

#### AC-3.3: Config CLI
- **WHEN** `moonbase config` is run
- **THEN** it shows current effective config (merged defaults + file + env)
- **SHALL** mask any detected environment variable values

---

### AC-4: Test Coverage

#### AC-4.1: CLI Command Tests
- **WHEN** `go test ./cmd/moonbase/...` runs
- **THEN** tests cover: install (creates dir, copies files), deploy (resolves agent, composes prompt), mission (creates pipeline)
- **SHALL** use temp directories and mock backends, not real AI calls

#### AC-4.2: Backend Tests
- **WHEN** `go test ./internal/backend/...` runs
- **THEN** tests cover: auto-detection logic, preferred selection, compose integration, error handling
- **SHALL** mock exec.LookPath for backend availability

#### AC-4.3: Config Tests
- **WHEN** `go test ./internal/config/...` runs
- **THEN** tests cover: default values, YAML load/save, missing file handling, migration from JSON

---

### AC-5: Clean Codebase

#### AC-5.1: Remove Dead Packages
- **WHEN** the codebase is inspected
- **THEN** no package exists that is unused or contains only stubs with no real functionality
- **SHALL** remove or consolidate: `internal/docs/`, `internal/history/`, `internal/platform/`, `internal/projects/` if they're stubs; keep if TUI uses them meaningfully

#### AC-5.2: No Stale References
- **WHEN** grep for `json.Unmarshal|json.Marshal|"encoding/json"` across non-config code
- **THEN** no remnants of old JSON agent loading exist
- **SHALL** JSON usage is limited to: snippets storage, legacy config migration, and external API responses

#### AC-5.3: Doctrine Consistency
- **WHEN** all doctrine files are read
- **THEN** none reference old format (config.json, Profile.md, vault/, mission-system/)
- **SHALL** each doctrine file either adds value beyond what's embedded in agents, or is removed

---

## Scope

### In Scope
- TUI pipeline wired to real backend execution
- CLI deploy with proper kiro-cli invocation (interactive mode)
- Config package rewrite (JSON → YAML, no secrets)
- Test files for cmd/, backend/, config/
- Dead code removal
- Doctrine cleanup

### Out of Scope
- New agents or agent content changes
- New TUI views or features beyond pipeline
- Multi-project support
- Agent marketplace / sharing
- Real-time streaming from AI backends (capture output after completion is fine)
- OpenAI/Anthropic direct API implementation (remain as stubs with clear error)

---

## Dependencies
- Existing: `gopkg.in/yaml.v3` (already in go.mod)
- Existing: `github.com/fsnotify/fsnotify` (already in go.mod for watcher)
- No new external dependencies required

---

## Risks

| Risk | Mitigation |
|------|-----------|
| kiro-cli flag interface may change | Use documented stable flags; graceful fallback |
| TUI pipeline execution blocks the event loop | Run backend invocation in goroutine with tea.Cmd |
| Config migration breaks existing users | Detect and auto-migrate JSON → YAML with warning |
| Removing "dead" packages breaks TUI imports | Verify TUI imports before removing |

---

## Rollback Note
- Config migration: keep JSON reader as fallback for one version
- TUI changes: keep simulated pipeline as fallback mode when no backend available
- Dead code removal: git history preserves everything
