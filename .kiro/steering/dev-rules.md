# Moonbase Development Rules

## Stack
- Go 1.26.5+
- Bubbletea (TUI framework, Elm architecture)
- Lipgloss (terminal styling)
- Bubbles (pre-built components: spinners, tables, viewports, inputs)
- gopkg.in/yaml.v3 (YAML parsing)

## Build & Test
```bash
go build ./...          # must pass before any PR
go test ./...           # must pass before any PR
make run                # go run cmd/moonbase/main.go
make build              # go build -o bin/moonbase cmd/moonbase/main.go
```

## Package Structure
- `cmd/moonbase/` — CLI entry point, subcommand routing
- `internal/agents/` — agent loading, parsing, registry
- `internal/backend/` — AI backend interface + implementations
- `internal/pipeline/` — pipeline state machine, risk gate, context
- `internal/discovery/` — project context discovery (.kiro/specs, steering, stack detection)
- `internal/tui/` — Bubbletea models, views, components, styles
- `internal/config/` — app configuration
- `agents/` — the actual agent .md files (source of truth)
- `doctrine/` — reference documents (not loaded by Go code)

## Conventions
- Errors: wrap with context using `fmt.Errorf("what failed: %w", err)`
- Naming: Go standard (camelCase locals, PascalCase exports)
- Tests: `*_test.go` in same package, table-driven where applicable
- No global state — pass dependencies through constructors
- Interfaces: define where consumed, not where implemented
- Files: one responsibility per file, max ~300 lines before splitting

## Agent Format Rules
- Agents are `.md` files with YAML frontmatter
- Frontmatter = machine-readable metadata
- Body = the agent's system prompt (sent to AI backend)
- Never modify agent content in Go code — pass through as-is
- The `agents/` directory is the single source of truth for agent definitions

## Testing
- Unit tests for all non-TUI logic
- Integration tests tagged with `//go:build integration`
- Mock backends for pipeline tests (don't call real AI in unit tests)
- Test real `.md` files from `agents/` directory (not synthetic test fixtures)
