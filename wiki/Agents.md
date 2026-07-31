# Agents

Each operative is a single self-contained Markdown file with YAML frontmatter. **One file = one complete agent** — copy it to any project's `.kiro/agents/` and it works. The `agents/` directory is the source of truth; agents are embedded into the binary via `go:embed`.

## The roster (14 operatives)

**Sector V — the core pipeline**

| # | Designation | Role |
|---|-------------|------|
| 1 | Nigel Uno | Requirements / Analysis |
| 2 | Hoagie Gilligan | Design / Architecture |
| 3 | Kuki Sanban | Implementation |
| 4 | Wallabee Beatles | QA / Verification |
| 5 | Abigail Lincoln | Review |

**Specialists — deploy on content signals**

| # | Role |
|---|------|
| 0 | Oversight (large/architectural changes) |
| 274 | Security |
| 362 | Infra / Deploy |
| 9 | Migrations / upgrades |
| 13 | Edge cases / chaos testing |
| 86 | Dead code / cleanup |
| 999 | Documentation |
| Sector Z | Legacy / code archaeology |

Plus **KND Council** — the full-lifecycle orchestrator.

## Agent file format

```yaml
---
name: numbuh-4
designation: Wallabee Beatles
role: QA / Verification
tools: [read, shell, grep, glob, code, knowledge, subagent]
auto_tools: [read, grep, glob]
shell:
  allowed_commands: ["go test ./...", "npm test", "mvn test"]
  read_only: true
write:
  auto: []
  denied: []
routing:
  available: [numbuh-2, numbuh-3, numbuh-5, numbuh-274, numbuh-362, numbuh-0]
  trusted: [numbuh-3, numbuh-5]
pipeline_position: 4
triggers: null
mcp_servers: []        # optional — see Kiro-Native-Interop
---

# Numbuh 4 — QA / Verification

## Identity
Australian. Blunt. Brave. Evidence-driven...

## Purpose
Core question: "Does it hold when I hit it?"
...
```

### Frontmatter fields

| Field | Meaning |
|-------|---------|
| `name` / `designation` / `role` | Identity + display |
| `tools` | Tools the agent may use |
| `auto_tools` | Tools auto-approved without a prompt |
| `shell.allowed_commands` / `shell.read_only` | Shell allowlist + read-only mode |
| `write.auto` / `write.denied` / `write.requires_approval` | Write-path policy |
| `routing.available` / `routing.trusted` | Which agents it can hand off to; whose output it trusts |
| `pipeline_position` | Slot in the core pipeline (nil = conditional specialist) |
| `triggers` | Content signals that activate a conditional specialist |
| `mcp_servers` | MCP servers to expose (compiled to Kiro's `mcpServers`) |

The Markdown body *is* the agent's system prompt. Validate all agents with `moonbase lint`. See [[Kiro-Native-Interop]] to compile these into native Kiro agent JSON.
