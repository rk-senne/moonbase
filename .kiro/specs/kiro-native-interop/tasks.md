# Tasks — Kiro Native Interop

Execute on branch `feat/kiro-native-interop`. Each phase should compile cleanly before
proceeding to the next.

## Phase 0 — De-risk & Schema Discovery

- [ ] 0.1 Run `kiro-cli agent validate --help` and `kiro-cli agent list` to confirm the
  native agent JSON schema expectations. Document the exact JSON fields Kiro expects in
  a scratch file and compare against the mapping table in R-3. (R-3)
- [ ] 0.2 Create a hand-written test JSON for numbuh-4 (read-only agent) and validate it
  via `kiro-cli agent validate`. Fix any schema mismatches before writing the compiler. (R-5)
- [ ] 0.3 Confirm `"prompt": "file://relative.md"` resolution behavior — does Kiro resolve
  relative to the JSON file location or CWD? Document finding. (D-2)
- [ ] 0.4 Create branch and open draft PR. (Quality gate)

## Phase 1 — MCP Server Frontmatter Field

- [ ] 1.1 Add `MCPServerConfig` struct to `internal/agents/agent.go`:
  ```go
  type MCPServerConfig struct {
      Name         string            `yaml:"name" json:"name"`
      Command      string            `yaml:"command" json:"command"`
      Args         []string          `yaml:"args,omitempty" json:"args,omitempty"`
      Env          map[string]string `yaml:"env,omitempty" json:"env,omitempty"`
      AllowedTools []string          `yaml:"allowed_tools,omitempty" json:"allowedTools,omitempty"`
  }
  ```
  Add `MCPServers []MCPServerConfig` field to `Agent` struct with tag
  `yaml:"mcp_servers,omitempty"`. (R-1.1, R-1.2)

- [ ] 1.2 Add validation in the parser: reject duplicate `name` values within one agent's
  `mcp_servers` list. Return a clear error: `"agent %s: duplicate mcp_server name %q"`.
  (R-1.3)

- [ ] 1.3 Add `HasMCPServers() bool` helper method on `Agent`. (Consistency with existing
  `HasShell()`, `HasWrite()` pattern.)

- [ ] 1.4 Write tests in `internal/agents/parse_test.go`:
  - Valid: agent with 2 MCP servers parses correctly, fields populated.
  - Missing: agent without `mcp_servers` → empty slice, no error.
  - Duplicate name: returns validation error.
  - Missing required field (`command` empty): returns error.
  (R-1.3, R-1.4)

- [ ] 1.5 Run `go build ./...` and `go test ./internal/agents/...` — green. (Quality gate)

## Phase 2 — Compiler Package

- [ ] 2.1 Create `internal/compile/types.go` with the Kiro JSON target structs:
  ```go
  type KiroAgent struct { ... }
  type KiroToolsSettings struct { ... }
  type KiroShellSettings struct { ... }
  type KiroWriteSettings struct { ... }
  type KiroHooks struct { ... }
  type KiroHook struct { ... }
  type KiroMCPServer struct { ... }
  ```
  Use `json` struct tags matching Kiro's schema (camelCase). (R-3, D-4)

- [ ] 2.2 Create `internal/compile/compile.go` with:
  - `func Compile(agent agents.Agent) (*KiroAgent, string, error)` — returns compiled
    struct + prompt body (the markdown after frontmatter) + error.
  - Mapping logic per R-3 table:
    - `tools` → `Tools`
    - `auto_tools` → `AllowedTools` (omit if empty, per R-3.4)
    - `shell.*` → `ToolsSettings.Shell.*`
    - `write.auto/denied` → `ToolsSettings.Write.AllowedPaths/DeniedPaths`
    - `hooks.*` → `Hooks.*` (camelCase key conversion)
    - `mcp_servers` → `MCPServers`
    - Body → `Prompt` = `"file://<name>.prompt.md"`
    - `shell.read_only && write == nil` → `Toolset = "read-only"` (R-3.3)
  (R-3)

- [ ] 2.3 Create `internal/compile/write.go` with:
  - `func WriteAgent(ka *KiroAgent, promptBody, targetDir string) error`
  - Writes `<name>.json` (indented JSON, 0644) and `<name>.prompt.md` (0644).
  - Validates: name is non-empty, no path traversal in name.
  (R-2.5, R-2.6)

- [ ] 2.4 Create `internal/compile/staleness.go` with:
  - `func IsStale(agentMDPath, compiledJSONPath string) (bool, error)`
  - Compares modification times; missing JSON = stale.
  (D-6)

- [ ] 2.5 Write `internal/compile/compile_test.go` — table-driven tests:
  - Compile numbuh-3 (full agent: shell, write, hooks, tools, auto_tools) → verify all
    JSON fields.
  - Compile numbuh-4 (read-only, no write) → verify `toolset: "read-only"`.
  - Compile agent with `mcp_servers` → verify `mcpServers` array in JSON.
  - Compile agent with no optional fields → verify minimal valid JSON.
  - `WriteAgent` produces valid JSON files (read back and unmarshal).
  - Staleness detection (newer source, older source, missing compiled).
  (R-3, R-5.3)

- [ ] 2.6 Run `go build ./...` and `go test ./internal/compile/...` — green. (Quality gate)

## Phase 3 — CLI Command (`moonbase compile`)

- [ ] 3.1 Create `cmd/moonbase/compile_cmd.go`:
  - Register `compile` subcommand in `main.go`.
  - Flags: `--out <dir>` (default `.kiro/agents`), `--validate`, `--agent <name>`.
  - Alias: register `install --native` as equivalent to `compile` (D-1 compromise).
  (R-2.1, R-2.2, R-2.3, R-2.4)

- [ ] 3.2 Create `cmd/moonbase/compile.go` — implementation:
  - Load all agents (or single if `--agent` specified).
  - For each: `compile.Compile(agent)` → `compile.WriteAgent(...)`.
  - Print per-agent status: `✅ numbuh-3 → .kiro/agents/numbuh-3.json`.
  - If `--validate`: invoke `kiro-cli agent validate <file>` per agent; print pass/fail.
  - If `kiro-cli` not found and `--validate`: warn and skip (R-5.2).
  (R-2, R-5)

- [ ] 3.3 Update `cmd/moonbase/main.go` to register the new command in the command table
  and help text. (R-2.1)

- [ ] 3.4 Update README CLI table with `moonbase compile` entry. (Docs)

- [ ] 3.5 Write integration test: `TestCompile_AllAgents` — loads embedded agents, compiles
  all 14 to `t.TempDir()`, verifies 14 `.json` + 14 `.prompt.md` files exist, each JSON
  unmarshals without error. (R-2, R-7.4)

- [ ] 3.6 Run full `go test ./...` — green. (Quality gate)

## Phase 4 — Native Deployment Path

- [ ] 4.1 Add `DeployNative(agentName string) (string, error)` method to the `Kiro` backend
  in `internal/backend/backends.go`:
  ```go
  func (k *Kiro) DeployNative(agentName string) (string, error) {
      args := []string{"chat", "--agent", agentName}
      if k.TrustTools {
          args = append(args, "--trust-all-tools", "--no-interactive")
      }
      cmd := exec.Command("kiro-cli", args...)
      // NOTE: No SafeEnv() — Kiro's engine handles env isolation.
      ...
  }
  ```
  (R-4.3)

- [ ] 4.2 Add `--native` flag to `cmd/moonbase/deploy_cmd.go`. When set:
  - Check compiled JSON exists and is not stale (D-6).
  - If stale + `--auto-compile`: recompile. If stale without: warn.
  - Call `Kiro.DeployNative(agent.Name)` instead of `DeployComposed`. (R-4.2, R-4.4)

- [ ] 4.3 Add config fields to `internal/config/config.go`:
  ```go
  type CompileConfig struct {
      OutDir       string `yaml:"out_dir"`
      AutoValidate bool   `yaml:"auto_validate"`
  }
  type DeployConfig struct {
      Mode        string `yaml:"mode"`         // "legacy" | "native"
      AutoCompile bool   `yaml:"auto_compile"`
  }
  type SafetyConfig struct {
      DelegateToKiro bool `yaml:"delegate_to_kiro"`
  }
  ```
  (D-8)

- [ ] 4.4 When `deploy.mode = "native"` in config, `moonbase deploy <n>` defaults to
  native path (no `--native` flag needed). Override with `--legacy` flag. (R-4.2)

- [ ] 4.5 When `safety.delegate_to_kiro = true` AND native mode: skip `SafeEnv()` call,
  skip hook guard checks in the deploy path. Add clear comment documenting this is
  intentional delegation. (R-4.3, R-6.4)

- [ ] 4.6 Write tests:
  - `TestDeployNative_InvokesKiroCLI` — mock `exec.Command`, verify `--agent <name>`.
  - `TestDeployNative_StaleWarning` — verify staleness detection triggers warning.
  - `TestDeployNative_AutoCompile` — verify recompilation when flag set.
  - `TestDeployLegacy_Unchanged` — existing path still works without native flag.
  (R-4, R-7.2)

- [ ] 4.7 Run full `go test ./...` — green. (Quality gate)

## Phase 5 — Validation & Status Integration

- [ ] 5.1 `moonbase status` gains a "Native Interop" section showing:
  - Compiled agents count vs. total agents.
  - Stale agents (source newer than compiled).
  - Kiro CLI version (if available).
  - Deploy mode (from config).
  (Observability)

- [ ] 5.2 `moonbase lint` gains validation of `mcp_servers` in all agent files:
  - Name present and non-empty.
  - Command present and non-empty.
  - No duplicate names within one agent.
  (R-1.3, R-5.3)

- [ ] 5.3 Update `moonbase list` output to show MCP server count per agent (if any).
  (UX)

- [ ] 5.4 Run full `go test ./...` — green. (Quality gate)

## Phase 6 — Safety Migration Documentation

- [ ] 6.1 Create `docs/MIGRATION-NATIVE.md` documenting:
  - Which safety mechanisms are delegated to Kiro in native mode.
  - Which remain in moonbase regardless of mode.
  - Step-by-step opt-in instructions.
  - Rollback: set `deploy.mode: legacy` to revert immediately.
  (R-6.1, R-6.2, R-6.3)

- [ ] 6.2 Add deprecation logging in `SafeEnv()` and hook-guard code paths:
  - If `safety.delegate_to_kiro = true` AND native mode: log once per session at `slog.Info`
    level that these checks are being skipped in favor of Kiro's engine.
  (R-6.4)

- [ ] 6.3 Document in MIGRATION-NATIVE.md the exact Kiro CLI version that supports each
  feature (hooks, MCP, toolsSettings), so users know the minimum required version. (Risk mitigation)

## Phase 7 — Integration & Quality Gates

- [ ] 7.1 Run full gate suite:
  ```bash
  go build ./...
  go vet ./...
  go test -race ./... -count=1 -timeout 300s
  go run ./cmd/moonbase lint
  ```
  All green. (R-8, Quality gate)

- [ ] 7.2 Manual validation: compile all 14 agents, run `kiro-cli agent validate` on each,
  deploy one agent natively (numbuh-1 — requirements, low risk), verify it loads and
  responds. (AC-4)

- [ ] 7.3 Verify `moonbase deploy 1 "test"` (without `--native`) still works via legacy
  path — backward compatibility confirmed. (R-7.2)

- [ ] 7.4 Verify `moonbase install` (without `--native`) still copies `.md` files — backward
  compatibility confirmed. (R-7.1)

- [ ] 7.5 Verify existing tests pass without modification (the new `mcp_servers` field
  defaults to nil/empty and does not affect any existing test fixture). (R-7.4)

## Phase 8 — Changelog & PR

- [ ] 8.1 CHANGELOG under `[Unreleased] → Added`:
  - `feat(agents): mcp_servers frontmatter field for MCP server declarations`
  - `feat(compile): moonbase compile emits Kiro-native agent JSON`
  - `feat(deploy): --native flag for Kiro-native agent deployment`
  - `feat(config): safety.delegate_to_kiro option for Kiro safety delegation`
  (Changelog discipline)

- [ ] 8.2 Update README:
  - Add `moonbase compile` to CLI commands table.
  - Add "Kiro Native Interop" section under Key Capabilities.
  - Update Project Structure with `internal/compile/`.
  (Docs)

- [ ] 8.3 PR description:
  - Summary: MCP support + Kiro-native agent compilation.
  - What was tested: unit tests, integration compile of all 14 agents, manual validation.
  - Migration: link to `docs/MIGRATION-NATIVE.md`.
  - Risk: LOW for the new code (additive), MEDIUM for native deploy path (config-gated).
  (Git discipline)

- [ ] 8.4 Merge PR after CI green. This is a MINOR bump (new feature, backward-compatible).
  (Release)

## Out of Scope (Explicitly)

- Auto-detecting MCP servers from project context (e.g., scanning `package.json` for MCP
  deps). Future enhancement.
- Bi-directional sync (editing JSON → updating `.md`). JSON is always derived, never edited.
- Removing the legacy deploy path. Both paths coexist indefinitely.
- Kiro agent grouping/workspace features beyond single-agent compilation.
- Implementing MCP server health checks in moonbase.
