package tui

import (
	"context"
	"fmt"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rk-senne/moonbase/internal/agents"
	"github.com/rk-senne/moonbase/internal/chat"
	"github.com/rk-senne/moonbase/internal/discovery"
	"github.com/rk-senne/moonbase/internal/pipeline"
)

// mockBackend implements backend.Backend for testing executePhase.
type mockBackend struct {
	name      string
	available bool
	output    string
	err       error
	delay     time.Duration
}

func (m *mockBackend) Name() string      { return m.name }
func (m *mockBackend) Available() bool    { return m.available }
func (m *mockBackend) Deploy(agent agents.Agent, ctx *discovery.ProjectContext, task string) (string, error) {
	if m.delay > 0 {
		time.Sleep(m.delay)
	}
	return m.output, m.err
}

// === animTick / clockTick / blinkTick / bootDone ===

func TestAnimTick_ReturnsCmd(t *testing.T) {
	cmd := animTick()
	if cmd == nil {
		t.Fatal("expected non-nil cmd from animTick")
	}
}

func TestClockTick_ReturnsCmd(t *testing.T) {
	cmd := clockTick()
	if cmd == nil {
		t.Fatal("expected non-nil cmd from clockTick")
	}
}

func TestBlinkTick_ReturnsCmd(t *testing.T) {
	cmd := blinkTick()
	if cmd == nil {
		t.Fatal("expected non-nil cmd from blinkTick")
	}
}

func TestBootDone_ReturnsMsg(t *testing.T) {
	cmd := bootDone()
	if cmd == nil {
		t.Fatal("expected non-nil cmd from bootDone")
	}
}

func TestBootTick_ReturnsCmd(t *testing.T) {
	cmd := bootTick()
	if cmd == nil {
		t.Fatal("expected non-nil cmd from bootTick")
	}
}

// === Init ===

func TestInit_ReturnsBatchCmd(t *testing.T) {
	app := NewApp()
	cmd := app.Init()
	if cmd == nil {
		t.Fatal("expected non-nil batch cmd from Init")
	}
}

// === detectSystem ===

func TestDetectSystem_ReturnsCmd(t *testing.T) {
	cmd := detectSystem()
	if cmd == nil {
		t.Fatal("expected non-nil cmd from detectSystem")
	}
	// Execute the cmd to get the systemInfoMsg
	msg := cmd()
	info, ok := msg.(systemInfoMsg)
	if !ok {
		t.Fatalf("expected systemInfoMsg, got %T", msg)
	}
	// branch may be empty in CI (detached HEAD from actions/checkout)
	// just verify the function executed without panic and returned valid msg
	_ = info.branch
}

// === pollStream ===

func TestPollStream_ReceivesChunk(t *testing.T) {
	ch := make(chan chat.StreamChunk, 1)
	ch <- chat.StreamChunk{Text: "hello", Done: false}

	cmd := pollStream(ch)
	if cmd == nil {
		t.Fatal("expected non-nil cmd from pollStream")
	}
	msg := cmd()
	chunk, ok := msg.(streamChunkMsg)
	if !ok {
		t.Fatalf("expected streamChunkMsg, got %T", msg)
	}
	if chunk.text != "hello" {
		t.Errorf("expected text='hello', got %q", chunk.text)
	}
	if chunk.done {
		t.Error("expected done=false")
	}
}

func TestPollStream_ClosedChannel(t *testing.T) {
	ch := make(chan chat.StreamChunk)
	close(ch)

	cmd := pollStream(ch)
	msg := cmd()
	chunk, ok := msg.(streamChunkMsg)
	if !ok {
		t.Fatalf("expected streamChunkMsg, got %T", msg)
	}
	if !chunk.done {
		t.Error("expected done=true for closed channel")
	}
}

func TestPollStream_WithError(t *testing.T) {
	ch := make(chan chat.StreamChunk, 1)
	ch <- chat.StreamChunk{Err: fmt.Errorf("api error"), Done: false}

	cmd := pollStream(ch)
	msg := cmd()
	chunk := msg.(streamChunkMsg)
	if chunk.err == nil {
		t.Error("expected non-nil error in stream chunk")
	}
}

// === pollAgentDir ===

func TestPollAgentDir_ReturnsCmd(t *testing.T) {
	app := NewApp()
	cmd := app.pollAgentDir()
	if cmd == nil {
		t.Fatal("expected non-nil cmd from pollAgentDir")
	}
}

// === pollWatcher ===

func TestPollWatcher_NilWatcher(t *testing.T) {
	app := NewApp()
	app.infra.Watcher = nil
	cmd := app.pollWatcher()
	if cmd != nil {
		t.Error("expected nil cmd when fileWatcher is nil")
	}
}

// === executePhase ===

func TestExecutePhase_AgentNotFound(t *testing.T) {
	reg := agents.NewRegistry("/nonexistent")
	// Don't load anything — registry is empty

	phase := pipeline.Phase{
		Number:    1,
		Name:      "Analysis",
		AgentName: "numbuh-1",
		Operative: "Numbuh 1",
	}
	be := &mockBackend{name: "test", available: true, output: "done"}
	pctx := pipeline.NewPipelineContext("test task")

	cmd := executePhase(context.Background(), phase, reg, be, nil, pctx, 120*time.Second)
	if cmd == nil {
		t.Fatal("expected non-nil cmd")
	}
	msg := cmd()
	result, ok := msg.(PhaseResultMsg)
	if !ok {
		t.Fatalf("expected PhaseResultMsg, got %T", msg)
	}
	if result.Err == nil {
		t.Error("expected error for missing agent")
	}
	if result.Phase != 1 {
		t.Errorf("expected phase=1, got %d", result.Phase)
	}
}

func TestExecutePhase_Success(t *testing.T) {
	reg := newTestRegistry()

	phase := pipeline.Phase{
		Number:    1,
		Name:      "Analysis",
		AgentName: "numbuh-1",
		Operative: "Numbuh 1",
	}
	be := &mockBackend{name: "test-backend", available: true, output: "Requirements gathered successfully."}
	pctx := pipeline.NewPipelineContext("add pagination")

	cmd := executePhase(context.Background(), phase, reg, be, nil, pctx, 120*time.Second)
	msg := cmd()
	result := msg.(PhaseResultMsg)

	if result.Err != nil {
		t.Errorf("expected no error, got %v", result.Err)
	}
	if result.Output != "Requirements gathered successfully." {
		t.Errorf("unexpected output: %s", result.Output)
	}
	if result.Phase != 1 {
		t.Errorf("expected phase=1, got %d", result.Phase)
	}
	if result.Elapsed <= 0 {
		t.Error("expected positive elapsed time")
	}
}

func TestExecutePhase_BackendError(t *testing.T) {
	reg := newTestRegistry()

	phase := pipeline.Phase{
		Number:    2,
		Name:      "Architecture",
		AgentName: "numbuh-2",
		Operative: "Numbuh 2",
	}
	be := &mockBackend{name: "clipboard", available: true, err: fmt.Errorf("clipboard failed")}
	pctx := pipeline.NewPipelineContext("test")

	cmd := executePhase(context.Background(), phase, reg, be, nil, pctx, 120*time.Second)
	msg := cmd()
	result := msg.(PhaseResultMsg)

	if result.Err == nil {
		t.Error("expected error from backend failure")
	}
}

func TestExecutePhase_ContextCancelled(t *testing.T) {
	reg := newTestRegistry()

	phase := pipeline.Phase{
		Number:    1,
		Name:      "Analysis",
		AgentName: "numbuh-1",
		Operative: "Numbuh 1",
	}
	be := &mockBackend{name: "slow", available: true, output: "done", delay: 5 * time.Second}
	pctx := pipeline.NewPipelineContext("test")

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	cmd := executePhase(ctx, phase, reg, be, nil, pctx, 120*time.Second)
	msg := cmd()
	result := msg.(PhaseResultMsg)

	if result.Err == nil {
		t.Error("expected error from cancelled context")
	}
}

// === startNextPhase ===

func TestStartNextPhase_NilPipelineState(t *testing.T) {
	app := NewApp()
	app.pipeline.State = nil
	app.pipeline.Running = true

	cmd := app.startNextPhase()
	if cmd != nil {
		t.Error("expected nil cmd for nil pipelineState")
	}
	if app.pipeline.Running {
		t.Error("expected pipelineRunning=false")
	}
}

func TestStartNextPhase_InactivePipeline(t *testing.T) {
	app := NewApp()
	app.pipeline.State = pipeline.New("test")
	app.pipeline.State.Active = false

	cmd := app.startNextPhase()
	if cmd != nil {
		t.Error("expected nil cmd for inactive pipeline")
	}
}

func TestStartNextPhase_ClipboardBackend(t *testing.T) {
	app := NewApp()
	app.pipeline.State = pipeline.New("test")
	app.backend.Active = &mockBackend{name: "clipboard", available: true}

	cmd := app.startNextPhase()
	if cmd != nil {
		t.Error("expected nil cmd for clipboard backend (simulated mode)")
	}
	if app.pipeline.Running {
		t.Error("expected pipelineRunning=false for clipboard backend")
	}
}

func TestStartNextPhase_NilBackend(t *testing.T) {
	app := NewApp()
	app.pipeline.State = pipeline.New("test")
	app.backend.Active = nil

	cmd := app.startNextPhase()
	if cmd != nil {
		t.Error("expected nil cmd for nil backend")
	}
}

func TestStartNextPhase_PipelineComplete(t *testing.T) {
	app := NewApp()
	app.pipeline.State = pipeline.New("test")
	app.backend.Active = &mockBackend{name: "real", available: true}
	// Move past all phases
	app.pipeline.State.Current = len(app.pipeline.State.Phases)

	cmd := app.startNextPhase()
	if cmd != nil {
		t.Error("expected nil cmd when past all phases")
	}
}

func TestStartNextPhase_ConditionalSkipped(t *testing.T) {
	app := NewApp()
	app.pipeline.State = pipeline.New("test")
	app.backend.Active = &mockBackend{name: "real", available: true}
	// Move to a conditional phase (index 5 = Oversight)
	app.pipeline.State.Current = 5

	cmd := app.startNextPhase()
	// Conditional phases are skipped when no trigger conditions met
	// With no real backend execution happening, we just check it doesn't crash
	// The function should recurse through conditional phases
	_ = cmd
}

func TestStartNextPhase_RealBackendExecutes(t *testing.T) {
	app := NewApp()
	app.pipeline.State = pipeline.New("test")
	app.backend.Active = &mockBackend{name: "test-backend", available: true, output: "done"}
	app.registry = newTestRegistry()

	cmd := app.startNextPhase()
	if cmd == nil {
		t.Error("expected non-nil cmd for real backend with pending phase")
	}
	if !app.pipeline.Running {
		t.Error("expected pipelineRunning=true when phase starts")
	}
}

// === handlePhaseResult additional cases ===

func TestHandlePhaseResult_RiskGate_High(t *testing.T) {
	app := NewApp()
	app.view = ViewPipeline
	app.boot.Ready = true
	app.pipeline.State = pipeline.New("test task")
	for i := 0; i < 3; i++ {
		app.pipeline.State.Advance()
	}
	app.pipeline.State.Phases[3].Status = pipeline.StatusRunning
	app.pipeline.Running = true

	msg := PhaseResultMsg{
		Phase:   4,
		Output:  "## Verdict\nHIGH\n\nMajor architectural issues found.",
		Err:     nil,
		Elapsed: 3 * time.Second,
	}

	model, _ := app.Update(msg)
	result := model.(App)

	if result.pipeline.State.Context.RiskLevel != "HIGH" {
		t.Errorf("expected risk level HIGH, got %s", result.pipeline.State.Context.RiskLevel)
	}
}

func TestHandlePhaseResult_PipelineComplete(t *testing.T) {
	app := NewApp()
	app.view = ViewPipeline
	app.boot.Ready = true
	app.pipeline.State = pipeline.New("test task")
	// Advance to the last mandatory phase (Review = phase 5, index 4)
	for i := 0; i < 4; i++ {
		app.pipeline.State.Advance()
	}
	app.pipeline.State.Phases[4].Status = pipeline.StatusRunning
	app.pipeline.Running = true
	app.backend.Active = nil // no backend means it won't try to execute next

	msg := PhaseResultMsg{
		Phase:   5,
		Output:  "Review complete. All good.",
		Err:     nil,
		Elapsed: 1 * time.Second,
	}

	cmd := app.handlePhaseResult(msg)
	// After phase 5, pipeline advances and conditional phases may be skipped
	// resulting in pipeline completion
	_ = cmd
	if app.pipeline.Running {
		t.Error("expected pipelineRunning=false after completion")
	}
}

func TestHandlePhaseResult_ErrorSetsFailedStatus(t *testing.T) {
	app := NewApp()
	app.pipeline.State = pipeline.New("test")
	app.pipeline.State.Phases[0].Status = pipeline.StatusRunning
	app.pipeline.Running = true

	msg := PhaseResultMsg{
		Phase:   1,
		Err:     fmt.Errorf("timeout"),
		Elapsed: 120 * time.Second,
	}

	cmd := app.handlePhaseResult(msg)
	if cmd != nil {
		t.Error("expected nil cmd on error")
	}
	if app.pipeline.State.Phases[0].Status != pipeline.StatusFailed {
		t.Errorf("expected StatusFailed, got %d", app.pipeline.State.Phases[0].Status)
	}
}

// === handleDocsKeys ===

func TestHandleDocsKeys_Esc(t *testing.T) {
	app := NewApp()
	app.view = ViewDocs
	app.boot.Ready = true
	app.docs = newDocsState(100, 40)

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyEscape})
	result := model.(App)
	if result.view != ViewDashboard {
		t.Errorf("expected ViewDashboard after esc, got %d", result.view)
	}
}

func TestHandleDocsKeys_NilDocs(t *testing.T) {
	app := NewApp()
	app.view = ViewDocs
	app.boot.Ready = true
	app.docs = nil

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	result := model.(App)
	_ = result // should not panic
}

func TestHandleDocsKeys_Navigate(t *testing.T) {
	app := NewApp()
	app.view = ViewDocs
	app.boot.Ready = true
	app.docs = newDocsState(100, 40)

	// Navigate down
	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	result := model.(App)
	if result.docs != nil && len(result.docs.files) > 1 {
		if result.docs.cursor != 1 {
			t.Errorf("expected cursor=1 after 'j', got %d", result.docs.cursor)
		}
	}

	// Navigate up
	model, _ = result.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	result = model.(App)
	if result.docs != nil {
		if result.docs.cursor != 0 {
			t.Errorf("expected cursor=0 after 'k', got %d", result.docs.cursor)
		}
	}
}

func TestHandleDocsKeys_Enter(t *testing.T) {
	app := NewApp()
	app.view = ViewDocs
	app.boot.Ready = true
	app.width = 100
	app.height = 40
	app.docs = newDocsState(100, 40)

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	result := model.(App)
	// Should load the doc (or not crash if no files)
	_ = result
}

func TestHandleDocsKeys_PageDown(t *testing.T) {
	app := NewApp()
	app.view = ViewDocs
	app.boot.Ready = true
	app.docs = newDocsState(100, 40)

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	result := model.(App)
	_ = result // should not panic
}

func TestHandleDocsKeys_PageUp(t *testing.T) {
	app := NewApp()
	app.view = ViewDocs
	app.boot.Ready = true
	app.docs = newDocsState(100, 40)

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyPgUp})
	result := model.(App)
	_ = result
}

// === renderDocs ===

func TestRenderDocs_NilDocs(t *testing.T) {
	app := NewApp()
	app.boot.Ready = true
	app.width = 100
	app.height = 40
	app.view = ViewDocs
	app.docs = nil

	result := app.renderDocs()
	if result == "" {
		t.Error("expected non-empty renderDocs output for nil docs")
	}
}

func TestRenderDocs_EmptyFiles(t *testing.T) {
	app := NewApp()
	app.boot.Ready = true
	app.width = 100
	app.height = 40
	app.view = ViewDocs
	app.docs = &DocsState{files: nil}

	result := app.renderDocs()
	if result == "" {
		t.Error("expected non-empty renderDocs output for empty files")
	}
}

func TestRenderDocs_WithFiles(t *testing.T) {
	app := NewApp()
	app.boot.Ready = true
	app.width = 100
	app.height = 40
	app.view = ViewDocs
	app.docs = newDocsState(100, 40)

	result := app.renderDocs()
	if result == "" {
		t.Error("expected non-empty renderDocs output")
	}
}

// === loadDoc ===

func TestLoadDoc_InvalidIndex(t *testing.T) {
	ds := newDocsState(100, 40)
	ds.loadDoc(-1, 70)
	ds.loadDoc(9999, 70)
	// Should not panic
}

func TestLoadDoc_ValidIndex(t *testing.T) {
	ds := newDocsState(100, 40)
	if len(ds.files) > 0 {
		ds.loadDoc(0, 70)
		if !ds.loaded {
			t.Error("expected loaded=true after loadDoc")
		}
		if ds.content == "" {
			t.Error("expected non-empty content after loadDoc")
		}
	}
}

// === runGitCmd ===

func TestRunGitCmd_ReturnsCmd(t *testing.T) {
	app := NewApp()
	cmd := app.runGitCmd("git status --short")
	if cmd == nil {
		t.Fatal("expected non-nil cmd from runGitCmd")
	}
	msg := cmd()
	result, ok := msg.(gitOutputMsg)
	if !ok {
		t.Fatalf("expected gitOutputMsg, got %T", msg)
	}
	// output should be non-empty (clean = "(clean — no output)")
	if result.output == "" {
		t.Error("expected non-empty git output")
	}
}

func TestRunGitCmd_InvalidCommand(t *testing.T) {
	app := NewApp()
	cmd := app.runGitCmd("git invalid-subcommand-xyz")
	msg := cmd()
	result := msg.(gitOutputMsg)
	// Should contain "failed" since the command doesn't exist
	if result.output == "" {
		t.Error("expected non-empty output for failed git command")
	}
}

// === execTermCmd ===

func TestExecTermCmd_CdHome(t *testing.T) {
	app := NewApp()
	cmd := app.execTermCmd("cd ~")
	// cd is handled inline, returns nil
	if cmd != nil {
		t.Error("expected nil cmd for cd (handled inline)")
	}
}

func TestExecTermCmd_CdInvalid(t *testing.T) {
	app := NewApp()
	cmd := app.execTermCmd("cd /nonexistent_path_xyz_12345")
	if cmd != nil {
		t.Error("expected nil cmd for cd (even invalid)")
	}
	// Should have error in termOutput
}

func TestExecTermCmd_Clear(t *testing.T) {
	app := NewApp()
	app.terminal.Output = []string{"line1", "line2"}
	cmd := app.execTermCmd("clear")
	if cmd != nil {
		t.Error("expected nil cmd for clear")
	}
	if len(app.terminal.Output) != 0 {
		t.Errorf("expected cleared terminal output, got %d lines", len(app.terminal.Output))
	}
}

func TestExecTermCmd_Echo(t *testing.T) {
	app := NewApp()
	cmd := app.execTermCmd("echo hello")
	if cmd == nil {
		t.Fatal("expected non-nil cmd for echo")
	}
	msg := cmd()
	result, ok := msg.(termOutputMsg)
	if !ok {
		t.Fatalf("expected termOutputMsg, got %T", msg)
	}
	if result.cmd != "echo hello" {
		t.Errorf("expected cmd='echo hello', got %q", result.cmd)
	}
	// Output should contain "hello" (may have errors in constrained environments)
	if result.output == "" {
		t.Error("expected non-empty output")
	}
}

// === runSpawnHook ===

func TestRunSpawnHook_NoHook(t *testing.T) {
	app := NewApp()
	app.registry = newTestRegistry()
	app.dashboard.Selected = 0

	cmd := app.runSpawnHook()
	if cmd == nil {
		t.Fatal("expected non-nil cmd")
	}
	msg := cmd()
	result, ok := msg.(spawnHookMsg)
	if !ok {
		t.Fatalf("expected spawnHookMsg, got %T", msg)
	}
	// Most agents don't have hooks
	_ = result
}

// === copyPrompt ===

func TestCopyPrompt_ReturnsCmd(t *testing.T) {
	app := NewApp()
	app.registry = newTestRegistry()
	app.dashboard.Selected = 0

	cmd := app.copyPrompt()
	if cmd == nil {
		t.Fatal("expected non-nil cmd from copyPrompt")
	}
	// Execute — on macOS with pbcopy available this should work
	msg := cmd()
	result, ok := msg.(copyDoneMsg)
	if !ok {
		t.Fatalf("expected copyDoneMsg, got %T", msg)
	}
	if result.agent == "" {
		t.Error("expected non-empty agent name")
	}
}

// === deployAgent ===

func TestDeployAgent_ReturnsCmd(t *testing.T) {
	app := NewApp()
	app.registry = newTestRegistry()
	app.dashboard.Selected = 0

	cmd := app.deployAgent()
	if cmd == nil {
		t.Fatal("expected non-nil cmd from deployAgent")
	}
	msg := cmd()
	result, ok := msg.(deployDoneMsg)
	if !ok {
		t.Fatalf("expected deployDoneMsg, got %T", msg)
	}
	if result.agent == "" {
		t.Error("expected non-empty agent name")
	}
}

// === launchTool ===

func TestLaunchTool_NotFound(t *testing.T) {
	app := NewApp()
	cmd := app.launchTool("nonexistent_tool_xyz_999")
	if cmd == nil {
		t.Fatal("expected non-nil cmd")
	}
	msg := cmd()
	result, ok := msg.(toolExitMsg)
	if !ok {
		t.Fatalf("expected toolExitMsg, got %T", msg)
	}
	if result.tool == "" {
		t.Error("expected non-empty tool name")
	}
}

// === launchNvim ===

func TestLaunchNvim_NotFound(t *testing.T) {
	// If nvim is not available, should return a "not found" cmd
	app := NewApp()
	cmd := app.launchNvim()
	if cmd == nil {
		t.Fatal("expected non-nil cmd from launchNvim")
	}
	// We can't execute tea.ExecProcess in tests, just verify non-nil
}

// === launchCmux ===

func TestLaunchCmux_ReturnsCmd(t *testing.T) {
	app := NewApp()
	cmd := app.launchCmux()
	if cmd == nil {
		t.Fatal("expected non-nil cmd from launchCmux")
	}
}

// === createPR ===

func TestCreatePR_ReturnsCmd(t *testing.T) {
	app := NewApp()
	cmd := app.createPR()
	if cmd == nil {
		t.Fatal("expected non-nil cmd from createPR")
	}
	msg := cmd()
	result, ok := msg.(prCreatedMsg)
	if !ok {
		t.Fatalf("expected prCreatedMsg, got %T", msg)
	}
	// Either gh not found or on main/master — both produce output
	if result.output == "" {
		t.Error("expected non-empty PR output")
	}
}

// === sendCommsMessage ===

func TestSendCommsMessage_EmptyInput(t *testing.T) {
	app := NewApp()
	app.comms = newCommsState("test-agent", "prompt", 80, 40)
	app.commsInput.SetValue("")

	cmd := app.sendCommsMessage()
	if cmd != nil {
		t.Error("expected nil cmd for empty message")
	}
}

func TestSendCommsMessage_WithMessage(t *testing.T) {
	app := NewApp()
	app.comms = newCommsState("test-agent", "prompt", 80, 40)
	app.commsInput.SetValue("Hello agent")

	cmd := app.sendCommsMessage()
	// cmd may be nil if ANTHROPIC_API_KEY is not set, but message should be added
	if app.commsInput.Value() != "" {
		t.Error("expected commsInput to be reset")
	}
	if !app.comms.streaming {
		t.Error("expected streaming=true after sendCommsMessage")
	}
	// Verify user message was added
	found := false
	for _, m := range app.comms.conv.Messages {
		if m.Role == chat.RoleUser && m.Content == "Hello agent" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected user message to be added to conversation")
	}
	// Should have a stream channel
	if app.pipeline.StreamCh == nil {
		t.Error("expected streamCh to be set")
	}
	if cmd == nil {
		t.Error("expected non-nil cmd (pollStream)")
	}
}

// === relayToAgent ===

func TestRelayToAgent_TargetNotFound(t *testing.T) {
	app := NewApp()
	app.registry = newTestRegistry()
	app.width = 100
	app.height = 40
	app.view = ViewComms
	app.comms = newCommsState("numbuh-1", "prompt", 80, 40)
	app.commsInput.Focus()
	app.commsInput.SetValue(">>nonexistent-agent-xyz hello")

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	result := model.(App)
	// Should not crash, agent stays numbuh-1
	if result.comms.agent != "numbuh-1" {
		t.Errorf("expected comms to stay as numbuh-1, got %s", result.comms.agent)
	}
}

func TestRelayToAgent_WithExplicitMessage(t *testing.T) {
	app := NewApp()
	app.registry = newTestRegistry()
	app.width = 100
	app.height = 40
	app.view = ViewComms
	app.comms = newCommsState("numbuh-1", "prompt", 80, 40)
	app.commsInput.Focus()
	app.commsInput.SetValue(">>numbuh-2 analyze this code")

	model, cmd := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	result := model.(App)

	// If registry failed to load agents (e.g., fd exhaustion), skip assertions
	if app.registry.Count() == 0 {
		t.Skip("registry empty (likely fd exhaustion)")
	}
	if result.comms.agent != "numbuh-2" {
		t.Errorf("expected comms switched to numbuh-2, got %s", result.comms.agent)
	}
	if !result.comms.streaming {
		t.Error("expected streaming=true after relay")
	}
	if cmd == nil {
		t.Error("expected non-nil cmd from relay")
	}
}

func TestRelayToAgent_EmptyMsgNoHistory(t *testing.T) {
	app := NewApp()
	app.registry = newTestRegistry()
	app.width = 100
	app.height = 40
	app.view = ViewComms
	app.comms = newCommsState("numbuh-1", "prompt", 80, 40)
	// No messages in conversation — relay with empty msg should fail
	app.commsInput.Focus()
	app.commsInput.SetValue(">numbuh-2")

	model, cmd := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	result := model.(App)
	_ = result
	// With no assistant message to relay, cmd should be nil
	_ = cmd
}

func TestRelayToAgent_EmptyMsgWithHistory(t *testing.T) {
	app := NewApp()
	app.registry = newTestRegistry()
	app.width = 100
	app.height = 40
	app.view = ViewComms
	app.comms = newCommsState("numbuh-1", "prompt", 80, 40)
	// Add an assistant message to relay
	app.comms.conv.Add(chat.RoleAssistant, "previous response")
	app.commsInput.Focus()
	app.commsInput.SetValue(">numbuh-2")

	model, cmd := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	result := model.(App)

	// If registry failed to load agents (e.g., fd exhaustion), skip assertions
	if app.registry.Count() == 0 {
		t.Skip("registry empty (likely fd exhaustion)")
	}
	if result.comms.agent != "numbuh-2" {
		t.Errorf("expected comms switched to numbuh-2, got %s", result.comms.agent)
	}
	if cmd == nil {
		t.Error("expected non-nil cmd from relay with history")
	}
}

// === renderHistory ===

func TestRenderHistory_Full(t *testing.T) {
	app := NewApp()
	app.boot.Ready = true
	app.width = 100
	app.height = 40
	app.view = ViewHistory

	result := app.renderHistory()
	if result == "" {
		t.Error("expected non-empty renderHistory output")
	}
}

// === renderPipeline ===

func TestRenderPipeline_WithState(t *testing.T) {
	app := NewApp()
	app.boot.Ready = true
	app.width = 100
	app.height = 40
	app.view = ViewPipeline
	app.pipeline.State = pipeline.New("test mission")
	app.pipeline.Chat = []PipelineMsg{
		{"", "━━━ MISSION: test mission ━━━"},
		{"Numbuh 1", "Starting analysis..."},
	}

	result := app.renderPipeline()
	if result == "" {
		t.Error("expected non-empty renderPipeline output")
	}
}

func TestRenderPipeline_NilState(t *testing.T) {
	app := NewApp()
	app.boot.Ready = true
	app.width = 100
	app.height = 40
	app.view = ViewPipeline
	app.pipeline.State = nil

	result := app.renderPipeline()
	if result == "" {
		t.Error("expected non-empty renderPipeline output even with nil state")
	}
}

func TestRenderPipeline_AbortPending(t *testing.T) {
	app := NewApp()
	app.boot.Ready = true
	app.width = 100
	app.height = 40
	app.view = ViewPipeline
	app.pipeline.State = pipeline.New("test")
	app.pipeline.Running = true
	app.pipeline.AbortPending = true
	app.pipeline.AbortAt = time.Now()

	result := app.renderPipeline()
	if result == "" {
		t.Error("expected non-empty pipeline render with abort pending")
	}
}

// === selectProject ===

func TestSelectProject_WithDocs(t *testing.T) {
	app := NewApp()
	app.width = 100
	app.height = 40
	app.projectNav = &ProjectsState{list: nil, cursor: 0}

	// Should not panic with nil list
	app.selectProject()
}

// === Update message types: clockTickMsg, blinkTickMsg, animTickMsg ===

func TestUpdate_ClockTickMsg(t *testing.T) {
	app := NewApp()
	app.boot.Ready = true
	app.view = ViewDashboard

	msg := clockTickMsg(time.Now())
	model, _ := app.Update(msg)
	result := model.(App)
	if result.chrome.Clock == "" {
		t.Error("expected clock to be updated")
	}
}

func TestUpdate_BlinkTickMsg(t *testing.T) {
	app := NewApp()
	app.boot.Ready = true
	app.view = ViewDashboard
	app.chrome.Blink = false

	msg := blinkTickMsg(time.Now())
	model, _ := app.Update(msg)
	result := model.(App)
	if !result.chrome.Blink {
		t.Error("expected blink to toggle to true")
	}
}

func TestUpdate_AnimTickMsg(t *testing.T) {
	app := NewApp()
	app.boot.Ready = true
	app.view = ViewDashboard
	initialFrame := app.chrome.Anim.frame

	msg := animTickMsg(time.Now())
	model, _ := app.Update(msg)
	result := model.(App)
	if result.chrome.Anim.frame != initialFrame+1 {
		t.Errorf("expected frame=%d, got %d", initialFrame+1, result.chrome.Anim.frame)
	}
}

func TestUpdate_CopyDoneMsg(t *testing.T) {
	app := NewApp()
	app.boot.Ready = true
	app.view = ViewDossier

	msg := copyDoneMsg{agent: "numbuh-1"}
	model, _ := app.Update(msg)
	result := model.(App)
	if result.view != ViewDashboard {
		t.Errorf("expected ViewDashboard after copyDone, got %d", result.view)
	}
}

func TestUpdate_DeployDoneMsg(t *testing.T) {
	app := NewApp()
	app.boot.Ready = true
	app.view = ViewDashboard

	msg := deployDoneMsg{agent: "numbuh-3"}
	model, _ := app.Update(msg)
	result := model.(App)
	// Should add intel entry
	found := false
	for _, entry := range result.intel {
		if entry.Message != "" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected intel entry for deploy done")
	}
}

func TestUpdate_GitOutputMsg(t *testing.T) {
	app := NewApp()
	app.boot.Ready = true
	app.view = ViewDashboard

	msg := gitOutputMsg{output: "M internal/tui/app.go\nM internal/tui/views.go"}
	model, _ := app.Update(msg)
	result := model.(App)
	if len(result.intel) == 0 {
		t.Error("expected intel entries for git output")
	}
}

func TestUpdate_SpawnHookMsg(t *testing.T) {
	app := NewApp()
	app.boot.Ready = true

	msg := spawnHookMsg{agent: "numbuh-4", output: "hook ran ok"}
	model, _ := app.Update(msg)
	result := model.(App)
	if len(result.intel) == 0 {
		t.Error("expected intel entries for spawn hook")
	}
}

func TestUpdate_ToolExitMsg(t *testing.T) {
	app := NewApp()
	app.boot.Ready = true

	msg := toolExitMsg{tool: "lazygit"}
	model, _ := app.Update(msg)
	result := model.(App)
	if len(result.intel) == 0 {
		t.Error("expected intel entry for tool exit")
	}
}

func TestUpdate_PRCreatedMsg(t *testing.T) {
	app := NewApp()
	app.boot.Ready = true

	msg := prCreatedMsg{output: "https://github.com/example/pr/1"}
	model, _ := app.Update(msg)
	result := model.(App)
	if len(result.intel) == 0 {
		t.Error("expected intel entry for PR created")
	}
}

func TestUpdate_AgentReloadMsg(t *testing.T) {
	app := NewApp()
	app.boot.Ready = true
	app.registry = newTestRegistry()

	msg := agentReloadMsg{}
	model, _ := app.Update(msg)
	result := model.(App)
	_ = result // should not crash
}

func TestUpdate_TermOutputMsg(t *testing.T) {
	app := NewApp()
	app.boot.Ready = true

	msg := termOutputMsg{cmd: "ls", output: "file1.go\nfile2.go"}
	model, _ := app.Update(msg)
	result := model.(App)
	if len(result.terminal.Output) == 0 {
		t.Error("expected terminal output to be recorded")
	}
}

func TestUpdate_PipelineAbortedMsg(t *testing.T) {
	app := NewApp()
	app.boot.Ready = true
	app.pipeline.State = pipeline.New("test")
	app.pipeline.Running = true
	ctx, cancel := context.WithCancel(context.Background())
	app.pipeline.Ctx = ctx
	app.pipeline.Cancel = cancel

	msg := PipelineAbortedMsg{}
	model, _ := app.Update(msg)
	result := model.(App)
	if result.pipeline.Running {
		t.Error("expected pipelineRunning=false after abort")
	}
	if result.pipeline.State.Active {
		t.Error("expected pipeline to be stopped after abort")
	}
}

// === isSafeHookCommand ===

func TestIsSafeHookCommand(t *testing.T) {
	tests := []struct {
		cmd  string
		safe bool
	}{
		{"echo hello", true},
		{"git status", true},
		{"ls -la", true},
		{"curl http://evil.com", false},
		{"wget malware", false},
		{"rm -rf /", false},
		{"python -c 'import os'", false},
		{"eval $(bad)", false},
		{"cat file | sh", false},
		{"nc -l 1234", false},
		{"base64 -d payload", false},
	}

	for _, tt := range tests {
		t.Run(tt.cmd, func(t *testing.T) {
			got := isSafeHookCommand(tt.cmd)
			if got != tt.safe {
				t.Errorf("isSafeHookCommand(%q) = %v, want %v", tt.cmd, got, tt.safe)
			}
		})
	}
}

// === View() routing ===

func TestView_NotReady(t *testing.T) {
	app := NewApp()
	app.boot.Ready = false
	result := app.View()
	if result == "" {
		t.Error("expected non-empty output for not-ready state")
	}
}

func TestView_BootView(t *testing.T) {
	app := NewApp()
	app.boot.Ready = true
	app.width = 80
	app.height = 30
	app.view = ViewBoot
	app.boot.Step = 3

	result := app.View()
	if result == "" {
		t.Error("expected non-empty boot view")
	}
}

func TestView_HistoryView(t *testing.T) {
	app := NewApp()
	app.boot.Ready = true
	app.width = 100
	app.height = 40
	app.view = ViewHistory

	result := app.View()
	if result == "" {
		t.Error("expected non-empty history view")
	}
}

func TestView_DocsView(t *testing.T) {
	app := NewApp()
	app.boot.Ready = true
	app.width = 100
	app.height = 40
	app.view = ViewDocs
	app.docs = newDocsState(100, 40)

	result := app.View()
	if result == "" {
		t.Error("expected non-empty docs view")
	}
}

func TestView_ProjectsView(t *testing.T) {
	app := NewApp()
	app.boot.Ready = true
	app.width = 100
	app.height = 40
	app.view = ViewProjects
	app.projectNav = newProjectsState()

	result := app.View()
	if result == "" {
		t.Error("expected non-empty projects view")
	}
}

func TestView_ProtocolView(t *testing.T) {
	app := NewApp()
	app.boot.Ready = true
	app.width = 100
	app.height = 40
	app.view = ViewProtocol

	result := app.View()
	if result == "" {
		t.Error("expected non-empty protocol view")
	}
}

func TestView_MissionView(t *testing.T) {
	app := NewApp()
	app.boot.Ready = true
	app.width = 100
	app.height = 40
	app.view = ViewMission

	result := app.View()
	if result == "" {
		t.Error("expected non-empty mission view")
	}
}

// === renderHeader / renderSidebar / renderMainPanel / renderRightPanel additional ===

func TestRenderHeader_Wide(t *testing.T) {
	app := NewApp()
	app.boot.Ready = true
	app.width = 200
	app.height = 40
	app.registry = newTestRegistry()

	result := app.renderHeader("Test Title")
	if result == "" {
		t.Error("expected non-empty header")
	}
}

func TestRenderSidebar_WithFiltered(t *testing.T) {
	app := NewApp()
	app.boot.Ready = true
	app.width = 120
	app.height = 40
	app.registry = newTestRegistry()
	app.search.Filtered = []int{0, 1, 2}
	app.search.Active = true

	result := app.renderSidebar(30, 30)
	if result == "" {
		t.Error("expected non-empty sidebar with filtered agents")
	}
}

func TestRenderMainPanel_DossierView(t *testing.T) {
	app := NewApp()
	app.boot.Ready = true
	app.width = 120
	app.height = 40
	app.view = ViewDossier
	app.registry = newTestRegistry()
	app.dashboard.Selected = 0

	result := app.renderMainPanel(60, 30)
	if result == "" {
		t.Error("expected non-empty main panel in dossier view")
	}
}

func TestRenderRightPanel_Pipeline(t *testing.T) {
	app := NewApp()
	app.boot.Ready = true
	app.width = 120
	app.height = 40
	app.pipeline.State = pipeline.New("test")
	app.registry = newTestRegistry()

	result := app.renderRightPanel(30, 30)
	if result == "" {
		t.Error("expected non-empty right panel")
	}
}

// === Comms relay via Update (enter key with ">" prefix) - additional coverage ===

func TestCommsKeys_RelayPrefix_NoHistory(t *testing.T) {
	app := NewApp()
	app.view = ViewComms
	app.boot.Ready = true
	app.width = 100
	app.height = 40
	app.registry = newTestRegistry()
	app.comms = newCommsState("numbuh-1", "prompt", 80, 40)
	app.commsInput.Focus()
	app.commsInput.SetValue(">numbuh-3")

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	result := model.(App)
	// Should reset input even if relay fails (no history to relay)
	if result.commsInput.Value() != "" {
		t.Errorf("expected commsInput reset after relay, got %q", result.commsInput.Value())
	}
}

// === Dashboard key 't' for spawn hook ===

func TestDossierKeys_T_SpawnHook(t *testing.T) {
	app := NewApp()
	app.view = ViewDossier
	app.boot.Ready = true
	app.browser.Active = false
	app.terminal.Active = false
	app.registry = newTestRegistry()
	app.dashboard.Selected = 0

	_, cmd := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	if cmd == nil {
		t.Error("expected non-nil cmd for 't' (spawn hook)")
	}
}

// === Dashboard keys: L, B, V, M, F (launch tools) ===

func TestDashboardKeys_LaunchTools(t *testing.T) {
	app := NewApp()
	app.view = ViewDashboard
	app.boot.Ready = true
	app.browser.Active = false
	app.terminal.Active = false

	keys := []rune{'L', 'B', 'V', 'M', 'F'}
	for _, k := range keys {
		_, cmd := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{k}})
		if cmd == nil {
			t.Errorf("expected non-nil cmd for key '%c'", k)
		}
	}
}

// === Dashboard keys: H (history), W (docs), p (projects) ===

func TestDashboardKeys_ViewSwitching(t *testing.T) {
	app := NewApp()
	app.view = ViewDashboard
	app.boot.Ready = true
	app.browser.Active = false
	app.terminal.Active = false
	app.width = 100
	app.height = 40

	// H -> History
	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'H'}})
	result := model.(App)
	if result.view != ViewHistory {
		t.Errorf("expected ViewHistory after 'H', got %d", result.view)
	}

	// W -> Docs
	app.view = ViewDashboard
	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'W'}})
	result = model.(App)
	if result.view != ViewDocs {
		t.Errorf("expected ViewDocs after 'W', got %d", result.view)
	}

	// p -> Projects
	app.view = ViewDashboard
	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	result = model.(App)
	if result.view != ViewProjects {
		t.Errorf("expected ViewProjects after 'p', got %d", result.view)
	}
}

// === BootTick when view is not boot (no-op) ===

func TestBootTick_NonBootView(t *testing.T) {
	app := NewApp()
	app.view = ViewDashboard
	app.boot.Ready = true

	model, _ := app.Update(bootTickMsg{})
	result := model.(App)
	if result.view != ViewDashboard {
		t.Errorf("expected view unchanged, got %d", result.view)
	}
}
