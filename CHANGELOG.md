# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.5.0] - 2026-07-15

### Added
- `moonbase setup` command — installs agents globally to `~/.moonbase/agents/` so moonbase works from any project directory without per-project agent installation
- `moonbase guide` command — per-agent usage guides with when-to-use, examples, and tools reference (aliases: `man`, `howto`)
  - `moonbase guide` — general operations overview
  - `moonbase guide <n>` — detailed guide for specific agent
  - `moonbase guide --all` — full guide for all 14 agents
- `make install` now installs agents globally alongside the binary

### Changed
- Agent directory resolution reordered: `~/.moonbase/agents/` is now checked before CWD, making global install the primary mechanism for portable usage
- Improved error messages when agents directory is not found — now suggests `moonbase setup`
- Removed `setup` alias from `init` command (reassigned to the new global setup command)

## [1.2.0] - 2026-07-03

### Added
- Cobra CLI framework for structured subcommand routing
- Glamour-based markdown rendering for docs and agent prompts
- Structured logging with `log/slog` (file-based, no stdout noise)
- Graceful shutdown with context cancellation for pipeline execution
- Comprehensive usage guide (`docs/guide.md`)
- Agent hot-reload — TUI detects changes in agents directory automatically
- Pre-commit hooks (`.githooks/pre-commit`)
- CONTRIBUTING.md with full developer guide

### Fixed
- P0: Pipeline context propagation for abort handling
- P1: TUI state cleanup on view transitions
- P2: File watcher goroutine leak on restart
- P3: Minor rendering glitches in pipeline chat view

### Changed
- CLI migrated from manual flag parsing to Cobra
- Backend selection now uses structured logging for observability
- Pipeline phases use context-aware execution for cancellation support

## [1.1.0] - 2026-06-15

### Added
- Full codebase sweep — 244 tests across all packages
- Dead code removal and security hardening
- CI-compatible tests (Ubuntu without clipboard/desktop)
- Comprehensive README with CLI reference, quality metrics, security section

### Changed
- Test suite expanded from initial coverage to 244 tests
- Security: input validation, SafeEnv, hook guard hardened

### Fixed
- Tests now pass on headless CI environments (no clipboard/display required)

## [1.0.0] - 2026-06-01

### Added
- Initial release — 14-agent AI development pipeline
- Moonbase TUI dashboard with Bubbletea (Elm architecture)
- Agent format: Markdown + YAML frontmatter (portable, versionable)
- Pipeline state machine with risk gate (LOW/MEDIUM/HIGH/CRITICAL)
- 5 core pipeline agents (Numbuh 1–5)
- 8 conditional specialist agents (Numbuh 0, 9, 13, 86, 274, 362, 999, Sector Z)
- KND Council meta-agent for full pipeline orchestration
- Project discovery (`.kiro/specs/`, steering rules, stack detection)
- Backend integrations: kiro-cli, codex, openai, anthropic, ollama, clipboard
- CLI commands: deploy, mission, init, install, status, lint, list, config, help
- File watcher for live monitoring
- Embedded terminal and file browser in TUI
- Multi-agent COMMS with streaming, relay, and snippet support
- Cross-platform clipboard support (macOS/Linux/Windows)
- GitHub Actions CI (vet + build + test on every push)
- goreleaser for cross-platform binary releases (darwin/linux × amd64/arm64)
- Security: SafeEnv isolation, hook command validation, input sanitization
- Mission history with persistence

[1.2.0]: https://github.com/f5508037/moonbase/compare/v1.1.0...v1.2.0
[1.1.0]: https://github.com/f5508037/moonbase/compare/v1.0.0...v1.1.0
[1.0.0]: https://github.com/f5508037/moonbase/releases/tag/v1.0.0
