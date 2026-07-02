# Tasks: Moonbase v3 — Production-Ready

## Milestone 1: CLI Deploy Fix (quick win)

### Task 1: Fix deploy to use syscall.Exec for interactive kiro-cli
- **Requirements:** AC-2.1, AC-2.2, AC-2.3
- **Files:** `cmd/moonbase/main.go`
- **Action:** Replace the current `exec.Command` approach with `syscall.Exec` that hands the terminal to kiro-cli. Accept optional task as additional args: `moonbase deploy 4 "check auth"`. Use proper kiro-cli flags (`--context` for system prompt file). Test which kiro-cli flags actually work.
- **Test:** Manual test: `moonbase deploy 1` opens interactive kiro-cli session with Numbuh 1's prompt. Verify with `--help` what flags kiro-cli accepts.
- **Status:** pending

### Task 2: Improve deploy fallback messaging
- **Requirements:** AC-2.2
- **Files:** `cmd/moonbase/main.go`
- **Action:** When clipboard fallback activates, show: agent name, role, prompt size, project context detected, and suggest next steps ("Paste into Claude/ChatGPT/Kiro IDE").
- **Test:** Deploy without kiro-cli shows informative message.
- **Status:** pending

### Checkpoint: CLI Deploy
- [ ] `moonbase deploy 4` opens interactive session (when kiro-cli available)
- [ ] `moonbase deploy 4 "fix the bug"` includes task in prompt
- [ ] Clipboard fallback shows useful info
- [ ] Build passes

---

## Milestone 2: Config Modernisation

### Task 3: Rewrite config package to YAML
- **Requirements:** AC-3.1, AC-3.2
- **Files:** `internal/config/config.go`
- **Action:** Replace JSON serialization with YAML. New schema: `default_backend`, `theme`, `agents_dir`, `agent_order`. Remove `Backends` map with API keys. Config path: `~/.config/moonbase/config.yaml`.
- **Test:** Unit tests for Load/Save/Defaults with temp dirs
- **Status:** pending

### Task 4: Add JSON → YAML migration
- **Requirements:** AC-3.1
- **Files:** `internal/config/migrate.go`, `internal/config/migrate_test.go`
- **Action:** Detect old `config.json` at same path. If found: read it, convert to new schema (drop API keys, keep preferences), write as `config.yaml`, rename old file to `config.json.bak`, log migration message.
- **Test:** Migration converts, preserves preferences, doesn't carry secrets
- **Status:** pending

### Task 5: Wire config into CLI and TUI
- **Requirements:** AC-3.1, AC-3.2, AC-3.3
- **Files:** `cmd/moonbase/main.go`, `internal/tui/app.go`
- **Action:** Load config at startup. Use `default_backend` for backend selection. Use `agents_dir` if set (otherwise auto-detect). Add `moonbase config` subcommand that prints current effective config.
- **Test:** `moonbase config` shows YAML output
- **Status:** pending

### Task 6: Config tests
- **Requirements:** AC-4.3
- **Files:** `internal/config/config_test.go`
- **Action:** Test: default values are correct, YAML load/save round-trips, missing file returns defaults, malformed YAML doesn't crash (returns defaults + warning), agent_order is optional.
- **Test:** `go test ./internal/config/...` passes
- **Status:** pending

### Checkpoint: Config
- [ ] Config loads from YAML
- [ ] `moonbase config` shows current settings
- [ ] Migration from JSON works (test with temp file)
- [ ] No API keys in config
- [ ] Old config users aren't broken (migration + fallback)

---

## Milestone 3: TUI Real Pipeline

### Task 7: Add project context and backend fields to App
- **Requirements:** AC-1.1
- **Files:** `internal/tui/app.go`
- **Action:** Add `projectCtx *discovery.ProjectContext` and `backend backend.Backend` fields to App struct. Initialize in `NewApp()` by calling `discovery.Discover(cwd)` and `backend.Preferred()`. Store for use during pipeline execution.
- **Test:** App initializes without panic when no .kiro/ exists
- **Status:** pending

### Task 8: Create pipeline execution tea.Cmd wrappers
- **Requirements:** AC-1.1, AC-1.2
- **Files:** `internal/tui/pipeline_exec.go`
- **Action:** Implement `PhaseResultMsg` type and `executePhase(phase)` tea.Cmd function. The Cmd runs the backend in a goroutine and returns the result as a message. Include a 120-second timeout.
- **Test:** Unit test with mock backend that returns canned output
- **Status:** pending

### Task 9: Wire pipeline execution into TUI Update()
- **Requirements:** AC-1.1, AC-1.2, AC-1.3, AC-1.4
- **Files:** `internal/tui/app.go`
- **Action:** When mission is entered: start phase 1 via `executePhase`. When `PhaseResultMsg` arrives: record output, update pipeline chat with real content, apply risk gate for phase 4, start next phase or stop. When user presses `esc`: set abort flag, mark pipeline stopped. Handle conditional phases with trigger evaluation.
- **Test:** Manual TUI test — enter mission, observe real pipeline execution
- **Status:** pending

### Task 10: Pipeline chat view shows real output
- **Requirements:** AC-1.1, AC-1.4
- **Files:** `internal/tui/views.go`
- **Action:** Update pipeline chat rendering to show: phase start message, real agent output (truncated to fit viewport), risk gate result, conditional trigger/skip messages. Replace hardcoded fake messages with dynamic content from PipelineContext.
- **Test:** Visual verification — pipeline chat shows real content
- **Status:** pending

### Task 11: Fallback simulated mode when no backend
- **Requirements:** AC-1.1
- **Files:** `internal/tui/app.go`
- **Action:** If `backend.Preferred()` returns only clipboard: show a message "No AI backend available — showing pipeline simulation" and use current fake messages as fallback. Don't break the TUI for users without kiro-cli.
- **Test:** TUI launches and shows simulation mode without kiro-cli
- **Status:** pending

### Checkpoint: TUI Pipeline
- [ ] Mission in TUI invokes real backend for each phase
- [ ] Pipeline chat shows real agent output
- [ ] Risk gate routes correctly (visible in TUI)
- [ ] Conditional phases show trigger/skip reason
- [ ] `esc` aborts pipeline execution
- [ ] Fallback to simulation when no backend
- [ ] TUI doesn't freeze during backend calls (async via tea.Cmd)

---

## Milestone 4: Test Coverage

### Task 12: Backend package tests
- **Requirements:** AC-4.2
- **Files:** `internal/backend/backend_test.go`
- **Action:** Test `DetectAll()` returns all 6 backends. Test `DetectAvailable()` with mocked LookPath (use build tags or interface). Test `Preferred()` selection order. Test clipboard `Deploy` produces correct prompt composition.
- **Test:** `go test ./internal/backend/...` passes
- **Status:** pending

### Task 13: CLI command tests
- **Requirements:** AC-4.1
- **Files:** `cmd/moonbase/main_test.go`
- **Action:** Test `findAgentsSource()` with temp dirs. Test `isAgentsDir()` detection. Test install creates correct directory structure. Test deploy resolves agent names correctly (`4` → `numbuh-4.md`, `council` → `knd-council.md`, `z` → `sector-z.md`).
- **Test:** `go test ./cmd/moonbase/...` passes
- **Status:** pending

### Task 14: Pipeline execution integration test (TUI)
- **Requirements:** AC-1.1
- **Files:** `internal/tui/pipeline_exec_test.go`
- **Action:** Test `executePhase` with a mock backend. Verify PhaseResultMsg is returned with output. Test timeout behavior. Test abort flag.
- **Test:** `go test ./internal/tui/...` passes
- **Status:** pending

### Checkpoint: Test Coverage
- [ ] `go test ./...` passes with NO package failures
- [ ] At least 6 packages have test files (agents, discovery, pipeline, config, backend, cmd)
- [ ] Total test count > 80

---

## Milestone 5: Codebase Cleanup

### Task 15: Audit and fix all doctrine files
- **Requirements:** AC-5.3
- **Files:** `doctrine/*.md`
- **Action:** Grep all doctrine for: `config.json`, `Profile.md`, `vault/`, `mission-system/`, `permissions/`, `*.json`. Fix or remove stale references. Remove doctrines that add zero value beyond what's in agent files.
- **Test:** `grep -r "config.json\|Profile.md\|vault/\|mission-system" doctrine/` returns 0 results
- **Status:** pending

### Task 16: Audit internal packages for dead code
- **Requirements:** AC-5.1, AC-5.2
- **Files:** `internal/*/`
- **Action:** Check each internal package: is it imported by TUI or CLI? Does it contain stale JSON references to old agent format? Remove dead imports, fix stale patterns. Specifically check `internal/chat/`, `internal/docs/`, `internal/history/`, `internal/projects/` for old-era cruft.
- **Test:** `go vet ./...` clean, no unused imports
- **Status:** pending

### Task 17: Remove unused `encoding/json` from non-config code
- **Requirements:** AC-5.2
- **Files:** Various
- **Action:** Grep for `encoding/json` imports across all Go files. JSON is valid in: config (migration), snippets (storage), history (storage). If found elsewhere, remove or justify.
- **Test:** `go build ./...` clean after removal
- **Status:** pending

### Task 18: Update docs/design.md to match reality
- **Requirements:** AC-5.3
- **Files:** `docs/design.md`
- **Action:** Update architecture diagram, file list, tech stack, and milestone status to reflect v3 state. Remove references to features that were never built (backends.md, agents.md). Update views section to match current TUI functionality.
- **Test:** Docs accurately describe the project
- **Status:** pending

### Checkpoint: Cleanup
- [ ] No stale doctrine references
- [ ] No dead internal packages
- [ ] `go vet ./...` clean
- [ ] `go build ./...` clean
- [ ] docs/design.md matches reality

---

## Final Verification

- [ ] All milestones complete
- [ ] `go vet ./...` clean
- [ ] `go build ./...` clean
- [ ] `go test ./...` passes (target: 80+ tests, 6+ packages covered)
- [ ] `make build` produces working binary
- [ ] `moonbase help` shows all commands
- [ ] `moonbase deploy 4` works (interactive or clipboard)
- [ ] `moonbase mission "fix the bug"` executes pipeline
- [ ] `moonbase install --all` installs agents
- [ ] `moonbase config` shows YAML config
- [ ] TUI launches without panic
- [ ] TUI pipeline executes real backend calls (when available)
- [ ] TUI falls back to simulation gracefully (when no backend)
- [ ] No hardcoded developer-specific paths
- [ ] No API keys in config files
- [ ] All doctrine files are current
