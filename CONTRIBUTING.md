# Contributing to Moonbase

Thanks for your interest in contributing to Moonbase! This guide covers setup, conventions, and workflow.

## Build & Test

```bash
make build              # compile binary to bin/moonbase
make test               # run all tests
make lint               # go vet + moonbase lint (validates agents)
make coverage           # generate coverage report
make run                # run from source
```

All three (`build`, `test`, `lint`) must pass before submitting a PR.

## Pre-commit Hooks

Enable the project's pre-commit hook to catch issues before pushing:

```bash
git config core.hooksPath .githooks
```

Or use the Makefile shortcut:

```bash
make hooks
```

This runs `go vet`, `go build`, and `go test` automatically on every commit.

## Adding a New Agent

1. Create a new `.md` file in `agents/` (e.g., `agents/numbuh-99.md`)
2. Add YAML frontmatter with required fields
3. Write the agent's system prompt as the markdown body

Minimal example:

```markdown
---
name: numbuh-99
designation: Your Name
role: Your Role
tools: [read, shell, grep, glob, code]
shell:
  allowed_commands: ["go test ./..."]
  read_only: true
routing:
  available: [numbuh-1, numbuh-5]
  trusted: [numbuh-5]
pipeline_position: null
triggers: null
---

# Numbuh 99 — Your Role

## Identity
Personality and style...

## Purpose
Core question this agent answers...

## Output Formats
How it structures responses...

## Operating Protocol
Rules it follows...
```

See [docs/agent-format.md](docs/agent-format.md) for the full specification including all frontmatter fields, trigger conditions, hooks, and routing.

Run `moonbase lint` to validate your agent file.

## Adding a New Backend

Backends live in `internal/backend/`. To add one:

1. Create a new file (e.g., `internal/backend/mybackend.go`)
2. Implement the `Backend` interface:

```go
type Backend interface {
    Name() string
    Available() bool
    Deploy(agent agents.Agent, context *discovery.ProjectContext, task string) (string, error)
}
```

3. Add your backend to the `DetectAll()` slice in `internal/backend/backend.go`
4. Write tests in `internal/backend/backend_test.go`

The `Available()` method should check if the tool is installed (e.g., `exec.LookPath`). The `Deploy()` method composes the agent prompt with project context and sends it to the AI tool.

## Testing TUI Changes

The TUI uses [Bubbletea](https://github.com/charmbracelet/bubbletea) (Elm architecture):

- **Model**: `internal/tui/app.go` — state + `Update()` + `View()`
- **Views**: `internal/tui/views.go` — rendering functions
- **Styles**: `internal/tui/styles.go` — Lipgloss styles

To test TUI changes:

```bash
make run                # launch the TUI and interact manually
make test               # run unit tests (TUI tests use tea.Model assertions)
```

TUI unit tests should exercise `Update()` with synthetic messages and assert on model state — not on rendered strings.

## PR Guidelines

1. Branch from `main`
2. Keep commits focused and well-described
3. `make build` must pass
4. `make test` must pass
5. `make lint` must pass
6. Add tests for new functionality
7. Update docs if adding user-facing features

## Code Conventions

From the project's development rules:

- **Error wrapping**: Always wrap with context — `fmt.Errorf("what failed: %w", err)`
- **Naming**: Go standard — `camelCase` for locals, `PascalCase` for exports
- **File size**: Max ~300 lines before splitting into a new file
- **No global state**: Pass dependencies through constructors
- **Interfaces**: Define where consumed, not where implemented
- **Tests**: `*_test.go` in same package, table-driven where applicable
- **One responsibility per file**

## Project Structure

```
cmd/moonbase/        CLI entry point + subcommands
internal/agents/     Agent loading, parsing, registry
internal/backend/    AI backend interface + implementations
internal/pipeline/   Pipeline state machine, risk gate
internal/discovery/  Project context discovery
internal/tui/        Bubbletea models, views, styles
internal/config/     App configuration
agents/              Agent .md files (source of truth)
doctrine/            Operating doctrine (reference only)
```
