# CLI Reference

Run `moonbase` with no arguments to launch the TUI dashboard. Otherwise:

## Commands

| Command | Description |
|---------|-------------|
| `moonbase` | Launch the TUI dashboard |
| `moonbase init` | Scaffold `.kiro/` in any project (specs, steering, agents, skills, prompts) |
| `moonbase init --data-access` | Also generate a stack-aware `data-access-performance.md` steering rule |
| `moonbase deploy <n> [task]` | Deploy operative by numbuh (interactive kiro-cli session) |
| `moonbase mission <task>` | Run the KND Council pipeline on a task (adaptive depth by default) |
| `moonbase install [--all] [--global]` | Install agents to `.kiro/agents/` (or `~/.kiro/agents/` with `--global`) |
| `moonbase compile [--out] [--validate] [--agent]` | Compile agents to Kiro-native JSON (see [[Kiro-Native-Interop]]) |
| `moonbase setup` | Install embedded agents globally to `~/.moonbase/agents/` |
| `moonbase status` | Environment health check (incl. Native Interop section) |
| `moonbase lint` | Validate all agent `.md` files (incl. `mcp_servers`) |
| `moonbase config` | Show current YAML configuration |
| `moonbase list` | Show operative roster (with MCP server count per agent) |
| `moonbase guide [numbuh]` | Usage guide for agents (aliases: `man`, `howto`) |
| `moonbase history` | Show mission history (`--json`, `--all`, `--limit N`) |
| `moonbase replay <id>` | Replay a previous mission (`--dry-run`) |
| `moonbase export <id>` | Export mission details |
| `moonbase snippet save/list <name>` | Manage reusable prompt snippets |
| `moonbase flywheel` | Pipeline learning insights + token/cost summary (see [[Flywheel-and-Observability]]) |
| `moonbase version` | Print version information |

## `moonbase mission` flags

| Flag | Effect |
|------|--------|
| *(none)* | **Auto-classify** task complexity → run the minimum viable depth, escalating if QA flags risk (see [[The-Pipeline]]) |
| `--fast` | Skip analysis/architecture → Implementation + QA only (no escalation) |
| `--full` | Force all 5 mandatory phases regardless of complexity |
| `--depth trivial\|simple\|complex` | Override auto-classification (starting depth; escalation still allowed) |
| `--sequential` | Disable parallel specialist fan-out for this mission |
| `--dry-run` | Print the execution plan (depth + active/skipped phases) without invoking backends |
| `--trace` | Emit trace-level info (TraceID, phase timestamps, output sizes) |
| `--force` | Override the WIP lock if another mission is running |

> `--fast`, `--full`, and `--depth` are mutually exclusive.

## Pipe mode

```bash
echo "fix the auth bug" | moonbase              # pipe to the KND Council
echo "check security"  | moonbase deploy 274     # pipe to a specific agent
```

## Examples

```bash
moonbase mission "fix typo in README"            # → trivial (Implement + QA)
moonbase mission "add rate limiting to the API"  # → complex (full council)
moonbase mission --depth simple "any task"       # override starting depth
moonbase compile --validate                      # emit + validate Kiro agent JSON
moonbase deploy 3 --native                       # deploy via kiro-cli chat --agent
```
