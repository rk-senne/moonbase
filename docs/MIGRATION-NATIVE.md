# Migration Guide: Kiro Native Interop

## Overview

Moonbase can now compile its `.md` agent definitions into Kiro-native agent JSON.
When deployed natively (`kiro-cli chat --agent <name>`), agents inherit Kiro's
permission engine, hook system, and MCP infrastructure — eliminating the need for
moonbase to reimplement these enforcement mechanisms.

This guide covers the opt-in migration from "legacy" mode (moonbase enforces safety)
to "native" mode (Kiro enforces safety).

## Safety Delegation Table

| Moonbase mechanism | Kiro equivalent | Action in native mode |
|---|---|---|
| `SafeEnv` (env var filtering) | Kiro's sandboxed execution | Retired — Kiro isolates |
| Hook guard (shell blocklist) | `toolsSettings.shell.allowedCommands` | Retired — compiled into JSON |
| Shell allowlist enforcement | `toolsSettings.shell.allowedCommands` + `denyByDefault` | Retired — compiled into JSON |
| Write path enforcement | `toolsSettings.write.allowedPaths/deniedPaths` | Retired — compiled into JSON |
| Hook execution (`on_activate`, etc.) | `hooks.*` in agent JSON | Retired — Kiro executes hooks |
| MCP server lifecycle | `mcpServers` in agent JSON | Retired — Kiro manages MCP |
| `input_validation` regex (guardrails) | No Kiro equivalent | **Kept in moonbase** |
| `max_turns` / `max_output` | No Kiro equivalent | **Kept in moonbase** |
| Pipeline risk-gate | No Kiro equivalent | **Kept in moonbase** |
| Agent routing restrictions | No Kiro equivalent | **Kept in moonbase** |

## Opt-In Steps

### Step 1: Compile agents

```bash
moonbase compile --validate
```

This emits `.json` + `.prompt.md` files into `.kiro/agents/` and validates each
against `kiro-cli agent validate`.

### Step 2: Test native deployment

```bash
moonbase deploy 1 --native "test task"
```

Verify the agent loads correctly and responds. Check that hooks fire, shell
commands respect permissions, and MCP servers (if any) are available.

### Step 3: Set native as default (per-project)

Add to `~/.moonbase/config.yaml`:

```yaml
deploy:
  mode: native
  auto_compile: true   # Auto-recompile stale agents
```

### Step 4: Enable safety delegation

Once confident that Kiro's enforcement matches expectations:

```yaml
safety:
  delegate_to_kiro: true
```

This skips moonbase's `SafeEnv` and hook-guard checks entirely. Kiro's engine
owns all runtime safety enforcement.

## Rollback

Immediate rollback — set in config:

```yaml
deploy:
  mode: legacy
```

This returns to the original behaviour: moonbase composes raw prompts, applies
`SafeEnv`, enforces shell allowlists, and executes hooks itself. No code change
required. The compiled JSON files can remain on disk (they're inert in legacy mode).

## Minimum Kiro CLI Version

| Feature | Minimum kiro-cli version |
|---|---|
| `agent validate` | 0.3.0+ |
| `chat --agent` | 0.3.0+ |
| `hooks` (agentSpawn, stop, preToolUse, postToolUse) | 0.3.0+ |
| `toolsSettings.shell` (denyByDefault, autoAllowReadonly) | 0.3.0+ |
| `toolsSettings.write` (allowedPaths, deniedPaths) | 0.3.0+ |
| `mcpServers` | 0.3.0+ |
| `file://` prompt resolution | 0.3.0+ |

Run `kiro-cli --version` to check your installed version.

## FAQ

**Q: Can I use native mode for some agents and legacy for others?**
A: The `--native` flag is per-invocation. Config `deploy.mode` is the default but
   can be overridden with `--legacy` on any deploy command. Currently all agents
   compile to the same format — there's no per-agent mode selector.

**Q: What happens if I edit the `.md` file but forget to recompile?**
A: With `deploy.auto_compile: true`, moonbase will detect staleness and recompile
   automatically. Without it, moonbase warns that the compiled JSON is stale and
   suggests running `moonbase compile`.

**Q: Is the `.json` file safe to commit to git?**
A: Yes. It contains no secrets (env vars are referenced as `${VAR}`, not expanded).
   However, since it's a derived artifact, you may prefer to `.gitignore` it and
   compile on demand.

**Q: What about the `routing`, `pipeline_position`, and `triggers` fields?**
A: These are moonbase-only orchestration concepts with no Kiro equivalent. They
   remain in the `.md` source and are used by moonbase's pipeline orchestrator
   regardless of deploy mode.
