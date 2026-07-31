# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- feat(pipeline): adaptive pipeline depth — auto-classifies task complexity (trivial/simple/complex) and selects minimum viable pipeline depth
- feat(pipeline): mid-pipeline escalation — promotes shallow pipelines to deeper depth when QA flags insufficient analysis (MEDIUM/HIGH risk)
- feat(cli): `moonbase mission --full` flag to force all phases regardless of task complexity
- feat(cli): `moonbase mission --depth trivial|simple|complex` flag to override auto-classification
- feat(cli): `moonbase mission --dry-run` now shows depth classification and which phases are active/skipped
- feat(flywheel): `depth`, `depth_reason`, `escalated_from`, `escalated_to` fields on FlywheelEntry for depth observability

## [1.15.0] - 2026-07-31

### Added
- feat(pipeline): parallel fan-out execution of independent conditional specialists — triggered specialists now run concurrently after QA (RiskLow), with bounded concurrency (default 4, configurable 1–16)
- feat(pipeline): `RunSpecialists` orchestrator with `sync.WaitGroup` + semaphore for partial-failure tolerance (one failure does not cancel siblings)
- feat(pipeline): `IsIndependentSpecialist` classifies parallelizable agents from metadata (shell.read_only, tools list) — not hardcoded
- feat(pipeline): deterministic output aggregation via phase-number sorting, regardless of completion order
- feat(pipeline): `MergeSpecialistResults` on `PipelineContext` for stable merging with file de-duplication
- feat(config): `parallel_specialists` (default true) and `max_specialist_concurrency` (default 4) config fields
- feat(cli): `moonbase mission --sequential` flag to disable fan-out for a single mission
- feat(flywheel): `parallel_group` field on `FlywheelEntry` for correlating fan-out batch entries
- feat(checkpoint): `specialist_results` optional field on `Checkpoint` for persisting fan-out outcomes
- feat(tui): `FanOutCompleteMsg` Elm message for atomic batch updates; fan-out group header in pipeline sidebar

## [1.14.0] - 2026-07-31

### Added
- feat(discovery): progressive skill loading — `SkillRegistry` indexes skills by metadata at startup, loads full content on demand via `LoadContent(name)`
- feat(discovery): `@skill(name)` protocol — agents see a lightweight skill catalog and request full content on demand, saving context window tokens
- feat(discovery): `parseFrontmatterOnly` reads ≤1KB per skill file for YAML frontmatter extraction (name + description)
- feat(discovery): `EmitKiroSkillResources` writes Kiro-native `skill://` directory structure (`<name>/SKILL.md`)
- feat(discovery): backward-compatible legacy skill path — skills without YAML frontmatter are still loaded eagerly into `ProjectContext.Skills`
- feat(init): `moonbase init` now scaffolds example skill with YAML frontmatter in Kiro-native directory format

## [1.13.0] - 2026-07-31

### Added
- feat(flywheel): token consumption and estimated cost per pipeline phase — OpenAI, Anthropic, and Kimi backends report prompt/completion tokens; flywheel persists per-entry token counts and estimated USD cost
- feat(flywheel): `moonbase flywheel` shows Token & Cost Summary, per-agent cost breakdown, per-mission averages, and cost-heavy phase detection
- feat(pipeline): per-mission token budget with configurable warn threshold and hard cap (`token_budget` config section)
- feat(config): `model_pricing` config section for overriding default per-model token prices
- feat(backend): `UsageReporter` and `RawUsageReporter` optional interfaces for backends that report token usage

## [1.12.0] - 2026-07-30

### Added
- feat(agents): `mcp_servers` frontmatter field for MCP server declarations
- feat(compile): `moonbase compile` emits Kiro-native agent JSON (`--out`, `--validate`, `--agent`)
- feat(deploy): `--native` flag for Kiro-native agent deployment via `kiro-cli chat --agent`
- feat(config): `compile`, `deploy`, and `safety` config sections for native interop
- feat(status): Native Interop section showing compiled/stale counts and deploy mode
- feat(list): MCP server count per agent in roster output
- feat(lint): validates `mcp_servers` entries (name/command present, no duplicates)
- docs(migration): `docs/MIGRATION-NATIVE.md` — safety delegation table, opt-in steps, rollback

## [1.11.0] - 2026-07-30

### Changed
- deps/tui: migrated the terminal UI to the **Charm v2 stack** — `charm.land/bubbletea/v2` (v2.0.8), `charm.land/lipgloss/v2` (v2.0.5), `charm.land/bubbles/v2` (v2.1.1). No user-facing behavior change: the app now uses the declarative `View() tea.View` model, `tea.KeyPressMsg` keyboard events, and typed mouse messages (`MouseWheelMsg`/`MouseClickMsg`). A side benefit of lipgloss v2's border-width accounting is that dashboard panels now fit the target width exactly (v1 overflowed by a few columns). Golden snapshots were regenerated once; visible text is byte-for-byte identical.

## [1.10.0] - 2026-07-30

### Added
- tui: **click-to-select operatives** — left-clicking an operative in the dashboard roster now selects it and opens its dossier (the mouse equivalent of arrowing to it and pressing enter). Implemented with a drift-proof sidebar hit-map that shares the roster group-walk with the renderer, so click targets stay accurate as the roster changes. Completes the mouse-support follow-up noted in v1.9.0.

## [1.9.0] - 2026-07-30

### Added
- tui: **mouse support** — the mouse wheel now scrolls the active view (the COMMS transcript and document viewports, and the operative-roster / project-navigator cursors), and a click dismisses the boot splash. Enabled via cell-motion mode. Fully additive: every action still has a keyboard equivalent, so nothing changes for keyboard-only use. (Click-to-select an operative from the roster is a planned follow-up — it needs a render hit-map.)

## [1.8.0] - 2026-07-30

### Changed
- tui: the dashboard **THREAT LEVEL** gauge is now a real composite reflection of the working tree instead of a raw line count. It scores staged **and** unstaged change volume, breadth (files touched), untracked files, and security/infra-sensitive paths (auth, secrets, `.env`, Dockerfiles, CI workflows, terraform, k8s, migrations, payments…), shows a one-line reason, and forces at least **HIGH** whenever a sensitive file is in flight — mirroring the pipeline's Numbuh 274 trigger. Works in any git repository moonbase is launched in, not just moonbase itself.

## [1.7.0] - 2026-07-30

### Added
- MIT `LICENSE` — moonbase is now MIT-licensed (Copyright (c) 2026 Senne), so anyone can use, modify, and distribute it once the repository is public. Referenced from the README.
- `install.sh` — one-line installer (`curl -fsSL https://raw.githubusercontent.com/rk-senne/moonbase/main/install.sh | sh`) that detects your OS/arch, downloads the latest release, verifies its checksum, installs the binary to `~/.local/bin`, and runs `moonbase setup`. README now documents installation via the script, `go install`, release download, or source. (Requires the repository/releases to be publicly accessible.)
- Subsequence fuzzy agent search in the TUI — typing `n4` now matches `numbuh-4`, `arch` matches the architect, etc.; results are ranked by match quality (stdlib-only).
- Mission WIP lock — `moonbase mission` refuses to start a second concurrent mission (file lock at `~/.moonbase/mission.lock` with a liveness check and stale/corrupt-lock takeover); a `--force` flag overrides a running mission.
- `moonbase flywheel` now reports average mission lead time and the longest phase.

### Changed
- Graceful shutdown — `Ctrl+C`/`SIGTERM` during `moonbase mission` cancels the in-flight backend call, marks the phase interrupted, writes a flywheel entry, saves a checkpoint, and prints a `moonbase replay <id>` hint (no orphaned state).
- The mission pipeline is now backend-orthogonal — it runs with any configured backend via `backend.Preferred()` (kiro-cli, OpenAI, Anthropic, Ollama, or clipboard fallback), not only kiro-cli.
- All 14 agents gained a role-tailored `Reasoning Discipline` section (ReAct + Reflexion, complexity-scaled) and an inter-agent handoff protocol (shared context as distributed state), drawn from the research library.
- Risk gate now prefers the structured `__moonbase_meta` block over regex heuristics when present, falling back to regex only for non-conforming output.
- The self-updater derives the GitHub owner/repo from the module path baked into the binary (`debug.ReadBuildInfo`) instead of a hardcoded owner, and `.goreleaser.yml` auto-detects the repository from the git `origin` remote. Forks are followed automatically.
- O(1) agent-registry lookup (`GetByName` now uses a `byName` index instead of a linear scan).

### Fixed
- Retry budget is now bounded by the phase timeout — retries can no longer exceed a phase's deadline.
- Persisted data integrity: schema version (`v`) added to flywheel/checkpoint/history records (safe format evolution); checkpoint writes are now atomic (temp-then-rename); flywheel appends `fsync` for crash durability; backend SSE responses are bounded (10 MB) to prevent unbounded memory growth.

### Security
- deps: clear all `govulncheck` findings — bump `golang.org/x/text` v0.30.0→v0.39.0 (GO-2026-5970, infinite loop on invalid input), `github.com/yuin/goldmark` v1.7.13→v1.7.17 (GO-2026-5320, XSS in rendered markdown), and the Go toolchain to 1.26.5 (GO-2026-5856, crypto/tls Encrypted Client Hello privacy leak). `govulncheck ./...` now reports 0 vulnerabilities affecting the code.
- The self-updater's HTTP client now enforces a TLS 1.2 minimum (it downloads executable code).

## [1.6.0] - 2026-07-29

### Added
- Agents are now embedded into the `moonbase` binary (`go:embed`). `moonbase setup`, `moonbase install`, and `moonbase init` work from any directory without a repository checkout — when no on-disk agent source is found (or it would be the same directory as the target), agents are installed from the copy baked into the binary. This also lets `setup` self-heal a damaged `~/.moonbase/agents/`.
- TUI contextual footer — each view now shows a short, per-view key hint bar generated from the central key map, so available actions are discoverable without opening the full manual (`?`).
- `moonbase init --data-access` — opt-in flag that generates a stack-aware `data-access-performance.md` steering standard for projects with a data layer that warrants it: bound every query, push filtering/sorting/aggregation to the data layer, no N+1, no lazy loading across boundaries (ORM guidance adapts to Java/JPA, Django/SQLAlchemy, Prisma/TypeORM, Go, Rust), with a complexity budget that still allows bounded in-memory work and O(N log N) where justified. Not part of the default steering set.
- `moonbase init` now manages a `.gitignore` block for moonbase artifacts — excludes `.kiro/agents/` (installed, re-installable via `moonbase install`) and `.kiro/steering/data-access-performance.md` (opt-in, kept local to the projects that need it). Creates `.gitignore` if absent; appends only missing patterns idempotently (tolerates common pattern variants so it never duplicates).

### Changed
- refactor(tui): centralised all key bindings into a single `KeyMap` (`bubbles/key`) — key handling and both the full help manual and the new contextual footer are now generated from one source of truth, eliminating help/keybinding drift. Spec: `.kiro/specs/tui-refactor/`.
- refactor(tui): theme system is now immutable values (`Theme`/`Styles` + registry) carried by the model instead of mutated package-level globals — race-safe, `NO_COLOR`-aware, and extensible by registering a theme rather than editing a switch. Same four themes, same colours.
- refactor(tui): began decomposing the `App` god struct (57→47 fields) into focused value-type sub-models (`TerminalModel`, `DashboardModel`, `PipelineModel`); further extraction of remaining field groups deferred to a future phase.

### Fixed
- fix(install): `moonbase setup`/`install` no longer destroys agents when run outside the repo. `findAgentsSource()` falls back to `~/.moonbase/agents`, which is also the setup target; copying that directory onto itself truncated every agent to 0 bytes (`copyFile` opens the destination with `O_TRUNC`). Both `copyFile` and `installAgentsTo` now guard against source==target and skip safely.
- fix(history): history file path is now resolved lazily from `HOME` on each call instead of being cached at package `init()`, so tests can isolate history I/O. Behaviour is identical in production.
- fix(test): history tests no longer read or write the real `~/.config/moonbase/history.json` — they isolate to a temp `HOME` via `t.Setenv`. Previously they polluted the user's real history file (which had accumulated ~488 test-only entries) and could not be isolated because the path was fixed at init.
- fix(test): `captureStdout` test helper drained its `os.Pipe` only after the captured function returned, deadlocking on output larger than the OS pipe buffer (~64KB); it now drains concurrently. Surfaced by the history-command test as the accumulated history file crossed the threshold.

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

[1.15.0]: https://github.com/rk-senne/moonbase/compare/v1.14.0...v1.15.0
[1.14.0]: https://github.com/rk-senne/moonbase/compare/v1.13.0...v1.14.0
[1.13.0]: https://github.com/rk-senne/moonbase/compare/v1.12.0...v1.13.0
[1.12.0]: https://github.com/rk-senne/moonbase/compare/v1.11.0...v1.12.0
[1.11.0]: https://github.com/rk-senne/moonbase/compare/v1.10.0...v1.11.0
[1.10.0]: https://github.com/rk-senne/moonbase/compare/v1.9.0...v1.10.0
[1.9.0]: https://github.com/rk-senne/moonbase/compare/v1.8.0...v1.9.0
[1.8.0]: https://github.com/rk-senne/moonbase/compare/v1.7.0...v1.8.0
[1.7.0]: https://github.com/rk-senne/moonbase/compare/v1.6.0...v1.7.0
[1.6.0]: https://github.com/rk-senne/moonbase/compare/v1.5.0...v1.6.0
[1.2.0]: https://github.com/rk-senne/moonbase/compare/v1.1.0...v1.2.0
[1.1.0]: https://github.com/rk-senne/moonbase/compare/v1.0.0...v1.1.0
[1.0.0]: https://github.com/rk-senne/moonbase/releases/tag/v1.0.0
