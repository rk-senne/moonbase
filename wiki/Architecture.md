# Architecture

Moonbase is a single Go binary (~15,300 LOC, 17 internal packages, 8 direct dependencies). Agents are embedded via `go:embed`, so it runs in any project with no checkout.

## Project structure

```
moonbase/
├── agents/              ← 14 agent .md files (source of truth)
├── doctrine/            ← 9 operating-doctrine documents
├── .kiro/
│   ├── specs/           ← feature specs (requirements, design, tasks)
│   ├── steering/        ← project-wide dev rules
│   └── agents/          ← installed agents for kiro-cli
├── internal/            ← Go source (17 packages)
│   ├── agents/          ← YAML frontmatter parser + registry
│   ├── pipeline/        ← orchestrator, risk gate, triggers, depth, fan-out, flywheel, pricing
│   ├── discovery/       ← project context (.kiro/specs, steering, skills registry, stack)
│   ├── compile/         ← Kiro-native JSON compiler
│   ├── backend/         ← AI backends (kiro-cli, openai, anthropic, kimi, ollama, clipboard)
│   ├── config/          ← YAML config (no secrets)
│   ├── tui/             ← Bubble Tea v2 TUI (aggregate-root App)
│   └── …                ← chat, history, watcher, docs, snippets, projects, platform, clipboard, logging, updater, templates
├── cmd/moonbase/        ← CLI entry point + subcommands
└── .github/workflows/   ← CI + release automation
```

## Stack

| Layer | Tool |
|-------|------|
| Language | Go 1.26 (single binary, cross-platform) |
| TUI | Bubble Tea v2 + Lip Gloss v2 + Bubbles v2 (`charm.land/…/v2`) |
| Agents | Markdown + YAML frontmatter |
| Backends | Kiro CLI (primary), OpenAI, Anthropic, Kimi, Ollama, clipboard |
| CI / Releases | GitHub Actions · goreleaser (4 platform binaries) |

## Design notes

- **MVU/Elm TUI**: the `App` is an aggregate root of ~13 fields, delegating to owned sub-models (Dashboard, Pipeline, System, Search, Comms, Terminal, Chrome, Boot, Infra, Backend, Views…). A `KeyMap` is the single source of truth for keybindings and generated help. Golden tests pin rendered output.
- **Pipeline as a state machine**: `internal/pipeline` owns phases, the risk gate, triggers, adaptive depth, and parallel fan-out. The TUI and CLI both drive it.
- **Backend-agnostic**: a small `Backend` interface with optional extension interfaces (`RawDeployer`, `UsageReporter`) — each provider opts in to capabilities.
- **Security by default**: SafeEnv env allowlisting, shell-command hook guard, input validation, `0600`/`0700` file perms, TLS 1.2 minimum on the updater.

See `docs/architecture.md` and `docs/design.md` in the repo for deeper detail.
