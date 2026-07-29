//go:build integration

package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rk-senne/moonbase/internal/agents"
	"github.com/rk-senne/moonbase/internal/backend"
	"github.com/rk-senne/moonbase/internal/discovery"
	"github.com/rk-senne/moonbase/internal/pipeline"
)

// TestIntegration_MissionPipeline exercises the full mission pipeline path against
// a fake OpenAI-compatible SSE server. It verifies:
// - Phase loop executes all mandatory phases
// - Risk gate routes correctly on LOW verdict
// - Checkpoint file is written
// - Flywheel entries are logged
//
// No real network calls are made. The test isolates HOME to a temp directory.
func TestIntegration_MissionPipeline(t *testing.T) {
	// Stand up a fake OpenAI-compatible SSE server
	phaseResponses := map[int]string{
		1: "# Analysis\n\n## Acceptance Criteria\nAC-1: Feature works\n\n## Handoff\nNEXT_AGENT: numbuh-2\nRISK: LOW",
		2: "# Design\n\n## Approach\nSimple implementation.\n\n## Files\n- internal/foo.go\n\n## Handoff\nNEXT_AGENT: numbuh-3",
		3: "# Implementation\n\n## Files Updated\n- internal/foo.go\n\n## Tests\ngo test ./... — PASS\n\n## Handoff\nNEXT_AGENT: numbuh-4",
		4: "# QA Report\n\n## Verdict\nLOW\n\n## Route\n→ Numbuh 5",
		5: "# Review\n\nLGTM. Ship it.\n\n## Summary\nClean implementation.",
	}

	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++

		// Determine which phase this is based on request count
		phaseNum := requestCount
		if phaseNum > 5 {
			phaseNum = 5
		}

		response := phaseResponses[phaseNum]

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.WriteHeader(http.StatusOK)

		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("server does not support flushing")
		}

		// Send response as a single SSE chunk to preserve formatting
		data := fmt.Sprintf(`{"choices":[{"delta":{"content":"%s"}}]}`, escapeJSON(response))
		fmt.Fprintf(w, "data: %s\n\n", data)
		fmt.Fprintf(w, "data: [DONE]\n\n")
		flusher.Flush()
	}))
	defer server.Close()

	// Isolate HOME to temp directory
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("OPENAI_API_KEY", "test-key-integration")
	t.Setenv("OPENAI_BASE_URL", server.URL)
	t.Setenv("OPENAI_MODEL", "test-model")

	// Disable kiro-cli so Preferred() selects OpenAI
	t.Setenv("PATH", tmpHome+"/bin")

	// Find the real agents directory (repo-local)
	projectRoot := findProjectRootIntegration(t)
	agentsDir := filepath.Join(projectRoot, "agents")
	if _, err := os.Stat(agentsDir); os.IsNotExist(err) {
		t.Fatalf("agents directory not found at %s", agentsDir)
	}

	// Load agents from the repo
	reg := agents.NewRegistry(agentsDir)
	if err := reg.LoadSync(); err != nil {
		t.Fatalf("failed to load agents: %v", err)
	}

	// Discover project context (use a minimal temp project)
	tmpProject := filepath.Join(tmpHome, "project")
	os.MkdirAll(filepath.Join(tmpProject, ".kiro", "steering"), 0o700)
	os.WriteFile(filepath.Join(tmpProject, ".kiro", "steering", "test-rule.md"), []byte("# Test Rule\n\nBe good.\n"), 0o600)
	os.WriteFile(filepath.Join(tmpProject, "go.mod"), []byte("module test\n\ngo 1.22\n"), 0o600)

	ctx := discovery.Discover(tmpProject)
	if ctx == nil {
		t.Fatal("discovery returned nil")
	}

	// Create pipeline
	task := "integration test task"
	p := pipeline.New(task)

	// Create flywheel logger (writes to tmpHome)
	flywheelPath := filepath.Join(tmpHome, ".moonbase", "flywheel.jsonl")
	flywheel := &pipeline.FlywheelLog{}
	setFlywheelPath(flywheel, flywheelPath)

	// Verify backend selects OpenAI
	be := backend.Preferred()
	if be.Name() != "openai" {
		t.Fatalf("expected openai backend, got: %s", be.Name())
	}

	// Execute mandatory phases (1-5)
	for i := 0; i < len(p.Phases); i++ {
		phase := &p.Phases[i]

		// Skip conditional phases
		if phase.Conditional {
			phase.Status = pipeline.StatusSkipped
			continue
		}

		agent := reg.GetByName(phase.AgentName)
		if agent == nil {
			t.Fatalf("agent %s not found in registry", phase.AgentName)
		}

		phase.StartPhase()
		phaseInput := p.Context.ForPhase(phase.Number)

		// Deploy via the real OpenAI backend (hitting our fake server)
		output, err := be.Deploy(*agent, ctx, phaseInput)
		if err != nil {
			t.Fatalf("phase %d (%s) deploy failed: %v", phase.Number, phase.AgentName, err)
		}

		if output == "" {
			t.Fatalf("phase %d (%s) returned empty output", phase.Number, phase.AgentName)
		}

		p.Context.RecordPhase(phase.Number, output)
		phase.CompletePhase()

		flywheel.Append(pipeline.FlywheelEntry{
			Timestamp:   time.Now().UTC(),
			TraceID:     p.TraceID,
			Phase:       phase.Number,
			Agent:       phase.AgentName,
			Task:        task,
			Outcome:     "complete",
			DurationMs:  phase.ElapsedTime().Milliseconds(),
			OutputSize:  len(output),
			ReworkCount: p.Context.ReworkCount,
		})

		// Apply risk gate after QA (phase 4)
		if phase.Number == 4 {
			routing, rErr := p.ApplyRiskGate(output)
			if rErr != nil {
				t.Fatalf("risk gate error: %v", rErr)
			}
			if routing.Level != pipeline.RiskLow {
				t.Fatalf("expected LOW risk from QA, got: %s", routing.Level)
			}
		}

		p.Advance()
	}

	// Save checkpoint
	checkpointDir := filepath.Join(tmpHome, ".moonbase", "checkpoints")
	if err := pipeline.SaveCheckpoint(p, checkpointDir); err != nil {
		t.Fatalf("checkpoint save failed: %v", err)
	}

	// VERIFY: Checkpoint file was written
	checkpointPath := filepath.Join(checkpointDir, p.TraceID+".json")
	if _, err := os.Stat(checkpointPath); os.IsNotExist(err) {
		t.Fatalf("checkpoint file not written at %s", checkpointPath)
	}

	// VERIFY: Checkpoint content is valid
	cpData, err := os.ReadFile(checkpointPath)
	if err != nil {
		t.Fatalf("cannot read checkpoint: %v", err)
	}
	var cp pipeline.Checkpoint
	if err := json.Unmarshal(cpData, &cp); err != nil {
		t.Fatalf("checkpoint is not valid JSON: %v", err)
	}
	if cp.TraceID != p.TraceID {
		t.Errorf("checkpoint trace ID mismatch: got %s, want %s", cp.TraceID, p.TraceID)
	}
	if cp.Task != task {
		t.Errorf("checkpoint task mismatch: got %q, want %q", cp.Task, task)
	}

	// VERIFY: Flywheel entries were written
	if _, err := os.Stat(flywheelPath); os.IsNotExist(err) {
		t.Fatalf("flywheel log not written at %s", flywheelPath)
	}

	flywheelFile, err := os.Open(flywheelPath)
	if err != nil {
		t.Fatalf("cannot open flywheel log: %v", err)
	}
	defer flywheelFile.Close()

	var flywheelEntries []pipeline.FlywheelEntry
	scanner := bufio.NewScanner(flywheelFile)
	for scanner.Scan() {
		var entry pipeline.FlywheelEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			t.Fatalf("flywheel entry is not valid JSON: %v\nLine: %s", err, scanner.Text())
		}
		flywheelEntries = append(flywheelEntries, entry)
	}

	// Should have 5 entries (one per mandatory phase)
	if len(flywheelEntries) != 5 {
		t.Fatalf("expected 5 flywheel entries, got %d", len(flywheelEntries))
	}

	// Verify all entries have the same trace ID
	for i, entry := range flywheelEntries {
		if entry.TraceID != p.TraceID {
			t.Errorf("entry %d trace ID mismatch: got %s, want %s", i, entry.TraceID, p.TraceID)
		}
		if entry.Phase != i+1 {
			t.Errorf("entry %d phase: got %d, want %d", i, entry.Phase, i+1)
		}
		if entry.Outcome != "complete" {
			t.Errorf("entry %d outcome: got %s, want complete", i, entry.Outcome)
		}
	}

	// VERIFY: Pipeline completed all mandatory phases
	if requestCount < 5 {
		t.Errorf("expected at least 5 backend requests, got %d", requestCount)
	}

	t.Logf("Integration test passed: %d phases executed, checkpoint at %s, %d flywheel entries",
		requestCount, checkpointPath, len(flywheelEntries))
}

// TestIntegration_RiskGateRework exercises the rework loop by having the QA
// agent return MEDIUM risk on the first pass, triggering a rework back to Phase 3.
func TestIntegration_RiskGateRework(t *testing.T) {
	qaAttempt := 0
	requestCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++

		var response string
		// Simple heuristic: first 3 requests are phases 1-3, then QA
		switch {
		case requestCount == 1:
			response = "# Analysis\n\nAC-1: Works\n\nRISK: LOW"
		case requestCount == 2:
			response = "# Design\n\nSimple approach.\n\n## Files\n- foo.go"
		case requestCount == 3:
			response = "# Implementation v1\n\nDone."
		case requestCount == 4:
			qaAttempt++
			response = "# QA Report\n\n## Verdict\nMEDIUM\n\n## Findings\nMissing edge case test."
		case requestCount == 5:
			response = "# Implementation v2\n\nFixed edge case."
		case requestCount == 6:
			qaAttempt++
			response = "# QA Report\n\n## Verdict\nLOW\n\nAll good now."
		default:
			response = "# Review\n\nApproved."
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		flusher := w.(http.Flusher)

		// Send as single chunk for simplicity
		data := fmt.Sprintf(`{"choices":[{"delta":{"content":"%s"}}]}`, escapeJSON(response))
		fmt.Fprintf(w, "data: %s\n\n", data)
		fmt.Fprintf(w, "data: [DONE]\n\n")
		flusher.Flush()
	}))
	defer server.Close()

	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("OPENAI_API_KEY", "test-key-rework")
	t.Setenv("OPENAI_BASE_URL", server.URL)
	t.Setenv("OPENAI_MODEL", "test-model")
	t.Setenv("PATH", tmpHome+"/bin")

	be := backend.Preferred()
	if be.Name() != "openai" {
		t.Fatalf("expected openai backend, got: %s", be.Name())
	}

	projectRoot := findProjectRootIntegration(t)
	reg := agents.NewRegistry(filepath.Join(projectRoot, "agents"))
	if err := reg.LoadSync(); err != nil {
		t.Fatalf("failed to load agents: %v", err)
	}

	task := "rework integration test"
	p := pipeline.New(task)

	// Execute phases with rework loop
	for i := 0; i < len(p.Phases); i++ {
		phase := &p.Phases[i]
		if phase.Conditional {
			phase.Status = pipeline.StatusSkipped
			continue
		}

		agent := reg.GetByName(phase.AgentName)
		if agent == nil {
			t.Fatalf("agent %s not found", phase.AgentName)
		}

		phase.StartPhase()
		phaseInput := p.Context.ForPhase(phase.Number)

		output, err := be.Deploy(*agent, ctx(tmpHome), phaseInput)
		if err != nil {
			t.Fatalf("phase %d deploy failed: %v", phase.Number, err)
		}

		p.Context.RecordPhase(phase.Number, output)
		phase.CompletePhase()
		p.Advance()

		// Apply risk gate after QA
		if phase.Number == 4 {
			routing, rErr := p.ApplyRiskGate(output)
			if routing.Level == pipeline.RiskMedium || routing.Level == pipeline.RiskHigh {
				if rErr != nil {
					t.Fatalf("rework routing error: %v", rErr)
				}
				// Loop back to the target phase
				for j, ph := range p.Phases {
					if ph.Number == routing.TargetPhase {
						i = j - 1 // -1 because loop increments
						break
					}
				}
				continue
			}
		}
	}

	// Verify rework happened
	if qaAttempt < 2 {
		t.Errorf("expected at least 2 QA attempts (rework loop), got %d", qaAttempt)
	}
	if p.Context.ReworkCount < 1 {
		t.Errorf("expected rework count >= 1, got %d", p.Context.ReworkCount)
	}

	t.Logf("Rework integration test passed: %d requests, %d QA attempts, %d rework(s)",
		requestCount, qaAttempt, p.Context.ReworkCount)
}

// escapeJSON escapes a string for embedding in a JSON string value.
func escapeJSON(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	s = strings.ReplaceAll(s, "\t", `\t`)
	return s
}

// ctx returns a minimal ProjectContext for the integration test.
func ctx(tmpHome string) *discovery.ProjectContext {
	return &discovery.ProjectContext{
		RootDir: tmpHome,
		Stack:   discovery.StackInfo{Language: "go", BuildTool: "go", TestCommand: "go test ./..."},
	}
}

// setFlywheelPath sets the path on a FlywheelLog using the exported constructor pattern.
// Since FlywheelLog.path is unexported, we create a new log and override HOME.
func setFlywheelPath(fl *pipeline.FlywheelLog, path string) {
	// The FlywheelLog uses HOME internally via NewFlywheelLog().
	// Since we already set HOME to tmpHome via t.Setenv, NewFlywheelLog() will use it.
	// We just need to reassign the pointer.
	*fl = *pipeline.NewFlywheelLog()
}

// findProjectRootIntegration walks up from CWD to find the moonbase project root.
func findProjectRootIntegration(t *testing.T) string {
	t.Helper()

	// Start from the actual file location (the test is in cmd/moonbase/)
	// We need to find the repo root which has go.mod + agents/
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("cannot get working directory: %v", err)
	}

	for dir := wd; dir != "/"; dir = filepath.Dir(dir) {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			if _, err := os.Stat(filepath.Join(dir, "agents")); err == nil {
				return dir
			}
		}
	}
	t.Fatal("project root not found — ensure test runs from within moonbase repo")
	return ""
}
