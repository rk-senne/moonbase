# Design — Kiro Native Interop

## Strategy

A **new compile step** inserted between "agent source (`.md`)" and "Kiro deployment" that
translates the existing markdown format into Kiro's native agent JSON. This keeps the
`.md` as the single source of truth while giving operatives full access to Kiro's
permission engine, hook system, and MCP infrastructure.

The compile step is additive — no existing flow changes until the user opts in to native
deployment. The existing raw-prompt deployment path (`kiro-cli chat -- <prompt>`) remains
the default.

```
agents/*.md  →  moonbase compile  →  .kiro/agents/*.json + *.prompt.md
                                          ↓
                                   kiro-cli chat --agent <name>
                                   (inherits Kiro's hooks/permissions/MCP)
```

## Phase 0 Findings — Schema Corrections (verified live via `kiro-cli agent validate`)

The original R-3 mapping *guessed* several Kiro field names. Validated against the real
`kiro-cli` (`/opt/homebrew/bin/kiro-cli`), the following corrections are **authoritative**
and supersede any conflicting text below in this document and in requirements R-3:

1. **Hook triggers use Kiro lifecycle names** (not `onActivate`/`onComplete`), object-format,
   with `timeout_ms` (snake, milliseconds — NOT `timeoutMs`):
   - `hooks.on_activate` → `hooks.agentSpawn`
   - `hooks.pre_tool_use` → `hooks.preToolUse`
   - `hooks.post_tool_use` → `hooks.postToolUse`
   - `hooks.on_complete` → `hooks.stop`
   - Shape: `"hooks": { "agentSpawn": [ { "command": "…", "timeout_ms": 5000 } ] }`.
   (`onActivate`/`timeoutMs` **fail** validation: "did not match any variant of untagged enum".)

2. **No `toolset` and no `toolsSettings.shell.readOnly`** — not in Kiro's schema (they don't
   error but are silently ignored → false enforcement). Map moonbase `shell.read_only: true`
   to `toolsSettings.shell.denyByDefault: true` + `autoAllowReadonly: true`, and simply do NOT
   grant the `write` tool. **Drop R-3.3's `toolset: "read-only"` entirely.**

3. **`mcpServers` is a JSON object keyed by server name, NOT an array** (array fails:
   "invalid type: sequence, expected a map"). Emit
   `"mcpServers": { "<name>": { "command": …, "args": […], "env": {…} } }`. Omit when empty.

4. **Per-server `allowed_tools`** has no field inside the server object; scope MCP tools via the
   agent's top-level `allowedTools` with `@<name>/<tool>` patterns
   (e.g. `"allowedTools": ["read", "@github/create_pull_request"]`).

5. **Prompt reference**: relative `file://<name>.prompt.md` (companion file beside the JSON)
   passes schema validation; runtime resolution is confirmed in Task 7.2 (manual native deploy).

Minimal agent using the corrected schema (validated ✅ — zero error output from `kiro-cli agent validate`):
```json
{ "name":"nb4","description":"QA","prompt":"file://nb4.prompt.md",
  "tools":["read","shell","grep","glob"], "allowedTools":["read","grep","glob"],
  "toolsSettings":{"shell":{"allowedCommands":["go test ./..."],"denyByDefault":true,"autoAllowReadonly":true}},
  "hooks":{"agentSpawn":[{"command":"git branch --show-current","timeout_ms":5000}]} }
```

The `KiroAgent` struct MUST therefore use `MCPServers map[string]KiroMCPServer` (not a slice) and
`Hooks` keyed by `agentSpawn/preToolUse/postToolUse/stop`, and MUST NOT include `toolset`/`readOnly`.

## Design Decisions

### D-1 — `moonbase compile` as a Dedicated Command (not overloading `install`)

**Why:** `install` copies `.md` files for kiro-cli's existing raw-agent discovery.
`compile` is a semantically different operation (translation + code generation).
Overloading would conflate "copy source files" with "emit derived artifacts."

**Alternative considered:** `moonbase install --native` flag. Rejected because it
hides the compilation nature of the operation and makes `install` do two very
different things depending on a flag.

**Compromise:** Both work — `moonbase compile` is canonical, but `moonbase install
--native` is accepted as an alias for discoverability.

### D-2 — Prompt as Companion File (`file://` Reference)

**Why:** Kiro agent JSON supports `"prompt": "file://path.md"` for external prompt
files. Embedding the full markdown body as a JSON string would:
- Break readability (escaped newlines, no syntax highlighting)
- Bloat the JSON (some prompts are 3000+ chars)
- Complicate diffing

**Structure:**
```
.kiro/agents/
├── numbuh-3.json          # Agent config (tools, hooks, MCP, permissions)
├── numbuh-3.prompt.md     # Full prompt body (the markdown after frontmatter)
├── numbuh-4.json
├── numbuh-4.prompt.md
└── ...
```

The JSON references: `"prompt": "file://numbuh-3.prompt.md"` (relative path).

### D-3 — MCPServerConfig Schema

Aligned with Kiro's `mcpServers` JSON schema:

```go
// MCPServerConfig defines an MCP server available to the agent.
type MCPServerConfig struct {
    Name         string            `yaml:"name" json:"name"`
    Command      string            `yaml:"command" json:"command"`
    Args         []string          `yaml:"args,omitempty" json:"args,omitempty"`
    Env          map[string]string `yaml:"env,omitempty" json:"env,omitempty"`
    AllowedTools []string          `yaml:"allowed_tools,omitempty" json:"allowedTools,omitempty"`
}
```

Example in agent frontmatter:
```yaml
mcp_servers:
  - name: github
    command: npx
    args: ["-y", "@modelcontextprotocol/server-github"]
    env:
      GITHUB_TOKEN: "${GITHUB_TOKEN}"
    allowed_tools:
      - create_pull_request
      - list_issues
  - name: postgres
    command: npx
    args: ["-y", "@modelcontextprotocol/server-postgres"]
    env:
      DATABASE_URL: "${DATABASE_URL}"
```

### D-4 — Compiled JSON Schema

Target output (example for numbuh-3):

```json
{
  "name": "numbuh-3",
  "description": "Writes clean, readable, testable code...",
  "prompt": "file://numbuh-3.prompt.md",
  "tools": ["read", "write", "shell", "grep", "glob", "code", "knowledge"],
  "allowedTools": ["read", "write", "grep", "glob", "code"],
  "toolsSettings": {
    "shell": {
      "allowedCommands": ["mvn test", "npm test", "go test ./...", "..."],
      "readOnly": false
    },
    "write": {
      "allowedPaths": ["src/**", "lib/**", "app/**", "tests/**", "internal/**", "docs/**"],
      "deniedPaths": []
    }
  },
  "hooks": {
    "onActivate": [
      {
        "command": "echo \"Branch: $(git branch --show-current)\" ...",
        "timeoutMs": 5000
      }
    ]
  },
  "mcpServers": []
}
```

For a read-only agent (numbuh-4):
```json
{
  "name": "numbuh-4",
  "description": "Hits implementation with reality...",
  "prompt": "file://numbuh-4.prompt.md",
  "tools": ["read", "shell", "grep", "glob", "code", "knowledge", "subagent"],
  "allowedTools": ["read", "shell", "grep", "glob", "code", "knowledge"],
  "toolset": "read-only",
  "toolsSettings": {
    "shell": {
      "allowedCommands": ["mvn test", "npm test", "go test ./...", "..."],
      "readOnly": true
    }
  },
  "hooks": {
    "onActivate": [...]
  },
  "mcpServers": []
}
```

### D-5 — Compiler Architecture

New package: `internal/compile/`

```go
// Package compile translates moonbase Agent structs into Kiro-native agent JSON.
package compile

// KiroAgent is the target JSON structure for a compiled agent.
type KiroAgent struct {
    Name          string              `json:"name"`
    Description   string              `json:"description,omitempty"`
    Prompt        string              `json:"prompt"`
    Tools         []string            `json:"tools,omitempty"`
    AllowedTools  []string            `json:"allowedTools,omitempty"`
    Toolset       string              `json:"toolset,omitempty"`
    ToolsSettings *KiroToolsSettings  `json:"toolsSettings,omitempty"`
    Hooks         *KiroHooks          `json:"hooks,omitempty"`
    MCPServers    []KiroMCPServer     `json:"mcpServers,omitempty"`
}

// Compile translates a moonbase Agent into a KiroAgent.
func Compile(agent agents.Agent) (*KiroAgent, error) { ... }

// WriteAgent writes the compiled JSON and prompt file to the target directory.
func WriteAgent(agent *KiroAgent, promptBody string, targetDir string) error { ... }
```

**Separation of concerns:**
- `internal/compile/` — pure translation logic (no I/O in `Compile`, testable)
- `cmd/moonbase/compile_cmd.go` — CLI wiring (flags, I/O, validation invocation)
- `internal/agents/agent.go` — source struct (gains `MCPServers` field)

### D-6 — Staleness Detection

`moonbase deploy --native` checks:
1. Does the compiled `.json` exist?
2. Is `<agent>.json` modification time ≥ source `<agent>.md` modification time?

If stale or missing:
- With `--auto-compile`: recompile silently and proceed.
- Without: print warning and suggest `moonbase compile`.

### D-7 — Native Deployment Path

When deploying natively (`moonbase deploy <n> --native`):

```go
// Instead of:
//   composed := discovery.ComposePrompt(agent.Prompt, context, task)
//   kiro.DeployRaw(composed, task)
//
// Do:
//   kiro-cli chat --agent <name>
//   (Kiro loads the JSON, resolves file:// prompt, applies permissions)
```

This means moonbase delegates ALL of:
- Shell command validation → Kiro's `toolsSettings.shell.allowedCommands`
- Write path enforcement → Kiro's `toolsSettings.write.allowedPaths/deniedPaths`
- Hook execution → Kiro's `hooks.*`
- MCP server lifecycle → Kiro's `mcpServers`
- Environment isolation → Kiro's sandbox

Moonbase retains:
- Pipeline orchestration (phase transitions, risk gate)
- Agent routing (which agent can hand off to which)
- Guardrails (max_turns, max_output, input/output rules)
- Context injection (specs, steering, project discovery)

### D-8 — Config Integration

New config fields in `~/.moonbase/config.yaml`:

```yaml
compile:
  out_dir: ".kiro/agents"     # Default output directory
  auto_validate: true         # Run kiro-cli agent validate after compile

deploy:
  mode: "legacy"              # "legacy" (raw prompt) | "native" (compiled JSON)
  auto_compile: false         # Auto-recompile stale agents on deploy

safety:
  delegate_to_kiro: false     # When true + native mode, skip moonbase safety checks
```

### D-9 — Safety Delegation Architecture

```
┌─────────────────────────────────────────────────────────┐
│ deploy.mode = "legacy" (current)                         │
│                                                          │
│  moonbase SafeEnv ─→ kiro-cli chat -- <prompt>          │
│  moonbase hook guard                                     │
│  moonbase shell allowlist                                │
│  (all enforced by moonbase before/during deploy)         │
└─────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────┐
│ deploy.mode = "native" + safety.delegate_to_kiro = true  │
│                                                          │
│  moonbase pipeline orchestration only ─→                 │
│    kiro-cli chat --agent <name>                          │
│    (Kiro enforces: shell, write, hooks, MCP, env)        │
│                                                          │
│  moonbase retains: routing, risk gate, guardrails        │
└─────────────────────────────────────────────────────────┘
```

The transition is opt-in per-project. Both modes coexist indefinitely. The "retire"
path is: once Kiro covers a safety mechanism, moonbase's implementation becomes dead
code for native-mode users; it is removed only after a deprecation period (one minor
release cycle with a deprecation log).

## File Impact Map

| Area | Files | Requirements |
|---|---|---|
| Agent struct | `internal/agents/agent.go` | R-1 |
| Agent parser | `internal/agents/parse.go` (validation) | R-1.3 |
| Agent parser tests | `internal/agents/parse_test.go` | R-1.4 |
| Compiler package | `internal/compile/compile.go` (new) | R-3 |
| Compiler types | `internal/compile/types.go` (new) | R-3, D-4, D-5 |
| Compiler tests | `internal/compile/compile_test.go` (new) | R-3, R-5.3 |
| CLI compile cmd | `cmd/moonbase/compile_cmd.go` (new) | R-2 |
| CLI compile logic | `cmd/moonbase/compile.go` (new) | R-2 |
| Deploy enhancement | `cmd/moonbase/deploy_cmd.go` | R-4.2 |
| Backend native | `internal/backend/backends.go` (Kiro.DeployNative) | R-4.3 |
| Config | `internal/config/config.go` | D-8 |
| Main command reg | `cmd/moonbase/main.go` | R-2 |
| Staleness check | `internal/compile/staleness.go` (new) | D-6 |

## Test Alignment

| Requirement | Verified by |
|---|---|
| R-1 MCP field | `internal/agents/parse_test.go` — valid, missing, duplicate-name cases |
| R-2 compile cmd | Integration test: compile all 14 agents, check output files exist |
| R-3 field mapping | `internal/compile/compile_test.go` — table-driven per field |
| R-3.3 read-only | Compile numbuh-4, assert `toolset: "read-only"` present |
| R-4 native deploy | Integration test with mock `kiro-cli` (verify args passed) |
| R-5 validation | Test with `kiro-cli agent validate` available/unavailable |
| R-6 safety delegation | Unit test: verify SafeEnv skipped when delegate flag set |
| R-7 backward compat | Existing test suite passes unchanged |

## Estimated Effort

Medium. The compiler is mostly mechanical mapping. The risk is in Kiro's JSON schema
correctness (mitigated by `--validate`) and in the deploy-mode switchover (mitigated by
the opt-in config flag). Budget for validation round-trips with `kiro-cli agent validate`
to catch schema mismatches early.
