package tui

import (
	"strings"
	"testing"

	"github.com/rk-senne/moonbase/internal/mux"
	"github.com/rk-senne/moonbase/internal/pipeline"
)

func TestSpecialistPaneCommand(t *testing.T) {
	got := specialistPaneCommand("numbuh-274", "check auth")
	if !strings.Contains(got, "kiro-cli chat --agent numbuh-274") {
		t.Errorf("missing agent deploy: %q", got)
	}
	if !strings.Contains(got, "'check auth'") {
		t.Errorf("task should be single-quoted: %q", got)
	}
	// No task → no trailing prompt.
	if strings.Contains(specialistPaneCommand("numbuh-0", ""), "''") {
		t.Error("empty task should not add an empty quoted arg")
	}
}

func TestShellQuote_EscapesQuotes(t *testing.T) {
	got := shellQuote("it's a test")
	// it's -> 'it'\''s a test'
	if got != `'it'\''s a test'` {
		t.Errorf("unexpected quoting: %q", got)
	}
	// A value with no specials is simply wrapped.
	if shellQuote("plain") != `'plain'` {
		t.Errorf("plain quoting wrong: %q", shellQuote("plain"))
	}
}

func TestSpecialistPaneCommands(t *testing.T) {
	triggered := []pipeline.Phase{
		{AgentName: "numbuh-274"},
		{AgentName: "numbuh-0"},
	}
	cmds := specialistPaneCommands(triggered, "add rate limiting")
	if len(cmds) != 2 {
		t.Fatalf("expected 2 commands, got %d", len(cmds))
	}
	if !strings.Contains(cmds[0], "numbuh-274") || !strings.Contains(cmds[1], "numbuh-0") {
		t.Errorf("commands not built in order: %v", cmds)
	}
}

func TestUsePanesForFanOut(t *testing.T) {
	cmuxUp := mux.Mux{Kind: mux.Cmux, Bin: "/usr/bin/cmux"} // Available + InSession
	none := mux.Mux{Kind: mux.None}

	if usePanesForFanOut(false, cmuxUp) {
		t.Error("opt-out must never use panes")
	}
	if !usePanesForFanOut(true, cmuxUp) {
		t.Error("opt-in with an available cmux session should use panes")
	}
	if usePanesForFanOut(true, none) {
		t.Error("no multiplexer → must not use panes")
	}
}
