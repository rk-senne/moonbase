# Tasks: Moonbase v2

## Milestone 1: Agent Loader (Foundation)

### Task 1: Add YAML dependency
- **Requirements:** AC-1.1
- **Files:** `go.mod`, `go.sum`
- **Action:** `go get gopkg.in/yaml.v3`
- **Test:** `go build ./...` succeeds
- **Status:** done

### Task 2: Create frontmatter parser
- **Requirements:** AC-1.1, AC-1.2
- **Files:** `internal/agents/parser.go`, `internal/agents/parser_test.go`
- **Action:** Implement `SplitFrontmatter(content []byte) (yamlBytes, bodyBytes []byte, err error)` — splits on first `---` delimiters. Handle edge cases: no frontmatter, malformed delimiters, empty body.
- **Test:** Unit tests with valid agent, no frontmatter, malformed YAML, empty body
- **Status:** done

### Task 3: Update Agent struct
- **Requirements:** AC-1.1
- **Files:** `internal/agents/agent.go`
- **Action:** Replace old JSON-tagged struct with new YAML-tagged struct matching frontmatter schema. Add `Prompt` and `FilePath` non-YAML fields.
- **Test:** Struct compiles, YAML tags match frontmatter keys
- **Status:** done

### Task 4: Update registry loader
- **Requirements:** AC-1.1, AC-1.2, AC-1.3
- **Files:** `internal/agents/registry.go`
- **Action:** Change `loadFromDir` to glob `*.md` instead of `*.json`. Use parser to split frontmatter+body. Parse YAML into Agent struct, set Prompt field from body. Log warning if only `.json` files found.
- **Test:** Load real agent `.md` files from `agents/` dir, verify all 14 parse correctly
- **Status:** done

### Task 5: Update TUI agent display
- **Requirements:** AC-1.1
- **Files:** `internal/tui/views.go`, `internal/tui/app.go`
- **Action:** Update any references to old Agent fields (`WelcomeMessage`, `KeyboardShortcut` → `Shortcut`). Display `Designation`, `Role` in dossier view.
- **Test:** TUI renders agent list without panic
- **Status:** done

### Checkpoint: Agent Loading
- [x] All 14 `.md` files load without error
- [x] Agent struct populated with correct frontmatter data
- [x] Prompt field contains full markdown body
- [x] `go test ./internal/agents/...` passes
- [x] `go build ./...` succeeds

---

## Milestone 2: Project Discovery

### Task 6: Create discovery package
- **Requirements:** AC-3.1, AC-3.2
- **Files:** `internal/discovery/discovery.go`, `internal/discovery/discovery_test.go`
- **Action:** Implement `Discover(projectDir string) (*ProjectContext, error)`. Search for `.kiro/specs/`, `.kiro/steering/`, `package.json`/`pom.xml`/`go.mod`/`Cargo.toml`, `README.md`. Return structured context.
- **Test:** Unit tests with mock project dirs (with/without .kiro, various stacks)
- **Status:** done

### Task 7: Steering file loader with inclusion filter
- **Requirements:** AC-3.2
- **Files:** `internal/discovery/steering.go`, `internal/discovery/steering_test.go`
- **Action:** Load `.kiro/steering/*.md` files. Parse frontmatter to check for `inclusion: manual` — skip those unless explicitly included. Return content of included files.
- **Test:** File with `inclusion: manual` is skipped; file without it is included; file with `inclusion: auto` is included
- **Status:** done

### Task 8: Prompt composition with project context
- **Requirements:** AC-3.1, AC-3.2
- **Files:** `internal/backend/compose.go`, `internal/backend/compose_test.go`
- **Action:** Implement `ComposePrompt(agent Agent, context ProjectContext, task string) string`. Order: steering rules → agent prompt → spec summary → task.
- **Test:** Compose with full context, empty context, partial context
- **Status:** done

### Checkpoint: Project Discovery
- [x] Discovery finds .kiro/specs and steering in a real project
- [x] Manual-inclusion files are correctly filtered
- [x] Composed prompt includes steering + agent + context
- [x] Works gracefully when no .kiro/ exists
- [x] `go test ./internal/discovery/...` passes

---

## Milestone 3: Pipeline Execution

### Task 9: Pipeline context accumulator
- **Requirements:** AC-2.1
- **Files:** `internal/pipeline/context.go`, `internal/pipeline/context_test.go`
- **Action:** Implement `PipelineContext` that stores phase outputs, accumulated files changed, decisions recorded. Provide `ForPhase(n int) string` that composes context for the next phase (task + relevant prior outputs).
- **Test:** Context accumulates across phases, ForPhase includes relevant history
- **Status:** done

### Task 10: Risk gate parser
- **Requirements:** AC-2.2
- **Files:** `internal/pipeline/riskgate.go`, `internal/pipeline/riskgate_test.go`
- **Action:** Parse Numbuh 4's output to extract risk level. Look for `## Verdict` section with LOW/MEDIUM/HIGH/CRITICAL. Return parsed risk + routing decision.
- **Test:** Parse real QA report examples, handle missing verdict (default MEDIUM), handle mixed case
- **Status:** done

### Task 11: Pipeline orchestrator
- **Requirements:** AC-2.1, AC-2.2, AC-2.3
- **Files:** `internal/pipeline/pipeline.go`
- **Action:** Update Pipeline to actually execute phases: for each phase, resolve agent by `pipeline_position`, compose prompt with context, invoke backend, capture output, feed to next phase. Implement risk gate routing and max 2 rework loops.
- **Test:** Integration test with mock backend that returns canned responses
- **Status:** done

### Task 12: Conditional phase evaluation
- **Requirements:** AC-2.3
- **Files:** `internal/pipeline/triggers.go`, `internal/pipeline/triggers_test.go`
- **Action:** Implement trigger evaluation for conditional specialists. Parse `triggers` field from agent frontmatter. Check pipeline context (files changed, keywords, scope) against triggers. Return invoke/skip decision with reason.
- **Test:** Trigger fires when >5 files changed (Numbuh 0), trigger fires when auth code touched (Numbuh 274), trigger skips for small changes
- **Status:** done

### Checkpoint: Pipeline
- [x] Pipeline executes phases 1-5 in sequence with mock backend
- [x] Risk gate correctly routes MEDIUM→3, HIGH→2, CRITICAL→stop
- [x] Max 2 rework loops enforced
- [x] Conditional phases evaluate triggers
- [x] Context accumulates and feeds forward
- [x] `go test ./internal/pipeline/...` passes

---

## Milestone 4: Backend Integration

### Task 13: Kiro CLI backend (real execution)
- **Requirements:** AC-2.1
- **Files:** `internal/backend/kiro.go`
- **Action:** Implement real kiro-cli invocation. Spawn `kiro-cli chat` with composed system prompt as context and task as user message. Capture output. Handle timeout and errors.
- **Test:** Manual test with kiro-cli installed (integration test tagged)
- **Status:** done

### Task 14: Clipboard fallback with composed prompt
- **Requirements:** AC-1.2
- **Files:** `internal/backend/clipboard.go`
- **Action:** Update clipboard backend to compose full prompt (steering + agent + task) before copying. Show a summary of what was copied.
- **Test:** Prompt copied matches expected composition
- **Status:** done

### Checkpoint: Backend
- [x] Kiro CLI backend deploys agent with composed prompt
- [x] Clipboard fallback copies full composed prompt
- [x] Backend auto-detection still works
- [x] Error handling for missing kiro-cli

---

## Milestone 5: Install Command

### Task 15: Install subcommand
- **Requirements:** AC-4.1, AC-4.2
- **Files:** `cmd/moonbase/install.go`
- **Action:** Implement `moonbase install` CLI subcommand. Creates `.kiro/agents/` in current directory. Copies selected agent `.md` files into it. Interactive selection (bubbletea list) when no `--all` flag.
- **Test:** Creates directory structure, copies files, respects selection
- **Status:** done

### Task 16: Wire install into main
- **Requirements:** AC-4.1
- **Files:** `cmd/moonbase/main.go`
- **Action:** Add `install` to CLI subcommand routing. Parse `--all` flag.
- **Test:** `moonbase install --help` works, `moonbase install --all` copies all agents
- **Status:** done

### Checkpoint: Install
- [x] `moonbase install` creates .kiro/agents/ with selected agents
- [x] `moonbase install --all` copies all 14 agents
- [x] Agents in .kiro/agents/ are usable by kiro-cli directly
- [x] Overwrite prompt when file exists

---

## Milestone 6: Integration & Polish

### Task 17: TUI pipeline view update
- **Requirements:** AC-2.1, AC-2.3
- **Files:** `internal/tui/views.go`
- **Action:** Update pipeline view to show real phase execution, context accumulation indicator, risk gate status, and conditional phase trigger/skip reasons.
- **Test:** Visual verification in TUI
- **Status:** done

### Task 18: End-to-end test
- **Requirements:** All
- **Files:** `internal/pipeline/integration_test.go`
- **Action:** Integration test that loads real agents, discovers a mock project, runs pipeline with mock backend, verifies full flow including risk gate.
- **Test:** `go test ./internal/pipeline/...`
- **Status:** done

### Task 19: Update docs
- **Requirements:** All
- **Files:** `docs/design.md`, `README.md`
- **Action:** Update docs to reflect implemented behaviour. Add CLI usage examples for `moonbase install`.
- **Test:** Docs match reality
- **Status:** done

### Final Checkpoint
- [x] All tasks complete
- [x] `go test ./...` passes (67 tests)
- [x] `go build ./...` succeeds
- [x] TUI launches and shows agents from .md files
- [x] `moonbase install --all` works
- [x] Pipeline executes with context accumulation and risk gate
- [x] No regressions in existing TUI functionality
