# Kiro Native Interop

Moonbase operatives are Markdown `.md` files. **Kiro Native Interop** compiles them into Kiro's native agent JSON so they become first-class Kiro citizens — deployable via `kiro-cli chat --agent`, visible in `kiro-cli agent list`, and running under Kiro's own permission/hook/MCP engine. The `.md` format stays the single source of truth; JSON is a derived artifact.

## MCP server support

Declare Model Context Protocol servers in an operative's frontmatter:

```yaml
mcp_servers:
  - name: github
    command: npx
    args: ["-y", "@modelcontextprotocol/server-github"]
    env:
      GITHUB_TOKEN: "${GITHUB_TOKEN}"
    allowed_tools: [create_pull_request, list_issues]
```

On compile, these become Kiro's `mcpServers` object, and each `allowed_tools` entry becomes an `@github/<tool>` entry in the agent's top-level `allowedTools`.

## `moonbase compile`

```bash
moonbase compile                 # emit .kiro/agents/<name>.json + <name>.prompt.md for all 14
moonbase compile --agent numbuh-3
moonbase compile --out ./out
moonbase compile --validate      # run `kiro-cli agent validate` on each emitted file
```

Field mapping (Markdown → Kiro JSON):

| Moonbase | Kiro JSON |
|----------|-----------|
| `tools` | `tools` |
| `auto_tools` | `allowedTools` |
| `shell.allowed_commands` | `toolsSettings.shell.allowedCommands` |
| `shell.read_only: true` | `toolsSettings.shell.denyByDefault: true` + `autoAllowReadonly: true` |
| `write.auto` / `write.denied` | `toolsSettings.write.allowedPaths` / `deniedPaths` |
| `hooks.on_activate` / `pre_tool_use` / `post_tool_use` / `on_complete` | `hooks.agentSpawn` / `preToolUse` / `postToolUse` / `stop` |
| body | `prompt: "file://<name>.prompt.md"` |
| `mcp_servers` | `mcpServers` (object keyed by name) |

Moonbase-only orchestration fields (`routing`, `pipeline_position`, `triggers`, `guardrails`, `handoff`, `output_schema`) are intentionally omitted from the JSON.

> All 14 shipped operatives compile to JSON that passes `kiro-cli agent validate` cleanly.

## Native deployment

```bash
moonbase deploy 3 --native   # runs: kiro-cli chat --agent numbuh-3
```

In native mode, Kiro's engine owns shell/write enforcement, hooks, MCP lifecycle, and env isolation. Set `deploy.mode: native` in config to default to it; `safety.delegate_to_kiro: true` then lets moonbase skip its own (now-redundant) safety checks. Moonbase retains what Kiro doesn't cover: pipeline orchestration, the risk gate, agent routing, and guardrails.

See `docs/MIGRATION-NATIVE.md` in the repo for the full safety-delegation table and opt-in guide.
