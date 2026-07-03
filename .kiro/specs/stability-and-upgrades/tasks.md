# Tasks: Stability & Feature Upgrades

## Milestone 1: Critical Stability (P0)

### Task 1: Move pipelineRunning into App struct
- **Requirements:** AC-1.1
- **Files:** `internal/tui/pipeline_exec.go`, `internal/tui/app.go`
- **Action:** Delete `var pipelineRunning bool` from `pipeline_exec.go:32`. Add `pipelineRunning bool` field to `App` struct. Replace all `pipelineRunning` references with `a.pipelineRunning`. Update `startNextPhase()` and `handlePhaseResult()` accordingly (they're already methods on `*App`).
- **Test:** `go build ./...` passes. `go test ./internal/tui/ -race` shows no race warnings. Verify with `grep -r "var pipelineRunning" internal/` returns nothing.
- **Status:** pending

### Task 2: Move streamCh into App struct
- **Requirements:** AC-1.2
- **Files:** `internal/tui/app.go`
- **Action:** Delete `var streamCh <-chan chat.StreamChunk` from line 1281. Add `streamCh <-chan chat.StreamChunk` field to `App` struct. Update all references (likely in the COMMS streaming logic) to use `a.streamCh`.
- **Test:** `go build ./...` passes. `grep -r "var streamCh" internal/` returns nothing. COMMS streaming still works in TUI.
- **Status:** pending

### Task 3: Fix CI lint step
- **Requirements:** AC-2.1
- **Files:** `.github/workflows/ci.yml`
- **Action:** Replace line 37: `run: go run ./cmd/moonbase lint 2>/dev/null || echo "lint command not yet wired"` with `run: go run ./cmd/moonbase lint`. Remove the error suppression entirely.
- **Test:** Push to a branch. CI lint job runs and passes (or fails meaningfully if an agent is broken).
- **Status:** pending

### Checkpoint: P0 Stability
- [ ] No package-level mutable `var` in `internal/tui/`
- [ ] CI lint step has no `|| echo` or `2>/dev/null`
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes

---

## Milestone 2: Code Health (P1)

### Task 4: Remove dead Pipeline Backend interface
- **Requirements:** AC-5.1
- **Files:** `internal/pipeline/pipeline.go`
- **Action:** Delete the unused `Backend` interface (lines 53-55): `type Backend interface { Deploy(agentName string, prompt string, input string) (string, error) }`. Also delete its doc comment.
- **Test:** `go build ./...` passes. `grep -n "type Backend interface" internal/pipeline/` returns nothing.
- **Status:** pending

### Task 5: Consolidate duplicate envExists
- **Requirements:** AC-5.2
- **Files:** `cmd/moonbase/main.go`, `internal/backend/backends.go`
- **Action:** Remove `envExists` from `cmd/moonbase/main.go`. In `runList()` and `runStatus()`, replace calls with `os.Getenv("KEY") != ""` directly (it's only used twice). Keep the one in `backends.go` if it's used internally there, or inline it too.
- **Test:** `go build ./...` passes. `grep -rn "func envExists" .` returns at most one result.
- **Status:** pending

### Task 6: Extract agentsDir to shared location
- **Requirements:** AC-4.1, AC-4.2
- **Files:** `internal/agents/resolve.go` (new), `cmd/moonbase/main.go`, `internal/tui/app.go`
- **Action:** Create `internal/agents/resolve.go` with `func FindAgentsDir(configPath string) (string, error)` containing the resolution logic from `agentsDir()` in main.go. Update main.go's `agentsDir()` to call `agents.FindAgentsDir("")`. Update `NewApp()` in app.go to call `agents.FindAgentsDir("")` instead of hardcoding `"./agents"`.
- **Test:** `go build ./...` passes. Create a temp dir, run `go run ./cmd/moonbase list` from it (should find agents via executable-relative path or error cleanly).
- **Status:** pending

### Task 7: Split app.go into per-view update files
- **Requirements:** AC-3.1, AC-3.2
- **Files:** `internal/tui/app.go`, `internal/tui/update_dashboard.go` (new), `internal/tui/update_comms.go` (new), `internal/tui/update_pipeline.go` (new), `internal/tui/update_mission.go` (new), `internal/tui/update_common.go` (new)
- **Action:** 
  1. Extract the `tea.KeyMsg` handling into `handleKeyMsg(msg tea.KeyMsg)` that switches on `a.view` and delegates to per-view handlers.
  2. Move dashboard key logic to `update_dashboard.go`
  3. Move COMMS key + stream logic to `update_comms.go`
  4. Move pipeline key + PhaseResultMsg logic to `update_pipeline.go`
  5. Move mission input logic to `update_mission.go`
  6. Move boot/resize/tick handlers to `update_common.go`
  7. Keep in `app.go`: struct definition, `NewApp()`, `Init()`, top-level `Update()` dispatch, `View()`
- **Test:** `go build ./...` passes. `go test ./internal/tui/...` passes. `wc -l internal/tui/app.go` shows ≤400 lines. TUI launches and all views work.
- **Status:** pending

### Checkpoint: P1 Code Health
- [ ] No dead `Backend` interface in pipeline package
- [ ] No duplicate `envExists`
- [ ] `NewApp()` uses `FindAgentsDir()`, not `"./agents"`
- [ ] `app.go` ≤ 400 lines
- [ ] All new `update_*.go` files compile and pass tests
- [ ] TUI works from a non-project directory (shows error or finds agents)

---

## Milestone 3: Error Handling & Accuracy (P2)

### Task 8: Fix writeTemplate error handling
- **Requirements:** AC-6.1
- **Files:** `cmd/moonbase/init.go`
- **Action:** Change `writeTemplate` signature to `func writeTemplate(path, content string) error`. Return `os.WriteFile` error. Update all callers in `runInit()` to check the error and print a warning (don't exit — try remaining files).
- **Test:** `go build ./...` passes. Manually test with a read-only directory to verify error message appears.
- **Status:** pending

### Task 9: Add snippet name validation
- **Requirements:** AC-6.2
- **Files:** `cmd/moonbase/main.go` (or `cmd/moonbase/snippet.go` after cobra migration)
- **Action:** Before saving, validate: `len(name) <= 100`, no path separators (`/`, `\`), no control characters. Reject with clear error message.
- **Test:** `go build ./...` passes. Attempt `moonbase snippet save "../../../../etc/passwd"` → rejected.
- **Status:** pending

### Task 10: Make runList dynamic from registry
- **Requirements:** AC-7.1
- **Files:** `cmd/moonbase/main.go` (or `cmd/moonbase/list.go` after cobra migration)
- **Action:** Replace hardcoded agent slices with `reg := agents.NewRegistry(agentsDir()); all := reg.All()`. Group by `PipelinePosition`. Fall back to the hardcoded list only if registry load fails (with a warning).
- **Test:** Add a test agent `.md` file, run `moonbase list`, verify it appears. Remove it, verify it disappears.
- **Status:** pending

### Task 11: Update docs/design.md
- **Requirements:** AC-8.1
- **Files:** `docs/design.md`
- **Action:** Rewrite the Architecture section to reflect actual file structure: flat `internal/tui/` with `app.go`, `views.go`, `styles.go`, `keys.go`, `pipeline_exec.go`, `update_*.go`. Remove references to `internal/tui/views/`, `internal/tui/components/`, `internal/agents/validator.go`, `internal/agents/loader.go`. Add the new files from this spec.
- **Test:** Every file path mentioned in docs/design.md exists: `for f in $(grep -oP 'internal/\S+\.go' docs/design.md); do test -f "$f" || echo "MISSING: $f"; done`
- **Status:** pending

### Checkpoint: P2 Accuracy
- [ ] `writeTemplate` errors are reported (not silent)
- [ ] Snippet names are validated
- [ ] `moonbase list` reflects actual agents directory
- [ ] `docs/design.md` references only files that exist
- [ ] `go build ./...` and `go test ./...` pass

---

## Milestone 4: Cobra CLI Migration (Feature)

### Task 12: Add cobra dependency
- **Requirements:** AC-10.1
- **Files:** `go.mod`
- **Action:** `go get github.com/spf13/cobra@v1.8.1`. Pin exact version.
- **Test:** `go mod tidy` succeeds. `go build ./...` still passes.
- **Status:** pending

### Task 13: Create root command with persistent flags
- **Requirements:** AC-10.1, AC-12.2
- **Files:** `cmd/moonbase/root.go` (new)
- **Action:** Create `rootCmd` with `Use: "moonbase"`, persistent `--debug` flag, and default run that launches TUI or pipe mode. Add `var (version, commit, date string)` for ldflags injection.
- **Test:** `go build ./...` passes. `./bin/moonbase --help` shows all subcommands.
- **Status:** pending

### Task 14: Migrate subcommands to cobra
- **Requirements:** AC-10.1, AC-10.2
- **Files:** `cmd/moonbase/deploy.go` (new), `cmd/moonbase/mission.go` (modify), `cmd/moonbase/list.go` (new), `cmd/moonbase/init_cmd.go` (new), `cmd/moonbase/status_cmd.go` (new), `cmd/moonbase/lint_cmd.go` (new), `cmd/moonbase/config_cmd.go` (new), `cmd/moonbase/snippet.go` (new)
- **Action:** Convert each `case` in the old `main()` switch to a cobra command. Deploy: `Args: cobra.MinimumNArgs(1)`. Mission: `Args: cobra.MinimumNArgs(1)`. Add flag definitions. Add shell completion command via `rootCmd.CompletionOptions`.
- **Test:** All commands work: `moonbase deploy 4`, `moonbase mission "test"`, `moonbase list`, etc. `moonbase completion zsh` generates output.
- **Status:** pending

### Task 15: Add version command with ldflags
- **Requirements:** AC-10.3
- **Files:** `cmd/moonbase/version.go` (new), `Makefile`, `.goreleaser.yml`
- **Action:** Create version command printing `version`, `commit`, `date`. Update Makefile build target with `-ldflags`. Update goreleaser with same ldflags.
- **Test:** `make build && ./bin/moonbase version` shows version info. Dev builds show "dev".
- **Status:** pending

### Task 16: Add --dry-run to mission command
- **Requirements:** AC-11.1
- **Files:** `cmd/moonbase/mission.go`
- **Action:** Add `--dry-run` bool flag. When set, call `runMissionDryRun(task)` that creates a pipeline, evaluates conditional triggers, and prints the plan without invoking any backend.
- **Test:** `moonbase mission --dry-run "add auth"` prints phases without calling AI. No backend errors even if no AI is available.
- **Status:** pending

### Task 17: Remove old main.go switch-case routing
- **Requirements:** AC-10.1
- **Files:** `cmd/moonbase/main.go`
- **Action:** Delete the entire `os.Args` switch block. `main()` becomes just `rootCmd.Execute()`. Move pipe-mode handling into root command's `RunE`. Delete `runHelp()` (cobra generates help automatically).
- **Test:** `go build ./...` passes. All commands still work. `moonbase help` shows cobra-generated output.
- **Status:** pending

### Checkpoint: Cobra Migration
- [ ] All existing commands work through cobra
- [ ] `moonbase --help` shows all subcommands with descriptions
- [ ] `moonbase version` prints version/commit/date
- [ ] `moonbase mission --dry-run "test"` shows pipeline plan
- [ ] `moonbase completion zsh` generates valid completion script
- [ ] Pipe mode still works: `echo "test" | moonbase`
- [ ] `go test ./...` passes

---

## Milestone 5: Feature Upgrades

### Task 18: Add glamour markdown rendering
- **Requirements:** AC-9.1, AC-9.2
- **Files:** `internal/tui/render.go` (new), `internal/tui/update_comms.go`, `internal/tui/update_pipeline.go`
- **Action:** Create `render.go` with `RenderMarkdown(md string, width int) string` using glamour with auto style and word wrap. In COMMS view, render agent responses through it before display. In pipeline view, render `PipelineMsg` content where `Agent != ""` (skip system messages). Handle glamour errors by falling back to raw text.
- **Test:** `go build ./...` passes. In COMMS, send a message and verify code blocks render with syntax highlighting. Verify system messages (✅, ❌) are NOT rendered through glamour.
- **Status:** pending

### Task 19: Add structured logging
- **Requirements:** AC-12.1, AC-12.2
- **Files:** `internal/logging/logger.go` (new), `cmd/moonbase/root.go`, `internal/tui/pipeline_exec.go`, `internal/backend/backends.go`
- **Action:** Create logging package with `Init(debug bool)` function. Call `logging.Init(debug)` early in root command's PersistentPreRun. Add `slog.Info/Error` calls at: phase start/complete/fail, backend selection, context discovery, abort events. Debug logs write to `~/.config/moonbase/debug.log`. Non-debug mode only logs WARN+.
- **Test:** `moonbase --debug mission "test"` creates debug.log with JSON entries. Normal mode produces no log file. `cat ~/.config/moonbase/debug.log | jq .` parses cleanly.
- **Status:** pending

### Task 20: Add graceful shutdown with context
- **Requirements:** AC-14.1, AC-14.2
- **Files:** `internal/tui/app.go`, `internal/tui/pipeline_exec.go`, `internal/tui/update_pipeline.go`
- **Action:** Add `pipelineCtx context.Context` and `cancelPipeline context.CancelFunc` to App struct. In `startMission()`, create context with cancel. Pass context to `executePhase()`. In abort handler (Esc during pipeline), call `cancelPipeline()`. In `executePhase`, add `case <-ctx.Done()` to the select. On TUI quit, cancel any active context.
- **Test:** Start a mission, press Esc — pipeline stops within 2 seconds. No goroutine leaks (verify with `-race` flag). Backend subprocess is killed.
- **Status:** pending

### Checkpoint: Features
- [ ] COMMS view renders markdown (code blocks, headers, lists visible)
- [ ] Pipeline chat renders agent output with markdown styling
- [ ] `moonbase --debug status` creates a debug.log file
- [ ] Esc during pipeline execution cleanly cancels within 5s
- [ ] No temp files leaked after abort
- [ ] `go test ./...` passes

---

## Milestone 6: DX & Polish (P3)

### Task 21: Add Makefile lint and coverage targets
- **Requirements:** AC-13.1, AC-13.2
- **Files:** `Makefile`
- **Action:** Add `lint:` target running `go vet ./...` and `go run ./cmd/moonbase lint`. Add `coverage:` target generating coverage.out, printing summary, and creating HTML report. Add both to `.PHONY`.
- **Test:** `make lint` passes. `make coverage` produces coverage.html and prints percentage.
- **Status:** pending

### Task 22: Update Makefile build with version ldflags
- **Requirements:** AC-10.3
- **Files:** `Makefile`
- **Action:** Update `build` target to include `-ldflags "-X main.version=dev -X main.commit=$(shell git rev-parse --short HEAD 2>/dev/null || echo none) -X main.date=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)"`.
- **Test:** `make build && ./bin/moonbase version` shows commit hash and date.
- **Status:** pending

### Checkpoint: DX Polish
- [ ] `make lint` works
- [ ] `make coverage` produces HTML report
- [ ] `make build` injects version info
- [ ] All CI checks pass

---

## Final Verification

- [ ] `go build ./...` — clean
- [ ] `go test ./...` — all pass
- [ ] `go vet ./...` — clean
- [ ] `go test ./... -race` — no race conditions
- [ ] `wc -l internal/tui/app.go` ≤ 400
- [ ] `grep -r "var pipelineRunning\|var streamCh" internal/` — no results
- [ ] `moonbase version` — shows version
- [ ] `moonbase list` — dynamic from agents directory
- [ ] `moonbase mission --dry-run "test"` — shows plan without execution
- [ ] `moonbase --debug status` — creates debug.log
- [ ] CI passes on push
