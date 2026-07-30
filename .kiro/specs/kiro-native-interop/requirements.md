# Requirements — Kiro Native Interop

## Overview

Two coupled improvements that make moonbase operatives first-class Kiro citizens:

| Improvement | What |
|---|---|
| **A — MCP Server Support** | Add an `mcp_servers` frontmatter field so operatives can declare Model Context Protocol servers; compile to Kiro's `mcpServers` schema |
| **B — Native Agent JSON Emission** | Translate moonbase's `.md` operatives into valid Kiro agent JSON, so they are deployed via `kiro-cli chat --agent <name>`, inherit Kiro's native permission/hook/MCP engine, and appear in `kiro-cli agent list` |

The `.md` format remains the **single source of truth**. The JSON is a compiled artifact — never hand-edited, always regenerated from the markdown.

## Goals

- G-1: Operatives gain MCP server access without leaving the moonbase format.
- G-2: `moonbase compile` (or `moonbase install --native`) emits valid Kiro agent JSON that passes `kiro-cli agent validate`.
- G-3: Operatives deployed via Kiro's native engine inherit its permission model, hooks, and MCP plumbing — no moonbase-side reimplementation needed.
- G-4: Backward-compatible — existing `.md` agents work unchanged; the new `mcp_servers` field is optional.
- G-5: Clear migration path to retire moonbase's parallel safety enforcement (`SafeEnv`, hook guard, shell allowlist checks) where Kiro's engine provides equivalent coverage.

## Non-Goals

- Replacing the `.md` format with JSON — markdown stays source of truth.
- Implementing an MCP server runtime inside moonbase — Kiro's engine handles that.
- Changing agent semantics (pipeline position, routing, triggers) — those remain moonbase-only orchestration concerns not mapped to Kiro's schema.
- Supporting non-Kiro backends with this JSON format (OpenAI, Anthropic, Ollama still use the existing deploy path).

## Requirements

### R-1 — MCP Server Frontmatter Field

- **1.1** `internal/agents/agent.go` SHALL add an `MCPServers []MCPServerConfig` field (`yaml:"mcp_servers,omitempty"`) to the `Agent` struct.
- **1.2** `MCPServerConfig` SHALL support at minimum: `name` (string, required), `command` (string, required), `args` ([]string, optional), `env` (map[string]string, optional), `allowed_tools` ([]string, optional — maps to Kiro's scoped tool filtering).
- **1.3** The parser SHALL reject duplicate `name` values within one agent's `mcp_servers` list (validation error, not silent dedup).
- **1.4** Existing agents without `mcp_servers` SHALL parse identically to today (zero-value slice, no behavioral change).

### R-2 — Compile Command

- **2.1** A new `moonbase compile` command (or `moonbase install --native`) SHALL emit one Kiro agent JSON file per operative into a target directory (default: `.kiro/agents/`).
- **2.2** The command SHALL accept `--out <dir>` to override the target directory.
- **2.3** The command SHALL accept `--validate` to run `kiro-cli agent validate` on each emitted file and report pass/fail per agent.
- **2.4** The command SHALL accept `--agent <name>` to compile a single agent (default: all).
- **2.5** If the target directory does not exist, the command SHALL create it with `0755` permissions.
- **2.6** Emitted filenames SHALL be `<agent-name>.json` (e.g., `numbuh-4.json`).

### R-3 — Field Mapping (MD → Kiro JSON)

The compiler SHALL map frontmatter and body fields to Kiro's agent JSON schema:

| Moonbase field | Kiro JSON field | Mapping rule |
|---|---|---|
| `tools` | `tools` | Direct array copy |
| `auto_tools` | `allowedTools` | Direct array copy (tools that auto-approve) |
| `shell.allowed_commands` | `toolsSettings.shell.allowedCommands` | Direct array copy |
| `shell.read_only: true` | `toolsSettings.shell.readOnly` | Boolean |
| `write.auto` | `toolsSettings.write.allowedPaths` | Direct array copy |
| `write.denied` | `toolsSettings.write.deniedPaths` | Direct array copy |
| `hooks.on_activate` | `hooks.onActivate` | Map `command`→`command`, `timeout_ms`→`timeoutMs` |
| `hooks.pre_tool_use` | `hooks.preToolUse` | Same mapping |
| `hooks.post_tool_use` | `hooks.postToolUse` | Same mapping |
| `hooks.on_complete` | `hooks.onComplete` | Same mapping |
| Body (markdown prompt) | `prompt` | `"file://<agent-name>.prompt.md"` — prompt written as a companion file |
| `mcp_servers` | `mcpServers` | Map `name/command/args/env/allowed_tools` to Kiro's MCP schema |
| `name` | `name` | Direct copy |
| `description` | `description` | Direct copy |

- **3.1** Fields with no Kiro equivalent (`routing`, `pipeline_position`, `triggers`, `shortcut`, `guardrails`, `handoff`, `output_schema`) SHALL be omitted from the JSON (they are moonbase-orchestration-only).
- **3.2** The prompt body SHALL be emitted as a companion `.prompt.md` file alongside the JSON, referenced via `"file://<name>.prompt.md"` in the `prompt` field.
- **3.3** If `shell.read_only` is `true` AND `write` is nil/empty, the compiler SHALL set `toolset: "read-only"` in the JSON (Kiro's read-only agent mode).
- **3.4** If `auto_tools` is empty but `tools` is set, `allowedTools` SHALL be omitted (Kiro uses its default permission prompting).

### R-4 — Deployment via Kiro Native Engine

- **4.1** After compilation, operatives SHALL be deployable via `kiro-cli chat --agent <name>` and appear in `kiro-cli agent list`.
- **4.2** `moonbase deploy <n>` SHALL gain a `--native` flag (or respect a config option `backend: kiro-native`) that uses the compiled JSON path instead of composing a raw prompt.
- **4.3** When deploying natively, moonbase SHALL NOT apply its own SafeEnv filtering, hook execution, or shell command validation — Kiro's engine owns those responsibilities.
- **4.4** If the compiled JSON is stale (source `.md` is newer), the deploy command SHALL warn and offer to recompile (or auto-recompile with `--auto-compile`).

### R-5 — Validation

- **5.1** `moonbase compile --validate` SHALL invoke `kiro-cli agent validate <file>` for each emitted JSON and report results.
- **5.2** If `kiro-cli` is not available, `--validate` SHALL print a warning and skip (not fail the compile).
- **5.3** The compiler SHALL perform its own structural validation before writing: required fields present, no empty `name`, `command` in MCP configs, no path traversal in filenames.

### R-6 — Safety Migration Path

- **6.1** The spec SHALL document which moonbase safety mechanisms have Kiro-native equivalents and can be retired when `backend: kiro-native` is the deployment mode.
- **6.2** Mechanisms with no Kiro equivalent SHALL remain in moonbase (e.g., pipeline risk-gate routing, agent routing restrictions).
- **6.3** A `MIGRATION.md` document SHALL be generated alongside the spec listing the retirement plan per mechanism.

| Moonbase mechanism | Kiro equivalent | Retirement status |
|---|---|---|
| `SafeEnv` (env var filtering) | Kiro's sandboxed execution | Retire when native |
| Hook guard (shell blocklist) | Kiro's `toolsSettings.shell` | Retire when native |
| Shell allowlist enforcement | `toolsSettings.shell.allowedCommands` | Retire when native |
| Write path enforcement | `toolsSettings.write.allowedPaths/deniedPaths` | Retire when native |
| Hook execution (`on_activate`, etc.) | `hooks.*` in Kiro agent JSON | Retire when native |
| `input_validation` regex (guardrails) | No Kiro equivalent | Keep in moonbase |
| `max_turns` / `max_output` | No Kiro equivalent | Keep in moonbase |
| Pipeline risk-gate | No Kiro equivalent | Keep in moonbase |
| Agent routing restrictions | No Kiro equivalent | Keep in moonbase |

- **6.4** Retirement SHALL be gated behind a config flag (`safety.delegate_to_kiro: true`) so users can opt in incrementally.

### R-7 — Backward Compatibility

- **7.1** The existing `moonbase install` command SHALL continue to work unchanged (copies `.md` files for kiro-cli's native agent discovery).
- **7.2** The existing `moonbase deploy` command without `--native` SHALL continue using the current prompt-composition + raw-deployment path.
- **7.3** Agents without `mcp_servers` SHALL compile to valid JSON with an empty/absent `mcpServers` field.
- **7.4** No existing test SHALL break from adding the new field or command.

## Risks

- **Kiro agent JSON schema instability**: Kiro CLI is evolving; the JSON schema may change. Mitigate by validating against `kiro-cli agent validate` and version-pinning the schema mapping.
- **Prompt file reference resolution**: `file://` prompt references depend on Kiro resolving paths relative to the agent JSON location. Verify during implementation.
- **Dual-mode confusion**: Users may not know whether they're running native vs. legacy mode. Mitigate with clear `moonbase status` output showing deployment mode.
- **MCP server security**: Arbitrary commands in `mcp_servers.command` are a trust boundary. Mitigate by documenting that MCP servers run under Kiro's security model (not moonbase's SafeEnv) when deployed natively.

## Acceptance Criteria (Summary)

- AC-1: New `mcp_servers` field parses correctly; old agents unchanged (R-1).
- AC-2: `moonbase compile` emits valid JSON + prompt files for all 14 agents (R-2).
- AC-3: Field mapping covers all specified translations (R-3).
- AC-4: Compiled agents deploy via `kiro-cli chat --agent` and appear in `kiro-cli agent list` (R-4).
- AC-5: `--validate` invokes `kiro-cli agent validate` and reports results (R-5).
- AC-6: Safety migration documented; retirement gated behind config flag (R-6).
- AC-7: Zero breaking changes to existing commands and tests (R-7).
